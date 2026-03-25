package task

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/gobenpark/colign/gen/proto/task/v1"
	"github.com/gobenpark/colign/gen/proto/task/v1/taskv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

var _ taskv1connect.TaskServiceHandler = (*ConnectHandler)(nil)

func (h *ConnectHandler) extractClaims(ctx context.Context, header string) (*auth.Claims, error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) ListTasks(ctx context.Context, req *connect.Request[taskv1.ListTasksRequest]) (*connect.Response[taskv1.ListTasksResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	tasks, err := h.service.List(ctx, req.Msg.ChangeId, req.Msg.ProjectId, claims.OrgID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTasks := make([]*taskv1.Task, len(tasks))
	for i, t := range tasks {
		protoTasks[i] = taskToProto(&t)
	}

	return connect.NewResponse(&taskv1.ListTasksResponse{
		Tasks: protoTasks,
	}), nil
}

func (h *ConnectHandler) CreateTask(ctx context.Context, req *connect.Request[taskv1.CreateTaskRequest]) (*connect.Response[taskv1.CreateTaskResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	status := req.Msg.Status
	if status == "" {
		status = "todo"
	}

	userID := claims.UserID
	t := &models.Task{
		ChangeID:    req.Msg.ChangeId,
		Title:       req.Msg.Title,
		Description: req.Msg.Description,
		Status:      models.TaskStatus(status),
		SpecRef:     req.Msg.SpecRef,
		CreatorID:   &userID,
	}
	if req.Msg.AssigneeId != nil {
		t.AssigneeID = req.Msg.AssigneeId
	}

	if err := h.service.Create(ctx, t, req.Msg.ProjectId, claims.OrgID); err != nil {
		if errors.Is(err, ErrInvalidChange) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&taskv1.CreateTaskResponse{
		Task: taskToProto(t),
	}), nil
}

func (h *ConnectHandler) UpdateTask(ctx context.Context, req *connect.Request[taskv1.UpdateTaskRequest]) (*connect.Response[taskv1.UpdateTaskResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	t, err := h.service.Update(ctx, req.Msg.Id, req.Msg.Title, req.Msg.Description, req.Msg.Status, req.Msg.SpecRef, req.Msg.AssigneeId, req.Msg.ClearAssignee, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&taskv1.UpdateTaskResponse{
		Task: taskToProto(t),
	}), nil
}

func (h *ConnectHandler) DeleteTask(ctx context.Context, req *connect.Request[taskv1.DeleteTaskRequest]) (*connect.Response[taskv1.DeleteTaskResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.Delete(ctx, req.Msg.Id, claims.OrgID); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&taskv1.DeleteTaskResponse{}), nil
}

func (h *ConnectHandler) ReorderTasks(ctx context.Context, req *connect.Request[taskv1.ReorderTasksRequest]) (*connect.Response[taskv1.ReorderTasksResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	items := make([]ReorderItem, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = ReorderItem{
			ID:         item.Id,
			Status:     item.Status,
			OrderIndex: int(item.OrderIndex),
		}
	}

	if err := h.service.Reorder(ctx, req.Msg.ChangeId, req.Msg.ProjectId, items, claims.OrgID); err != nil {
		if errors.Is(err, ErrInvalidChange) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&taskv1.ReorderTasksResponse{}), nil
}

func taskToProto(t *models.Task) *taskv1.Task {
	pt := &taskv1.Task{
		Id:          t.ID,
		ChangeId:    t.ChangeID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		OrderIndex:  int32(t.OrderIndex),
		SpecRef:     t.SpecRef,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
	if t.AssigneeID != nil {
		pt.AssigneeId = t.AssigneeID
	}
	if t.CreatorID != nil {
		pt.CreatorId = t.CreatorID
	}
	if t.Assignee != nil {
		pt.AssigneeName = t.Assignee.Name
	}
	if t.Creator != nil {
		pt.CreatorName = t.Creator.Name
	}
	return pt
}
