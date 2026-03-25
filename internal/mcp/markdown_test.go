package mcp

import (
	"strings"
	"testing"
)

func TestMarkdownToHTMLSupportsCommonFormatting(t *testing.T) {
	input := "# Architecture\n\n기존 패턴 참고\n\n1. **생성 후 동작**: `router.push('/projects/{slug}')`\n2. **목록 갱신**: projects 페이지 refetch\n"
	html, err := markdownToHTML(input)
	if err != nil {
		t.Fatalf("markdownToHTML returned error: %v", err)
	}

	for _, want := range []string{
		"<h1>Architecture</h1>",
		"<ol>",
		"<strong>생성 후 동작</strong>",
		"<code>router.push('/projects/{slug}')</code>",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected HTML to contain %q, got %q", want, html)
		}
	}
}
