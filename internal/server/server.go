package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/config"
	"github.com/gobenpark/colign/internal/database"

	"github.com/gobenpark/colign/gen/proto/auth/v1/authv1connect"
	"github.com/gobenpark/colign/gen/proto/comment/v1/commentv1connect"
	"github.com/gobenpark/colign/gen/proto/organization/v1/organizationv1connect"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	"github.com/gobenpark/colign/gen/proto/workflow/v1/workflowv1connect"
	"github.com/gobenpark/colign/internal/comment"
	"github.com/gobenpark/colign/internal/organization"
	"github.com/gobenpark/colign/internal/project"
	"github.com/gobenpark/colign/internal/workflow"
)

type Server struct {
	mux        *http.ServeMux
	db         *bun.DB
	jwtManager *auth.JWTManager
	cfg        *config.Config
}

func New(cfg *config.Config) (*Server, error) {
	db := database.New(cfg.DatabaseURL, cfg.Debug)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)

	s := &Server{
		mux:        http.NewServeMux(),
		db:         db,
		jwtManager: jwtManager,
		cfg:        cfg,
	}

	s.setupRoutes(cfg)
	return s, nil
}

func (s *Server) setupRoutes(cfg *config.Config) {
	// Health check
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Auth service (Connect)
	authService := auth.NewService(s.db, s.jwtManager)
	oauthService := auth.NewOAuthService(s.db, s.jwtManager, auth.OAuthConfig{
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		RedirectBaseURL:    cfg.RedirectBaseURL,
	})

	authConnectHandler := auth.NewConnectHandler(authService, oauthService)
	authPath, authHandler := authv1connect.NewAuthServiceHandler(authConnectHandler)
	s.mux.Handle(authPath, authHandler)

	// OAuth redirect routes (REST)
	oauthHandler := auth.NewOAuthHandler(oauthService, cfg.FrontendURL)
	s.mux.HandleFunc("GET /api/auth/{provider}", oauthHandler.Redirect)
	s.mux.HandleFunc("GET /api/auth/{provider}/callback", oauthHandler.Callback)

	// Project service (Connect)
	projectService := project.NewService(s.db)
	projectConnectHandler := project.NewConnectHandler(projectService, s.jwtManager)
	projectPath, projectHandler := projectv1connect.NewProjectServiceHandler(projectConnectHandler)
	s.mux.Handle(projectPath, projectHandler)

	// Organization service (Connect)
	orgService := organization.NewService(s.db)
	orgConnectHandler := organization.NewConnectHandler(orgService, s.jwtManager)
	orgPath, orgHandler := organizationv1connect.NewOrganizationServiceHandler(orgConnectHandler)
	s.mux.Handle(orgPath, orgHandler)

	// Workflow service (Connect)
	workflowService := workflow.NewService(s.db)
	workflowConnectHandler := workflow.NewConnectHandler(workflowService, s.db, s.jwtManager)
	workflowPath, workflowHandler := workflowv1connect.NewWorkflowServiceHandler(workflowConnectHandler)
	s.mux.Handle(workflowPath, workflowHandler)

	// Comment service (Connect)
	commentService := comment.NewService(s.db)
	commentConnectHandler := comment.NewConnectHandler(commentService, s.jwtManager)
	commentPath, commentHandler := commentv1connect.NewCommentServiceHandler(commentConnectHandler)
	s.mux.Handle(commentPath, commentHandler)
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux, s.cfg.FrontendURL)
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
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Connect-Protocol-Version")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
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
