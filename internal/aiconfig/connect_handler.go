package aiconfig

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	aiconfigv1 "github.com/gobenpark/colign/gen/proto/aiconfig/v1"
	"github.com/gobenpark/colign/gen/proto/aiconfig/v1/aiconfigv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

// ConnectHandler implements the AIConfigService ConnectRPC handler.
type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
	testConnFunc      func(ctx context.Context, provider, model, apiKey string) error
}

var _ aiconfigv1connect.AIConfigServiceHandler = (*ConnectHandler)(nil)

// NewConnectHandler creates a new ConnectHandler.
func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator, testConnFunc func(ctx context.Context, provider, model, apiKey string) error) *ConnectHandler {
	return &ConnectHandler{
		service:           service,
		jwtManager:        jwtManager,
		apiTokenValidator: apiTokenValidator,
		testConnFunc:      testConnFunc,
	}
}

// GetAIConfig returns the AI configuration for a project.
func (h *ConnectHandler) GetAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.GetAIConfigRequest]) (*connect.Response[aiconfigv1.GetAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.verifyProjectOwnership(ctx, req.Msg.ProjectId, claims.OrgID); err != nil {
		return nil, err
	}

	cfg, err := h.service.GetByProjectID(ctx, req.Msg.ProjectId)
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: GetByProjectID failed",
			slog.Int64("project_id", req.Msg.ProjectId),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &aiconfigv1.GetAIConfigResponse{}
	if cfg != nil {
		decrypted, decErr := h.service.DecryptAPIKey(cfg)
		if decErr != nil {
			slog.ErrorContext(ctx, "aiconfig: decrypt api key failed",
				slog.Int64("project_id", req.Msg.ProjectId),
				slog.String("error", decErr.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, decErr)
		}
		resp.Config = configToProto(cfg, decrypted)
	}

	return connect.NewResponse(resp), nil
}

// SaveAIConfig creates or updates the AI configuration for a project.
func (h *ConnectHandler) SaveAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.SaveAIConfigRequest]) (*connect.Response[aiconfigv1.SaveAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.verifyProjectOwnership(ctx, req.Msg.ProjectId, claims.OrgID); err != nil {
		return nil, err
	}

	cfg, err := h.service.Upsert(ctx, req.Msg.ProjectId, UpsertInput{
		Provider:              req.Msg.Provider,
		Model:                 req.Msg.Model,
		APIKey:                req.Msg.ApiKey,
		IncludeProjectContext: req.Msg.IncludeProjectContext,
	})
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: Upsert failed",
			slog.Int64("project_id", req.Msg.ProjectId),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	decrypted, decErr := h.service.DecryptAPIKey(cfg)
	if decErr != nil {
		slog.ErrorContext(ctx, "aiconfig: decrypt api key after upsert failed",
			slog.Int64("project_id", req.Msg.ProjectId),
			slog.String("error", decErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, decErr)
	}

	return connect.NewResponse(&aiconfigv1.SaveAIConfigResponse{
		Config: configToProto(cfg, decrypted),
	}), nil
}

// TestConnection verifies that the provided AI provider credentials are valid.
func (h *ConnectHandler) TestConnection(ctx context.Context, req *connect.Request[aiconfigv1.TestConnectionRequest]) (*connect.Response[aiconfigv1.TestConnectionResponse], error) {
	if h.testConnFunc == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("test connection not configured"))
	}
	err := h.testConnFunc(ctx, req.Msg.Provider, req.Msg.Model, req.Msg.ApiKey)
	if err != nil {
		return connect.NewResponse(&aiconfigv1.TestConnectionResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}
	return connect.NewResponse(&aiconfigv1.TestConnectionResponse{
		Success: true,
	}), nil
}

// DeleteAIConfig removes the AI configuration for a project.
func (h *ConnectHandler) DeleteAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.DeleteAIConfigRequest]) (*connect.Response[aiconfigv1.DeleteAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.verifyProjectOwnership(ctx, req.Msg.ProjectId, claims.OrgID); err != nil {
		return nil, err
	}

	if err := h.service.Delete(ctx, req.Msg.ProjectId); err != nil {
		slog.ErrorContext(ctx, "aiconfig: Delete failed",
			slog.Int64("project_id", req.Msg.ProjectId),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&aiconfigv1.DeleteAIConfigResponse{}), nil
}

// GetOrgAIConfig returns the AI configuration for the caller's organization.
func (h *ConnectHandler) GetOrgAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.GetOrgAIConfigRequest]) (*connect.Response[aiconfigv1.GetOrgAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	cfg, err := h.service.GetByOrgID(ctx, claims.OrgID)
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: GetByOrgID failed",
			slog.Int64("org_id", claims.OrgID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &aiconfigv1.GetOrgAIConfigResponse{}
	if cfg != nil {
		decrypted, decErr := h.service.DecryptOrgAPIKey(cfg)
		if decErr != nil {
			slog.ErrorContext(ctx, "aiconfig: decrypt org api key failed",
				slog.Int64("org_id", claims.OrgID),
				slog.String("error", decErr.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, decErr)
		}
		resp.Config = orgConfigToProto(cfg, decrypted)
	}

	return connect.NewResponse(resp), nil
}

// SaveOrgAIConfig creates or updates the AI configuration for the caller's organization.
func (h *ConnectHandler) SaveOrgAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.SaveOrgAIConfigRequest]) (*connect.Response[aiconfigv1.SaveOrgAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	cfg, err := h.service.UpsertOrg(ctx, claims.OrgID, OrgUpsertInput{
		Provider: req.Msg.Provider,
		Model:    req.Msg.Model,
		APIKey:   req.Msg.ApiKey,
	})
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: UpsertOrg failed",
			slog.Int64("org_id", claims.OrgID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	decrypted, decErr := h.service.DecryptOrgAPIKey(cfg)
	if decErr != nil {
		slog.ErrorContext(ctx, "aiconfig: decrypt org api key after upsert failed",
			slog.Int64("org_id", claims.OrgID),
			slog.String("error", decErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, decErr)
	}

	return connect.NewResponse(&aiconfigv1.SaveOrgAIConfigResponse{
		Config: orgConfigToProto(cfg, decrypted),
	}), nil
}

// DeleteOrgAIConfig removes the AI configuration for the caller's organization.
func (h *ConnectHandler) DeleteOrgAIConfig(ctx context.Context, req *connect.Request[aiconfigv1.DeleteOrgAIConfigRequest]) (*connect.Response[aiconfigv1.DeleteOrgAIConfigResponse], error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.service.DeleteOrg(ctx, claims.OrgID); err != nil {
		slog.ErrorContext(ctx, "aiconfig: DeleteOrg failed",
			slog.Int64("org_id", claims.OrgID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&aiconfigv1.DeleteOrgAIConfigResponse{}), nil
}

// orgConfigToProto converts an OrgAIConfig model to a proto message.
func orgConfigToProto(cfg *OrgAIConfig, decryptedKey string) *aiconfigv1.OrgAIConfigProto {
	return &aiconfigv1.OrgAIConfigProto{
		Id:         cfg.ID,
		OrgId:      cfg.OrgID,
		Provider:   cfg.Provider,
		Model:      cfg.Model,
		ApiKeyMasked: MaskAPIKey(decryptedKey),
		CreatedAt:  timestamppb.New(cfg.CreatedAt),
		UpdatedAt:  timestamppb.New(cfg.UpdatedAt),
	}
}

// verifyProjectOwnership checks that the given project belongs to the given org.
// Returns a connect.CodeNotFound error if the project does not exist in the org.
func (h *ConnectHandler) verifyProjectOwnership(ctx context.Context, projectID, orgID int64) error {
	exists, err := h.service.db.NewSelect().
		Model((*models.Project)(nil)).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: project ownership check failed",
			slog.Int64("project_id", projectID),
			slog.Int64("org_id", orgID),
			slog.String("error", err.Error()),
		)
		return connect.NewError(connect.CodeInternal, err)
	}
	if !exists {
		return connect.NewError(connect.CodeNotFound, errors.New("project not found"))
	}
	return nil
}

// configToProto converts an AIConfig model to a proto message.
// decryptedKey is the plaintext API key (used for masking display only).
func configToProto(cfg *AIConfig, decryptedKey string) *aiconfigv1.AIConfigProto {
	return &aiconfigv1.AIConfigProto{
		Id:                    cfg.ID,
		ProjectId:             cfg.ProjectID,
		Provider:              cfg.Provider,
		Model:                 cfg.Model,
		ApiKeyMasked:          MaskAPIKey(decryptedKey),
		IncludeProjectContext: cfg.IncludeProjectContext,
		CreatedAt:             timestamppb.New(cfg.CreatedAt),
		UpdatedAt:             timestamppb.New(cfg.UpdatedAt),
	}
}
