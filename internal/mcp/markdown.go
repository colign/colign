package mcp

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// markdownToHTML converts markdown to HTML for TipTap editor rendering.
func markdownToHTML(md string) (string, error) {
	var buf bytes.Buffer
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	if err := parser.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("failed to convert markdown: %w", err)
	}
	return buf.String(), nil
}
