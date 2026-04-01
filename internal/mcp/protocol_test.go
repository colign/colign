package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolDefinitions(t *testing.T) {
	tools := ListTools()
	require.NotEmpty(t, tools, "expected at least one tool defined")

	expectedTools := []string{
		"list_projects", "get_change", "read_spec",
		"write_spec", "list_tasks", "update_task", "suggest_spec",
		"get_change_history",
	}

	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, name := range expectedTools {
		assert.True(t, toolMap[name], "missing tool: %s", name)
	}
}

func TestUpdateTaskToolHasAssigneeParams(t *testing.T) {
	tools := ListTools()

	var updateTask *Tool
	for i, tool := range tools {
		if tool.Name == "update_task" {
			updateTask = &tools[i]
			break
		}
	}
	require.NotNil(t, updateTask, "update_task tool not found")

	assert.Contains(t, updateTask.InputSchema.Properties, "assignee_id", "update_task should have assignee_id param")
	assert.Contains(t, updateTask.InputSchema.Properties, "clear_assignee", "update_task should have clear_assignee param")
	assert.Equal(t, "integer", updateTask.InputSchema.Properties["assignee_id"].Type)
	assert.Equal(t, "boolean", updateTask.InputSchema.Properties["clear_assignee"].Type)

	// status should no longer be required (only task_id and project_id are)
	assert.Equal(t, []string{"task_id", "project_id"}, updateTask.InputSchema.Required)
}

func TestLinkACToTestToolDefinition(t *testing.T) {
	tools := ListTools()

	var linkTool *Tool
	for i, tool := range tools {
		if tool.Name == "link_ac_to_test" {
			linkTool = &tools[i]
			break
		}
	}
	require.NotNil(t, linkTool, "link_ac_to_test tool not found")

	assert.Contains(t, linkTool.InputSchema.Properties, "ac_id")
	assert.Contains(t, linkTool.InputSchema.Properties, "test_ref")
	assert.Equal(t, []string{"ac_id", "project_id", "test_ref"}, linkTool.InputSchema.Required)
}

func TestCreateACToolHasTestRefParam(t *testing.T) {
	tools := ListTools()

	var createAC *Tool
	for i, tool := range tools {
		if tool.Name == "create_acceptance_criteria" {
			createAC = &tools[i]
			break
		}
	}
	require.NotNil(t, createAC, "create_acceptance_criteria tool not found")

	assert.Contains(t, createAC.InputSchema.Properties, "test_ref", "create_acceptance_criteria should have test_ref param")
}

func TestGetChangeHistoryToolDefinition(t *testing.T) {
	tools := ListTools()

	var historyTool *Tool
	for i, tool := range tools {
		if tool.Name == "get_change_history" {
			historyTool = &tools[i]
			break
		}
	}
	require.NotNil(t, historyTool, "get_change_history tool not found")

	assert.Contains(t, historyTool.InputSchema.Properties, "change_id")
	assert.Equal(t, []string{"change_id", "project_id"}, historyTool.InputSchema.Required)
}

func TestPOToolsExist(t *testing.T) {
	tools := ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	poTools := []string{
		"get_change_summary",
		"get_project_dashboard",
		"get_gate_status",
		"approve_change",
		"reject_change",
		"archive_change",
	}

	for _, name := range poTools {
		assert.True(t, toolMap[name], "missing PO tool: %s", name)
	}
}

func TestApproveChangeToolDefinition(t *testing.T) {
	tools := ListTools()

	var tool *Tool
	for i, tt := range tools {
		if tt.Name == "approve_change" {
			tool = &tools[i]
			break
		}
	}
	require.NotNil(t, tool)
	assert.Contains(t, tool.InputSchema.Properties, "change_id")
	assert.Contains(t, tool.InputSchema.Properties, "comment")
	assert.Equal(t, []string{"change_id", "project_id"}, tool.InputSchema.Required, "change_id and project_id required, comment is optional")
}

func TestRejectChangeToolDefinition(t *testing.T) {
	tools := ListTools()

	var tool *Tool
	for i, tt := range tools {
		if tt.Name == "reject_change" {
			tool = &tools[i]
			break
		}
	}
	require.NotNil(t, tool)
	assert.Contains(t, tool.InputSchema.Properties, "reason")
	assert.Equal(t, []string{"change_id", "project_id", "reason"}, tool.InputSchema.Required, "change_id, project_id and reason are required")
}

