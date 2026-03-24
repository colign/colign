package ai

import (
	"strings"
)

// SectionChunk represents a piece of text belonging to a named section.
type SectionChunk struct {
	Section string // "problem", "scope", "outOfScope", "approach"
	Text    string
}

// SectionParser tracks the current section while processing streaming chunks.
type SectionParser struct {
	currentSection string
	buffer         string // incomplete line buffer
}

// NewSectionParser creates a new SectionParser.
func NewSectionParser() *SectionParser {
	return &SectionParser{}
}

// Feed processes a chunk of text and returns any SectionChunk values found.
// It detects "---SECTION:xxx---" delimiters and routes text to the current section.
func (p *SectionParser) Feed(chunk string) []SectionChunk {
	if chunk == "" {
		return nil
	}

	p.buffer += chunk

	// Split by newlines; the last element is the incomplete line remainder.
	lines := strings.Split(p.buffer, "\n")
	// Keep the last (potentially incomplete) line in the buffer.
	p.buffer = lines[len(lines)-1]
	completeLines := lines[:len(lines)-1]

	var result []SectionChunk
	for _, line := range completeLines {
		if strings.HasPrefix(line, "---SECTION:") && strings.HasSuffix(line, "---") {
			// Extract section name between "---SECTION:" and "---"
			inner := line[len("---SECTION:"):]
			name := inner[:len(inner)-len("---")]
			p.currentSection = name
			continue
		}
		if p.currentSection != "" {
			result = append(result, SectionChunk{
				Section: p.currentSection,
				Text:    line + "\n",
			})
		}
	}

	return result
}
