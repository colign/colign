package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/schema"
	"github.com/uptrace/bun"
)

// ToolExecutor handles execution of AI agent tools.
// changeID is injected from the request — tools don't need to specify it.
type ToolExecutor struct {
	db       *bun.DB
	changeID int64
}

// NewToolExecutor creates a new ToolExecutor for a specific change.
func NewToolExecutor(db *bun.DB, changeID int64) *ToolExecutor {
	return &ToolExecutor{db: db, changeID: changeID}
}

// ToolDefs returns the tool definitions for the AI model.
// Note: change_id is NOT a parameter — it's injected server-side from the request.
func ToolDefs() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		// Read tools
		{
			Name: "read_proposal",
			Desc: "Read the current proposal content (problem, scope, outOfScope) for this change.",
		},
		{
			Name: "list_acceptance_criteria",
			Desc: "List all acceptance criteria (BDD scenarios) for this change.",
		},
		{
			Name: "read_spec",
			Desc: "Read the spec (technical design) document for this change.",
		},
		{
			Name: "get_change_info",
			Desc: "Get metadata about this change: name, current stage, identifier.",
		},
		// Write tools (require user confirmation)
		{
			Name: "write_proposal",
			Desc: "Update the proposal content. Requires user confirmation before execution.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"problem":      {Type: schema.String, Desc: "Problem section content"},
				"scope":        {Type: schema.String, Desc: "Scope section content"},
				"out_of_scope": {Type: schema.String, Desc: "Out of scope section content"},
			}),
		},
		{
			Name: "create_acceptance_criteria",
			Desc: "Create new acceptance criteria. Requires user confirmation before execution.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"criteria": {Type: schema.Array, Desc: "Array of criteria with scenario and steps", ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
				}},
			}),
		},
	}
}

// IsWriteTool returns true if the tool requires user confirmation.
func IsWriteTool(name string) bool {
	switch name {
	case "write_proposal", "create_acceptance_criteria":
		return true
	default:
		return false
	}
}

// ExecuteReadTool executes a read-only tool and returns the result as a string.
func (e *ToolExecutor) ExecuteReadTool(ctx context.Context, name string, _ string) (string, error) {
	switch name {
	case "read_proposal":
		return e.readProposal(ctx, e.changeID)
	case "list_acceptance_criteria":
		return e.listAC(ctx, e.changeID)
	case "read_spec":
		return e.readSpec(ctx, e.changeID)
	case "get_change_info":
		return e.getChangeInfo(ctx, e.changeID)
	default:
		return "", fmt.Errorf("unknown read tool: %s", name)
	}
}

func (e *ToolExecutor) readProposal(ctx context.Context, changeID int64) (string, error) {
	var content string
	err := e.db.NewSelect().
		ColumnExpr("content").
		TableExpr("documents").
		Where("change_id = ?", changeID).
		Where("type = ?", "proposal").
		Scan(ctx, &content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "No proposal document exists yet for this change.", nil
		}
		return "", err
	}
	if content == "" {
		return "Proposal exists but is empty.", nil
	}
	return content, nil
}

func (e *ToolExecutor) listAC(ctx context.Context, changeID int64) (string, error) {
	var rows []struct {
		Scenario string `bun:"scenario"`
		Steps    string `bun:"steps"`
		Met      bool   `bun:"met"`
	}
	err := e.db.NewSelect().
		ColumnExpr("scenario, steps, met").
		TableExpr("acceptance_criteria").
		Where("change_id = ?", changeID).
		OrderExpr("sort_order ASC").
		Scan(ctx, &rows)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "No acceptance criteria defined yet.", nil
		}
		return "", err
	}
	if len(rows) == 0 {
		return "No acceptance criteria defined yet.", nil
	}

	result, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (e *ToolExecutor) readSpec(ctx context.Context, changeID int64) (string, error) {
	var content string
	err := e.db.NewSelect().
		ColumnExpr("content").
		TableExpr("documents").
		Where("change_id = ?", changeID).
		Where("type = ?", "spec").
		Scan(ctx, &content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "No spec document exists yet for this change.", nil
		}
		return "", err
	}
	if content == "" {
		return "Spec exists but is empty.", nil
	}
	return content, nil
}

