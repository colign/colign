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
)

func (s *Server) callTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "list_projects":
		return s.handleListProjects(ctx)
	case "get_change":
		return s.handleGetChange(ctx, args)
	case "read_spec":
		return s.handleReadSpec(ctx, args)
	case "write_spec":
		return s.handleWriteSpec(ctx, args)
	case "create_task":
		return s.handleCreateTask(ctx, args)
	case "list_tasks":
		return s.handleListTasks(ctx, args)
	case "update_task":
		return s.handleUpdateTask(ctx, args)
	case "suggest_spec":
		return s.handleSuggestSpec(ctx, args)
	case "list_acceptance_criteria":
		return s.handleListAC(ctx, args)
	case "create_acceptance_criteria":
		return s.handleCreateAC(ctx, args)
	case "toggle_acceptance_criteria":
		return s.handleToggleAC(ctx, args)
	case "create_project":
		return s.handleCreateProject(ctx, args)
	case "update_project":
		return s.handleUpdateProject(ctx, args)
	case "create_change":
		return s.handleCreateChange(ctx, args)
	case "list_changes":
		return s.handleListChanges(ctx, args)
	case "advance_stage":
		return s.handleAdvanceStage(ctx, args)
	case "list_comments":
		return s.handleListComments(ctx, args)
	case "create_comment":
		return s.handleCreateComment(ctx, args)
	case "delete_task":
		return s.handleDeleteTask(ctx, args)
	case "get_memory":
		return s.handleGetMemory(ctx, args)
	case "save_memory":
		return s.handleSaveMemory(ctx, args)
	case "get_change_history":
		return s.handleGetChangeHistory(ctx, args)
	case "link_ac_to_test":
		return s.handleLinkACToTest(ctx, args)
	case "get_change_summary":
		return s.handleGetChangeSummary(ctx, args)
	case "get_project_dashboard":
		return s.handleGetProjectDashboard(ctx, args)
	case "get_gate_status":
		return s.handleGetGateStatus(ctx, args)
	case "approve_change":
		return s.handleApproveChange(ctx, args)
	case "reject_change":
		return s.handleRejectChange(ctx, args)
	case "archive_change":
		return s.handleArchiveChange(ctx, args)
	case "get_work_context":
		return s.handleGetWorkContext(ctx, args)
	case "get_me":
		return s.handleGetMe(ctx)
	case "list_members":
		return s.handleListMembers(ctx)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
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
	if params.DocType != "proposal" {
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
		"suggestion":      "Review the current content and suggest improvements based on the document type. For proposals, ensure Problem, Scope, and Approach sections are clear. For designs, ensure architecture decisions and implementation steps are well-defined.",
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
		"id":   c.Id,
		"name": c.Name,
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
		Stage        string       `json:"stage"`
		TaskProgress progressInfo `json:"task_progress"`
		ACProgress   progressInfo `json:"ac_progress"`
	}

	changes := make([]changeInfo, len(resp.Msg.Changes))
	for i, c := range resp.Msg.Changes {
		ci := changeInfo{ID: c.Id, Name: c.Name, Stage: c.Stage}

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

func (s *Server) handleAdvanceStage(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.workflow.Advance(ctx, connect.NewRequest(&workflowv1.AdvanceRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"change_id": params.ChangeID.Int64(),
		"new_stage": resp.Msg.NewStage,
		"advanced":  true,
	}, nil
}

func (s *Server) handleListComments(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.ListComments(ctx, connect.NewRequest(&commentv1.ListCommentsRequest{
		ChangeId:  params.ChangeID.Int64(),
		ProjectId: params.ProjectID.Int64(),
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
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
		Content   string    `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.CreateComment(ctx, connect.NewRequest(&commentv1.CreateCommentRequest{
		ChangeId:  params.ChangeID.Int64(),
		Body:      params.Content,
		ProjectId: params.ProjectID.Int64(),
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
		"change_id": c.Id,
		"name":      c.Name,
		"stage":     c.Stage,
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
