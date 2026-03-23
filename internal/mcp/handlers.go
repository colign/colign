package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/yuin/goldmark"

	acceptancev1 "github.com/gobenpark/colign/gen/proto/acceptance/v1"
	commentv1 "github.com/gobenpark/colign/gen/proto/comment/v1"
	documentv1 "github.com/gobenpark/colign/gen/proto/document/v1"
	memoryv1 "github.com/gobenpark/colign/gen/proto/memory/v1"
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
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
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
		ChangeID int64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.GetChange(ctx, connect.NewRequest(&projectv1.GetChangeRequest{
		Id: params.ChangeID,
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
		ChangeID int64  `json:"change_id"`
		DocType  string `json:"doc_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.document.GetDocument(ctx, connect.NewRequest(&documentv1.GetDocumentRequest{
		ChangeId: params.ChangeID,
		Type:     params.DocType,
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
	return map[string]any{
		"id":        d.Id,
		"change_id": d.ChangeId,
		"doc_type":  d.Type,
		"title":     d.Title,
		"content":   d.Content,
		"version":   d.Version,
		"exists":    true,
	}, nil
}

func (s *Server) handleWriteSpec(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64  `json:"change_id"`
		DocType  string `json:"doc_type"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// For non-proposal doc types, convert markdown to HTML and route through Hocuspocus
	// for real-time sync with web editors via Y.js CRDT
	if params.DocType != "proposal" {
		content := markdownToHTML(params.Content)

		if s.clients.hocuspocusURL != "" {
			if err := s.updateViaHocuspocus(params.ChangeID, params.DocType, content); err != nil {
				log.Printf("hocuspocus update failed, falling back to direct save: %v", err)
			} else {
				return map[string]any{
					"change_id": params.ChangeID,
					"doc_type":  params.DocType,
					"saved":     true,
					"via":       "hocuspocus",
				}, nil
			}
		}

		// Fallback: direct DB save
		resp, err := s.clients.document.SaveDocument(ctx, connect.NewRequest(&documentv1.SaveDocumentRequest{
			ChangeId: params.ChangeID,
			Type:     params.DocType,
			Title:    params.DocType,
			Content:  content,
		}))
		if err != nil {
			return nil, err
		}
		d := resp.Msg.Document
		return map[string]any{
			"id":      d.Id,
			"version": d.Version,
			"saved":   true,
		}, nil
	}

	// Proposal: direct DB save (JSON format, not TipTap)
	resp, err := s.clients.document.SaveDocument(ctx, connect.NewRequest(&documentv1.SaveDocumentRequest{
		ChangeId: params.ChangeID,
		Type:     params.DocType,
		Title:    params.DocType,
		Content:  params.Content,
	}))
	if err != nil {
		return nil, err
	}

	d := resp.Msg.Document
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
		ChangeID    int64  `json:"change_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Status == "" {
		params.Status = "todo"
	}

	resp, err := s.clients.task.CreateTask(ctx, connect.NewRequest(&taskv1.CreateTaskRequest{
		ChangeId:    params.ChangeID,
		Title:       params.Title,
		Description: params.Description,
		Status:      params.Status,
	}))
	if err != nil {
		return nil, err
	}

	t := resp.Msg.Task
	result := map[string]any{
		"id":     t.Id,
		"title":  t.Title,
		"status": t.Status,
	}

	s.publishEvent("task_created", params.ChangeID, result)

	return result, nil
}

func (s *Server) handleListTasks(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.task.ListTasks(ctx, connect.NewRequest(&taskv1.ListTasksRequest{
		ChangeId: params.ChangeID,
	}))
	if err != nil {
		return nil, err
	}

	type taskInfo struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		OrderIndex  int32  `json:"order_index"`
		SpecRef     string `json:"spec_ref,omitempty"`
	}

	tasks := make([]taskInfo, len(resp.Msg.Tasks))
	for i, t := range resp.Msg.Tasks {
		tasks[i] = taskInfo{
			ID:          t.Id,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			OrderIndex:  t.OrderIndex,
			SpecRef:     t.SpecRef,
		}
	}

	return map[string]any{
		"change_id": params.ChangeID,
		"tasks":     tasks,
		"total":     len(tasks),
	}, nil
}

func (s *Server) handleUpdateTask(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		TaskID int64  `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.task.UpdateTask(ctx, connect.NewRequest(&taskv1.UpdateTaskRequest{
		Id:     params.TaskID,
		Status: &params.Status,
	}))
	if err != nil {
		return nil, err
	}

	t := resp.Msg.Task
	result := map[string]any{
		"id":     t.Id,
		"title":  t.Title,
		"status": t.Status,
	}

	s.publishEvent("task_updated", t.ChangeId, result)

	return result, nil
}

func (s *Server) handleSuggestSpec(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64  `json:"change_id"`
		DocType  string `json:"doc_type"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Read the current document first
	resp, err := s.clients.document.GetDocument(ctx, connect.NewRequest(&documentv1.GetDocumentRequest{
		ChangeId: params.ChangeID,
		Type:     params.DocType,
	}))
	if err != nil {
		return nil, err
	}

	if resp.Msg.Document == nil || resp.Msg.Document.Content == "" {
		return map[string]any{
			"suggestion": "No document found. Create the document first using write_spec.",
		}, nil
	}

	// Return the current content for the AI client to analyze and suggest improvements
	return map[string]any{
		"current_content": resp.Msg.Document.Content,
		"doc_type":        params.DocType,
		"suggestion":      "Review the current content and suggest improvements based on the document type. For proposals, ensure Problem, Scope, and Approach sections are clear. For designs, ensure architecture decisions and implementation steps are well-defined.",
	}, nil
}

func (s *Server) handleListAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.acceptance.ListAC(ctx, connect.NewRequest(&acceptancev1.ListACRequest{
		ChangeId: params.ChangeID,
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
		}
	}

	return map[string]any{
		"change_id": params.ChangeID,
		"criteria":  items,
		"total":     len(items),
	}, nil
}

func (s *Server) handleCreateAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64  `json:"change_id"`
		Scenario string `json:"scenario"`
		Steps    string `json:"steps"`
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
		ChangeId: params.ChangeID,
		Scenario: params.Scenario,
		Steps:    protoSteps,
	}))
	if err != nil {
		return nil, err
	}

	ac := resp.Msg.Criteria
	return map[string]any{
		"id":       ac.Id,
		"scenario": ac.Scenario,
		"created":  true,
	}, nil
}

func (s *Server) handleToggleAC(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ID  int64 `json:"id"`
		Met bool  `json:"met"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.acceptance.ToggleAC(ctx, connect.NewRequest(&acceptancev1.ToggleACRequest{
		Id:  params.ID,
		Met: params.Met,
	}))
	if err != nil {
		return nil, err
	}

	ac := resp.Msg.Criteria
	return map[string]any{
		"id":       ac.Id,
		"scenario": ac.Scenario,
		"met":      ac.Met,
		"toggled":  true,
	}, nil
}

func (s *Server) handleUpdateProject(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID   int64   `json:"project_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Readme      *string `json:"readme"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	req := &projectv1.UpdateProjectRequest{
		Id:          params.ProjectID,
		Name:        params.Name,
		Description: params.Description,
	}
	if params.Readme != nil {
		// Convert markdown to HTML for Tiptap editor
		var buf bytes.Buffer
		if err := goldmark.Convert([]byte(*params.Readme), &buf); err != nil {
			return nil, fmt.Errorf("failed to convert markdown: %w", err)
		}
		html := buf.String()
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
		ProjectID int64  `json:"project_id"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.CreateChange(ctx, connect.NewRequest(&projectv1.CreateChangeRequest{
		ProjectId: params.ProjectID,
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
		ProjectID int64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.project.ListChanges(ctx, connect.NewRequest(&projectv1.ListChangesRequest{
		ProjectId: params.ProjectID,
	}))
	if err != nil {
		return nil, err
	}

	type changeInfo struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Stage string `json:"stage"`
	}
	changes := make([]changeInfo, len(resp.Msg.Changes))
	for i, c := range resp.Msg.Changes {
		changes[i] = changeInfo{ID: c.Id, Name: c.Name, Stage: c.Stage}
	}

	return map[string]any{
		"project_id": params.ProjectID,
		"changes":    changes,
		"total":      len(changes),
	}, nil
}

func (s *Server) handleAdvanceStage(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.workflow.Advance(ctx, connect.NewRequest(&workflowv1.AdvanceRequest{
		ChangeId: params.ChangeID,
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"change_id": params.ChangeID,
		"new_stage": resp.Msg.NewStage,
		"advanced":  true,
	}, nil
}

func (s *Server) handleListComments(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64 `json:"change_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.ListComments(ctx, connect.NewRequest(&commentv1.ListCommentsRequest{
		ChangeId: params.ChangeID,
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
		"change_id": params.ChangeID,
		"comments":  comments,
		"total":     len(comments),
	}, nil
}

func (s *Server) handleCreateComment(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ChangeID int64  `json:"change_id"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.comment.CreateComment(ctx, connect.NewRequest(&commentv1.CreateCommentRequest{
		ChangeId: params.ChangeID,
		Body:     params.Content,
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
		TaskID int64 `json:"task_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	_, err := s.clients.task.DeleteTask(ctx, connect.NewRequest(&taskv1.DeleteTaskRequest{
		Id: params.TaskID,
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"task_id": params.TaskID,
		"deleted": true,
	}, nil
}

func (s *Server) handleGetMemory(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID int64 `json:"project_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.clients.memory.GetMemory(ctx, connect.NewRequest(&memoryv1.GetMemoryRequest{
		ProjectId: params.ProjectID,
	}))
	if err != nil {
		return nil, err
	}

	if resp.Msg.Memory == nil {
		return map[string]any{
			"project_id": params.ProjectID,
			"content":    "",
			"exists":     false,
		}, nil
	}

	return map[string]any{
		"project_id": params.ProjectID,
		"content":    resp.Msg.Memory.Content,
		"exists":     true,
	}, nil
}

func (s *Server) handleSaveMemory(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		ProjectID int64  `json:"project_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	_, err := s.clients.memory.SaveMemory(ctx, connect.NewRequest(&memoryv1.SaveMemoryRequest{
		ProjectId: params.ProjectID,
		Content:   params.Content,
	}))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"project_id": params.ProjectID,
		"saved":      true,
	}, nil
}
