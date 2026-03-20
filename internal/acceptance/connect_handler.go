package acceptance

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	acceptancev1 "github.com/gobenpark/colign/gen/proto/acceptance/v1"
	"github.com/gobenpark/colign/gen/proto/acceptance/v1/acceptancev1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service    *Service
	jwtManager *auth.JWTManager
}

var _ acceptancev1connect.AcceptanceCriteriaServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager}
}

func (h *ConnectHandler) CreateAC(ctx context.Context, req *connect.Request[acceptancev1.CreateACRequest]) (*connect.Response[acceptancev1.CreateACResponse], error) {
	if _, err := auth.ExtractClaims(h.jwtManager, req.Header().Get("Authorization")); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ac := &models.AcceptanceCriteria{
		ChangeID:  req.Msg.ChangeId,
		Scenario:  req.Msg.Scenario,
		Steps:     protoStepsToModel(req.Msg.Steps),
		SortOrder: int(req.Msg.SortOrder),
	}
	if err := h.service.Create(ctx, ac); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&acceptancev1.CreateACResponse{
		Criteria: acToProto(ac),
	}), nil
}

func (h *ConnectHandler) ListAC(ctx context.Context, req *connect.Request[acceptancev1.ListACRequest]) (*connect.Response[acceptancev1.ListACResponse], error) {
	items, err := h.service.List(ctx, req.Msg.ChangeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protos := make([]*acceptancev1.AcceptanceCriteria, len(items))
	for i := range items {
		protos[i] = acToProto(&items[i])
	}

	return connect.NewResponse(&acceptancev1.ListACResponse{
		Criteria: protos,
	}), nil
}

func (h *ConnectHandler) UpdateAC(ctx context.Context, req *connect.Request[acceptancev1.UpdateACRequest]) (*connect.Response[acceptancev1.UpdateACResponse], error) {
	if _, err := auth.ExtractClaims(h.jwtManager, req.Header().Get("Authorization")); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ac, err := h.service.Update(ctx, req.Msg.Id, req.Msg.Scenario, protoStepsToModel(req.Msg.Steps), int(req.Msg.SortOrder))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&acceptancev1.UpdateACResponse{
		Criteria: acToProto(ac),
	}), nil
}

func (h *ConnectHandler) ToggleAC(ctx context.Context, req *connect.Request[acceptancev1.ToggleACRequest]) (*connect.Response[acceptancev1.ToggleACResponse], error) {
	if _, err := auth.ExtractClaims(h.jwtManager, req.Header().Get("Authorization")); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ac, err := h.service.Toggle(ctx, req.Msg.Id, req.Msg.Met)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&acceptancev1.ToggleACResponse{
		Criteria: acToProto(ac),
	}), nil
}

func (h *ConnectHandler) DeleteAC(ctx context.Context, req *connect.Request[acceptancev1.DeleteACRequest]) (*connect.Response[acceptancev1.DeleteACResponse], error) {
	if _, err := auth.ExtractClaims(h.jwtManager, req.Header().Get("Authorization")); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := h.service.Delete(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&acceptancev1.DeleteACResponse{}), nil
}

func protoStepsToModel(steps []*acceptancev1.ACStep) []models.ACStep {
	result := make([]models.ACStep, len(steps))
	for i, s := range steps {
		result[i] = models.ACStep{Keyword: s.Keyword, Text: s.Text}
	}
	return result
}

func acToProto(ac *models.AcceptanceCriteria) *acceptancev1.AcceptanceCriteria {
	steps := make([]*acceptancev1.ACStep, len(ac.Steps))
	for i, s := range ac.Steps {
		steps[i] = &acceptancev1.ACStep{Keyword: s.Keyword, Text: s.Text}
	}
	return &acceptancev1.AcceptanceCriteria{
		Id:        ac.ID,
		ChangeId:  ac.ChangeID,
		Scenario:  ac.Scenario,
		Steps:     steps,
		Met:       ac.Met,
		SortOrder: int32(ac.SortOrder),
		CreatedAt: timestamppb.New(ac.CreatedAt),
		UpdatedAt: timestamppb.New(ac.UpdatedAt),
	}
}
