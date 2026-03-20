package organization

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	organizationv1 "github.com/gobenpark/colign/gen/proto/organization/v1"
	"github.com/gobenpark/colign/gen/proto/organization/v1/organizationv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service    *Service
	jwtManager *auth.JWTManager
}

var _ organizationv1connect.OrganizationServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager}
}

func (h *ConnectHandler) extractClaims(header string) (*auth.Claims, error) {
	claims, err := auth.ExtractClaims(h.jwtManager, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) ListOrganizations(ctx context.Context, req *connect.Request[organizationv1.ListOrganizationsRequest]) (*connect.Response[organizationv1.ListOrganizationsResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	orgs, err := h.service.ListByUser(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoOrgs := make([]*organizationv1.Organization, len(orgs))
	for i, o := range orgs {
		protoOrgs[i] = orgToProto(&o)
	}

	return connect.NewResponse(&organizationv1.ListOrganizationsResponse{
		Organizations: protoOrgs,
		CurrentOrgId:  claims.OrgID,
	}), nil
}

func (h *ConnectHandler) SwitchOrganization(ctx context.Context, req *connect.Request[organizationv1.SwitchOrganizationRequest]) (*connect.Response[organizationv1.SwitchOrganizationResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	// Verify user is member of target org
	isMember, err := h.service.IsMember(ctx, req.Msg.OrganizationId, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !isMember {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not a member of this organization"))
	}

	org, err := h.service.GetByID(ctx, req.Msg.OrganizationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Generate new token pair with new org_id
	tokenPair, err := h.jwtManager.GenerateTokenPair(claims.UserID, claims.Email, claims.Name, req.Msg.OrganizationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.SwitchOrganizationResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		Organization: orgToProto(org),
	}), nil
}

func (h *ConnectHandler) UpdateOrganization(ctx context.Context, req *connect.Request[organizationv1.UpdateOrganizationRequest]) (*connect.Response[organizationv1.UpdateOrganizationResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	// Only allow updating the current org
	if req.Msg.Id != claims.OrgID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("can only update current organization"))
	}

	org, err := h.service.Update(ctx, req.Msg.Id, req.Msg.Name)
	if err != nil {
		if errors.Is(err, ErrOrgNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.UpdateOrganizationResponse{
		Organization: orgToProto(org),
	}), nil
}

func orgToProto(o *models.Organization) *organizationv1.Organization {
	return &organizationv1.Organization{
		Id:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		CreatedAt: timestamppb.New(o.CreatedAt),
	}
}
