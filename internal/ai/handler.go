package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/aiconfig"
	"github.com/gobenpark/colign/internal/auth"
)

// Sentinel errors used by resolveAIConfig and writeAIError.
var (
	errUnauthenticated = errors.New("unauthenticated")
	errRateLimited     = errors.New("rate_limited")
	errBadRequest      = errors.New("bad_request")
	errNotFound        = errors.New("not_found")
	errAINotConfigured = errors.New("ai_not_configured")
)

// handler interfaces — defined by the consumer (Go idiom).
//
//go:generate mockgen -source=handler.go -destination=mock_handler_test.go -package=ai
type proposalGenerator interface {
	GenerateProposal(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateProposalInput) (<-chan SectionChunk, error)
}

type acGenerator interface {
	GenerateAC(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateACInput) ([]GeneratedAC, error)
}

// Handler serves HTTP endpoints for AI generation.
type Handler struct {
	proposalGen proposalGenerator
	acGen       acGenerator
	jwtManager  *auth.JWTManager
	configSvc   *aiconfig.Service
	db          *bun.DB
	limiter     *OrgRateLimiter
}

// NewHandler creates a new Handler.
func NewHandler(proposalGen proposalGenerator, acGen acGenerator, jwtManager *auth.JWTManager, configSvc *aiconfig.Service, db *bun.DB) *Handler {
	return &Handler{
		proposalGen: proposalGen,
		acGen:       acGen,
		jwtManager:  jwtManager,
		configSvc:   configSvc,
		db:          db,
		limiter:     NewOrgRateLimiter(),
	}
}

type generateRequest struct {
	ChangeID    int64  `json:"changeId"`
	Description string `json:"description,omitempty"` // for proposal
}

func (h *Handler) resolveAIConfig(r *http.Request) (*aiconfig.AIConfig, *auth.Claims, *generateRequest, error) {
	ctx := r.Context()

	// 1. Auth
	claims, err := auth.ResolveFromHeader(h.jwtManager, nil, ctx, r.Header.Get("Authorization"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", errUnauthenticated, err)
	}

	// 2. Rate limit
	if !h.limiter.Allow(claims.OrgID) {
		return nil, nil, nil, errRateLimited
	}

	// 3. Parse request body
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", errBadRequest, err)
	}

	// 4. changeId → projectId → verify org ownership
	var projectID int64
	err = h.db.NewSelect().
		ColumnExpr("p.id").
		TableExpr("changes c").
		Join("JOIN projects p ON p.id = c.project_id").
		Where("c.id = ?", req.ChangeID).
		Where("p.organization_id = ?", claims.OrgID).
		Scan(ctx, &projectID)
	if err != nil {
		return nil, nil, nil, errNotFound
	}

	// 5. Load AI config (project-level first, then org-level fallback)
	cfg, err := h.configSvc.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("internal: %w", err)
	}
	if cfg == nil {
		// Fallback to org-level config
		orgCfg, orgErr := h.configSvc.GetByOrgID(ctx, claims.OrgID)
		if orgErr != nil {
			return nil, nil, nil, fmt.Errorf("internal: %w", orgErr)
		}
		if orgCfg == nil {
			return nil, nil, nil, errAINotConfigured
		}
		// Convert org config to project config shape for downstream consumers
		cfg = &aiconfig.AIConfig{
			ID:              orgCfg.ID,
			Provider:        orgCfg.Provider,
			Model:           orgCfg.Model,
			APIKeyEncrypted: orgCfg.APIKeyEncrypted,
			KeyVersion:      orgCfg.KeyVersion,
			CreatedAt:       orgCfg.CreatedAt,
			UpdatedAt:       orgCfg.UpdatedAt,
		}
	}

	return cfg, claims, &req, nil
}

// HandleGenerateProposal streams an SSE response with proposal section chunks.
func (h *Handler) HandleGenerateProposal(w http.ResponseWriter, r *http.Request) {
	cfg, _, req, err := h.resolveAIConfig(r)
	if err != nil {
		writeAIError(w, err)
		return
	}

	writeSSEProposal(w, r, h.proposalGen, cfg, GenerateProposalInput{
		Description: req.Description,
	})
}

// HandleGenerateAC returns a JSON response with generated acceptance criteria.
func (h *Handler) HandleGenerateAC(w http.ResponseWriter, r *http.Request) {
	cfg, _, req, err := h.resolveAIConfig(r)
	if err != nil {
		writeAIError(w, err)
		return
	}

	// Fetch proposal document from DB
	var proposalContent string
	err = h.db.NewSelect().
		ColumnExpr("content").
		TableExpr("documents").
		Where("change_id = ?", req.ChangeID).
		Where("type = ?", "proposal").
		Scan(r.Context(), &proposalContent)
	if err != nil {
		slog.ErrorContext(r.Context(), "ai: fetch proposal failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"proposal not found"}`, http.StatusNotFound)
		return
	}

	writeACJSON(w, r, h.acGen, cfg, GenerateACInput{
		Proposal: proposalContent,
	})
}

// writeSSEProposal is the testable SSE writer, separated from HTTP handler wiring.
func writeSSEProposal(w http.ResponseWriter, r *http.Request, gen proposalGenerator, cfg *aiconfig.AIConfig, input GenerateProposalInput) {
	ch, err := gen.GenerateProposal(r.Context(), cfg, input)
	if err != nil {
		slog.ErrorContext(r.Context(), "ai: generate proposal failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for chunk := range ch {
		data, err := json.Marshal(chunk)
		if err != nil {
			slog.ErrorContext(r.Context(), "ai: marshal chunk failed", slog.String("error", err.Error()))
			continue
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return // client disconnected
		}
		flusher.Flush()
	}

	if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
		return // client disconnected
	}
	flusher.Flush()
}

// writeACJSON is the testable JSON writer, separated from HTTP handler wiring.
func writeACJSON(w http.ResponseWriter, r *http.Request, gen acGenerator, cfg *aiconfig.AIConfig, input GenerateACInput) {
	acs, err := gen.GenerateAC(r.Context(), cfg, input)
	if err != nil {
		slog.ErrorContext(r.Context(), "ai: generate ac failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(acs); err != nil {
		slog.ErrorContext(r.Context(), "ai: encode ac response failed", slog.String("error", err.Error()))
	}
}

// writeAIError maps sentinel errors to HTTP status codes.
func writeAIError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unauthenticated"):
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	case strings.Contains(msg, "rate_limited"):
		http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
	case strings.Contains(msg, "bad_request"):
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
	case strings.Contains(msg, "not_found"):
		http.Error(w, `{"error":"change not found"}`, http.StatusNotFound)
	case strings.Contains(msg, "ai_not_configured"):
		http.Error(w, `{"error":"AI not configured for this project"}`, http.StatusPreconditionFailed)
	default:
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
	}
}
