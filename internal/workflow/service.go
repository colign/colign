package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrChangeNotFound    = errors.New("change not found")
	ErrInvalidTransition = errors.New("invalid stage transition")
	ErrGateNotMet        = errors.New("gate conditions not met")
	ErrChangeArchived    = errors.New("cannot modify archived change")
	ErrInvalidSubStatus  = errors.New("invalid sub-status")
)

type Service struct {
	db   *bun.DB
	gate *GateChecker
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db, gate: NewGateChecker()}
}

// EvaluateAndAdvance checks gate conditions and auto-advances the change if all are met.
// Returns true if the stage was advanced.
func (s *Service) EvaluateAndAdvance(ctx context.Context, changeID int64, orgID int64) (bool, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrChangeNotFound
		}
		return false, err
	}

	if change.ArchivedAt != nil {
		return false, ErrChangeArchived
	}

	input, err := s.buildGateInput(ctx, change)
	if err != nil {
		return false, err
	}

	if !s.gate.AllMet(change.Stage, *input) {
		return false, nil
	}

	next, ok := NextStage(change.Stage)
	if !ok {
		return false, nil // already at final stage
	}

	// Advance
	subStatus := models.SubStatusInProgress
	if next == models.StageApproved {
		subStatus = ""
	}
	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("stage = ?", next).
		Set("sub_status = ?", subStatus).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", changeID).
		Exec(ctx)
	if err != nil {
		return false, err
	}

	// Record event
	event := &models.WorkflowEvent{
		ChangeID:  changeID,
		FromStage: string(change.Stage),
		ToStage:   string(next),
		Action:    "auto_advance",
		Reason:    "gate conditions met",
	}
	_, _ = s.db.NewInsert().Model(event).Exec(ctx)

	// TODO: trigger task generation when entering Approved stage

	return true, nil
}

// Advance manually moves the change to the next stage.
// If force is false and gate conditions are not met, it returns ErrGateNotMet.
// If force is true, gate validation is bypassed.
func (s *Service) Advance(ctx context.Context, changeID int64, userID int64, orgID int64, force bool) (models.ChangeStage, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrChangeNotFound
		}
		return "", err
	}

	if change.ArchivedAt != nil {
		return "", ErrChangeArchived
	}

	next, ok := NextStage(change.Stage)
	if !ok {
		return change.Stage, fmt.Errorf("already at final stage")
	}

	if !force {
		input, err := s.buildGateInput(ctx, change)
		if err != nil {
			return "", err
		}
		if !s.gate.AllMet(change.Stage, *input) {
			return "", ErrGateNotMet
		}
	}

	subStatus := models.SubStatusInProgress
	if next == models.StageApproved {
		subStatus = ""
	}
	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("stage = ?", next).
		Set("sub_status = ?", subStatus).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", changeID).
		Exec(ctx)
	if err != nil {
		return "", err
	}

	event := &models.WorkflowEvent{
		ChangeID:  changeID,
		FromStage: string(change.Stage),
		ToStage:   string(next),
		Action:    "advance",
		Reason:    "manually advanced",
		UserID:    userID,
	}
	_, _ = s.db.NewInsert().Model(event).Exec(ctx)

	return next, nil
}

// Revert moves the change to the previous stage with a recorded reason.
func (s *Service) Revert(ctx context.Context, changeID int64, userID int64, reason string, orgID int64) error {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrChangeNotFound
		}
		return err
	}

	if change.ArchivedAt != nil {
		return ErrChangeArchived
	}

	prev, ok := PrevStage(change.Stage)
	if !ok {
		return fmt.Errorf("cannot revert from %s", change.Stage)
	}

	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("stage = ?", prev).
		Set("sub_status = ?", models.SubStatusInProgress).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", changeID).
		Exec(ctx)
	if err != nil {
		return err
	}

	event := &models.WorkflowEvent{
		ChangeID:  changeID,
		FromStage: string(change.Stage),
		ToStage:   string(prev),
		Action:    "revert",
		Reason:    reason,
		UserID:    userID,
	}
	_, _ = s.db.NewInsert().Model(event).Exec(ctx)

	return nil
}

// GetStatus returns the current stage, sub-status, and gate conditions for a change.
func (s *Service) GetStatus(ctx context.Context, changeID int64, orgID int64) (models.ChangeStage, models.SubStatus, []GateCondition, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, ErrChangeNotFound
		}
		return "", "", nil, err
	}

	input, err := s.buildGateInput(ctx, change)
	if err != nil {
		return "", "", nil, err
	}

	conditions := s.gate.Check(change.Stage, *input)
	return change.Stage, change.SubStatus, conditions, nil
}

// SetSubStatus sets the sub-status for a change.
func (s *Service) SetSubStatus(ctx context.Context, changeID int64, subStatus models.SubStatus, orgID int64) error {
	if !subStatus.IsValid() {
		return ErrInvalidSubStatus
	}

	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrChangeNotFound
		}
		return err
	}

	if change.ArchivedAt != nil {
		return ErrChangeArchived
	}

	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("sub_status = ?", subStatus).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", changeID).
		Exec(ctx)
	return err
}

func (s *Service) buildGateInput(ctx context.Context, change *models.Change) (*GateInput, error) {
	input := &GateInput{}

	// Check if proposal document exists
	proposalExists, err := s.db.NewSelect().Model((*models.Document)(nil)).
		Where("change_id = ?", change.ID).
		Where("type = ?", models.DocProposal).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	input.HasProposal = proposalExists

	// Check if spec document exists
	specExists, err := s.db.NewSelect().Model((*models.Document)(nil)).
		Where("change_id = ?", change.ID).
		Where("type = ?", models.DocSpec).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	input.HasDesign = specExists

	// Count approvals
	policy := new(models.ApprovalPolicy)
	err = s.db.NewSelect().Model(policy).
		Where("project_id = ?", change.ProjectID).
		Scan(ctx)
	if err != nil {
		// No policy means no approvals needed
		input.ApprovalsNeeded = 0
		input.ApprovalsDone = 0
		return input, nil
	}

	input.ApprovalsNeeded = policy.MinCount

	approvalCount, err := s.db.NewSelect().Model((*models.Approval)(nil)).
		Where("change_id = ?", change.ID).
		Where("status = ?", "approved").
		Count(ctx)
	if err != nil {
		return nil, err
	}
	input.ApprovalsDone = approvalCount

	return input, nil
}
