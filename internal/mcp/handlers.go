package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"connectrpc.com/connect"

	acceptancev1 "github.com/gobenpark/colign/gen/proto/acceptance/v1"
	authv1 "github.com/gobenpark/colign/gen/proto/auth/v1"
	commentv1 "github.com/gobenpark/colign/gen/proto/comment/v1"
	documentv1 "github.com/gobenpark/colign/gen/proto/document/v1"
	memoryv1 "github.com/gobenpark/colign/gen/proto/memory/v1"
	organizationv1 "github.com/gobenpark/colign/gen/proto/organization/v1"
	projectv1 "github.com/gobenpark/colign/gen/proto/project/v1"
	taskv1 "github.com/gobenpark/colign/gen/proto/task/v1"
	workflowv1 "github.com/gobenpark/colign/gen/proto/workflow/v1"
	"github.com/gobenpark/colign/internal/events"
	"github.com/gobenpark/colign/internal/models"
)

func init() {
	// ── project ──────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "list_projects",
		Description: "List all projects the user has access to",
		InputSchema: InputSchema{Type: "object"},
		ReadOnly:    true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListProjects(ctx)
		},
	})
	RegisterTool(Tool{
		Name:        "create_project",
		Description: "Create a new project in the current organization",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name":        {Type: "string", Description: "Project name"},
				"description": {Type: "string", Description: "Short project description (one-liner, optional)"},
			},
			Required: []string{"name"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleCreateProject(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "update_project",
		Description: "Update a project's name, description, or README",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id":  {Type: "integer", Description: "Project ID"},
				"name":        {Type: "string", Description: "New project name (optional)"},
				"description": {Type: "string", Description: "Short project description (one-liner)"},
				"readme":      {Type: "string", Description: "Project README content in markdown (auto-converted to HTML)"},
			},
			Required: []string{"project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleUpdateProject(ctx, args)
		},
	})

	// ── change ───────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "get_change",
		Description: "Get details of a specific change including its stage and artifacts",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetChange(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "list_changes",
		Description: "List all changes in a project",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListChanges(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "get_work_context",
		Description: "Get all context needed to start working on a change in one call: change info, proposal, tasks, acceptance criteria, memory, gate conditions, and recent comments.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id": {Type: "integer", Description: "Change ID"},
			},
			Required: []string{"change_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetWorkContext(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "get_change_summary",
		Description: "Get an aggregated summary of a change: stage, task progress, AC progress, and gate conditions. Ideal for quick status checks.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id": {Type: "integer", Description: "Change ID"},
			},
			Required: []string{"change_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetChangeSummary(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "get_change_history",
		Description: "Get the workflow event history for a change. Returns stage transitions, approvals, and rejections in chronological order.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetChangeHistory(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "get_project_dashboard",
		Description: "Get all active (non-archived) changes in a project with their progress summaries. Shows task and AC progress for each change.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetProjectDashboard(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "create_change",
		Description: "Create a new change (feature/initiative) in a project",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {Type: "integer", Description: "Project ID"},
				"name":       {Type: "string", Description: "Change name"},
			},
			Required: []string{"project_id", "name"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleCreateChange(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "update_change",
		Description: "Update a change's name and/or workflow status. Use status for composite workflow states like draft(ready) or spec(in_progress). Approved changes must still go through approve_change. At least one of name, sub_status, or status must be provided.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":     {Type: "integer", Description: "Change ID"},
				"project_id":    {Type: "integer", Description: "Project ID"},
				"name":          {Type: "string", Description: "New name for the change (optional)"},
				"sub_status":    {Type: "string", Description: "Sub-status only: in_progress or ready (optional). Cannot be combined with status."},
				"status":        {Type: "string", Description: "Composite workflow status: draft(in_progress), draft(ready), spec(in_progress), or spec(ready) (optional). Cannot be used to transition to approved."},
				"status_reason": {Type: "string", Description: "Reason recorded when moving the change backward to an earlier stage (optional)."},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleUpdateChange(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "archive_change",
		Description: "Archive a change (cancel or shelve it).",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleArchiveChange(ctx, args)
		},
	})

	// ── spec ─────────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "read_spec",
		Description: "Read a spec document for a change. For proposals, the content field is a JSON string with keys: problem, scope, outOfScope, designLinks (optional), documentLinks (optional).",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"doc_type":   {Type: "string", Description: "Document type: proposal, spec, tasks"},
			},
			Required: []string{"change_id", "project_id", "doc_type"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleReadSpec(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "suggest_spec",
		Description: "Get AI suggestions for improving a spec document",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"doc_type":   {Type: "string", Description: "Document type to improve"},
			},
			Required: []string{"change_id", "project_id", "doc_type"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleSuggestSpec(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "write_spec",
		Description: "Write or update a spec document for a change. For proposals, content must be a JSON string with keys: problem (required), scope (required), outOfScope (optional), designLinks (optional string array), documentLinks (optional string array). Example: {\"problem\":\"...\",\"scope\":\"...\",\"outOfScope\":\"...\",\"documentLinks\":[\"https://...\"]}. For other doc types, content is plain markdown.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"doc_type":   {Type: "string", Description: "Document type: proposal, spec, tasks"},
				"content":    {Type: "string", Description: "For proposal: JSON with problem, scope, outOfScope. For others: markdown text."},
			},
			Required: []string{"change_id", "project_id", "doc_type", "content"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleWriteSpec(ctx, args)
		},
	})

	// ── task ─────────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "list_tasks",
		Description: "List implementation tasks for a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListTasks(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "create_task",
		Description: "Create a new implementation task for a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":   {Type: "integer", Description: "Change ID"},
				"project_id":  {Type: "integer", Description: "Project ID"},
				"title":       {Type: "string", Description: "Task title"},
				"description": {Type: "string", Description: "Task description (optional)"},
				"status":      {Type: "string", Description: "Initial status: todo, in_progress, done (default: todo)"},
			},
			Required: []string{"change_id", "project_id", "title"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleCreateTask(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "update_task",
		Description: "Update a task's status or assignee",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"task_id":        {Type: "integer", Description: "Task ID"},
				"project_id":     {Type: "integer", Description: "Project ID"},
				"status":         {Type: "string", Description: "New status: todo, in_progress, done"},
				"assignee_id":    {Type: "integer", Description: "User ID to assign the task to (optional)"},
				"clear_assignee": {Type: "boolean", Description: "Set to true to remove the current assignee (optional)"},
			},
			Required: []string{"task_id", "project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleUpdateTask(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "delete_task",
		Description: "Delete a task",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"task_id":    {Type: "integer", Description: "Task ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"task_id", "project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleDeleteTask(ctx, args)
		},
	})

	// ── ac (acceptance criteria) ─────────────────────────────────────
	RegisterTool(Tool{
		Name:        "list_acceptance_criteria",
		Description: "List acceptance criteria (Given/When/Then scenarios) for a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListAC(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "create_acceptance_criteria",
		Description: "Create an acceptance criteria with BDD-style Given/When/Then steps for a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"scenario":   {Type: "string", Description: "Scenario name describing the test case"},
				"steps":      {Type: "string", Description: "JSON array of steps, each with keyword (Given/When/Then/And/But) and text. Example: [{\"keyword\":\"Given\",\"text\":\"a user is logged in\"},{\"keyword\":\"When\",\"text\":\"they click logout\"},{\"keyword\":\"Then\",\"text\":\"they are redirected to login page\"}]"},
				"test_ref":   {Type: "string", Description: "Reference to test that verifies this criteria, e.g. 'tests/checkout_test.go::TestPaymentSuccess' (optional)"},
			},
			Required: []string{"change_id", "project_id", "scenario", "steps"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleCreateAC(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "toggle_acceptance_criteria",
		Description: "Toggle an acceptance criteria's met/unmet status",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"id":         {Type: "integer", Description: "Acceptance criteria ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"met":        {Type: "boolean", Description: "Whether the criteria is met (true) or unmet (false)"},
			},
			Required: []string{"id", "project_id", "met"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleToggleAC(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "link_ac_to_test",
		Description: "Link an acceptance criteria to a test reference. Set test_ref to empty string to unlink.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"ac_id":      {Type: "integer", Description: "Acceptance criteria ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"test_ref":   {Type: "string", Description: "Test reference, e.g. 'tests/checkout_test.go::TestPaymentSuccess'"},
			},
			Required: []string{"ac_id", "project_id", "test_ref"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleLinkACToTest(ctx, args)
		},
	})

	// ── workflow ──────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "get_gate_status",
		Description: "Get the gate conditions for a change's current stage and whether it can advance.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetGateStatus(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "approve_change",
		Description: "Approve a change in spec stage. If approval policy is met, the change advances automatically.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"comment":    {Type: "string", Description: "Optional approval comment"},
			},
			Required: []string{"change_id", "project_id"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleApproveChange(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "reject_change",
		Description: "Request changes on a review, sending the change back to draft stage.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":  {Type: "integer", Description: "Change ID"},
				"project_id": {Type: "integer", Description: "Project ID"},
				"reason":     {Type: "string", Description: "Reason for requesting changes"},
			},
			Required: []string{"change_id", "project_id", "reason"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleRejectChange(ctx, args)
		},
	})

	// ── comment ──────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "list_comments",
		Description: "List comments on a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":     {Type: "integer", Description: "Change ID"},
				"project_id":    {Type: "integer", Description: "Project ID"},
				"document_type": {Type: "string", Description: "Document type to filter comments (proposal, spec, tasks)"},
			},
			Required: []string{"change_id", "project_id", "document_type"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListComments(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "create_comment",
		Description: "Add a comment to a change",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"change_id":     {Type: "integer", Description: "Change ID"},
				"project_id":    {Type: "integer", Description: "Project ID"},
				"content":       {Type: "string", Description: "Comment text"},
				"document_type": {Type: "string", Description: "Document type for the comment (proposal, spec, tasks)"},
			},
			Required: []string{"change_id", "project_id", "content", "document_type"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleCreateComment(ctx, args)
		},
	})

	// ── memory ───────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "get_memory",
		Description: "Get the project memory (shared context like CLAUDE.md). Contains conventions, decisions, and context that AI should know about this project.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {Type: "integer", Description: "Project ID"},
			},
			Required: []string{"project_id"},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetMemory(ctx, args)
		},
	})
	RegisterTool(Tool{
		Name:        "save_memory",
		Description: "Save or update the project memory (shared context). Use this to persist important conventions, decisions, and context about the project.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"project_id": {Type: "integer", Description: "Project ID"},
				"content":    {Type: "string", Description: "Memory content in markdown"},
			},
			Required: []string{"project_id", "content"},
		},
		ReadOnly: false,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleSaveMemory(ctx, args)
		},
	})

	// ── auth ─────────────────────────────────────────────────────────
	RegisterTool(Tool{
		Name:        "get_me",
		Description: "Get the current authenticated user's information (ID, name, email, organization).",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]Property{},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleGetMe(ctx)
		},
	})
	RegisterTool(Tool{
		Name:        "list_members",
		Description: "List all members in the current organization. Use this to find user IDs for task assignment.",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]Property{},
		},
		ReadOnly: true,
		Handler: func(s *Server, ctx context.Context, args json.RawMessage) (any, error) {
			return s.handleListMembers(ctx)
		},
	})
}

