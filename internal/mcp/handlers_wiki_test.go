package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	wikiv1 "github.com/gobenpark/colign/gen/proto/wiki/v1"
)

func TestWikiToolsExist(t *testing.T) {
	tools := ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	wikiTools := []string{
		"list_wiki_pages",
		"get_wiki_page",
		"create_wiki_page",
		"update_wiki_page",
		"delete_wiki_page",
	}

	for _, name := range wikiTools {
		assert.True(t, toolMap[name], "missing wiki tool: %s", name)
	}
}

func TestListWikiPagesToolDefinition(t *testing.T) {
	tool := findTool(t, "list_wiki_pages")

	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Equal(t, []string{"project_id"}, tool.InputSchema.Required)
	assert.True(t, tool.Annotations.ReadOnlyHint)
}

func TestGetWikiPageToolDefinition(t *testing.T) {
	tool := findTool(t, "get_wiki_page")

	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Contains(t, tool.InputSchema.Properties, "page_id")
	assert.Equal(t, "string", tool.InputSchema.Properties["page_id"].Type)
	assert.Equal(t, []string{"project_id", "page_id"}, tool.InputSchema.Required)
	assert.True(t, tool.Annotations.ReadOnlyHint)
}

func TestCreateWikiPageToolDefinition(t *testing.T) {
	tool := findTool(t, "create_wiki_page")

	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Contains(t, tool.InputSchema.Properties, "parent_id")
	assert.Contains(t, tool.InputSchema.Properties, "title")
	assert.Equal(t, []string{"project_id"}, tool.InputSchema.Required)
	assert.False(t, tool.Annotations.ReadOnlyHint)
}

func TestUpdateWikiPageToolDefinition(t *testing.T) {
	tool := findTool(t, "update_wiki_page")

	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Contains(t, tool.InputSchema.Properties, "page_id")
	assert.Contains(t, tool.InputSchema.Properties, "title")
	assert.Contains(t, tool.InputSchema.Properties, "icon")
	assert.Contains(t, tool.InputSchema.Properties, "content")
	assert.Equal(t, []string{"project_id", "page_id"}, tool.InputSchema.Required)
	assert.False(t, tool.Annotations.ReadOnlyHint)
}

func TestDeleteWikiPageToolDefinition(t *testing.T) {
	tool := findTool(t, "delete_wiki_page")

	assert.Contains(t, tool.InputSchema.Properties, "project_id")
	assert.Contains(t, tool.InputSchema.Properties, "page_id")
	assert.Equal(t, []string{"project_id", "page_id"}, tool.InputSchema.Required)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)
}

func TestWikiPageToMap(t *testing.T) {
	now := timestamppb.Now()
	page := &wikiv1.WikiPage{
		Id:          "abc-123",
		ProjectId:   1,
		ParentId:    "parent-456",
		Title:       "Test Page",
		Icon:        "📝",
		SortOrder:   2,
		ContentText: "hello world",
		CreatorName: "Ben",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := wikiPageToMap(page)
	assert.Equal(t, "abc-123", m["id"])
	assert.Equal(t, int64(1), m["project_id"])
	assert.Equal(t, "parent-456", m["parent_id"])
	assert.Equal(t, "Test Page", m["title"])
	assert.Equal(t, "📝", m["icon"])
	assert.Equal(t, int32(2), m["sort_order"])
	assert.Equal(t, "hello world", m["content_text"])
	assert.Equal(t, "Ben", m["creator_name"])
	assert.Contains(t, m, "created_at")
	assert.Contains(t, m, "updated_at")
}

func TestWikiPageToMapOmitsEmptyOptionalFields(t *testing.T) {
	page := &wikiv1.WikiPage{
		Id:        "abc-123",
		ProjectId: 1,
		Title:     "Root Page",
	}

	m := wikiPageToMap(page)
	assert.NotContains(t, m, "parent_id")
	assert.NotContains(t, m, "content_text")
	assert.NotContains(t, m, "created_at")
	assert.NotContains(t, m, "updated_at")
}

func findTool(t *testing.T, name string) *Tool {
	t.Helper()
	tools := ListTools()
	for i, tt := range tools {
		if tt.Name == name {
			return &tools[i]
		}
	}
	t.Fatalf("tool %q not found", name)
	return nil
}
