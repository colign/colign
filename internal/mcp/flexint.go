package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexInt64 unmarshals both JSON numbers and JSON strings to int64.
// MCP clients may send numeric arguments as strings (e.g. "26" instead of 26).
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(data []byte) error {
	// null → zero value
	if string(data) == "null" {
		*f = 0
		return nil
	}

	// Try number first
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("FlexInt64: cannot unmarshal %s into int64", string(data))
	}

	s := n.String()

	// Reject floats — must be a whole integer
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		if _, intErr := strconv.ParseInt(s, 10, 64); intErr != nil {
			return fmt.Errorf("FlexInt64: %s is not a valid integer", s)
		}
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("FlexInt64: cannot parse %q as int64: %w", s, err)
	}

	*f = FlexInt64(v)
	return nil
}

// Int64 returns the underlying int64 value.
func (f FlexInt64) Int64() int64 {
	return int64(f)
}

// Int64Ptr returns a pointer to the underlying int64 value.
func (f FlexInt64) Int64Ptr() *int64 {
	v := int64(f)
	return &v
}
