package authz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/casbin/casbin/v2"
	"github.com/uptrace/bun"

	commentv1 "github.com/gobenpark/colign/gen/proto/comment/v1"
	"github.com/gobenpark/colign/internal/auth"
)

// ProjectScoped is implemented by protobuf Request messages that include a project_id field.
type ProjectScoped interface {
	GetProjectId() int64
}

// RBACInterceptor enforces role-based access control on Connect RPCs using Casbin.
type RBACInterceptor struct {
	db                *bun.DB
	enforcer          *casbin.Enforcer
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

// NewRBACInterceptor creates a new RBAC interceptor.
func NewRBACInterceptor(db *bun.DB, enforcer *casbin.Enforcer, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *RBACInterceptor {
	return &RBACInterceptor{db: db, enforcer: enforcer, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

// WrapUnary implements connect.Interceptor.
func (i *RBACInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure

		// Skip RPCs that don't need RBAC
		if IsSkipped(procedure) {
			return next(ctx, req)
		}

		// Look up the auth rule
		rule, ok := GetRule(procedure)
		if !ok {
			slog.Warn("unmapped RPC denied", "procedure", procedure)
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
		}

		projectID, err := i.resolveProjectID(ctx, req.Any())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		// Authenticate: resolve user from Authorization header
		header := req.Header().Get("Authorization")
		claims, err := auth.ResolveFromHeader(i.jwtManager, i.apiTokenValidator, ctx, header)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}

		// Look up user's role in the project
		role, err := i.lookupRole(ctx, projectID, claims.UserID)
		if err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not a project member"))
		}

		// Enforce with Casbin
		allowed, err := i.enforcer.Enforce(role, rule.Resource, rule.Action)
		if err != nil {
			slog.Error("casbin enforcement error", "error", err, "procedure", procedure)
			return nil, connect.NewError(connect.CodeInternal, errors.New("authorization error"))
		}

		if !allowed {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("insufficient permissions"))
		}

		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for streaming).
func (i *RBACInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor (no-op for streaming).
func (i *RBACInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// lookupRole queries project_members to find the user's role in the project.
func (i *RBACInterceptor) lookupRole(ctx context.Context, projectID, userID int64) (string, error) {
	var role string
	err := i.db.NewSelect().
		TableExpr("project_members").
		ColumnExpr("role").
		Where("project_id = ?", projectID).
		Where("user_id = ?", userID).
		Scan(ctx, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("not a member")
		}
		return "", err
	}
	return role, nil
}

func (i *RBACInterceptor) resolveProjectID(ctx context.Context, req any) (int64, error) {
	switch msg := req.(type) {
	case *commentv1.CreateCommentRequest:
		return i.lookupProjectIDByChange(ctx, msg.GetChangeId())
	case *commentv1.ListCommentsRequest:
		return i.lookupProjectIDByChange(ctx, msg.GetChangeId())
	case *commentv1.ResolveCommentRequest:
		return i.lookupProjectIDByComment(ctx, msg.GetCommentId())
	case *commentv1.DeleteCommentRequest:
		return i.lookupProjectIDByComment(ctx, msg.GetCommentId())
	case *commentv1.CreateReplyRequest:
		return i.lookupProjectIDByComment(ctx, msg.GetCommentId())
	}

	scoped, ok := req.(ProjectScoped)
	if !ok {
		return 0, errors.New("missing project context")
	}
	projectID := scoped.GetProjectId()
	if projectID == 0 {
		return 0, errors.New("project_id is required")
	}
	return projectID, nil
}

func (i *RBACInterceptor) lookupProjectIDByChange(ctx context.Context, changeID int64) (int64, error) {
	if changeID == 0 {
		return 0, errors.New("change_id is required")
	}

	var projectID int64
	err := i.db.NewSelect().
		TableExpr("changes").
		Column("project_id").
		Where("id = ?", changeID).
		Scan(ctx, &projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("change not found")
		}
		return 0, err
	}
	return projectID, nil
}

func (i *RBACInterceptor) lookupProjectIDByComment(ctx context.Context, commentID int64) (int64, error) {
	if commentID == 0 {
		return 0, errors.New("comment_id is required")
	}

	var projectID int64
	err := i.db.NewSelect().
		TableExpr("comments AS c").
		Join("JOIN changes AS ch ON ch.id = c.change_id").
		ColumnExpr("ch.project_id").
		Where("c.id = ?", commentID).
		Scan(ctx, &projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("comment not found")
		}
		return 0, err
	}
	return projectID, nil
}
