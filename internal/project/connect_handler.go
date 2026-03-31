package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	projectv1 "github.com/gobenpark/colign/gen/proto/project/v1"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	"github.com/gobenpark/colign/internal/archive"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/events"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service           *Service
	archiveService    *archive.Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
	hub               *events.Hub
}

var _ projectv1connect.ProjectServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, archiveService *archive.Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator, hub *events.Hub) *ConnectHandler {
	return &ConnectHandler{service: service, archiveService: archiveService, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator, hub: hub}
}

func (h *ConnectHandler) extractClaims(ctx context.Context, header string) (*auth.Claims, error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) CreateProject(ctx context.Context, req *connect.Request[projectv1.CreateProjectRequest]) (*connect.Response[projectv1.CreateProjectResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
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
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	project, members, labels, err := h.service.GetBySlug(ctx, req.Msg.Slug, claims.OrgID)
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

	pp := projectToProto(project)
	for i := range labels {
		pp.Labels = append(pp.Labels, labelToProto(&labels[i]))
	}

	return connect.NewResponse(&projectv1.GetProjectResponse{
		Project: pp,
		Members: protoMembers,
	}), nil
}

func (h *ConnectHandler) ListProjects(ctx context.Context, req *connect.Request[projectv1.ListProjectsRequest]) (*connect.Response[projectv1.ListProjectsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
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
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	input := UpdateProjectInput{
		ID:          req.Msg.Id,
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Identifier:  req.Msg.Identifier,
	}

	if req.Msg.Status != nil {
		input.Status = req.Msg.Status
	}
	if req.Msg.Priority != nil {
		input.Priority = req.Msg.Priority
	}
	if req.Msg.Health != nil {
		input.Health = req.Msg.Health
	}
	if req.Msg.LeadId != nil {
		v := *req.Msg.LeadId
		if v == 0 {
			input.ClearLead = true
		} else {
			input.LeadID = &v
		}
	}
	if req.Msg.StartDate != nil {
		v := *req.Msg.StartDate
		if v == "" {
			input.ClearStart = true
		} else {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			input.StartDate = &t
		}
	}
	if req.Msg.TargetDate != nil {
		v := *req.Msg.TargetDate
		if v == "" {
			input.ClearTarget = true
		} else {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			input.TargetDate = &t
		}
	}
	if req.Msg.Icon != nil {
		input.Icon = req.Msg.Icon
	}
	if req.Msg.Color != nil {
		input.Color = req.Msg.Color
	}
	if req.Msg.Readme != nil {
		input.Readme = req.Msg.Readme
	}

	project, err := h.service.Update(ctx, input, claims.OrgID)
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
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.Delete(ctx, req.Msg.Id, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.DeleteProjectResponse{}), nil
}

func (h *ConnectHandler) InviteMember(ctx context.Context, req *connect.Request[projectv1.InviteMemberRequest]) (*connect.Response[projectv1.InviteMemberResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	member, err := h.service.InviteMember(ctx, InviteMemberInput{
		ProjectID: req.Msg.ProjectId,
		Email:     req.Msg.Email,
		Role:      models.Role(req.Msg.Role),
		OrgID:     claims.OrgID,
	})
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.InviteMemberResponse{
		Member: memberToProto(member),
	}), nil
}

func (h *ConnectHandler) CreateLabel(ctx context.Context, req *connect.Request[projectv1.CreateLabelRequest]) (*connect.Response[projectv1.CreateLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	label, err := h.service.CreateLabel(ctx, claims.OrgID, req.Msg.Name, req.Msg.Color)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.CreateLabelResponse{
		Label: labelToProto(label),
	}), nil
}

func (h *ConnectHandler) ListLabels(ctx context.Context, req *connect.Request[projectv1.ListLabelsRequest]) (*connect.Response[projectv1.ListLabelsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	labels, err := h.service.ListLabels(ctx, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLabels := make([]*projectv1.ProjectLabel, len(labels))
	for i, l := range labels {
		protoLabels[i] = labelToProto(&l)
	}

	return connect.NewResponse(&projectv1.ListLabelsResponse{
		Labels: protoLabels,
	}), nil
}

func (h *ConnectHandler) DeleteLabel(ctx context.Context, req *connect.Request[projectv1.DeleteLabelRequest]) (*connect.Response[projectv1.DeleteLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteLabel(ctx, req.Msg.Id, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.DeleteLabelResponse{}), nil
}

func (h *ConnectHandler) AssignLabel(ctx context.Context, req *connect.Request[projectv1.AssignLabelRequest]) (*connect.Response[projectv1.AssignLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.AssignLabel(ctx, req.Msg.ProjectId, req.Msg.LabelId, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.AssignLabelResponse{}), nil
}

func (h *ConnectHandler) RemoveLabel(ctx context.Context, req *connect.Request[projectv1.RemoveLabelRequest]) (*connect.Response[projectv1.RemoveLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.RemoveLabel(ctx, req.Msg.ProjectId, req.Msg.LabelId, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.RemoveLabelResponse{}), nil
}

func (h *ConnectHandler) CreateChange(ctx context.Context, req *connect.Request[projectv1.CreateChangeRequest]) (*connect.Response[projectv1.CreateChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	change, err := h.service.CreateChange(ctx, req.Msg.ProjectId, req.Msg.Name, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	h.publishChangeEvent("change_created", change)

	pid, _ := h.service.GetProjectIdentifier(ctx, req.Msg.ProjectId)
	return connect.NewResponse(&projectv1.CreateChangeResponse{
		Change: changeToProto(change, pid),
	}), nil
}

func (h *ConnectHandler) ListChanges(ctx context.Context, req *connect.Request[projectv1.ListChangesRequest]) (*connect.Response[projectv1.ListChangesResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	filter := "active"
	if req.Msg.Filter != nil {
		filter = *req.Msg.Filter
	}
	changes, err := h.service.ListChanges(ctx, req.Msg.ProjectId, filter, claims.OrgID, req.Msg.LabelIds)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pid, _ := h.service.GetProjectIdentifier(ctx, req.Msg.ProjectId)
	protoChanges := make([]*projectv1.Change, len(changes))
	for i, c := range changes {
		protoChanges[i] = changeToProto(&c, pid)
	}

	return connect.NewResponse(&projectv1.ListChangesResponse{
		Changes: protoChanges,
	}), nil
}

func (h *ConnectHandler) GetChange(ctx context.Context, req *connect.Request[projectv1.GetChangeRequest]) (*connect.Response[projectv1.GetChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	change, err := h.service.GetChange(ctx, req.Msg.Id, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pid, _ := h.service.GetProjectIdentifier(ctx, change.ProjectID)
	return connect.NewResponse(&projectv1.GetChangeResponse{
		Change: changeToProto(change, pid),
	}), nil
}

func (h *ConnectHandler) UpdateChange(ctx context.Context, req *connect.Request[projectv1.UpdateChangeRequest]) (*connect.Response[projectv1.UpdateChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	change, err := h.service.UpdateChange(ctx, req.Msg.Id, req.Msg.Name, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	h.publishChangeEvent("change_updated", change)

	pid, _ := h.service.GetProjectIdentifier(ctx, req.Msg.ProjectId)
	return connect.NewResponse(&projectv1.UpdateChangeResponse{
		Change: changeToProto(change, pid),
	}), nil
}

func (h *ConnectHandler) DeleteChange(ctx context.Context, req *connect.Request[projectv1.DeleteChangeRequest]) (*connect.Response[projectv1.DeleteChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteChange(ctx, req.Msg.Id, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.DeleteChangeResponse{}), nil
}

func (h *ConnectHandler) ArchiveChange(ctx context.Context, req *connect.Request[projectv1.ArchiveChangeRequest]) (*connect.Response[projectv1.ArchiveChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	change, err := h.archiveService.Archive(ctx, req.Msg.ChangeId, claims.UserID, claims.OrgID)
	if err != nil {
		switch {
		case errors.Is(err, archive.ErrChangeNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, archive.ErrNotReady), errors.Is(err, archive.ErrAlreadyArchived):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	h.publishChangeEvent("change_updated", change)

	apid, _ := h.service.GetProjectIdentifier(ctx, change.ProjectID)
	return connect.NewResponse(&projectv1.ArchiveChangeResponse{
		Change: changeToProto(change, apid),
	}), nil
}

func (h *ConnectHandler) UnarchiveChange(ctx context.Context, req *connect.Request[projectv1.UnarchiveChangeRequest]) (*connect.Response[projectv1.UnarchiveChangeResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	change, err := h.archiveService.Unarchive(ctx, req.Msg.ChangeId, claims.UserID, claims.OrgID)
	if err != nil {
		switch {
		case errors.Is(err, archive.ErrChangeNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, archive.ErrNotArchived):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	h.publishChangeEvent("change_updated", change)

	upid, _ := h.service.GetProjectIdentifier(ctx, change.ProjectID)
	return connect.NewResponse(&projectv1.UnarchiveChangeResponse{
		Change: changeToProto(change, upid),
	}), nil
}

func (h *ConnectHandler) publishChangeEvent(eventType string, change *models.Change) {
	if h.hub == nil || change == nil {
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"changeId": change.ID,
		"stage":    change.Stage,
	})
	h.hub.Publish(events.Event{
		Type:     eventType,
		ChangeID: change.ID,
		Payload:  string(payload),
	})
}

func (h *ConnectHandler) GetArchivePolicy(ctx context.Context, req *connect.Request[projectv1.GetArchivePolicyRequest]) (*connect.Response[projectv1.GetArchivePolicyResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	policy, err := h.archiveService.GetPolicy(ctx, req.Msg.ProjectId, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.GetArchivePolicyResponse{
		Policy: &projectv1.ArchivePolicyMsg{
			ProjectId: policy.ProjectID,
			Mode:      string(policy.Mode),
			Trigger:   string(policy.TriggerType),
			DaysDelay: int32(policy.DaysDelay),
		},
	}), nil
}

func (h *ConnectHandler) UpdateArchivePolicy(ctx context.Context, req *connect.Request[projectv1.UpdateArchivePolicyRequest]) (*connect.Response[projectv1.UpdateArchivePolicyResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	policy, err := h.archiveService.UpdatePolicy(ctx, &models.ArchivePolicy{
		ProjectID:   req.Msg.ProjectId,
		Mode:        models.ArchiveMode(req.Msg.Mode),
		TriggerType: models.ArchiveTrigger(req.Msg.Trigger),
		DaysDelay:   int(req.Msg.DaysDelay),
	}, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&projectv1.UpdateArchivePolicyResponse{
		Policy: &projectv1.ArchivePolicyMsg{
			ProjectId: policy.ProjectID,
			Mode:      string(policy.Mode),
			Trigger:   string(policy.TriggerType),
			DaysDelay: int32(policy.DaysDelay),
		},
	}), nil
}

func projectToProto(p *models.Project) *projectv1.Project {
	pp := &projectv1.Project{
		Id:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Readme:      p.Readme,
		Status:      string(p.Status),
		Priority:    string(p.Priority),
		Health:      string(p.Health),
		Icon:        p.Icon,
		Color:       p.Color,
		Identifier:  p.Identifier,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}

	if p.LeadID != nil {
		pp.LeadId = p.LeadID
	}
	if p.Lead != nil {
		pp.LeadName = p.Lead.Name
		pp.LeadAvatarUrl = p.Lead.AvatarURL
	}
	if p.StartDate != nil {
		s := p.StartDate.Format("2006-01-02")
		pp.StartDate = &s
	}
	if p.TargetDate != nil {
		s := p.TargetDate.Format("2006-01-02")
		pp.TargetDate = &s
	}

	return pp
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

func (h *ConnectHandler) AssignChangeLabel(ctx context.Context, req *connect.Request[projectv1.AssignChangeLabelRequest]) (*connect.Response[projectv1.AssignChangeLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.AssignChangeLabel(ctx, req.Msg.ChangeId, req.Msg.LabelId, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.AssignChangeLabelResponse{}), nil
}

func (h *ConnectHandler) RemoveChangeLabel(ctx context.Context, req *connect.Request[projectv1.RemoveChangeLabelRequest]) (*connect.Response[projectv1.RemoveChangeLabelResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.RemoveChangeLabel(ctx, req.Msg.ChangeId, req.Msg.LabelId, claims.OrgID); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&projectv1.RemoveChangeLabelResponse{}), nil
}

func changeToProto(c *models.Change, projectIdentifier string) *projectv1.Change {
	pc := &projectv1.Change{
		Id:        c.ID,
		ProjectId: c.ProjectID,
		Name:      c.Name,
		Stage:     string(c.Stage),
		SubStatus: string(c.SubStatus),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
		Number:    int32(c.Number),
	}
	if projectIdentifier != "" {
		pc.Identifier = fmt.Sprintf("%s-%d", projectIdentifier, c.Number)
	}
	if c.ArchivedAt != nil {
		pc.ArchivedAt = timestamppb.New(*c.ArchivedAt)
	}
	for i := range c.Labels {
		pc.Labels = append(pc.Labels, labelToProto(&c.Labels[i]))
	}
	return pc
}

func labelToProto(l *models.ProjectLabel) *projectv1.ProjectLabel {
	return &projectv1.ProjectLabel{
		Id:    l.ID,
		Name:  l.Name,
		Color: l.Color,
	}
}

func (h *ConnectHandler) Search(ctx context.Context, req *connect.Request[projectv1.SearchRequest]) (*connect.Response[projectv1.SearchResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	results, err := h.service.Search(ctx, req.Msg.Query, claims.UserID, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoResults := make([]*projectv1.SearchResult, len(results))
	for i, r := range results {
		protoResults[i] = &projectv1.SearchResult{
			Type:      r.Type,
			Id:        r.ID,
			Title:     r.Title,
			Subtitle:  r.Subtitle,
			Slug:      r.Slug,
			ProjectId: r.ProjectID,
		}
	}

	return connect.NewResponse(&projectv1.SearchResponse{
		Results: protoResults,
	}), nil
}
