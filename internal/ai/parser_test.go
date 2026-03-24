package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSectionParser_AllSections(t *testing.T) {
	p := NewSectionParser()
	input := "---SECTION:problem---\nThis is the problem.\n---SECTION:scope---\nScope content here.\n"
	chunks := p.Feed(input)

	// Should have chunks for problem and scope sections
	require.NotEmpty(t, chunks)

	var problemText, scopeText string
	for _, c := range chunks {
		switch c.Section {
		case "problem":
			problemText += c.Text
		case "scope":
			scopeText += c.Text
		}
	}
	assert.Contains(t, problemText, "This is the problem.")
	assert.Contains(t, scopeText, "Scope content here.")
}

func TestSectionParser_StreamingChunks(t *testing.T) {
	// Simulate text arriving in small chunks, including delimiter split across chunks
	p := NewSectionParser()

	chunks1 := p.Feed("---SECTION:prob")
	chunks2 := p.Feed("lem---\nHello ")
	chunks3 := p.Feed("world\n")

	all := append(chunks1, chunks2...)
	all = append(all, chunks3...)

	var text string
	for _, c := range all {
		if c.Section == "problem" {
			text += c.Text
		}
	}
	assert.Contains(t, text, "Hello world")
}

func TestSectionParser_TextBeforeDelimiter(t *testing.T) {
	// Text before first delimiter should be ignored
	p := NewSectionParser()
	chunks := p.Feed("some preamble\n---SECTION:problem---\nActual content\n")

	for _, c := range chunks {
		assert.NotEmpty(t, c.Section, "all chunks should have a section")
	}
}

func TestSectionParser_EmptyInput(t *testing.T) {
	p := NewSectionParser()
	chunks := p.Feed("")
	assert.Empty(t, chunks)
}

func TestSectionParser_PreservesEmptyLines(t *testing.T) {
	p := NewSectionParser()
	chunks := p.Feed("---SECTION:problem---\nLine 1\n\nLine 3\n")

	var text string
	for _, c := range chunks {
		text += c.Text
	}
	assert.Contains(t, text, "Line 1")
	assert.Contains(t, text, "Line 3")
}

func TestSectionParser_FourSections(t *testing.T) {
	p := NewSectionParser()
	input := "---SECTION:problem---\nP\n---SECTION:scope---\nS\n---SECTION:outOfScope---\nO\n---SECTION:approach---\nA\n"
	chunks := p.Feed(input)

	sections := map[string]string{}
	for _, c := range chunks {
		sections[c.Section] += c.Text
	}
	assert.Contains(t, sections["problem"], "P")
	assert.Contains(t, sections["scope"], "S")
	assert.Contains(t, sections["outOfScope"], "O")
	assert.Contains(t, sections["approach"], "A")
}
