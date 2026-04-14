package mcp

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// markdownToHTML converts markdown to HTML for TipTap editor rendering.
// WithUnsafe is required to preserve raw HTML such as wiki-link <span> elements
// injected by resolveWikiLinks before this conversion step.
func markdownToHTML(md string) (string, error) {
	var buf bytes.Buffer
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	if err := parser.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("failed to convert markdown: %w", err)
	}
	return buf.String(), nil
}