func (s *Server) handleCreateProject(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.CreateProject(ctx, connect.NewRequest(&projectv1.CreateProjectRequest{
		Name:        params.Name,
		Description: params.Description,
	}))
	if err != nil {
		return nil, err
	}

	p := resp.Msg.Project
	return map[string]any{
		"id":          p.Id,
		"name":        p.Name,
		"slug":        p.Slug,
		"description": p.Description,
	}, nil
}

func (s *Server) handleListProjects(ctx context.Context) (any, error) {
	resp, err := s.clients.project.ListProjects(ctx, connect.NewRequest(&projectv1.ListProjectsRequest{}))
	if err != nil {
		return nil, err
	}

	type projectInfo struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}

	projects := make([]projectInfo, len(resp.Msg.Projects))
	for i, p := range resp.Msg.Projects {
		projects[i] = projectInfo{
			ID:          p.Id,
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
		}
	}

	return projects, nil
}

func (s *Server) handleGetChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{
		Id:        params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	c := resp.Msg.Change
	return map[string]any{
		"id":         c.Id,
		"project_id": c.ProjectId,
		"name":       c.Name,
		"stage":      c.Stage,
		"sub_status": c.SubStatus,
		"number":     c.Number,
		"identifier": c.Identifier,
	}, nil
}

