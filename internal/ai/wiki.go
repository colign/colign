package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/gobenpark/colign/internal/aiconfig"
)

// WikiPageUpdate represents a single wiki page update instruction from the LLM.
type WikiPageUpdate struct {
	Action    string `json:"action"`     // "update" or "create"
	PageID    string `json:"page_id"`    // UUID for update, empty for create
	PageTitle string `json:"page_title"` // page title
	Content   string `json:"content"`    // markdown content with [[Page Title]] links
}

// WikiUpdateInput contains the context needed for wiki auto-update.
type WikiUpdateInput struct {
	ChangeName    string
	Proposal      string   // JSON: {problem, scope, outOfScope}
	Spec          string   // markdown
	Tasks         string   // summary of tasks with statuses
	Acceptance    string   // BDD scenarios
	Comments      string   // key discussions
	ExistingPages []string // titles of existing wiki pages
}

// GenerateWikiUpdate asks the LLM to determine which wiki pages to create or
// update based on a change's artifacts.
func (s *Service) GenerateWikiUpdate(ctx context.Context, cfg *aiconfig.AIConfig, input WikiUpdateInput) ([]WikiPageUpdate, error) {
	decryptedKey, err := s.configSvc.DecryptAPIKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("ai: decrypt api key: %w", err)
	}

	chatModel, err := NewChatModel(ctx, cfg.Provider, cfg.Model, decryptedKey)
	if err != nil {
		return nil, fmt.Errorf("ai: create chat model: %w", err)
	}

	return generateWikiUpdateWithModel(ctx, chatModel, input)
}

func generateWikiUpdateWithModel(ctx context.Context, chatModel model.BaseChatModel, input WikiUpdateInput) ([]WikiPageUpdate, error) {
	systemPrompt := wikiUpdateSystemPrompt(input.ExistingPages)

	var userContent strings.Builder
	fmt.Fprintf(&userContent, "# Change: %s\n\n", input.ChangeName)

	if input.Proposal != "" {
		fmt.Fprintf(&userContent, "## Proposal\n%s\n\n", input.Proposal)
	}
	if input.Spec != "" {
		fmt.Fprintf(&userContent, "## Specification\n%s\n\n", input.Spec)
	}
	if input.Tasks != "" {
		fmt.Fprintf(&userContent, "## Tasks\n%s\n\n", input.Tasks)
	}
	if input.Acceptance != "" {
		fmt.Fprintf(&userContent, "## Acceptance Criteria\n%s\n\n", input.Acceptance)
	}
	if input.Comments != "" {
		fmt.Fprintf(&userContent, "## Key Discussions\n%s\n\n", input.Comments)
	}

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: userContent.String()},
	}

	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ai: generate wiki update: %w", err)
	}

	var result []WikiPageUpdate
	content := extractJSON(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		slog.WarnContext(ctx, "ai: wiki update parse failed, retrying", slog.String("error", err.Error()))

		response, err = chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ai: generate wiki update retry: %w", err)
		}
		content = extractJSON(response.Content)
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return nil, fmt.Errorf("ai: parse wiki update response: %w", err)
		}
	}

	return result, nil
}

func wikiUpdateSystemPrompt(existingPages []string) string {
	var sb strings.Builder
	sb.WriteString(`You are a technical documentation assistant maintaining a project wiki.
Given the artifacts from a completed change (proposal, spec, tasks, acceptance criteria, discussions),
determine which wiki pages should be updated or created.

Rules:
- Preserve existing content where possible — append or update sections, do not overwrite entire pages
- Use [[Page Title]] syntax for cross-references between wiki pages
- Write in the same language as the input artifacts
- Focus on architectural decisions, API changes, and important design choices
- Each page should be self-contained and useful for onboarding new team members
- Keep content concise and factual

Return a JSON array of page updates:
[
  {
    "action": "update",
    "page_id": "existing-page-uuid",
    "page_title": "Page Title",
    "content": "Full markdown content of the updated page"
  },
  {
    "action": "create",
    "page_title": "New Page Title",
    "content": "Full markdown content"
  }
]

If no updates are needed, return an empty array: []
`)

	if len(existingPages) > 0 {
		sb.WriteString("\nExisting wiki pages:\n")
		for _, title := range existingPages {
			fmt.Fprintf(&sb, "- %s\n", title)
		}
	} else {
		sb.WriteString("\nThe wiki is currently empty. Create initial pages to document this change.\n")
	}

	return sb.String()
}
