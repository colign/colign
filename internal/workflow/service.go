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
func (s *Service) EvaluateAndAdvance(ctx context.Context, changeID int64) (bool, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).Where("id = ?", changeID).Scan(ctx)
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
	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("stage = ?", next).
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

	// TODO: trigger task generation when entering Ready stage

	return true, nil
}

// Advance manually moves the change to the next stage.
func (s *Service) Advance(ctx context.Context, changeID int64, userID int64) (models.ChangeStage, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).Where("id = ?", changeID).Scan(ctx)
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

	_, err = s.db.NewUpdate().Model((*models.Change)(nil)).
		Set("stage = ?", next).
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
func (s *Service) Revert(ctx context.Context, changeID int64, userID int64, reason string) error {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).Where("id = ?", changeID).Scan(ctx)
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

// GetStatus returns the current gate conditions for a change.
func (s *Service) GetStatus(ctx context.Context, changeID int64) (models.ChangeStage, []GateCondition, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).Where("id = ?", changeID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, ErrChangeNotFound
		}
		return "", nil, err
	}

	input, err := s.buildGateInput(ctx, change)
	if err != nil {
		return "", nil, err
	}

	conditions := s.gate.Check(change.Stage, *input)
	return change.Stage, conditions, nil
}

func (s *Service) buildGateInput(_ context.Context, _ *models.Change) (*GateInput, error) {
	input := &GateInput{}

	// TODO: integrate with document storage
	input.HasProposal = true
	input.HasDesign = false

	return input, nil
}
