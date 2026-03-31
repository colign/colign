package ai

// This file is placed in the ai package temporarily for convenience.
// It provides a change activity timeline endpoint.

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/auth"
)

// ActivityHandler serves the unified activity timeline for a change.
type ActivityHandler struct {
	db                *bun.DB
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(db *bun.DB, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *ActivityHandler {
	return &ActivityHandler{db: db, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

type activityItem struct {
	Type      string    `json:"type" bun:"type"`
	Title     string    `json:"title" bun:"title"`
	UserID    int64     `json:"userId" bun:"user_id"`
	UserName  string    `json:"userName" bun:"user_name"`
	CreatedAt time.Time `json:"createdAt" bun:"created_at"`
	Detail    string    `json:"detail,omitempty" bun:"detail"`
}

// HandleGetActivities returns a unified activity timeline for a change.
func (h *ActivityHandler) HandleGetActivities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	changeIDStr := r.URL.Query().Get("changeId")
	changeID, err := strconv.ParseInt(changeIDStr, 10, 64)
	if err != nil || changeID <= 0 {
		http.Error(w, `{"error":"invalid changeId"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership
	var exists bool
	exists, err = h.db.NewSelect().
		TableExpr("changes c").
		Join("JOIN projects p ON p.id = c.project_id").
		Where("c.id = ?", changeID).
		Where("p.organization_id = ?", claims.OrgID).
		Exists(ctx)
	if err != nil || !exists {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var items []activityItem
	err = h.db.NewRaw(`
		SELECT type, title, user_id, user_name, created_at, detail FROM (
			SELECT
				'stage' AS type,
				we.action AS title,
				we.user_id,
				COALESCE(u.name, '') AS user_name,
				we.created_at AS created_at,
				we.from_stage || ' → ' || we.to_stage AS detail
			FROM colign.workflow_events we
			LEFT JOIN colign.users u ON u.id = we.user_id
			WHERE we.change_id = ?

			UNION ALL

			SELECT
				'task_created' AS type,
				t.title,
				COALESCE(t.creator_id, 0),
				COALESCE(u.name, '') AS user_name,
				t.created_at AS created_at,
				'' AS detail
			FROM colign.tasks t
			LEFT JOIN colign.users u ON u.id = t.creator_id
			WHERE t.change_id = ?

			UNION ALL

			SELECT
				'doc_' || d.type AS type,
				CASE d.type WHEN 'proposal' THEN 'Proposal updated' WHEN 'spec' THEN 'Spec updated' ELSE d.title END,
				COALESCE(d.updated_by, 0),
				COALESCE(u.name, '') AS user_name,
				d.updated_at AS created_at,
				'' AS detail
			FROM colign.documents d
			LEFT JOIN colign.users u ON u.id = d.updated_by
			WHERE d.change_id = ?

			UNION ALL

			SELECT
				'ac_created' AS type,
				ac.scenario,
				COALESCE(ac.created_by, 0),
				COALESCE(u.name, '') AS user_name,
				ac.created_at AS created_at,
				'' AS detail
			FROM colign.acceptance_criteria ac
			LEFT JOIN colign.users u ON u.id = ac.created_by
			WHERE ac.change_id = ?

			UNION ALL

			SELECT
				'comment' AS type,
				LEFT(c.body, 80) AS title,
				c.user_id,
				COALESCE(u.name, '') AS user_name,
				c.created_at AS created_at,
				'' AS detail
			FROM colign.comments c
			LEFT JOIN colign.users u ON u.id = c.user_id
			WHERE c.change_id = ?
		) AS activities
		ORDER BY created_at DESC
		LIMIT 50
	`, changeID, changeID, changeID, changeID, changeID).Scan(ctx, &items)

	if err != nil {
		slog.ErrorContext(ctx, "activity: query failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.ErrorContext(ctx, "activity: encode failed", slog.String("error", err.Error()))
	}
}
