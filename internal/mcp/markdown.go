package mcp

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
)

// markdownToHTML converts markdown to HTML for TipTap editor rendering.
func markdownToHTML(md string) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("failed to convert markdown: %w", err)
	}
	return buf.String(), nil
}
