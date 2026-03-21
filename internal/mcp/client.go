package mcp

import (
	"net/http"

	"github.com/gobenpark/colign/gen/proto/document/v1/documentv1connect"
	"github.com/gobenpark/colign/gen/proto/project/v1/projectv1connect"
	"github.com/gobenpark/colign/gen/proto/task/v1/taskv1connect"
)

type apiClients struct {
	project  projectv1connect.ProjectServiceClient
	document documentv1connect.DocumentServiceClient
	task     taskv1connect.TaskServiceClient
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

func newAPIClients(apiURL, apiToken string) *apiClients {
	httpClient := &http.Client{
		Transport: &tokenTransport{
			token: apiToken,
			base:  http.DefaultTransport,
		},
	}

	return &apiClients{
		project:  projectv1connect.NewProjectServiceClient(httpClient, apiURL),
		document: documentv1connect.NewDocumentServiceClient(httpClient, apiURL),
		task:     taskv1connect.NewTaskServiceClient(httpClient, apiURL),
	}
}
