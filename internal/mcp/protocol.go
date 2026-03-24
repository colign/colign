package mcp

import "encoding/json"

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func ListTools() []Tool {
	return []Tool{
		{
			Name:        "list_projects",
			Description: "List all projects the user has access to",
			InputSchema: InputSchema{Type: "object"},
		},
		{
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
		},
		{
			Name:        "read_spec",
			Description: "Read a spec document for a change. For proposals, the content field is a JSON string with keys: problem, scope, outOfScope, approach.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
					"doc_type":   {Type: "string", Description: "Document type: proposal, design, spec, tasks"},
				},
				Required: []string{"change_id", "project_id", "doc_type"},
			},
		},
		{
			Name:        "write_spec",
			Description: "Write or update a spec document for a change. For proposals, content must be a JSON string with keys: problem (required), scope (required), outOfScope (optional), approach (optional). Example: {\"problem\":\"...\",\"scope\":\"...\",\"outOfScope\":\"...\",\"approach\":\"...\"}. For other doc types, content is plain markdown.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
					"doc_type":   {Type: "string", Description: "Document type: proposal, design, spec, tasks"},
					"content":    {Type: "string", Description: "For proposal: JSON with problem, scope, outOfScope, approach. For others: markdown text."},
				},
				Required: []string{"change_id", "project_id", "doc_type", "content"},
			},
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
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
		},
		{
			Name:        "list_changes",
			Description: "List all changes in a project",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_id": {Type: "integer", Description: "Project ID"},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "advance_stage",
			Description: "Advance a change to the next workflow stage (draft -> design -> review -> ready)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
				},
				Required: []string{"change_id", "project_id"},
			},
		},
		{
			Name:        "list_comments",
			Description: "List comments on a change",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
				},
				Required: []string{"change_id", "project_id"},
			},
		},
		{
			Name:        "create_comment",
			Description: "Add a comment to a change",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
					"content":    {Type: "string", Description: "Comment text"},
				},
				Required: []string{"change_id", "project_id", "content"},
			},
		},
		{
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
		},
		{
			Name:        "get_memory",
			Description: "Get the project memory (shared context like CLAUDE.md). Contains conventions, decisions, and context that AI should know about this project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_id": {Type: "integer", Description: "Project ID"},
				},
				Required: []string{"project_id"},
			},
		},
		{
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
		},
		{
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
		},
		{
			Name:        "get_change_summary",
			Description: "Get an aggregated summary of a change: stage, task progress, AC progress, and gate conditions. Ideal for quick status checks.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id": {Type: "integer", Description: "Change ID"},
				},
				Required: []string{"change_id"},
			},
		},
		{
			Name:        "get_project_dashboard",
			Description: "Get all active (non-archived) changes in a project with their progress summaries. Shows task and AC progress for each change.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_id": {Type: "integer", Description: "Project ID"},
				},
				Required: []string{"project_id"},
			},
		},
		{
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
		},
		{
			Name:        "approve_change",
			Description: "Approve a change in review stage. If approval policy is met, the change advances automatically.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
					"comment":    {Type: "string", Description: "Optional approval comment"},
				},
				Required: []string{"change_id", "project_id"},
			},
		},
		{
			Name:        "reject_change",
			Description: "Request changes on a review, sending the change back to design stage.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id":  {Type: "integer", Description: "Change ID"},
					"project_id": {Type: "integer", Description: "Project ID"},
					"reason":     {Type: "string", Description: "Reason for requesting changes"},
				},
				Required: []string{"change_id", "project_id", "reason"},
			},
		},
		{
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
		},
		{
			Name:        "get_work_context",
			Description: "Get all context needed to start working on a change in one call: change info, proposal, tasks, acceptance criteria, memory, gate conditions, and recent comments.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"change_id": {Type: "integer", Description: "Change ID"},
				},
				Required: []string{"change_id"},
			},
		},
	}
}
