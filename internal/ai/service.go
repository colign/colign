package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/aiconfig"
)

// GenerateProposalInput holds the input for proposal generation.
type GenerateProposalInput struct {
	Description string // user's one-line description
}

// GenerateACInput holds the input for acceptance criteria generation.
type GenerateACInput struct {
	Proposal string // proposal JSON string
}

// GeneratedAC represents a single generated acceptance criterion.
type GeneratedAC struct {
	Scenario string   `json:"scenario"`
	Steps    []ACStep `json:"steps"`
}

// ACStep is a single BDD step within an acceptance criterion.
type ACStep struct {
	Keyword string `json:"keyword"`
	Text    string `json:"text"`
}

// Service provides AI generation capabilities backed by an aiconfig.Service.
type Service struct {
	configSvc *aiconfig.Service
	db        *bun.DB
}

// NewService creates a new AI Service.
func NewService(configSvc *aiconfig.Service, db *bun.DB) *Service {
	return &Service{
		configSvc: configSvc,
		db:        db,
	}
}

// GenerateProposal streams proposal section chunks for the given input using
// the AI configuration supplied in cfg.
func (s *Service) GenerateProposal(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateProposalInput) (<-chan SectionChunk, error) {
	decryptedKey, err := s.configSvc.DecryptAPIKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("ai: decrypt api key: %w", err)
	}

	chatModel, err := NewChatModel(ctx, cfg.Provider, cfg.Model, decryptedKey)
	if err != nil {
		return nil, fmt.Errorf("ai: create chat model: %w", err)
	}

	systemPrompt := ProposalSystemPrompt(cfg.IncludeProjectContext, "", nil)
	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: input.Description},
	}

	sr, err := chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ai: stream: %w", err)
	}

	ch := streamProposal(ctx, sr)
	return ch, nil
}

// GenerateAC generates acceptance criteria for the given proposal using the
// AI configuration in cfg. On JSON parse failure it retries once.
func (s *Service) GenerateAC(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateACInput) ([]GeneratedAC, error) {
	decryptedKey, err := s.configSvc.DecryptAPIKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("ai: decrypt api key: %w", err)
	}

	chatModel, err := NewChatModel(ctx, cfg.Provider, cfg.Model, decryptedKey)
	if err != nil {
		return nil, fmt.Errorf("ai: create chat model: %w", err)
	}

	return generateACWithModel(ctx, chatModel, input.Proposal)
}

// generateProposalWithModel is the testable inner function that streams
// proposal chunks from the given chat model.
func generateProposalWithModel(ctx context.Context, chatModel model.BaseChatModel, description string) (<-chan SectionChunk, error) {
	systemPrompt := ProposalSystemPrompt(false, "", nil)
	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: description},
	}

	sr, err := chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ai: stream: %w", err)
	}

	ch := streamProposal(ctx, sr)
	return ch, nil
}

// streamProposal consumes a StreamReader in a goroutine and feeds chunks
// through SectionParser, sending results to the returned channel.
func streamProposal(ctx context.Context, sr *schema.StreamReader[*schema.Message]) <-chan SectionChunk {
	ch := make(chan SectionChunk, 16)
	go func() {
		defer close(ch)
		defer sr.Close()
		parser := NewSectionParser()
		for {
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				slog.ErrorContext(ctx, "ai: stream recv error", slog.String("error", err.Error()))
				break
			}
			for _, chunk := range parser.Feed(msg.Content) {
				ch <- chunk
			}
		}
	}()
	return ch
}

// generateACWithModel is the testable inner function for AC generation.
// It retries once on JSON parse failure.
func generateACWithModel(ctx context.Context, chatModel model.BaseChatModel, proposal string) ([]GeneratedAC, error) {
	systemPrompt := ACSystemPrompt(false, nil, "", "")
	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: proposal},
	}

	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ai: generate ac: %w", err)
	}

	var result []GeneratedAC
	content := extractJSON(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		slog.WarnContext(ctx, "ai: ac parse failed, retrying", slog.String("error", err.Error()))

		// Retry once.
		response, err = chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ai: generate ac retry: %w", err)
		}
		content = extractJSON(response.Content)
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return nil, fmt.Errorf("ai: parse ac response: %w", err)
		}
	}

	return result, nil
}

// extractJSON strips markdown code fences (```json ... ```) if present and
// attempts to isolate the first JSON array from the string.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Strip ```json ... ``` or ``` ... ``` fences.
	if strings.HasPrefix(s, "```") {
		// Find the end of the opening fence line.
		newline := strings.Index(s, "\n")
		if newline != -1 {
			s = s[newline+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		return strings.TrimSpace(s)
	}

	// Try to extract first [...] block from content that has extra text.
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(s[start : end+1])
	}

	return s
}
