package ai

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel implements model.BaseChatModel for testing.
type mockChatModel struct {
	generateFunc func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFunc   func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return m.generateFunc(ctx, input, opts...)
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.streamFunc(ctx, input, opts...)
}

// --- extractJSON tests ---

func TestExtractJSON_Plain(t *testing.T) {
	input := `[{"scenario":"test","steps":[]}]`
	got := extractJSON(input)
	assert.Equal(t, input, got)
}

func TestExtractJSON_WithCodeFence(t *testing.T) {
	input := "```json\n[{\"scenario\":\"test\",\"steps\":[]}]\n```"
	got := extractJSON(input)
	assert.Equal(t, `[{"scenario":"test","steps":[]}]`, got)
}

func TestExtractJSON_WithExtraText(t *testing.T) {
	input := `Here is the JSON:
[{"scenario":"test","steps":[]}]
Done.`
	got := extractJSON(input)
	assert.Equal(t, `[{"scenario":"test","steps":[]}]`, got)
}

// --- JSON parsing tests ---

func TestParseACResponse_Valid(t *testing.T) {
	input := `[
		{
			"scenario": "User logs in successfully",
			"steps": [
				{"keyword": "Given", "text": "a registered user"},
				{"keyword": "When", "text": "they enter valid credentials"},
				{"keyword": "Then", "text": "they are redirected to the dashboard"}
			]
		}
	]`

	var result []GeneratedAC
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "User logs in successfully", result[0].Scenario)
	require.Len(t, result[0].Steps, 3)
	assert.Equal(t, "Given", result[0].Steps[0].Keyword)
	assert.Equal(t, "a registered user", result[0].Steps[0].Text)
}

func TestParseACResponse_Invalid(t *testing.T) {
	input := `not json at all`
	var result []GeneratedAC
	err := json.Unmarshal([]byte(input), &result)
	assert.Error(t, err)
}

// --- GenerateProposal streaming test ---

func TestGenerateProposal_StreamsChunks(t *testing.T) {
	ctx := context.Background()

	// Build a stream of messages using schema.Pipe.
	sr, sw := schema.Pipe[*schema.Message](4)
	sw.Send(&schema.Message{Role: schema.Assistant, Content: "---SECTION:problem---\n"}, nil)
	sw.Send(&schema.Message{Role: schema.Assistant, Content: "Something is broken\n"}, nil)
	sw.Close()

	mock := &mockChatModel{
		streamFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
			return sr, nil
		},
	}

	ch, err := generateProposalWithModel(ctx, mock, "fix the bug")
	require.NoError(t, err)

	var chunks []SectionChunk
	for c := range ch {
		chunks = append(chunks, c)
	}

	require.Len(t, chunks, 1)
	assert.Equal(t, "problem", chunks[0].Section)
	assert.Equal(t, "Something is broken\n", chunks[0].Text)
}

// --- GenerateAC test ---

func TestGenerateAC_ParsesResponse(t *testing.T) {
	ctx := context.Background()

	response := `[{"scenario":"Happy path","steps":[{"keyword":"Given","text":"valid input"}]}]`

	mock := &mockChatModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return &schema.Message{Role: schema.Assistant, Content: response}, nil
		},
	}

	result, err := generateACWithModel(ctx, mock, "my proposal")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Happy path", result[0].Scenario)
}

func TestGenerateAC_RetriesOnParseFailure(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	mock := &mockChatModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			callCount++
			if callCount == 1 {
				return &schema.Message{Role: schema.Assistant, Content: "not json"}, nil
			}
			return &schema.Message{Role: schema.Assistant, Content: `[{"scenario":"retry","steps":[]}]`}, nil
		},
	}

	result, err := generateACWithModel(ctx, mock, "proposal text")
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
	require.Len(t, result, 1)
	assert.Equal(t, "retry", result[0].Scenario)
}

func TestGenerateAC_ErrorsAfterTwoFailures(t *testing.T) {
	ctx := context.Background()

	mock := &mockChatModel{
		generateFunc: func(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
			return &schema.Message{Role: schema.Assistant, Content: "not json"}, nil
		},
	}

	_, err := generateACWithModel(ctx, mock, "proposal text")
	assert.Error(t, err)
}

// --- io.EOF import used via schema.StreamReader ---
var _ = io.EOF
