package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnforcer(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)
	require.NotNil(t, e)
}

func TestRoleHierarchy(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)

	t.Run("owner inherits editor permissions", func(t *testing.T) {
		allowed, err := e.Enforce("owner", "task", "create")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("owner inherits viewer permissions", func(t *testing.T) {
		allowed, err := e.Enforce("owner", "task", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("editor inherits viewer permissions", func(t *testing.T) {
		allowed, err := e.Enforce("editor", "change", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("viewer cannot access editor actions", func(t *testing.T) {
		allowed, err := e.Enforce("viewer", "task", "create")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("editor cannot access owner actions", func(t *testing.T) {
		allowed, err := e.Enforce("editor", "project", "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestViewerPermissions(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)

	readResources := []string{"project", "change", "task", "comment", "document", "memory", "workflow", "ac", "archive_policy"}
	for _, res := range readResources {
		t.Run("viewer can read "+res, func(t *testing.T) {
			allowed, err := e.Enforce("viewer", res, "read")
			require.NoError(t, err)
			assert.True(t, allowed, "viewer should be able to read %s", res)
		})
	}

	t.Run("viewer cannot create task", func(t *testing.T) {
		allowed, err := e.Enforce("viewer", "task", "create")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("viewer cannot delete change", func(t *testing.T) {
		allowed, err := e.Enforce("viewer", "change", "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestEditorPermissions(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)

	editorAllowed := []struct{ resource, action string }{
		{"change", "create"}, {"change", "update"}, {"change", "archive"}, {"change", "unarchive"}, {"change", "delete"},
		{"task", "create"}, {"task", "update"}, {"task", "delete"}, {"task", "reorder"},
		{"comment", "create"}, {"comment", "resolve"}, {"comment", "delete"}, {"comment", "reply"},
		{"document", "save"},
		{"ac", "create"}, {"ac", "update"}, {"ac", "toggle"}, {"ac", "delete"},
		{"memory", "save"},
		{"workflow", "advance"}, {"workflow", "revert"}, {"workflow", "approve"},
	}

	for _, tc := range editorAllowed {
		t.Run("editor can "+tc.action+" "+tc.resource, func(t *testing.T) {
			allowed, err := e.Enforce("editor", tc.resource, tc.action)
			require.NoError(t, err)
			assert.True(t, allowed)
		})
	}

	editorDenied := []struct{ resource, action string }{
		{"project", "delete"}, {"project", "update"}, {"project", "invite"},
		{"workflow", "set_policy"},
	}

	for _, tc := range editorDenied {
		t.Run("editor cannot "+tc.action+" "+tc.resource, func(t *testing.T) {
			allowed, err := e.Enforce("editor", tc.resource, tc.action)
			require.NoError(t, err)
			assert.False(t, allowed)
		})
	}
}

func TestOwnerPermissions(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)

	ownerOnly := []struct{ resource, action string }{
		{"project", "update"}, {"project", "delete"}, {"project", "invite"},
		{"project", "assign_label"}, {"project", "remove_label"},
		{"workflow", "set_policy"},
		{"archive_policy", "update"},
	}

	for _, tc := range ownerOnly {
		t.Run("owner can "+tc.action+" "+tc.resource, func(t *testing.T) {
			allowed, err := e.Enforce("owner", tc.resource, tc.action)
			require.NoError(t, err)
			assert.True(t, allowed)
		})
	}
}

func TestUnknownRole(t *testing.T) {
	e, err := NewEnforcer()
	require.NoError(t, err)

	allowed, err := e.Enforce("unknown", "task", "read")
	require.NoError(t, err)
	assert.False(t, allowed)
}
