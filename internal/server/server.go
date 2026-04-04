package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/authz"
	"github.com/gobenpark/colign/internal/config"
	"github.com/gobenpark/colign/internal/database"

	eeoauth "github.com/gobenpark/colign/ee/mcp/oauth"
	"github.com/gobenpark/colign/gen/proto/acceptance/v1/acceptancev1connect"
	aiconfigv1connect "github.com/gobenpark/colign/gen/proto/aiconfig/v1/aiconfigv1connect"
	"github.com/gobenpark/colign/gen/proto/apitoken/v1/apitokenv1connect"
	"github.com/gobenpark/colign/gen/proto/auth/v1/authv1connect"
	"github.com/gobenpark/colign/gen/proto/comment/v1/commentv1connect"
	"github.com/gobenpark/colign/gen/proto/document/v1/documentv1connect"
	"github.com/gobenpark/colign/gen/proto/memory/v1/memoryv1connect"
	"github.com/gobenpark/colign/gen/proto/notification/v1/notificationv1connect"
	"github.com/gobenpark/colign/gen/proto/organization/v1/organizationv1connect"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	taskv1connect "github.com/gobenpark/colign/gen/proto/task/v1/taskv1connect"
	"github.com/gobenpark/colign/gen/proto/workflow/v1/workflowv1connect"
	"github.com/gobenpark/colign/internal/acceptance"
	"github.com/gobenpark/colign/internal/ai"
	"github.com/gobenpark/colign/internal/aiconfig"
	"github.com/gobenpark/colign/internal/apitoken"
	"github.com/gobenpark/colign/internal/archive"
	"github.com/gobenpark/colign/internal/comment"
	"github.com/gobenpark/colign/internal/document"
	"github.com/gobenpark/colign/internal/events"
	mcpserver "github.com/gobenpark/colign/internal/mcp"
	"github.com/gobenpark/colign/internal/memory"
	"github.com/gobenpark/colign/internal/middleware"
	"github.com/gobenpark/colign/internal/notification"
	"github.com/gobenpark/colign/internal/organization"
	"github.com/gobenpark/colign/internal/project"
	"github.com/gobenpark/colign/internal/push"
	"github.com/gobenpark/colign/internal/task"
	"github.com/gobenpark/colign/internal/workflow"
)

type Server struct {
	mux        *http.ServeMux
	db         *bun.DB
	jwtManager *auth.JWTManager
	cfg        *config.Config
	EventHub   *events.Hub
}

