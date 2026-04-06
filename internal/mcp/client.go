package mcp

import (
	"net/http"

	"github.com/gobenpark/colign/gen/proto/acceptance/v1/acceptancev1connect"
	"github.com/gobenpark/colign/gen/proto/auth/v1/authv1connect"
	"github.com/gobenpark/colign/gen/proto/comment/v1/commentv1connect"
	"github.com/gobenpark/colign/gen/proto/document/v1/documentv1connect"
	"github.com/gobenpark/colign/gen/proto/memory/v1/memoryv1connect"
	"github.com/gobenpark/colign/gen/proto/organization/v1/organizationv1connect"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	"github.com/gobenpark/colign/gen/proto/task/v1/taskv1connect"
	"github.com/gobenpark/colign/gen/proto/wiki/v1/wikiv1connect"
	"github.com/gobenpark/colign/gen/proto/workflow/v1/workflowv1connect"
	"github.com/gobenpark/colign/internal/events"
)

type apiClients struct {
	project             projectv1connect.ProjectServiceClient
	document            documentv1connect.DocumentServiceClient
	task                taskv1connect.TaskServiceClient
	acceptance          acceptancev1connect.AcceptanceCriteriaServiceClient
	comment             commentv1connect.CommentServiceClient
	workflow            workflowv1connect.WorkflowServiceClient
	memory              memoryv1connect.MemoryServiceClient
	auth                authv1connect.AuthServiceClient
	organization        organizationv1connect.OrganizationServiceClient
	wiki                wikiv1connect.WikiServiceClient
	hocuspocusURL       string
	hocuspocusAPISecret string
	eventHub            *events.Hub
}

// tokenTransport injects the Authorization header into every request.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

func newAPIClients(apiURL, apiToken string, opts ...clientOption) *apiClients {
	httpClient := &http.Client{
		Transport: &tokenTransport{
			token: apiToken,
			base:  http.DefaultTransport,
		},
	}

	c := &apiClients{
		project:      projectv1connect.NewProjectServiceClient(httpClient, apiURL),
		document:     documentv1connect.NewDocumentServiceClient(httpClient, apiURL),
		task:         taskv1connect.NewTaskServiceClient(httpClient, apiURL),
		acceptance:   acceptancev1connect.NewAcceptanceCriteriaServiceClient(httpClient, apiURL),
		comment:      commentv1connect.NewCommentServiceClient(httpClient, apiURL),
		workflow:     workflowv1connect.NewWorkflowServiceClient(httpClient, apiURL),
		memory:       memoryv1connect.NewMemoryServiceClient(httpClient, apiURL),
		auth:         authv1connect.NewAuthServiceClient(httpClient, apiURL),
		organization: organizationv1connect.NewOrganizationServiceClient(httpClient, apiURL),
		wiki:         wikiv1connect.NewWikiServiceClient(httpClient, apiURL),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures additional options for MCP API clients.
type ClientOption = clientOption

type clientOption func(*apiClients)

func WithHocuspocus(url, secret string) clientOption {
	return func(c *apiClients) {
		c.hocuspocusURL = url
		c.hocuspocusAPISecret = secret
	}
}

func WithEventHub(hub *events.Hub) clientOption {
	return func(c *apiClients) {
		c.eventHub = hub
	}
}