func (s *Server) handleReadSpec(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		DocType   string    `json:"doc_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.document.GetDocument(ctx, connect.NewRequest(&documentv1.GetDocumentRequest{
		ChangeId:  params.ChangeID.Int64(),
		Type:      params.DocType,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	if resp.Msg.Document == nil {
		return map[string]any{
			"change_id": params.ChangeID,
			"doc_type":  params.DocType,
			"content":   "",
			"exists":    false,
		}, nil
	}

	d := resp.Msg.Document
	content := d.Content
	if params.DocType == "proposal" {
		converted, err := convertProposalToMarkdown(d.Content)
		if err == nil {
			content = converted
		}
	} else {
		exported, err := exportDocumentToMarkdown(d.Content)
		if err != nil {
			return nil, err
		}
		content = exported
	}
	return map[string]any{
		"id":        d.Id,
		"change_id": d.ChangeId,
		"doc_type":  d.Type,
		"title":     d.Title,
		"content":   content,
		"version":   d.Version,
		"exists":    true,
	}, nil
}

func (s *Server) handleWriteSpec(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		DocType   string    `json:"doc_type"`
		Content   string    `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// For non-proposal doc types, convert markdown to HTML and route through Hocuspocus
	// for real-time sync with web editors via Y.js CRDT
	if params.DocType != "proposal" {
		content, err := markdownToHTML(params.Content)
		if err != nil {
			return nil, err
		}

		if s.clients.hocuspocusURL != "" {
			if err := s.updateViaHocuspocus(params.ChangeID.Int64(), params.DocType, content); err != nil {
				log.Printf("hocuspocus update failed: %v", err)
			} else {
				s.publishEvent("document_updated", params.ChangeID.Int64(), map[string]any{
					"docType": params.DocType,
				})
				return map[string]any{
					"change_id": params.ChangeID.Int64(),
					"doc_type":  params.DocType,
					"saved":     true,
					"via":       "hocuspocus",
				}, nil
			}
		}

		return nil, fmt.Errorf("write_spec for %q requires hocuspocus to persist ProseMirror JSON", params.DocType)
	}

	// Proposal: direct DB save (JSON format, not TipTap)
	resp, err := s.clients.document.SaveDocument(ctx, connect.NewRequest(&documentv1.SaveDocumentRequest{
		ChangeId:  params.ChangeID.Int64(),
		Type:      params.DocType,
		Title:     params.DocType,
		Content:   params.Content,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	d := resp.Msg.Document
	s.publishEvent("document_updated", params.ChangeID.Int64(), map[string]any{
		"docType": params.DocType,
		"version": d.Version,
	})
	return map[string]any{
		"id":      d.Id,
		"version": d.Version,
		"saved":   true,
	}, nil
}

// updateViaHocuspocus sends an HTML document update to the Hocuspocus REST API,
// which applies it as a Y.js CRDT update for real-time sync with web editors.
func (s *Server) updateViaHocuspocus(changeID int64, docType, htmlContent string) error {
	documentName := fmt.Sprintf("change-%d-%s", changeID, docType)

	body, err := json.Marshal(map[string]string{
		"document_name": documentName,
		"content":       htmlContent,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.clients.hocuspocusURL+"/api/documents", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.clients.hocuspocusAPISecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("hocuspocus request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hocuspocus returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *Server) publishEvent(eventType string, changeID int64, data any) {
	if s.clients.eventHub == nil {
		return
	}
	payload, _ := json.Marshal(data)
	s.clients.eventHub.Publish(events.Event{
		Type:     eventType,
		ChangeID: changeID,
		Payload:  string(payload),
	})
}

func (s *Server) handleCreateTask(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID    FlexInt64 `json:"change_id"`
		ProjectID   FlexInt64 `json:"project_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Status == "" {
		params.Status = "todo"
	}

	resp, err := s.clients.task.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		ChangeId:    params.ChangeID.Int64(),
		ProjectId:   params.ProjectID.Int64(),
		Title:       params.Title,
		Description: params.Description,
		Status:      params.Status,
	}))
	if err != nil {
		return nil, err
	}

	t := resp.Msg.Task
	result := map[string]any{
		"id":            t.Id,
		"title":         t.Title,
		"status":        t.Status,
		"assignee_id":   t.AssigneeId,
		"assignee_name": t.AssigneeName,
	}

	s.publishEvent("task_created", params.ChangeID.Int64(), result)

	return result, nil
}

func (s *Server) handleListTasks(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	type taskInfo struct {
		ID           int64  `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Status       string `json:"status"`
		OrderIndex   int32  `json:"order_index"`
		SpecRef      string `json:"spec_ref,omitempty"`
		AssigneeID   *int64 `json:"assignee_id"`
		AssigneeName string `json:"assignee_name"`
	}

	tasks := make([]taskInfo, len(resp.Msg.Tasks))
	for i, t := range resp.Msg.Tasks {
		tasks[i] = taskInfo{
			ID:           t.Id,
			Title:        t.Title,
			Description:  t.Description,
			Status:       t.Status,
			OrderIndex:   t.OrderIndex,
			SpecRef:      t.SpecRef,
			AssigneeID:   t.AssigneeId,
			AssigneeName: t.AssigneeName,
		}
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"tasks":     tasks,
		"total":     len(tasks),
	}, nil
}

func (s *Server) handleUpdateTask(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		TaskID        FlexInt64  `json:"task_id"`
		ProjectID     FlexInt64  `json:"project_id"`
		Status        string     `json:"status"`
		AssigneeID    *FlexInt64 `json:"assignee_id"`
		ClearAssignee bool       `json:"clear_assignee"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	req := &taskv1.UpdateTaskRequest{
		Id:        params.TaskID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}
	if params.Status != "" {
		req.Status = &params.Status
	}
	if params.AssigneeID != nil {
		req.AssigneeId = params.AssigneeID.Int64Ptr()
	}
	if params.ClearAssignee {
		req.ClearAssignee = true
	}

	resp, err := s.clients.task.UpdateTask(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}

	t := resp.Msg.Task
	result := map[string]any{
		"id":            t.Id,
		"title":         t.Title,
		"status":        t.Status,
		"assignee_id":   t.AssigneeId,
		"assignee_name": t.AssigneeName,
	}

	s.publishEvent("task_updated", t.ChangeId, result)

	return result, nil
}

func (s *Server) handleSuggestSpec(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		DocType   string    `json:"doc_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Read the current document first
	resp, err := s.clients.document.GetDocument(ctx, connect.NewRequest(&documentv1.GetDocumentRequest{
		ChangeId:  params.ChangeID.Int64(),
		Type:      params.DocType,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	if resp.Msg.Document == nil || resp.Msg.Document.Content == "" {
		return map[string]any{
			"suggestion": "No document found. Create the document first using write_spec.",
		}, nil
	}

	currentContent := resp.Msg.Document.Content
	if params.DocType != "proposal" {
		exported, err := exportDocumentToMarkdown(resp.Msg.Document.Content)
		if err != nil {
			return nil, err
		}
		currentContent = exported
	}

	// Return the current content for the AI client to analyze and suggest improvements
	return map[string]any{
		"current_content": currentContent,
		"doc_type":        params.DocType,
		"suggestion":      "Review the current content and suggest improvements based on the document type. For proposals, ensure Problem, Scope, and Out of Scope sections are clear. For specs, ensure architecture decisions and implementation steps are well-defined.",
	}, nil
}

func (s *Server) handleListAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	type stepInfo struct {
		Keyword string `json:"keyword"`
		Text    string `json:"text"`
	}
	type acInfo struct {
		ID       int64      `json:"id"`
		Scenario string     `json:"scenario"`
		Steps    []stepInfo `json:"steps"`
		Met      bool       `json:"met"`
		TestRef  string     `json:"test_ref"`
	}

	items := make([]acInfo, len(resp.Msg.Criteria))
	for i, ac := range resp.Msg.Criteria {
		steps := make([]stepInfo, len(ac.Steps))
		for j, s := range ac.Steps {
			steps[j] = stepInfo{Keyword: s.Keyword, Text: s.Text}
		}
		items[i] = acInfo{
			ID:       ac.Id,
			Scenario: ac.Scenario,
			Steps:    steps,
			Met:      ac.Met,
			TestRef:  ac.TestRef,
		}
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"criteria":  items,
		"total":     len(items),
	}, nil
}

func (s *Server) handleCreateAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		Scenario  string    `json:"scenario"`
		Steps     string    `json:"steps"`
		TestRef   string    `json:"test_ref"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	var steps []struct {
		Keyword string `json:"keyword"`
		Text    string `json:"text"`
	}
	if err := json.Unmarshal([]byte(params.Steps), &steps); err != nil {
		return nil, fmt.Errorf("invalid steps JSON: %w", err)
	}

	protoSteps := make([]*acceptancev1.ACStep, len(steps))
	for i, s := range steps {
		protoSteps[i] = &acceptancev1.ACStep{Keyword: s.Keyword, Text: s.Text}
	}

	resp, err := s.clients.acceptance.CreateAC(ctx, connect.NewRequest(&acceptancev1.CreateACRequest{
		ChangeId:  params.ChangeID.Int64(),
		Scenario:  params.Scenario,
		Steps:     protoSteps,
		TestRef:   params.TestRef,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	ac := resp.Msg.Criteria
	return map[string]any{
		"id":       ac.Id,
		"scenario": ac.Scenario,
		"test_ref": ac.TestRef,
		"created":  true,
	}, nil
}

func (s *Server) handleToggleAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ID        FlexInt64 `json:"id"`
		ProjectID FlexInt64 `json:"project_id"`
		Met       bool      `json:"met"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.acceptance.ToggleAC(ctx, connect.NewRequest(&acceptancev1.ToggleACRequest{
		Id:        params.ID.Int64(),
		Met:       params.Met,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	ac := resp.Msg.Criteria
	return map[string]any{
		"id":       ac.Id,
		"scenario": ac.Scenario,
		"met":      ac.Met,
		"test_ref": ac.TestRef,
		"toggled":  true,
	}, nil
}

func (s *Server) handleLinkACToTest(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ACID      FlexInt64 `json:"ac_id"`
		ProjectID FlexInt64 `json:"project_id"`
		TestRef   string    `json:"test_ref"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.acceptance.UpdateAC(ctx, connect.NewRequest(&acceptancev1.UpdateACRequest{
		Id:        params.ACID.Int64(),
		TestRef:   params.TestRef,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	ac := resp.Msg.Criteria
	return map[string]any{
		"id":       ac.Id,
		"scenario": ac.Scenario,
		"test_ref": ac.TestRef,
		"linked":   true,
	}, nil
}

func (s *Server) handleUpdateProject(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID   FlexInt64 `json:"project_id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Readme      *string   `json:"readme"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	req := &projectv1.UpdateProjectRequest{
		Id:          params.ProjectID.Int64(),
		Name:        params.Name,
		Description: params.Description,
		ProjectId:   params.ProjectID.Int64(),
	}
	if params.Readme != nil {
		// Convert markdown to HTML for Tiptap editor
		html, err := markdownToHTML(*params.Readme)
		if err != nil {
			return nil, fmt.Errorf("failed to convert markdown: %w", err)
		}
		req.Readme = &html
	}

	resp, err := s.clients.project.UpdateProject(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}

	p := resp.Msg.Project
	return map[string]any{
		"id":          p.Id,
		"name":        p.Name,
		"description": p.Description,
		"readme":      p.Readme,
		"updated":     true,
	}, nil
}

func (s *Server) handleCreateChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID FlexInt64 `json:"project_id"`
		Name      string    `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.CreateChange(ctx, connect.NewRequest(&projectv1.CreateChangeRequest{
		ProjectId: params.ProjectID.Int64(),
		Name:      params.Name,
	}))
	if err != nil {
		return nil, err
	}

	c := resp.Msg.Change
	return map[string]any{
		"id":         c.Id,
		"name":       c.Name,
		"number":     c.Number,
		"identifier": c.Identifier,
	}, nil
}

func (s *Server) handleListChanges(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.ListChanges(ctx, connect.NewRequest(&projectv1.ListChangesRequest{
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	type progressInfo struct {
		Total int `json:"total"`
		Done  int `json:"done"`
	}
	type changeInfo struct {
		ID           int64        `json:"id"`
		Name         string       `json:"name"`
		Identifier   string       `json:"identifier"`
		Stage        string       `json:"stage"`
		SubStatus    string       `json:"sub_status,omitempty"`
		TaskProgress progressInfo `json:"task_progress"`
		ACProgress   progressInfo `json:"ac_progress"`
	}

	changes := make([]changeInfo, len(resp.Msg.Changes))
	for i, c := range resp.Msg.Changes {
		subStatus := c.SubStatus
		if c.Stage == "approved" {
			subStatus = ""
		}
		ci := changeInfo{ID: c.Id, Name: c.Name, Identifier: c.Identifier, Stage: c.Stage, SubStatus: subStatus}

		if taskResp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{ChangeId: c.Id, ProjectId: params.ProjectID.Int64()})); err == nil {
			ci.TaskProgress.Total = len(taskResp.Msg.Tasks)
			for _, t := range taskResp.Msg.Tasks {
				if t.Status == "done" {
					ci.TaskProgress.Done++
				}
			}
		}
		if acResp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{ChangeId: c.Id, ProjectId: params.ProjectID.Int64()})); err == nil {
			ci.ACProgress.Total = len(acResp.Msg.Criteria)
			for _, ac := range acResp.Msg.Criteria {
				if ac.Met {
					ci.ACProgress.Done++
				}
			}
		}

		changes[i] = ci
	}

	return map[string]any{
		"project_id": params.ProjectID.Int64(),
		"changes":    changes,
		"total":      len(changes),
	}, nil
}

func (s *Server) handleListComments(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID     FlexInt64 `json:"change_id"`
		ProjectID    FlexInt64 `json:"project_id"`
		DocumentType string    `json:"document_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.ListComments(ctx, connect.NewRequest(&commentv1.ListCommentsRequest{
		ChangeId:     params.ChangeID.Int64(),
		ProjectId:    params.ProjectID.Int64(),
		DocumentType: params.DocumentType,
	}))
	if err != nil {
		return nil, err
	}

	type commentInfo struct {
		ID         int64  `json:"id"`
		Content    string `json:"content"`
		AuthorName string `json:"author_name"`
	}
	comments := make([]commentInfo, len(resp.Msg.Comments))
	for i, c := range resp.Msg.Comments {
		comments[i] = commentInfo{ID: c.Id, Content: c.Body, AuthorName: c.UserName}
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"comments":  comments,
		"total":     len(comments),
	}, nil
}

func (s *Server) handleCreateComment(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID     FlexInt64 `json:"change_id"`
		ProjectID    FlexInt64 `json:"project_id"`
		Content      string    `json:"content"`
		DocumentType string    `json:"document_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.CreateComment(ctx, connect.NewRequest(&commentv1.CreateCommentRequest{
		ChangeId:     params.ChangeID.Int64(),
		Body:         params.Content,
		ProjectId:    params.ProjectID.Int64(),
		DocumentType: params.DocumentType,
	}))
	if err != nil {
		return nil, err
	}

	c := resp.Msg.Comment
	return map[string]any{
		"id":      c.Id,
		"content": c.Body,
		"created": true,
	}, nil
}

func (s *Server) handleDeleteTask(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		TaskID    FlexInt64 `json:"task_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	_, err := s.clients.task.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{
		Id:        params.TaskID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"task_id": params.TaskID.Int64(),
		"deleted": true,
	}, nil
}

func (s *Server) handleGetMemory(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.memory.GetMemory(ctx, connect.NewRequest(&memoryv1.GetMemoryRequest{
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	if resp.Msg.Memory == nil {
		return map[string]any{
			"project_id": params.ProjectID.Int64(),
			"content":    "",
			"exists":     false,
		}, nil
	}

	return map[string]any{
		"project_id": params.ProjectID.Int64(),
		"content":    resp.Msg.Memory.Content,
		"exists":     true,
	}, nil
}

func (s *Server) handleSaveMemory(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID FlexInt64 `json:"project_id"`
		Content   string    `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	_, err := s.clients.memory.SaveMemory(ctx, connect.NewRequest(&memoryv1.SaveMemoryRequest{
		ProjectId: params.ProjectID.Int64(),
		Content:   params.Content,
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"project_id": params.ProjectID.Int64(),
		"saved":      true,
	}, nil
}

func (s *Server) handleGetChangeHistory(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.workflow.GetHistory(ctx, connect.NewRequest(&workflowv1.GetHistoryRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	type eventInfo struct {
		ID        int64  `json:"id"`
		FromStage string `json:"from_stage"`
		ToStage   string `json:"to_stage"`
		Action    string `json:"action"`
		Reason    string `json:"reason"`
		UserName  string `json:"user_name"`
		CreatedAt string `json:"created_at"`
	}

	events := make([]eventInfo, len(resp.Msg.Events))
	for i, e := range resp.Msg.Events {
		createdAt := ""
		if e.CreatedAt != nil {
			createdAt = e.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z")
		}
		events[i] = eventInfo{
			ID:        e.Id,
			FromStage: e.FromStage,
			ToStage:   e.ToStage,
			Action:    e.Action,
			Reason:    e.Reason,
			UserName:  e.UserName,
			CreatedAt: createdAt,
		}
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"events":    events,
		"total":     len(events),
	}, nil
}

func (s *Server) handleGetChangeSummary(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID FlexInt64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	changeResp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{Id: params.ChangeID.Int64()}))
	if err != nil {
		return nil, err
	}
	c := changeResp.Msg.Change

	result := map[string]any{
		"change_id":  c.Id,
		"name":       c.Name,
		"stage":      c.Stage,
		"sub_status": c.SubStatus,
	}

	// Task progress
	taskProgress := map[string]int{"total": 0, "todo": 0, "in_progress": 0, "done": 0}
	if taskResp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
		taskProgress["total"] = len(taskResp.Msg.Tasks)
		for _, t := range taskResp.Msg.Tasks {
			taskProgress[t.Status]++
		}
	}
	result["task_progress"] = taskProgress

	// AC progress
	acProgress := map[string]int{"total": 0, "met": 0, "unmet": 0}
	if acResp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
		acProgress["total"] = len(acResp.Msg.Criteria)
		for _, ac := range acResp.Msg.Criteria {
			if ac.Met {
				acProgress["met"]++
			} else {
				acProgress["unmet"]++
			}
		}
	}
	result["ac_progress"] = acProgress

	// Gate conditions
	if statusResp, err := s.clients.workflow.GetStatus(ctx, connect.NewRequest(&workflowv1.GetStatusRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
		result["gate_conditions"] = statusResp.Msg.Conditions
	}

	return result, nil
}

func (s *Server) handleGetProjectDashboard(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Get project name — list_projects to find slug, then get project
	listResp, err := s.clients.project.ListProjects(ctx, connect.NewRequest(&projectv1.ListProjectsRequest{}))
	if err != nil {
		return nil, err
	}
	var projectName string
	for _, p := range listResp.Msg.Projects {
		if p.Id == params.ProjectID.Int64() {
			projectName = p.Name
			break
		}
	}

	changesResp, err := s.clients.project.ListChanges(ctx, connect.NewRequest(&projectv1.ListChangesRequest{ProjectId: params.ProjectID.Int64()}))
	if err != nil {
		return nil, err
	}

	type changeSummary struct {
		ID           int64          `json:"id"`
		Name         string         `json:"name"`
		Stage        string         `json:"stage"`
		TaskProgress map[string]int `json:"task_progress"`
		ACProgress   map[string]int `json:"ac_progress"`
	}

	var changes []changeSummary
	for _, c := range changesResp.Msg.Changes {
		// Skip archived changes
		if c.ArchivedAt != nil {
			continue
		}

		cs := changeSummary{
			ID:           c.Id,
			Name:         c.Name,
			Stage:        c.Stage,
			TaskProgress: map[string]int{"total": 0, "done": 0},
			ACProgress:   map[string]int{"total": 0, "met": 0},
		}

		if taskResp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{ChangeId: c.Id, ProjectId: params.ProjectID.Int64()})); err == nil {
			cs.TaskProgress["total"] = len(taskResp.Msg.Tasks)
			for _, t := range taskResp.Msg.Tasks {
				if t.Status == "done" {
					cs.TaskProgress["done"]++
				}
			}
		}
		if acResp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{ChangeId: c.Id, ProjectId: params.ProjectID.Int64()})); err == nil {
			cs.ACProgress["total"] = len(acResp.Msg.Criteria)
			for _, ac := range acResp.Msg.Criteria {
				if ac.Met {
					cs.ACProgress["met"]++
				}
			}
		}

		changes = append(changes, cs)
	}

	return map[string]any{
		"project_id":    params.ProjectID.Int64(),
		"project_name":  projectName,
		"changes":       changes,
		"total_changes": len(changes),
	}, nil
}

func (s *Server) handleGetGateStatus(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.workflow.GetStatus(ctx, connect.NewRequest(&workflowv1.GetStatusRequest{ChangeId: params.ChangeID.Int64(), ProjectId: params.ProjectID.Int64()}))
	if err != nil {
		return nil, err
	}

	canAdvance := true
	for _, c := range resp.Msg.Conditions {
		if !c.Met {
			canAdvance = false
			break
		}
	}

	return map[string]any{
		"change_id":     params.ChangeID.Int64(),
		"current_stage": resp.Msg.Stage,
		"sub_status":    resp.Msg.SubStatus,
		"conditions":    resp.Msg.Conditions,
		"can_advance":   canAdvance,
	}, nil
}

func (s *Server) handleApproveChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		Comment   string    `json:"comment"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.workflow.Approve(ctx, connect.NewRequest(&workflowv1.ApproveRequest{
		ChangeId:  params.ChangeID.Int64(),
		Comment:   params.Comment,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"approved":  true,
		"advanced":  resp.Msg.Advanced,
		"new_stage": resp.Msg.NewStage,
	}, nil
}

func (s *Server) handleRejectChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		Reason    string    `json:"reason"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	resp, err := s.clients.workflow.RequestChanges(ctx, connect.NewRequest(&workflowv1.RequestChangesRequest{
		ChangeId:  params.ChangeID.Int64(),
		Reason:    params.Reason,
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"rejected":  true,
		"new_stage": resp.Msg.NewStage,
	}, nil
}

func (s *Server) handleUpdateChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params updateChangeParams
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Name == "" && params.SubStatus == "" && params.Status == "" {
		return nil, fmt.Errorf("at least one of name, sub_status, or status must be provided")
	}
	if params.SubStatus != "" && params.Status != "" {
		return nil, fmt.Errorf("sub_status and status cannot be provided together")
	}

	var currentChange *projectv1.Change
	var originalName string
	if params.Status != "" || (params.Name != "" && params.SubStatus != "") {
		changeResp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{
			Id:        params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
		}))
		if err != nil {
			return nil, err
		}
		currentChange = changeResp.Msg.Change
		originalName = currentChange.Name
	}

	result := map[string]any{
		"id":      params.ChangeID.Int64(),
		"updated": true,
	}

	if params.Name != "" {
		resp, err := s.clients.project.UpdateChange(ctx, connect.NewRequest(&projectv1.UpdateChangeRequest{
			Id:        params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
			Name:      params.Name,
		}))
		if err != nil {
			return nil, err
		}
		result["name"] = resp.Msg.Change.Name
		result["identifier"] = resp.Msg.Change.Identifier
	}

	if params.SubStatus != "" || params.Status != "" {
		stage, subStatus, err := s.applyChangeStatusUpdate(ctx, params, currentChange)
		if err != nil {
			if params.Name != "" && originalName != "" {
				_, rollbackErr := s.clients.project.UpdateChange(ctx, connect.NewRequest(&projectv1.UpdateChangeRequest{
					Id:        params.ChangeID.Int64(),
					ProjectId: params.ProjectID.Int64(),
					Name:      originalName,
				}))
				if rollbackErr != nil {
					return nil, fmt.Errorf("status update failed: %w; rollback failed: %v", err, rollbackErr)
				}
			}
			return nil, err
		}
		result["stage"] = string(stage)
		if stage != models.StageApproved {
			result["sub_status"] = string(subStatus)
		}
		result["status"] = formatCompositeChangeStatus(stage, subStatus)
	}

	return result, nil
}

type updateChangeParams struct {
	ChangeID     FlexInt64 `json:"change_id"`
	ProjectID    FlexInt64 `json:"project_id"`
	Name         string    `json:"name"`
	SubStatus    string    `json:"sub_status"`
	Status       string    `json:"status"`
	StatusReason string    `json:"status_reason"`
}

func (s *Server) applyChangeStatusUpdate(ctx context.Context, params updateChangeParams, currentChange *projectv1.Change) (models.ChangeStage, models.SubStatus, error) {
	if params.Status == "" {
		if currentChange == nil {
			changeResp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{
				Id:        params.ChangeID.Int64(),
				ProjectId: params.ProjectID.Int64(),
			}))
			if err != nil {
				return "", "", err
			}
			currentChange = changeResp.Msg.Change
		}
		_, err := s.clients.workflow.SetSubStatus(ctx, connect.NewRequest(&workflowv1.SetSubStatusRequest{
			ChangeId:  params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
			SubStatus: params.SubStatus,
		}))
		return models.ChangeStage(currentChange.Stage), models.SubStatus(params.SubStatus), err
	}

	if currentChange == nil {
		changeResp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{
			Id:        params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
		}))
		if err != nil {
			return "", "", err
		}
		currentChange = changeResp.Msg.Change
	}

	targetStage, targetSubStatus, err := parseCompositeChangeStatus(params.Status)
	if err != nil {
		return "", "", err
	}

	currentStage := models.ChangeStage(currentChange.Stage)
	currentSubStatus := models.SubStatus(currentChange.SubStatus)
	revertReason := params.StatusReason
	if revertReason == "" {
		revertReason = fmt.Sprintf("set status via MCP to %s", params.Status)
	}

	for currentStage != targetStage {
		if stageRank(currentStage) < stageRank(targetStage) {
			resp, err := s.clients.workflow.Advance(ctx, connect.NewRequest(&workflowv1.AdvanceRequest{
				ChangeId:  params.ChangeID.Int64(),
				ProjectId: params.ProjectID.Int64(),
			}))
			if err != nil {
				return "", "", err
			}
			currentStage = models.ChangeStage(resp.Msg.NewStage)
			currentSubStatus = models.SubStatusInProgress
			continue
		}

		resp, err := s.clients.workflow.Revert(ctx, connect.NewRequest(&workflowv1.RevertRequest{
			ChangeId:  params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
			Reason:    revertReason,
		}))
		if err != nil {
			return "", "", err
		}
		currentStage = models.ChangeStage(resp.Msg.NewStage)
		currentSubStatus = models.SubStatusInProgress
	}

	if targetStage != models.StageApproved && currentSubStatus != targetSubStatus {
		_, err := s.clients.workflow.SetSubStatus(ctx, connect.NewRequest(&workflowv1.SetSubStatusRequest{
			ChangeId:  params.ChangeID.Int64(),
			ProjectId: params.ProjectID.Int64(),
			SubStatus: string(targetSubStatus),
		}))
		if err != nil {
			return "", "", err
		}
		currentSubStatus = targetSubStatus
	}

	return currentStage, currentSubStatus, nil
}

func parseCompositeChangeStatus(raw string) (models.ChangeStage, models.SubStatus, error) {
	switch raw {
	case "draft(in_progress)":
		return models.StageDraft, models.SubStatusInProgress, nil
	case "draft(ready)":
		return models.StageDraft, models.SubStatusReady, nil
	case "spec(in_progress)":
		return models.StageSpec, models.SubStatusInProgress, nil
	case "spec(ready)":
		return models.StageSpec, models.SubStatusReady, nil
	case "approved":
		return "", "", fmt.Errorf("status %q is not supported by update_change; use approve_change to move a change to approved", raw)
	default:
		return "", "", fmt.Errorf("invalid status %q; expected one of draft(in_progress), draft(ready), spec(in_progress), spec(ready)", raw)
	}
}

func formatCompositeChangeStatus(stage models.ChangeStage, subStatus models.SubStatus) string {
	if stage == models.StageApproved {
		return string(models.StageApproved)
	}
	return fmt.Sprintf("%s(%s)", stage, subStatus)
}

func stageRank(stage models.ChangeStage) int {
	switch stage {
	case models.StageDraft:
		return 0
	case models.StageSpec:
		return 1
	case models.StageApproved:
		return 2
	default:
		return -1
	}
}

func (s *Server) handleArchiveChange(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	_, err := s.clients.project.ArchiveChange(ctx, connect.NewRequest(&projectv1.ArchiveChangeRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"archived":  true,
	}, nil
}

func (s *Server) handleGetWorkContext(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID FlexInt64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// First get the change to obtain project_id
	changeResp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{Id: params.ChangeID.Int64()}))
	if err != nil {
		return nil, err
	}
	c := changeResp.Msg.Change

	result := map[string]any{
		"change": map[string]any{
			"id":    c.Id,
			"name":  c.Name,
			"stage": c.Stage,
		},
	}

	// Parallel fetch remaining data
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Gate conditions
	wg.Add(1)
	go func() {
		defer wg.Done()
		if resp, err := s.clients.workflow.GetStatus(ctx, connect.NewRequest(&workflowv1.GetStatusRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
			mu.Lock()
			result["gate_conditions"] = resp.Msg.Conditions
			mu.Unlock()
		}
	}()

	// Proposal
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := s.clients.document.GetDocument(ctx, connect.NewRequest(&documentv1.GetDocumentRequest{
			ChangeId:  params.ChangeID.Int64(),
			Type:      "proposal",
			ProjectId: c.ProjectId,
		}))
		if err == nil && resp.Msg.Document != nil && resp.Msg.Document.Content != "" {
			var proposal map[string]any
			if json.Unmarshal([]byte(resp.Msg.Document.Content), &proposal) == nil {
				convertProposalFieldsToMarkdown(proposal)
				mu.Lock()
				result["proposal"] = proposal
				mu.Unlock()
			}
		}
	}()

	// Tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		if resp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
			type taskInfo struct {
				ID           int64  `json:"id"`
				Title        string `json:"title"`
				Description  string `json:"description"`
				Status       string `json:"status"`
				AssigneeID   *int64 `json:"assignee_id"`
				AssigneeName string `json:"assignee_name"`
				OrderIndex   int32  `json:"order_index"`
			}
			tasks := make([]taskInfo, len(resp.Msg.Tasks))
			for i, t := range resp.Msg.Tasks {
				tasks[i] = taskInfo{
					ID: t.Id, Title: t.Title, Description: t.Description,
					Status: t.Status, AssigneeID: t.AssigneeId, AssigneeName: t.AssigneeName,
					OrderIndex: t.OrderIndex,
				}
			}
			mu.Lock()
			result["tasks"] = tasks
			mu.Unlock()
		}
	}()

	// Acceptance Criteria
	wg.Add(1)
	go func() {
		defer wg.Done()
		if resp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
			type stepInfo struct {
				Keyword string `json:"keyword"`
				Text    string `json:"text"`
			}
			type acInfo struct {
				ID       int64      `json:"id"`
				Scenario string     `json:"scenario"`
				Steps    []stepInfo `json:"steps"`
				Met      bool       `json:"met"`
				TestRef  string     `json:"test_ref"`
			}
			items := make([]acInfo, len(resp.Msg.Criteria))
			for i, ac := range resp.Msg.Criteria {
				steps := make([]stepInfo, len(ac.Steps))
				for j, st := range ac.Steps {
					steps[j] = stepInfo{Keyword: st.Keyword, Text: st.Text}
				}
				items[i] = acInfo{ID: ac.Id, Scenario: ac.Scenario, Steps: steps, Met: ac.Met, TestRef: ac.TestRef}
			}
			mu.Lock()
			result["acceptance_criteria"] = items
			mu.Unlock()
		}
	}()

	// Memory
	wg.Add(1)
	go func() {
		defer wg.Done()
		if resp, err := s.clients.memory.GetMemory(ctx, connect.NewRequest(&memoryv1.GetMemoryRequest{ProjectId: c.ProjectId})); err == nil && resp.Msg.Memory != nil {
			mu.Lock()
			result["memory"] = resp.Msg.Memory.Content
			mu.Unlock()
		}
	}()

	// Recent comments (last 5)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if resp, err := s.clients.comment.ListComments(ctx, connect.NewRequest(&commentv1.ListCommentsRequest{ChangeId: params.ChangeID.Int64(), ProjectId: c.ProjectId})); err == nil {
			type commentInfo struct {
				ID         int64  `json:"id"`
				Content    string `json:"content"`
				AuthorName string `json:"author_name"`
			}
			comments := resp.Msg.Comments
			// Take last 5 (newest)
			if len(comments) > 5 {
				comments = comments[len(comments)-5:]
			}
			items := make([]commentInfo, len(comments))
			for i, cm := range comments {
				items[i] = commentInfo{ID: cm.Id, Content: cm.Body, AuthorName: cm.UserName}
			}
			mu.Lock()
			result["recent_comments"] = items
			mu.Unlock()
		}
	}()

	wg.Wait()

	return result, nil
}

func (s *Server) handleGetMe(ctx context.Context) (any, error) {
	res, err := s.clients.auth.Me(ctx, connect.NewRequest(&authv1.MeRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return map[string]any{
		"user_id":    res.Msg.UserId,
		"name":       res.Msg.Name,
		"email":      res.Msg.Email,
		"org_id":     res.Msg.OrgId,
		"avatar_url": res.Msg.AvatarUrl,
	}, nil
}

func (s *Server) handleListMembers(ctx context.Context) (any, error) {
	res, err := s.clients.organization.ListMembers(ctx, connect.NewRequest(&organizationv1.ListMembersRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	members := make([]map[string]any, 0, len(res.Msg.Members))
	for _, m := range res.Msg.Members {
		members = append(members, map[string]any{
			"user_id": m.UserId,
			"name":    m.UserName,
			"email":   m.UserEmail,
			"role":    m.Role,
		})
	}

	return map[string]any{
		"members": members,
	}, nil
}
