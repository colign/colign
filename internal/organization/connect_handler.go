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

// OrgSwitcher generates a new session scoped to a different organization.
type OrgSwitcher interface {
	SwitchOrg(ctx context.Context, userID int64, email, name string, newOrgID int64) (*auth.TokenPair, error)
}

type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
	orgSwitcher       OrgSwitcher
}

var _ organizationv1connect.OrganizationServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator, orgSwitcher OrgSwitcher) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator, orgSwitcher: orgSwitcher}
}

func (h *ConnectHandler) extractClaims(ctx context.Context, header string) (*auth.Claims, error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) ListOrganizations(ctx context.Context, req *connect.Request[organizationv1.ListOrganizationsRequest]) (*connect.Response[organizationv1.ListOrganizationsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
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
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
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

	// Create new session scoped to the target org
	tokenPair, err := h.orgSwitcher.SwitchOrg(ctx, claims.UserID, claims.Email, claims.Name, req.Msg.OrganizationId)
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

func (h *ConnectHandler) CreateOrganization(ctx context.Context, req *connect.Request[organizationv1.CreateOrganizationRequest]) (*connect.Response[organizationv1.CreateOrganizationResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	org, err := h.service.Create(ctx, claims.UserID, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&organizationv1.CreateOrganizationResponse{
		Organization: orgToProto(org),
	}), nil
}

func (h *ConnectHandler) UpdateOrganization(ctx context.Context, req *connect.Request[organizationv1.UpdateOrganizationRequest]) (*connect.Response[organizationv1.UpdateOrganizationResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
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

func (h *ConnectHandler) ListMembers(ctx context.Context, req *connect.Request[organizationv1.ListMembersRequest]) (*connect.Response[organizationv1.ListMembersResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	members, err := h.service.ListMembers(ctx, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoMembers := make([]*organizationv1.OrganizationMember, len(members))
	for i, m := range members {
		protoMembers[i] = memberToProto(&m)
	}

	return connect.NewResponse(&organizationv1.ListMembersResponse{
		Members: protoMembers,
	}), nil
}

func (h *ConnectHandler) InviteOrgMember(ctx context.Context, req *connect.Request[organizationv1.InviteOrgMemberRequest]) (*connect.Response[organizationv1.InviteOrgMemberResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	role := models.OrgRole(req.Msg.Role)
	if role == "" {
		role = "member"
	}

	invitation, err := h.service.InviteMember(ctx, claims.OrgID, claims.UserID, req.Msg.Email, role)
	if err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.InviteOrgMemberResponse{
		Invitation: invitationToProto(invitation),
	}), nil
}

func (h *ConnectHandler) AcceptInvitation(ctx context.Context, req *connect.Request[organizationv1.AcceptInvitationRequest]) (*connect.Response[organizationv1.AcceptInvitationResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	member, err := h.service.AcceptInvitation(ctx, req.Msg.Token, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.AcceptInvitationResponse{
		Member: memberToProto(member),
	}), nil
}

func (h *ConnectHandler) GetInvitation(ctx context.Context, req *connect.Request[organizationv1.GetInvitationRequest]) (*connect.Response[organizationv1.GetInvitationResponse], error) {
	invitation, err := h.service.GetInvitationByToken(ctx, req.Msg.Token)
	if err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.GetInvitationResponse{
		Invitation: invitationToProto(invitation),
	}), nil
}

func (h *ConnectHandler) ListInvitations(ctx context.Context, req *connect.Request[organizationv1.ListInvitationsRequest]) (*connect.Response[organizationv1.ListInvitationsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	invitations, err := h.service.ListInvitations(ctx, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoInvitations := make([]*organizationv1.OrgInvitation, len(invitations))
	for i, inv := range invitations {
		protoInvitations[i] = invitationToProto(&inv)
	}

	return connect.NewResponse(&organizationv1.ListInvitationsResponse{
		Invitations: protoInvitations,
	}), nil
}

func (h *ConnectHandler) RevokeInvitation(ctx context.Context, req *connect.Request[organizationv1.RevokeInvitationRequest]) (*connect.Response[organizationv1.RevokeInvitationResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.RevokeInvitation(ctx, claims.OrgID, req.Msg.InvitationId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.RevokeInvitationResponse{}), nil
}

func (h *ConnectHandler) SetAllowedDomains(ctx context.Context, req *connect.Request[organizationv1.SetAllowedDomainsRequest]) (*connect.Response[organizationv1.SetAllowedDomainsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	org, err := h.service.SetAllowedDomains(ctx, claims.OrgID, req.Msg.Domains)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.SetAllowedDomainsResponse{
		Organization: orgToProto(org),
	}), nil
}

func (h *ConnectHandler) RemoveOrgMember(ctx context.Context, req *connect.Request[organizationv1.RemoveOrgMemberRequest]) (*connect.Response[organizationv1.RemoveOrgMemberResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if req.Msg.UserId == claims.UserID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot remove yourself"))
	}

	if err := h.service.RemoveMember(ctx, claims.OrgID, req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.RemoveOrgMemberResponse{}), nil
}

func (h *ConnectHandler) UpdateOrgMemberRole(ctx context.Context, req *connect.Request[organizationv1.UpdateOrgMemberRoleRequest]) (*connect.Response[organizationv1.UpdateOrgMemberRoleResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	member, err := h.service.UpdateMemberRole(ctx, claims.OrgID, req.Msg.UserId, models.OrgRole(req.Msg.Role))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.UpdateOrgMemberRoleResponse{
		Member: memberToProto(member),
	}), nil
}

func (h *ConnectHandler) DeleteOrganization(ctx context.Context, req *connect.Request[organizationv1.DeleteOrganizationRequest]) (*connect.Response[organizationv1.DeleteOrganizationResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	// Verify request org_id matches JWT org_id
	if req.Msg.OrganizationId != claims.OrgID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("can only delete the current organization"))
	}

	nextOrgID, err := h.service.Delete(ctx, claims.OrgID, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrNotOwner) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if errors.Is(err, ErrLastOrganization) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if errors.Is(err, ErrOrgNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&organizationv1.DeleteOrganizationResponse{
		NextOrganizationId: nextOrgID,
	}), nil
}

func memberToProto(m *models.OrganizationMember) *organizationv1.OrganizationMember {
	proto := &organizationv1.OrganizationMember{
		Id:             m.ID,
		OrganizationId: m.OrganizationID,
		UserId:         m.UserID,
		Role:           string(m.Role),
	}
	if m.User != nil {
		proto.UserName = m.User.Name
		proto.UserEmail = m.User.Email
	}
	return proto
}

func orgToProto(o *models.Organization) *organizationv1.Organization {
	return &organizationv1.Organization{
		Id:             o.ID,
		Name:           o.Name,
		Slug:           o.Slug,
		AllowedDomains: o.AllowedDomains,
		CreatedAt:      timestamppb.New(o.CreatedAt),
	}
}

func invitationToProto(inv *models.OrgInvitation) *organizationv1.OrgInvitation {
	proto := &organizationv1.OrgInvitation{
		Id:             inv.ID,
		OrganizationId: inv.OrganizationID,
		Email:          inv.Email,
		Role:           string(inv.Role),
		Token:          inv.Token,
		Status:         string(inv.Status),
		ExpiresAt:      timestamppb.New(inv.ExpiresAt),
		CreatedAt:      timestamppb.New(inv.CreatedAt),
	}
	if inv.Organization != nil {
		proto.Organization = orgToProto(inv.Organization)
	}
	return proto
}
