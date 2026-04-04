package middleware

import (
	"context"

	"connectrpc.com/connect"

	commentv1 "github.com/gobenpark/colign/gen/proto/comment/v1"
	workflowv1 "github.com/gobenpark/colign/gen/proto/workflow/v1"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/events"
)

// NotificationInterceptor publishes NotificationEvents to the EventHub
// after successful RPC calls.
type NotificationInterceptor struct {
	hub               *events.Hub
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

func NewNotificationInterceptor(hub *events.Hub, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *NotificationInterceptor {
	return &NotificationInterceptor{hub: hub, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

func (n *NotificationInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}

		procedure := req.Spec().Procedure
		evt, ok := n.extractEvent(ctx, procedure, req, resp)
		if !ok {
			return resp, nil
		}

		// Resolve actor
		claims, authErr := auth.ResolveFromHeader(n.jwtManager, n.apiTokenValidator, ctx, req.Header().Get("Authorization"))
		if authErr == nil {
			evt.ActorID = claims.UserID
		}

		n.hub.PublishNotification(evt)
		return resp, nil
	}
}

func (n *NotificationInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (n *NotificationInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (n *NotificationInterceptor) extractEvent(_ context.Context, procedure string, req connect.AnyRequest, resp connect.AnyResponse) (events.NotificationEvent, bool) {
	switch procedure {
	case "/workflow.v1.WorkflowService/Advance":
		r, ok := resp.Any().(*workflowv1.AdvanceResponse)
		if !ok {
			return events.NotificationEvent{}, false
		}
		advReq, _ := req.Any().(*workflowv1.AdvanceRequest)
		return events.NotificationEvent{
			Type:      "stage_change",
			ChangeID:  advReq.GetChangeId(),
			ProjectID: advReq.GetProjectId(),
			Metadata: map[string]any{
				"new_stage": r.GetNewStage(),
			},
		}, true

	case "/workflow.v1.WorkflowService/Approve":
		approveReq, _ := req.Any().(*workflowv1.ApproveRequest)
		return events.NotificationEvent{
			Type:      "approve",
			ChangeID:  approveReq.GetChangeId(),
			ProjectID: approveReq.GetProjectId(),
			Metadata: map[string]any{
				"comment": approveReq.GetComment(),
			},
		}, true

	case "/workflow.v1.WorkflowService/Revert":
		revertResp, ok := resp.Any().(*workflowv1.RevertResponse)
		if !ok {
			return events.NotificationEvent{}, false
		}
		revertReq, ok := req.Any().(*workflowv1.RevertRequest)
		if !ok {
			return events.NotificationEvent{}, false
		}
		return events.NotificationEvent{
			Type:      "stage_change",
			ChangeID:  revertReq.GetChangeId(),
			ProjectID: revertReq.GetProjectId(),
			Metadata: map[string]any{
				"new_stage": revertResp.GetNewStage(),
				"reason":    revertReq.GetReason(),
				"action":    "revert",
			},
		}, true

	case "/workflow.v1.WorkflowService/RequestChanges":
		rcReq, _ := req.Any().(*workflowv1.RequestChangesRequest)
		return events.NotificationEvent{
			Type:      "reject",
			ChangeID:  rcReq.GetChangeId(),
			ProjectID: rcReq.GetProjectId(),
			Metadata: map[string]any{
				"reason": rcReq.GetReason(),
			},
		}, true

	case "/comment.v1.CommentService/CreateComment":
		ccReq, _ := req.Any().(*commentv1.CreateCommentRequest)
		metadata := map[string]any{
			"preview": truncate(ccReq.GetBody(), 140),
		}
		if len(ccReq.GetMentionedUserIds()) > 0 {
			metadata["mentioned_user_ids"] = ccReq.GetMentionedUserIds()
		}
		return events.NotificationEvent{
			Type:      "comment",
			ChangeID:  ccReq.GetChangeId(),
			ProjectID: ccReq.GetProjectId(),
			Metadata:  metadata,
		}, true

	case "/comment.v1.CommentService/CreateReply":
		crReq, _ := req.Any().(*commentv1.CreateReplyRequest)
		metadata := map[string]any{
			"preview":    truncate(crReq.GetBody(), 140),
			"comment_id": crReq.GetCommentId(),
		}
		if len(crReq.GetMentionedUserIds()) > 0 {
			metadata["mentioned_user_ids"] = crReq.GetMentionedUserIds()
		}
		return events.NotificationEvent{
			Type:      "comment",
			ProjectID: crReq.GetProjectId(),
			Metadata:  metadata,
		}, true

	default:
		return events.NotificationEvent{}, false
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
