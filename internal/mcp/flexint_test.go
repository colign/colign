package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexInt64_UnmarshalJSON(t *testing.T) {
	type payload struct {
		ChangeID  FlexInt64 `json:"change_id"`
		ProjectID FlexInt64 `json:"project_id"`
	}

	tests := []struct {
		name    string
		input   string
		wantCID int64
		wantPID int64
		wantErr bool
	}{
		{
			name:    "both numbers",
			input:   `{"change_id": 26, "project_id": 1}`,
			wantCID: 26,
			wantPID: 1,
		},
		{
			name:    "both strings",
			input:   `{"change_id": "26", "project_id": "1"}`,
			wantCID: 26,
			wantPID: 1,
		},
		{
			name:    "mixed string and number",
			input:   `{"change_id": "26", "project_id": 1}`,
			wantCID: 26,
			wantPID: 1,
		},
		{
			name:    "zero values",
			input:   `{"change_id": 0, "project_id": "0"}`,
			wantCID: 0,
			wantPID: 0,
		},
		{
			name:    "negative number",
			input:   `{"change_id": -5, "project_id": "-10"}`,
			wantCID: -5,
			wantPID: -10,
		},
		{
			name:    "large number",
			input:   `{"change_id": 9223372036854775807, "project_id": "9223372036854775807"}`,
			wantCID: 9223372036854775807,
			wantPID: 9223372036854775807,
		},
		{
			name:    "invalid string",
			input:   `{"change_id": "abc"}`,
			wantErr: true,
		},
		{
			name:    "boolean value",
			input:   `{"change_id": true}`,
			wantErr: true,
		},
		{
			name:    "null stays zero",
			input:   `{"change_id": null, "project_id": 1}`,
			wantCID: 0,
			wantPID: 1,
		},
		{
			name:    "float truncation rejected",
			input:   `{"change_id": 3.14}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p payload
			err := json.Unmarshal([]byte(tt.input), &p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCID, int64(p.ChangeID))
			assert.Equal(t, tt.wantPID, int64(p.ProjectID))
		})
	}
}

func TestFlexInt64Ptr_UnmarshalJSON(t *testing.T) {
	type payload struct {
		AssigneeID *FlexInt64 `json:"assignee_id"`
	}

	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal int64
		wantErr bool
	}{
		{
			name:    "number pointer",
			input:   `{"assignee_id": 42}`,
			wantVal: 42,
		},
		{
			name:    "string pointer",
			input:   `{"assignee_id": "42"}`,
			wantVal: 42,
		},
		{
			name:    "null pointer",
			input:   `{"assignee_id": null}`,
			wantNil: true,
		},
		{
			name:    "absent field",
			input:   `{}`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p payload
			err := json.Unmarshal([]byte(tt.input), &p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, p.AssigneeID)
			} else {
				require.NotNil(t, p.AssigneeID)
				assert.Equal(t, tt.wantVal, int64(*p.AssigneeID))
			}
		})
	}
}

func TestFlexInt64_Int64(t *testing.T) {
	f := FlexInt64(42)
	assert.Equal(t, int64(42), f.Int64())
}

func TestFlexInt64Ptr_Int64Ptr(t *testing.T) {
	f := FlexInt64(42)
	ptr := f.Int64Ptr()
	require.NotNil(t, ptr)
	assert.Equal(t, int64(42), *ptr)
}
