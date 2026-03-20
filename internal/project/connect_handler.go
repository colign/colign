package project

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	projectv1 "github.com/gobenpark/colign/gen/proto/project/v1"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service    *Service
	jwtManager *auth.JWTManager
}

var _ projectv1connect.ProjectServiceHandler = (*ConnectHandler)(nil)

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

func (h *ConnectHandler) CreateProject(ctx context.Context, req *connect.Request[projectv1.CreateProjectRequest]) (*connect.Response[projectv1.CreateProjectResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	project, err := h.service.Create(ctx, CreateProjectInput{
		Name:           req.Msg.Name,
		Description:    req.Msg.Description,
		UserID:         claims.UserID,
		OrganizationID: claims.OrgID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.CreateProjectResponse{
		Project: projectToProto(project),
	}), nil
}

func (h *ConnectHandler) GetProject(ctx context.Context, req *connect.Request[projectv1.GetProjectRequest]) (*connect.Response[projectv1.GetProjectResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	project, members, err := h.service.GetBySlug(ctx, req.Msg.Slug, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoMembers := make([]*projectv1.ProjectMember, len(members))
	for i, m := range members {
		protoMembers[i] = memberToProto(&m)
	}

	return connect.NewResponse(&projectv1.GetProjectResponse{
		Project: projectToProto(project),
		Members: protoMembers,
	}), nil
}

func (h *ConnectHandler) ListProjects(ctx context.Context, req *connect.Request[projectv1.ListProjectsRequest]) (*connect.Response[projectv1.ListProjectsResponse], error) {
	claims, err := h.extractClaims(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	projects, err := h.service.ListByUser(ctx, claims.UserID, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoProjects := make([]*projectv1.Project, len(projects))
	for i, p := range projects {
		protoProjects[i] = projectToProto(&p)
	}

	return connect.NewResponse(&projectv1.ListProjectsResponse{
		Projects: protoProjects,
	}), nil
}

func (h *ConnectHandler) UpdateProject(ctx context.Context, req *connect.Request[projectv1.UpdateProjectRequest]) (*connect.Response[projectv1.UpdateProjectResponse], error) {
	project, err := h.service.Update(ctx, UpdateProjectInput{
		ID:          req.Msg.Id,
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
	})
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.UpdateProjectResponse{
		Project: projectToProto(project),
	}), nil
}

func (h *ConnectHandler) DeleteProject(ctx context.Context, req *connect.Request[projectv1.DeleteProjectRequest]) (*connect.Response[projectv1.DeleteProjectResponse], error) {
	if err := h.service.Delete(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.DeleteProjectResponse{}), nil
}

func (h *ConnectHandler) InviteMember(ctx context.Context, req *connect.Request[projectv1.InviteMemberRequest]) (*connect.Response[projectv1.InviteMemberResponse], error) {
	member, err := h.service.InviteMember(ctx, InviteMemberInput{
		ProjectID: req.Msg.ProjectId,
		Email:     req.Msg.Email,
		Role:      models.Role(req.Msg.Role),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.InviteMemberResponse{
		Member: memberToProto(member),
	}), nil
}

func (h *ConnectHandler) CreateChange(ctx context.Context, req *connect.Request[projectv1.CreateChangeRequest]) (*connect.Response[projectv1.CreateChangeResponse], error) {
	change, err := h.service.CreateChange(ctx, req.Msg.ProjectId, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.CreateChangeResponse{
		Change: changeToProto(change),
	}), nil
}

func (h *ConnectHandler) ListChanges(ctx context.Context, req *connect.Request[projectv1.ListChangesRequest]) (*connect.Response[projectv1.ListChangesResponse], error) {
	changes, err := h.service.ListChanges(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoChanges := make([]*projectv1.Change, len(changes))
	for i, c := range changes {
		protoChanges[i] = changeToProto(&c)
	}

	return connect.NewResponse(&projectv1.ListChangesResponse{
		Changes: protoChanges,
	}), nil
}

func (h *ConnectHandler) GetChange(ctx context.Context, req *connect.Request[projectv1.GetChangeRequest]) (*connect.Response[projectv1.GetChangeResponse], error) {
	change, err := h.service.GetChange(ctx, req.Msg.Id)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.GetChangeResponse{
		Change: changeToProto(change),
	}), nil
}

func (h *ConnectHandler) DeleteChange(ctx context.Context, req *connect.Request[projectv1.DeleteChangeRequest]) (*connect.Response[projectv1.DeleteChangeResponse], error) {
	if err := h.service.DeleteChange(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.DeleteChangeResponse{}), nil
}

func projectToProto(p *models.Project) *projectv1.Project {
	return &projectv1.Project{
		Id:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func memberToProto(m *models.ProjectMember) *projectv1.ProjectMember {
	pm := &projectv1.ProjectMember{
		Id:        m.ID,
		ProjectId: m.ProjectID,
		UserId:    m.UserID,
		Role:      string(m.Role),
	}
	if m.User != nil {
		pm.UserName = m.User.Name
		pm.UserEmail = m.User.Email
	}
	return pm
}

func changeToProto(c *models.Change) *projectv1.Change {
	return &projectv1.Change{
		Id:        c.ID,
		ProjectId: c.ProjectID,
		Name:      c.Name,
		Stage:     string(c.Stage),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}
