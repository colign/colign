package ai

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProposalSystemPrompt_WithoutContext(t *testing.T) {
	prompt := ProposalSystemPrompt(false, nil)
	assert.Contains(t, prompt, "---SECTION:problem---")
	assert.Contains(t, prompt, "---SECTION:scope---")
	assert.Contains(t, prompt, "---SECTION:outOfScope---")
	assert.NotContains(t, prompt, "---SECTION:approach---")
	assert.NotContains(t, prompt, "Project context:")
}

func TestProposalSystemPrompt_WithContext(t *testing.T) {
	prompt := ProposalSystemPrompt(true, []string{"feat: add login", "fix: button alignment"})
	assert.Contains(t, prompt, "Project context:")
	assert.Contains(t, prompt, "feat: add login")
	assert.Contains(t, prompt, "fix: button alignment")
}

func TestProposalSystemPrompt_MaxChanges(t *testing.T) {
	changes := make([]string, 15)
	for i := range changes {
		changes[i] = fmt.Sprintf("change %d", i)
	}
	prompt := ProposalSystemPrompt(true, changes)
	assert.Contains(t, prompt, "change 0")
	assert.Contains(t, prompt, "change 9")
	assert.NotContains(t, prompt, "change 10") // should be truncated
}

func TestACSystemPrompt_WithoutContext(t *testing.T) {
	prompt := ACSystemPrompt(false, nil, "", "")
	assert.Contains(t, prompt, "QA engineer")
	assert.Contains(t, prompt, "JSON array")
	assert.NotContains(t, prompt, "Additional context:")
}

func TestACSystemPrompt_WithContext(t *testing.T) {
	prompt := ACSystemPrompt(true, []string{"Login scenario"}, "design doc content", "spec content")
	assert.Contains(t, prompt, "Additional context:")
	assert.Contains(t, prompt, "Login scenario")
	assert.Contains(t, prompt, "design doc content")
	assert.Contains(t, prompt, "spec content")
}
