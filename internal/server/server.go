package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/config"
	"github.com/gobenpark/colign/internal/database"

	eeoauth "github.com/gobenpark/colign/ee/mcp/oauth"
	"github.com/gobenpark/colign/gen/proto/acceptance/v1/acceptancev1connect"
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
	"github.com/gobenpark/colign/internal/apitoken"
	"github.com/gobenpark/colign/internal/archive"
	"github.com/gobenpark/colign/internal/comment"
	"github.com/gobenpark/colign/internal/document"
	"github.com/gobenpark/colign/internal/events"
	mcpserver "github.com/gobenpark/colign/internal/mcp"
	"github.com/gobenpark/colign/internal/memory"
	"github.com/gobenpark/colign/internal/notification"
	"github.com/gobenpark/colign/internal/organization"
	"github.com/gobenpark/colign/internal/project"
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

	s.setupRoutes(cfg)
	return s, nil
}

func (s *Server) setupRoutes(cfg *config.Config) {
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

	// API Token service (Connect) — must be created first as it serves as APITokenValidator
	apiTokenService := apitoken.NewService(s.db)
	apiTokenConnectHandler := apitoken.NewConnectHandler(apiTokenService, s.jwtManager)
	apiTokenPath, apiTokenHandler := apitokenv1connect.NewApiTokenServiceHandler(apiTokenConnectHandler)
	s.mux.Handle(apiTokenPath, apiTokenHandler)

	// Project service (Connect)
	projectService := project.NewService(s.db)
	archiveService := archive.NewService(s.db)
	cronCtx, cronCancel := context.WithCancel(context.Background())
	_ = cronCancel // TODO: call on server shutdown
	archiveService.StartCron(cronCtx)
	projectConnectHandler := project.NewConnectHandler(projectService, archiveService, s.jwtManager, apiTokenService)
	projectPath, projectHandler := projectv1connect.NewProjectServiceHandler(projectConnectHandler)
	s.mux.Handle(projectPath, projectHandler)

	// Organization service (Connect handler)
	orgConnectHandler := organization.NewConnectHandler(orgService, s.jwtManager, apiTokenService, authService)
	orgPath, orgHandler := organizationv1connect.NewOrganizationServiceHandler(orgConnectHandler)
	s.mux.Handle(orgPath, orgHandler)

	// Workflow service (Connect)
	workflowService := workflow.NewService(s.db)
	workflowConnectHandler := workflow.NewConnectHandler(workflowService, s.db, s.jwtManager, apiTokenService)
	workflowPath, workflowHandler := workflowv1connect.NewWorkflowServiceHandler(workflowConnectHandler)
	s.mux.Handle(workflowPath, workflowHandler)

	// Comment service (Connect)
	commentService := comment.NewService(s.db)
	commentConnectHandler := comment.NewConnectHandler(commentService, s.jwtManager, apiTokenService)
	commentPath, commentHandler := commentv1connect.NewCommentServiceHandler(commentConnectHandler)
	s.mux.Handle(commentPath, commentHandler)

	// Document service (Connect)
	documentService := document.NewService(s.db)
	documentConnectHandler := document.NewConnectHandler(documentService, s.jwtManager, apiTokenService)
	documentPath, documentHandler := documentv1connect.NewDocumentServiceHandler(documentConnectHandler)
	s.mux.Handle(documentPath, documentHandler)

	// Task service (Connect)
	taskService := task.NewService(s.db, task.WithArchiveEvaluator(archiveService))
	taskConnectHandler := task.NewConnectHandler(taskService, s.jwtManager, apiTokenService)
	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(taskConnectHandler)
	s.mux.Handle(taskPath, taskHandler)

	// Acceptance Criteria service (Connect)
	acService := acceptance.NewService(s.db)
	acConnectHandler := acceptance.NewConnectHandler(acService, s.jwtManager, apiTokenService)
	acPath, acHandler := acceptancev1connect.NewAcceptanceCriteriaServiceHandler(acConnectHandler)
	s.mux.Handle(acPath, acHandler)

	// Notification service (Connect)
	notifService := notification.NewService(s.db)
	notifConnectHandler := notification.NewConnectHandler(notifService, s.jwtManager, apiTokenService, s.EventHub)
	notifPath, notifHandler := notificationv1connect.NewNotificationServiceHandler(notifConnectHandler)
	s.mux.Handle(notifPath, notifHandler)

	// Memory service (Connect)
	memoryService := memory.NewService(s.db)
	memoryConnectHandler := memory.NewConnectHandler(memoryService, s.jwtManager, apiTokenService)
	memoryPath, memoryHandler := memoryv1connect.NewMemoryServiceHandler(memoryConnectHandler)
	s.mux.Handle(memoryPath, memoryHandler)

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
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(browserCookieAuthMiddleware(s.mux), s.cfg.FrontendURL)
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
		if origin == allowOrigin || strings.HasPrefix(origin, "http://localhost:") {
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
