package mcp

import (
	"strings"
	"testing"
)

func TestProseMirrorJSONToMarkdown(t *testing.T) {
	input := `{
		"type":"doc",
		"content":[
			{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Design"}]},
			{"type":"paragraph","content":[
				{"type":"text","text":"Use "},
				{"type":"text","text":"Client","marks":[{"type":"code"}]}
			]},
			{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println(1)"}]}
		]
	}`

	got, err := proseMirrorJSONToMarkdown(input)
	if err != nil {
		t.Fatalf("proseMirrorJSONToMarkdown returned error: %v", err)
	}

	for _, want := range []string{
		"## Design",
		"Use `Client`",
		"```go",
		"fmt.Println(1)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, got)
		}
	}
}

func TestExportDocumentToMarkdownHTMLFallback(t *testing.T) {
	got, err := exportDocumentToMarkdown("<h2>Design</h2><p>Hello <code>world()</code></p>")
	if err != nil {
		t.Fatalf("exportDocumentToMarkdown returned error: %v", err)
	}

	for _, want := range []string{"## Design", "Hello `world()`"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, got)
		}
	}
}
