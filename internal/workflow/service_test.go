package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gobenpark/colign/internal/models"
)

func TestShouldAutoAdvance(t *testing.T) {
	gate := NewGateChecker()

	// Draft with proposal done → should advance
	assert.True(t, gate.AllMet(models.StageDraft, GateInput{HasProposal: true}),
		"should auto-advance from Draft when proposal exists")

	// Spec with spec and approvals done → should advance
	assert.True(t, gate.AllMet(models.StageSpec, GateInput{
		HasDesign:       true,
		ApprovalsNeeded: 2,
		ApprovalsDone:   2,
	}), "should auto-advance from Spec when spec complete and approvals met")

	// Spec without enough approvals → should not advance
	assert.False(t, gate.AllMet(models.StageSpec, GateInput{
		HasDesign:       true,
		ApprovalsNeeded: 2,
		ApprovalsDone:   1,
	}), "should not auto-advance from Spec with insufficient approvals")
}

func TestSetSubStatus(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "sub_status", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "draft", "in_progress", "feature", time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.SetSubStatus(ctx, 1, models.SubStatusReady, 1)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetSubStatus_InvalidStatus(t *testing.T) {
	db, _ := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.SetSubStatus(ctx, 1, models.SubStatus("invalid"), 1)
	require.ErrorIs(t, err, ErrInvalidSubStatus)
}

func TestSetSubStatus_ArchivedChange(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	archivedAt := time.Now()
	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "sub_status", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "draft", "in_progress", "feature", time.Now(), time.Now(), archivedAt)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	err := svc.SetSubStatus(ctx, 1, models.SubStatusReady, 1)
	require.ErrorIs(t, err, ErrChangeArchived)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatus_IncludesSubStatus(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "sub_status", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "draft", "ready", "feature", time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	stage, subStatus, conditions, err := svc.GetStatus(ctx, 1, 1)
	require.NoError(t, err)
	require.Equal(t, models.StageDraft, stage)
	require.Equal(t, models.SubStatusReady, subStatus)
	require.NotNil(t, conditions)
	require.NoError(t, mock.ExpectationsWereMet())
}
