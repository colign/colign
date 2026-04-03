package middleware

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workflowv1 "github.com/gobenpark/colign/gen/proto/workflow/v1"
	"github.com/gobenpark/colign/internal/events"
)

// fakeAnyRequest wraps a proto message so it satisfies connect.AnyRequest.
type fakeAnyRequest struct {
	connect.AnyRequest
	msg any
}

func (f *fakeAnyRequest) Any() any { return f.msg }

// fakeAnyResponse wraps a proto message so it satisfies connect.AnyResponse.
type fakeAnyResponse struct {
	connect.AnyResponse
	msg any
}

func (f *fakeAnyResponse) Any() any { return f.msg }

func TestExtractEvent_Revert(t *testing.T) {
	n := &NotificationInterceptor{}

	req := &fakeAnyRequest{
		msg: &workflowv1.RevertRequest{
			ChangeId:  42,
			Reason:    "wrong stage",
			ProjectId: 7,
		},
	}
	resp := &fakeAnyResponse{
		msg: &workflowv1.RevertResponse{
			NewStage: "draft",
		},
	}

	evt, ok := n.extractEvent(context.Background(), "/workflow.v1.WorkflowService/Revert", req, resp)
	require.True(t, ok)
	assert.Equal(t, "stage_change", evt.Type)
	assert.Equal(t, int64(42), evt.ChangeID)
	assert.Equal(t, int64(7), evt.ProjectID)
	assert.Equal(t, "draft", evt.Metadata["new_stage"])
	assert.Equal(t, "wrong stage", evt.Metadata["reason"])
	assert.Equal(t, "revert", evt.Metadata["action"])
}

func TestExtractEvent_Revert_BadResponse(t *testing.T) {
	n := &NotificationInterceptor{}

	req := &fakeAnyRequest{
		msg: &workflowv1.RevertRequest{ChangeId: 1, ProjectId: 1},
	}
	resp := &fakeAnyResponse{
		msg: "not a RevertResponse",
	}

	_, ok := n.extractEvent(context.Background(), "/workflow.v1.WorkflowService/Revert", req, resp)
	assert.False(t, ok)
}

func TestExtractEvent_Revert_BadRequest(t *testing.T) {
	n := &NotificationInterceptor{}

	req := &fakeAnyRequest{
		msg: "not a RevertRequest",
	}
	resp := &fakeAnyResponse{
		msg: &workflowv1.RevertResponse{NewStage: "draft"},
	}

	_, ok := n.extractEvent(context.Background(), "/workflow.v1.WorkflowService/Revert", req, resp)
	assert.False(t, ok)
}

func TestExtractEvent_Advance(t *testing.T) {
	n := &NotificationInterceptor{}

	req := &fakeAnyRequest{
		msg: &workflowv1.AdvanceRequest{
			ChangeId:  10,
			ProjectId: 3,
		},
	}
	resp := &fakeAnyResponse{
		msg: &workflowv1.AdvanceResponse{
			NewStage: "spec",
		},
	}

	evt, ok := n.extractEvent(context.Background(), "/workflow.v1.WorkflowService/Advance", req, resp)
	require.True(t, ok)
	assert.Equal(t, "stage_change", evt.Type)
	assert.Equal(t, int64(10), evt.ChangeID)
	assert.Equal(t, "spec", evt.Metadata["new_stage"])
}

func TestExtractEvent_UnknownProcedure(t *testing.T) {
	n := &NotificationInterceptor{}

	req := &fakeAnyRequest{msg: nil}
	resp := &fakeAnyResponse{msg: nil}

	_, ok := n.extractEvent(context.Background(), "/unknown.Service/Method", req, resp)
	assert.False(t, ok)
}

// Ensure NotificationEvent type matches expected shape.
func TestNotificationEventShape(t *testing.T) {
	evt := events.NotificationEvent{
		Type:      "stage_change",
		ActorID:   1,
		ChangeID:  2,
		ProjectID: 3,
		Metadata: map[string]any{
			"new_stage": "draft",
			"reason":    "test",
		},
	}
	assert.Equal(t, "stage_revert", evt.Type)
	assert.Equal(t, int64(2), evt.ChangeID)
}