func New(cfg *config.Config) (*Server, error) {
	db := database.New(cfg.DatabaseURL, cfg.Debug)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)
	hub := events.NewHub()

	s := &Server{
		mux:        http.NewServeMux(),
		db:         db,
		jwtManager: jwtManager,
		cfg:        cfg,
		EventHub:   hub,
	}

	if err := s.setupRoutes(cfg); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) setupRoutes(cfg *config.Config) error {
	cookieOpts := auth.BrowserSessionOptions{
		Domain: cfg.CookieDomain,
		Secure: cfg.CookieSecure,
	}
	// Liveness — process alive
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Readiness — dependencies healthy
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := s.db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "db": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "connected"})
	})

	// Organization service (created early for OrgJoiner injection)
	orgService := organization.NewService(s.db)

	// Auth service (Connect)
	authService := auth.NewService(s.db, s.jwtManager)
	authService.SetOrgJoiner(orgService)
	oauthService := auth.NewOAuthService(s.db, s.jwtManager, auth.OAuthConfig{
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		RedirectBaseURL:    cfg.RedirectBaseURL,
	}, orgService)

	authConnectHandler := auth.NewConnectHandler(authService, oauthService, cookieOpts)
	authPath, authHandler := authv1connect.NewAuthServiceHandler(authConnectHandler)
	s.mux.Handle(authPath, authHandler)

	// OAuth redirect routes (REST)
	oauthHandler := auth.NewOAuthHandler(oauthService, cfg.FrontendURL, cookieOpts)
	s.mux.HandleFunc("GET /api/auth/providers", oauthHandler.Providers)
	s.mux.HandleFunc("GET /api/auth/{provider}", oauthHandler.Redirect)
	s.mux.HandleFunc("GET /api/auth/{provider}/callback", oauthHandler.Callback)

	// Logout — clears HttpOnly session cookies
	s.mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		auth.ClearBrowserSessionCookies(w, cookieOpts)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// API Token service (Connect) — must be created first as it serves as APITokenValidator
	apiTokenService := apitoken.NewService(s.db)
	authService.SetAPITokenValidator(apiTokenService)
	apiTokenConnectHandler := apitoken.NewConnectHandler(apiTokenService, s.jwtManager)
	apiTokenPath, apiTokenHandler := apitokenv1connect.NewApiTokenServiceHandler(apiTokenConnectHandler)
	s.mux.Handle(apiTokenPath, apiTokenHandler)

	// Push service
	pushService := push.NewService(s.db, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	pushHandler := push.NewHandler(pushService, s.jwtManager, apiTokenService)
	s.mux.HandleFunc("GET /api/push/vapid-key", pushHandler.HandleVAPIDKey)
	s.mux.HandleFunc("POST /api/push/subscribe", pushHandler.HandleSubscribe)
	s.mux.HandleFunc("POST /api/push/unsubscribe", pushHandler.HandleUnsubscribe)

	// RBAC interceptor
	enforcer, err := authz.NewEnforcer()
	if err != nil {
		return fmt.Errorf("creating RBAC enforcer: %w", err)
	}
	rbacInterceptor := authz.NewRBACInterceptor(s.db, enforcer, s.jwtManager, apiTokenService)
	notifInterceptor := middleware.NewNotificationInterceptor(s.EventHub, s.jwtManager, apiTokenService)
	rbacOpts := connect.WithInterceptors(rbacInterceptor, notifInterceptor)

	// Project service (Connect)
	projectService := project.NewService(s.db)
	archiveService := archive.NewService(s.db)
	cronCtx, cronCancel := context.WithCancel(context.Background())
	_ = cronCancel // TODO: call on server shutdown
	archiveService.StartCron(cronCtx)
	projectConnectHandler := project.NewConnectHandler(projectService, archiveService, s.jwtManager, apiTokenService, s.EventHub)
	projectPath, projectHandler := projectv1connect.NewProjectServiceHandler(projectConnectHandler, rbacOpts)
	s.mux.Handle(projectPath, projectHandler)

	// Organization service (Connect handler)
	orgConnectHandler := organization.NewConnectHandler(orgService, s.jwtManager, apiTokenService, authService, cookieOpts)
	orgPath, orgHandler := organizationv1connect.NewOrganizationServiceHandler(orgConnectHandler)
	s.mux.Handle(orgPath, orgHandler)

	// Workflow service (Connect)
	workflowService := workflow.NewService(s.db)
	workflowConnectHandler := workflow.NewConnectHandler(workflowService, s.db, s.jwtManager, apiTokenService, s.EventHub)
	workflowPath, workflowHandler := workflowv1connect.NewWorkflowServiceHandler(workflowConnectHandler, rbacOpts)
	s.mux.Handle(workflowPath, workflowHandler)

	// Notification service (Connect)
	notifService := notification.NewService(s.db)
	notifConnectHandler := notification.NewConnectHandler(notifService, s.jwtManager, apiTokenService, s.EventHub)
	notifPath, notifHandler := notificationv1connect.NewNotificationServiceHandler(notifConnectHandler, rbacOpts)
	s.mux.Handle(notifPath, notifHandler)

	// Notification consumer (fan-in from EventHub)
	notifConsumer := notification.NewConsumer(s.db, notifService, pushService, s.EventHub)
	notifConsumer.Start(context.Background())

	// Comment service (Connect)
	commentService := comment.NewService(s.db, notifService)
	commentConnectHandler := comment.NewConnectHandler(commentService, s.jwtManager, apiTokenService)
	commentPath, commentHandler := commentv1connect.NewCommentServiceHandler(commentConnectHandler, rbacOpts)
	s.mux.Handle(commentPath, commentHandler)

	// Document service (Connect)
	documentService := document.NewService(s.db)
	documentConnectHandler := document.NewConnectHandler(documentService, s.jwtManager, apiTokenService, s.EventHub)
	documentPath, documentHandler := documentv1connect.NewDocumentServiceHandler(documentConnectHandler, rbacOpts)
	s.mux.Handle(documentPath, documentHandler)

	// Task service (Connect)
	taskService := task.NewService(s.db, task.WithArchiveEvaluator(archiveService))
	taskConnectHandler := task.NewConnectHandler(taskService, s.jwtManager, apiTokenService, s.EventHub)
	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(taskConnectHandler, rbacOpts)
	s.mux.Handle(taskPath, taskHandler)

	// Acceptance Criteria service (Connect)
	acService := acceptance.NewService(s.db)
	acConnectHandler := acceptance.NewConnectHandler(acService, s.jwtManager, apiTokenService)
	acPath, acHandler := acceptancev1connect.NewAcceptanceCriteriaServiceHandler(acConnectHandler, rbacOpts)
	s.mux.Handle(acPath, acHandler)

	// Memory service (Connect)
	memoryService := memory.NewService(s.db)
	memoryConnectHandler := memory.NewConnectHandler(memoryService, s.jwtManager, apiTokenService)
	memoryPath, memoryHandler := memoryv1connect.NewMemoryServiceHandler(memoryConnectHandler, rbacOpts)
	s.mux.Handle(memoryPath, memoryHandler)

	// AI Config service (Connect)
	aiConfigService := aiconfig.NewService(s.db, []byte(cfg.AIEncryptionKey))
	aiConfigConnectHandler := aiconfig.NewConnectHandler(aiConfigService, s.jwtManager, apiTokenService, ai.TestConnection)
	aiConfigPath, aiConfigHandler := aiconfigv1connect.NewAIConfigServiceHandler(aiConfigConnectHandler)
	s.mux.Handle(aiConfigPath, aiConfigHandler)

	// AI generation endpoints (REST)
	aiService := ai.NewService(aiConfigService, s.db)
	aiHandler := ai.NewHandler(aiService, aiService, s.jwtManager, aiConfigService, s.db)
	s.mux.HandleFunc("POST /api/ai/generate-proposal", aiHandler.HandleGenerateProposal)
	s.mux.HandleFunc("POST /api/ai/generate-ac", aiHandler.HandleGenerateAC)
	s.mux.HandleFunc("POST /api/ai/chat", aiHandler.HandleChat)
	s.mux.HandleFunc("POST /api/ai/execute-tool", aiHandler.HandleExecuteTool)
	activityHandler := ai.NewActivityHandler(s.db, s.jwtManager, apiTokenService)
	s.mux.HandleFunc("GET /api/changes/activities", activityHandler.HandleGetActivities)

	// Public OG metadata endpoint (no auth required)
	s.mux.HandleFunc("GET /api/og/projects/{slug}/changes/{changeId}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		changeID, err := strconv.ParseInt(r.PathValue("changeId"), 10, 64)
		if err != nil {
			http.Error(w, "invalid change id", http.StatusBadRequest)
			return
		}
		changeName, projectName, stage, err := projectService.GetChangeOG(r.Context(), slug, changeID)
		if err != nil {
			slog.Debug("og metadata not found", "slug", slug, "changeId", changeID, "error", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"changeName":  changeName,
			"projectName": projectName,
			"stage":       stage,
		})
	})

	s.mux.HandleFunc("GET /api/og/projects/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		projectName, description, err := projectService.GetProjectOG(r.Context(), slug)
		if err != nil {
			slog.Debug("og project metadata not found", "slug", slug, "error", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"projectName": projectName,
			"description": description,
		})
	})

	// MCP Streamable HTTP endpoint
	apiURL := fmt.Sprintf("http://localhost:%s", cfg.Port)
	var mcpOpts []mcpserver.ClientOption
	if cfg.HocuspocusURL != "" {
		mcpOpts = append(mcpOpts, mcpserver.WithHocuspocus(cfg.HocuspocusURL, cfg.HocuspocusAPISecret))
	}
	mcpOpts = append(mcpOpts, mcpserver.WithEventHub(s.EventHub))
	mcpHandler := mcpserver.NewStreamableHandlerWithAuth(apiURL, mcpOpts...)

	if cfg.Edition == "ee" {
		// Enterprise: OAuth discovery + 401 middleware for MCP
		mcpMiddleware := eeoauth.RegisterRoutes(s.mux, s.db, s.jwtManager, apiTokenService, cfg.RedirectBaseURL)
		s.mux.Handle("/mcp", mcpMiddleware(mcpHandler))
	} else {
		// Community: Bearer token only
		s.mux.Handle("/mcp", mcpHandler)
	}
	return nil
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(corsMiddleware(browserCookieAuthMiddleware(s.mux), s.cfg.FrontendURL))
}

func (s *Server) Close() error {
	return s.db.Close()
}

func corsMiddleware(next http.Handler, allowOrigin string) http.Handler {
	if allowOrigin == "" {
		allowOrigin = "http://localhost:3000"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Connect-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Mcp-Session-Id")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", int((12*time.Hour).Seconds())))
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func browserCookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if accessToken, _ := auth.BrowserSessionFromRequest(r); accessToken != "" {
				r.Header.Set("Authorization", "Bearer "+accessToken)
			}
		}
		next.ServeHTTP(w, r)
	})
}
