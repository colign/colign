package ai

import (
	"fmt"
	"strings"
)

const maxChanges = 10

// ProposalSystemPrompt returns the system prompt for proposal generation.
func ProposalSystemPrompt(includeContext bool, recentChanges []string) string {
	var sb strings.Builder
	sb.WriteString(`You are a product specification writer for software projects.
Given a one-line description, generate a structured proposal.
Output each section with a delimiter line, then the content:

---SECTION:problem---
What problem does this solve? Why now?
---SECTION:scope---
What's included in this change?
---SECTION:outOfScope---
What's explicitly excluded?

Write in the same language as the user's input.`)

	if includeContext && len(recentChanges) > 0 {
		sb.WriteString("\n\nProject context:")
		changes := recentChanges
		if len(changes) > maxChanges {
			changes = changes[:maxChanges]
		}
		sb.WriteString("\n- Recent changes:")
		for _, c := range changes {
			fmt.Fprintf(&sb, "\n  - %s", c)
		}
	}

	return sb.String()
}

// ACSystemPrompt returns the system prompt for acceptance criteria generation.
func ACSystemPrompt(includeContext bool, existingAC []string, designDoc, specDoc string) string {
	var sb strings.Builder
	sb.WriteString(`You are a QA engineer. Generate BDD acceptance criteria for the following proposal.
Each criterion must have:
- scenario: descriptive name
- steps: array of {keyword: "Given"|"When"|"Then"|"And"|"But", text: "..."}

Return as a JSON array. Generate 3-8 scenarios covering:
- Happy path
- Edge cases
- Error cases

Write in the same language as the proposal.`)

	if includeContext && (len(existingAC) > 0 || designDoc != "" || specDoc != "") {
		sb.WriteString("\n\nAdditional context:")
		if len(existingAC) > 0 {
			sb.WriteString("\n- Existing AC:")
			for _, ac := range existingAC {
				fmt.Fprintf(&sb, "\n  - %s", ac)
			}
		}
		if designDoc != "" {
			fmt.Fprintf(&sb, "\n- Design document:\n%s", designDoc)
		}
		if specDoc != "" {
			fmt.Fprintf(&sb, "\n- Spec document:\n%s", specDoc)
		}
	}

	return sb.String()
}