// ExecuteWriteTool executes a write tool that was confirmed by the user.
func (e *ToolExecutor) ExecuteWriteTool(ctx context.Context, name string, argsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("parse tool args: %w", err)
	}
	// Inject changeID from executor (not from AI-generated args)
	args["change_id"] = float64(e.changeID)

	switch name {
	case "write_proposal":
		return e.writeProposal(ctx, args)
	case "create_acceptance_criteria":
		return e.createAC(ctx, args)
	default:
		return "", fmt.Errorf("unknown write tool: %s", name)
	}
}

func (e *ToolExecutor) writeProposal(ctx context.Context, args map[string]any) (string, error) {
	changeID := int64(args["change_id"].(float64))

	proposal := map[string]string{}
	if v, ok := args["problem"].(string); ok {
		proposal["problem"] = v
	}
	if v, ok := args["scope"].(string); ok {
		proposal["scope"] = v
	}
	if v, ok := args["out_of_scope"].(string); ok {
		proposal["outOfScope"] = v
	}

	content, err := json.Marshal(proposal)
	if err != nil {
		return "", fmt.Errorf("marshal proposal: %w", err)
	}

	_, dbErr := e.db.NewRaw(
		"INSERT INTO documents (change_id, type, title, content) VALUES (?, 'proposal', 'proposal', ?) "+
			"ON CONFLICT (change_id, type) DO UPDATE SET content = EXCLUDED.content, updated_at = now()",
		changeID, string(content),
	).Exec(ctx)
	if dbErr != nil {
		return "", fmt.Errorf("save proposal: %w", dbErr)
	}

	return "Proposal updated successfully.", nil
}

func (e *ToolExecutor) createAC(ctx context.Context, args map[string]any) (string, error) {
	changeID := int64(args["change_id"].(float64))
	criteriaRaw, ok := args["criteria"]
	if !ok {
		return "", errors.New("criteria field is required")
	}

	criteriaJSON, err := json.Marshal(criteriaRaw)
	if err != nil {
		return "", fmt.Errorf("marshal criteria: %w", err)
	}

	var criteria []struct {
		Scenario string `json:"scenario"`
		Steps    []struct {
			Keyword string `json:"keyword"`
			Text    string `json:"text"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(criteriaJSON, &criteria); err != nil {
		return "", fmt.Errorf("parse criteria: %w", err)
	}

	// Get current max sort_order
	var maxOrder int
	_ = e.db.NewSelect().
		ColumnExpr("COALESCE(MAX(sort_order), -1)").
		TableExpr("acceptance_criteria").
		Where("change_id = ?", changeID).
		Scan(ctx, &maxOrder)

	for i, ac := range criteria {
		stepsJSON, marshalErr := json.Marshal(ac.Steps)
		if marshalErr != nil {
			continue
		}
		_, insertErr := e.db.NewRaw(
			"INSERT INTO acceptance_criteria (change_id, scenario, steps, sort_order, project_id) "+
				"VALUES (?, ?, ?, ?, (SELECT project_id FROM changes WHERE id = ?))",
			changeID, ac.Scenario, string(stepsJSON), maxOrder+1+i, changeID,
		).Exec(ctx)
		if insertErr != nil {
			slog.ErrorContext(ctx, "ai: create AC failed",
				slog.String("scenario", ac.Scenario),
				slog.String("error", insertErr.Error()),
			)
		}
	}

	return fmt.Sprintf("%d acceptance criteria created.", len(criteria)), nil
}

func (e *ToolExecutor) getChangeInfo(ctx context.Context, changeID int64) (string, error) {
	var info struct {
		Name       string `bun:"name" json:"name"`
		Identifier string `bun:"identifier" json:"identifier"`
		Stage      string `bun:"stage" json:"stage"`
	}

	err := e.db.NewSelect().
		ColumnExpr("name, COALESCE(number::text, '') AS identifier, stage").
		TableExpr("changes").
		Where("id = ?", changeID).
		Scan(ctx, &info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "Change not found.", nil
		}
		slog.ErrorContext(ctx, "ai: get change info failed", slog.String("error", err.Error()))
		return "", err
	}

	result, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