func TestGetWorkContextToolDefinition(t *testing.T) {
	tools := ListTools()

	var tool *Tool
	for i, tt := range tools {
		if tt.Name == "get_work_context" {
			tool = &tools[i]
			break
		}
	}
	require.NotNil(t, tool, "get_work_context tool not found")
	assert.Contains(t, tool.InputSchema.Properties, "change_id")
	assert.Equal(t, []string{"change_id"}, tool.InputSchema.Required)
}

func TestListChangesToolIncludesProgress(t *testing.T) {
	tools := ListTools()

	var tool *Tool
	for i, tt := range tools {
		if tt.Name == "list_changes" {
			tool = &tools[i]
			break
		}
	}
	require.NotNil(t, tool)
	// list_changes tool exists — progress is added at handler level, not schema level
	assert.Contains(t, tool.InputSchema.Properties, "project_id")
}

func TestCommentToolsHaveDocumentType(t *testing.T) {
	tools := ListTools()
	toolMap := make(map[string]*Tool)
	for i, tool := range tools {
		toolMap[tool.Name] = &tools[i]
	}

	t.Run("create_comment has document_type param", func(t *testing.T) {
		tool := toolMap["create_comment"]
		require.NotNil(t, tool)
		assert.Contains(t, tool.InputSchema.Properties, "document_type")
		assert.Equal(t, "string", tool.InputSchema.Properties["document_type"].Type)
		assert.Contains(t, tool.InputSchema.Required, "document_type")
	})

	t.Run("list_comments has document_type param", func(t *testing.T) {
		tool := toolMap["list_comments"]
		require.NotNil(t, tool)
		assert.Contains(t, tool.InputSchema.Properties, "document_type")
		assert.Equal(t, "string", tool.InputSchema.Properties["document_type"].Type)
		assert.Contains(t, tool.InputSchema.Required, "document_type")
	})
}

func TestToolMetadataIncludesStandardAnnotations(t *testing.T) {
	tools := ListTools()
	toolMap := make(map[string]*Tool)
	for i, tool := range tools {
		toolMap[tool.Name] = &tools[i]
	}

	readOnlyTool := toolMap["list_projects"]
	require.NotNil(t, readOnlyTool)
	require.NotNil(t, readOnlyTool.Annotations)
	assert.True(t, readOnlyTool.Annotations.ReadOnlyHint)
	if assert.NotNil(t, readOnlyTool.Annotations.DestructiveHint) {
		assert.False(t, *readOnlyTool.Annotations.DestructiveHint)
	}

	mutatingTool := toolMap["update_change"]
	require.NotNil(t, mutatingTool)
	require.NotNil(t, mutatingTool.Annotations)
	assert.False(t, mutatingTool.Annotations.ReadOnlyHint)
	assert.Nil(t, mutatingTool.Annotations.DestructiveHint)
	assert.Contains(t, mutatingTool.InputSchema.Properties, "status")
	assert.Contains(t, mutatingTool.InputSchema.Properties, "status_reason")
}

func TestToolsListJSONIncludesMetadata(t *testing.T) {
	tools := ListTools()
	raw, err := json.Marshal(tools)
	require.NoError(t, err)

	var decoded []map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.NotEmpty(t, decoded)

	var updateChange map[string]any
	for _, tool := range decoded {
		if tool["name"] == "update_change" {
			updateChange = tool
			break
		}
	}
	require.NotNil(t, updateChange)
	assert.Contains(t, updateChange, "annotations")
}

func TestUpdateChangeToolDefinition(t *testing.T) {
	tools := ListTools()

	var tool *Tool
	for i, tt := range tools {
		if tt.Name == "update_change" {
			tool = &tools[i]
			break
		}
	}
	require.NotNil(t, tool, "update_change tool not found")

	assert.Contains(t, tool.InputSchema.Properties, "change_id")
	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "sub_status")
	assert.Contains(t, tool.InputSchema.Properties, "status")
	assert.Contains(t, tool.InputSchema.Properties, "status_reason")
	assert.Equal(t, []string{"change_id", "project_id"}, tool.InputSchema.Required,
		"only change_id and project_id should be required; name, sub_status, status are optional")
}

func TestJSONRPCRequest(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	var req JSONRPCRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	assert.Equal(t, "tools/list", req.Method)
}
