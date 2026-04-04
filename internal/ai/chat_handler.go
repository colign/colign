package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/gobenpark/colign/internal/aiconfig"
	"github.com/gobenpark/colign/internal/auth"
)

type chatRequest struct {
	ChangeID int64         `json:"changeId"`
	Messages []chatMessage `json:"messages"`
	Mode     string        `json:"mode"` // "proposal", "ac", "general"
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// HandleChat streams a multi-turn conversation response via SSE with tool calling support.
// Read tools are auto-executed. Write tools are sent as confirmation events to the frontend.
func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Auth
	claims, err := auth.ResolveFromHeader(h.jwtManager, nil, ctx, r.Header.Get("Authorization"))
	if err != nil {
		writeAIError(w, fmt.Errorf("%w: %w", errUnauthenticated, err))
		return
	}

	// 2. Rate limit
	if !h.limiter.Allow(claims.OrgID) {
		writeAIError(w, errRateLimited)
		return
	}

	// 3. Parse request body
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAIError(w, fmt.Errorf("%w: %w", errBadRequest, err))
		return
	}

	if len(req.Messages) == 0 {
		writeAIError(w, fmt.Errorf("%w: messages required", errBadRequest))
		return
	}

	// 4. Resolve AI config (project → org fallback)
	var cfg *aiconfig.AIConfig
	if req.ChangeID > 0 {
		var projectID int64
		err = h.db.NewSelect().
			ColumnExpr("p.id").
			TableExpr("changes c").
			Join("JOIN projects p ON p.id = c.project_id").
			Where("c.id = ?", req.ChangeID).
			Where("p.organization_id = ?", claims.OrgID).
			Scan(ctx, &projectID)
		if err != nil {
			writeAIError(w, errNotFound)
			return
		}

		cfg, err = h.configSvc.GetByProjectID(ctx, projectID)
		if err != nil {
			writeAIError(w, fmt.Errorf("internal: %w", err))
			return
		}
	}

	if cfg == nil {
		orgCfg, orgErr := h.configSvc.GetByOrgID(ctx, claims.OrgID)
		if orgErr != nil {
			writeAIError(w, fmt.Errorf("internal: %w", orgErr))
			return
		}
		if orgCfg == nil {
			writeAIError(w, errAINotConfigured)
			return
		}
		cfg = &aiconfig.AIConfig{
			ID:              orgCfg.ID,
			Provider:        orgCfg.Provider,
			Model:           orgCfg.Model,
			APIKeyEncrypted: orgCfg.APIKeyEncrypted,
			KeyVersion:      orgCfg.KeyVersion,
		}
	}

	// 5. Decrypt key + create model
	decryptedKey, err := h.configSvc.DecryptAPIKey(cfg)
	if err != nil {
		slog.ErrorContext(ctx, "ai: chat decrypt key failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	chatModel, err := NewChatModel(ctx, cfg.Provider, cfg.Model, decryptedKey)
	if err != nil {
		slog.ErrorContext(ctx, "ai: chat create model failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"failed to create AI model"}`, http.StatusInternalServerError)
		return
	}

	// 6. Prepare tool definitions
	tools := ToolDefs()
	toolOpt := model.WithTools(tools)

	// 7. Build system prompt + message history
	systemPrompt := chatSystemPrompt(req.Mode)
	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
	}
	for _, m := range req.Messages {
		role := schema.User
		if m.Role == "assistant" {
			role = schema.Assistant
		}
		messages = append(messages, &schema.Message{Role: role, Content: m.Content})
	}

	// 8. Tool calling loop (max 5 iterations to prevent infinite loops)
	toolExec := NewToolExecutor(h.db, req.ChangeID)
	const maxToolRounds = 5

	for range maxToolRounds {
		// Generate (non-streaming) to check for tool calls
		response, genErr := chatModel.Generate(ctx, messages, toolOpt)
		if genErr != nil {
			slog.ErrorContext(ctx, "ai: chat generate failed", slog.String("error", genErr.Error()))
			http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
			return
		}

		// No tool calls → stream the final response
		if len(response.ToolCalls) == 0 {
			break
		}

		// Process tool calls
		messages = append(messages, response)

		allReadTools := true
		for _, tc := range response.ToolCalls {
			if IsWriteTool(tc.Function.Name) {
				allReadTools = false
			}
		}

		if !allReadTools {
			// Has write tools — send confirmation event and stream text so far via SSE
			writeSSEChatWithToolCalls(w, response, messages)
			return
		}

		// All read tools — auto-execute and continue the loop
		for _, tc := range response.ToolCalls {
			result, execErr := toolExec.ExecuteReadTool(ctx, tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				slog.ErrorContext(ctx, "ai: execute read tool failed",
					slog.String("tool", tc.Function.Name),
					slog.String("error", execErr.Error()),
				)
				result = fmt.Sprintf("Error: %s", execErr.Error())
			}
			messages = append(messages, schema.ToolMessage(result, tc.ID))
		}
	}

	// 9. Stream final response via SSE
	sr, err := chatModel.Stream(ctx, messages, toolOpt)
	if err != nil {
		slog.ErrorContext(ctx, "ai: chat stream failed", slog.String("error", err.Error()))
		http.Error(w, `{"error":"generation failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	defer sr.Close()
	for {
		msg, recvErr := sr.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			slog.ErrorContext(ctx, "ai: chat stream recv error", slog.String("error", recvErr.Error()))
			break
		}

		chunk := map[string]string{"content": msg.Content}
		data, marshalErr := json.Marshal(chunk)
		if marshalErr != nil {
			continue
		}
		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", data); writeErr != nil {
			return
		}
		flusher.Flush()
	}

	if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
		return
	}
	flusher.Flush()
}

// HandleExecuteTool executes a confirmed write tool and optionally continues the conversation.
func (h *Handler) HandleExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, err := auth.ResolveFromHeader(h.jwtManager, nil, ctx, r.Header.Get("Authorization"))
	if err != nil {
		writeAIError(w, fmt.Errorf("%w: %w", errUnauthenticated, err))
		return
	}

	var req struct {
		ChangeID int64  `json:"changeId"`
		ToolName string `json:"toolName"`
		ToolArgs string `json:"toolArgs"`
		ToolID   string `json:"toolId"`
		Approved bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAIError(w, fmt.Errorf("%w: %w", errBadRequest, err))
		return
	}

	// Verify change ownership — changeID is required for write tools
	if req.ChangeID <= 0 {
		writeAIError(w, fmt.Errorf("%w: changeId is required", errBadRequest))
		return
	}
	var exists bool
	exists, err = h.db.NewSelect().
		TableExpr("changes c").
		Join("JOIN projects p ON p.id = c.project_id").
		Where("c.id = ?", req.ChangeID).
		Where("p.organization_id = ?", claims.OrgID).
		Exists(ctx)
	if err != nil || !exists {
		writeAIError(w, errNotFound)
		return
	}

	toolExec := NewToolExecutor(h.db, req.ChangeID)
	var result string

	if req.Approved {
		result, err = toolExec.ExecuteWriteTool(ctx, req.ToolName, req.ToolArgs)
		if err != nil {
			slog.ErrorContext(ctx, "ai: execute write tool failed",
				slog.String("tool", req.ToolName),
				slog.String("error", err.Error()),
			)
			http.Error(w, fmt.Sprintf(`{"error":"tool execution failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
	} else {
		result = "User declined the action."
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"result": result}); err != nil {
		slog.ErrorContext(ctx, "ai: encode execute-tool response failed", slog.String("error", err.Error()))
	}
}

// writeSSEChatWithToolCalls sends the text content + tool_call events via SSE.
func writeSSEChatWithToolCalls(w http.ResponseWriter, response *schema.Message, _ []*schema.Message) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send any text content first
	if response.Content != "" {
		chunk := map[string]string{"content": response.Content}
		data, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data) // SSE write; network errors handled by client disconnect
		flusher.Flush()
	}

	// Send write tool calls as confirmation events
	for _, tc := range response.ToolCalls {
		if IsWriteTool(tc.Function.Name) {
			event := map[string]any{
				"tool_call": map[string]any{
					"id":   tc.ID,
					"name": tc.Function.Name,
					"args": json.RawMessage(tc.Function.Arguments),
				},
			}
			data, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data) // SSE write; network errors handled by client disconnect
			flusher.Flush()
		}
	}

	_, _ = fmt.Fprintf(w, "data: [DONE]\n\n") // SSE write; network errors handled by client disconnect
	flusher.Flush()
}

