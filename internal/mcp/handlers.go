package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	documentv1 "github.com/gobenpark/colign/gen/proto/document/v1"
	projectv1 "github.com/gobenpark/colign/gen/proto/project/v1"
	taskv1 "github.com/gobenpark/colign/gen/proto/task/v1"
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
	case "list_tasks":
		return s.handleListTasks(ctx, args)
	case "update_task":
		return s.handleUpdateTask(ctx, args)
	case "suggest_spec":
		return s.handleSuggestSpec(ctx, args)
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
	return map[string]any{
		"id":     t.Id,
		"title":  t.Title,
		"status": t.Status,
	}, nil
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
