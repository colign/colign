package notification

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	notificationv1 "github.com/gobenpark/colign/gen/proto/notification/v1"
	"github.com/gobenpark/colign/gen/proto/notification/v1/notificationv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/events"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
	hub               *events.Hub
}

var _ notificationv1connect.NotificationServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator, hub *events.Hub) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator, hub: hub}
}

func (h *ConnectHandler) extractClaims(ctx context.Context, header string) (*auth.Claims, error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) ListNotifications(ctx context.Context, req *connect.Request[notificationv1.ListNotificationsRequest]) (*connect.Response[notificationv1.ListNotificationsResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	notifications, unreadCount, err := h.service.List(ctx, claims.UserID, req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoNotifs := make([]*notificationv1.Notification, len(notifications))
	for i, n := range notifications {
		protoNotifs[i] = notificationToProto(&n)
	}

	return connect.NewResponse(&notificationv1.ListNotificationsResponse{
		Notifications: protoNotifs,
		UnreadCount:   int32(unreadCount),
	}), nil
}

func (h *ConnectHandler) MarkRead(ctx context.Context, req *connect.Request[notificationv1.MarkReadRequest]) (*connect.Response[notificationv1.MarkReadResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.MarkRead(ctx, claims.UserID, req.Msg.Id, req.Msg.Read); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&notificationv1.MarkReadResponse{}), nil
}

func (h *ConnectHandler) MarkAllRead(ctx context.Context, req *connect.Request[notificationv1.MarkAllReadRequest]) (*connect.Response[notificationv1.MarkAllReadResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.MarkAllRead(ctx, claims.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&notificationv1.MarkAllReadResponse{}), nil
}

func (h *ConnectHandler) GetUnreadCount(ctx context.Context, req *connect.Request[notificationv1.GetUnreadCountRequest]) (*connect.Response[notificationv1.GetUnreadCountResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	count, err := h.service.GetUnreadCount(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&notificationv1.GetUnreadCountResponse{
		Count: int32(count),
	}), nil
}

func (h *ConnectHandler) Subscribe(ctx context.Context, req *connect.Request[notificationv1.SubscribeRequest], stream *connect.ServerStream[notificationv1.SubscribeResponse]) error {
	if _, err := h.extractClaims(ctx, req.Header().Get("Authorization")); err != nil {
		return err
	}

	sub := h.hub.Subscribe(req.Msg.ChangeId)
	defer h.hub.Unsubscribe(sub)

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-sub.Events():
			if !ok {
				return nil
			}
			if err := stream.Send(&notificationv1.SubscribeResponse{
				Type:      evt.Type,
				ChangeId:  evt.ChangeID,
				Payload:   evt.Payload,
				Timestamp: timestamppb.Now(),
			}); err != nil {
				return err
			}
		}
	}
}

func notificationToProto(n *models.Notification) *notificationv1.Notification {
	proto := &notificationv1.Notification{
		Id:             n.ID,
		UserId:         n.UserID,
		Type:           string(n.Type),
		Read:           n.Read,
		ActorId:        n.ActorID,
		ChangeId:       n.ChangeID,
		ProjectId:      n.ProjectID,
		Stage:          n.Stage,
		CommentPreview: n.CommentPreview,
		CreatedAt:      timestamppb.New(n.CreatedAt),
	}
	if n.Actor != nil {
		proto.ActorName = n.Actor.Name
	}
	if n.Change != nil {
		proto.ChangeName = n.Change.Name
	}
	if n.Project != nil {
		proto.ProjectName = n.Project.Name
		proto.ProjectSlug = n.Project.Slug
	}
	return proto
}