// chatSystemPrompt returns the system prompt based on chat mode.
func chatSystemPrompt(mode string) string {
	var sb strings.Builder

	sb.WriteString(`You are an AI assistant for Colign, a software change management platform.
You have access to tools that let you read and modify change documents.

IMPORTANT RULES:
- Use read tools to get current document state before making changes.
- For write operations, the user will be asked to confirm before execution.
- Write in the same language as the user's input.
- Be concise and focused.

`)

	switch mode {
	case "proposal":
		sb.WriteString(`You are in Proposal mode. Help the user draft or refine a proposal.
A proposal has three sections: problem, scope, and outOfScope.
- Use read_proposal to check the current content before suggesting changes.
- Use write_proposal to update the proposal after discussing with the user.
- Ask clarifying questions when the user's intent is unclear.`)

	case "ac":
		sb.WriteString(`You are in Acceptance Criteria mode. Help the user create or refine BDD acceptance criteria.
- Use read_proposal to understand what the change is about.
- Use list_acceptance_criteria to see what already exists.
- Use create_acceptance_criteria to add new scenarios.
- Each criterion needs a scenario name and Given/When/Then/And/But steps.`)

	default:
		sb.WriteString(`You are in General mode. Help the user think through their software changes.
- Use available tools to read current documents and provide informed suggestions.
- You can read proposals, specs, acceptance criteria, and change metadata.`)
	}

	return sb.String()
}
