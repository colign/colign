package workflow

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var changeColumns = []string{"id", "project_id", "name", "stage", "sub_status", "change_type", "created_at", "updated_at", "archived_at"}

// mockGateQueries sets up sqlmock expectations for buildGateInput DB queries.
func mockGateQueries(mock sqlmock.Sqlmock, proposalExists, specExists bool, approvalsDone int, policyMinCount *int) {
	// 1. Proposal document EXISTS check (always returns one row)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(proposalExists))

	// 2. Spec document EXISTS check (always returns one row)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(specExists))

	// 3. Approval policy lookup
	if policyMinCount != nil {
		policyRows := sqlmock.NewRows([]string{"id", "project_id", "policy", "min_count", "created_at", "updated_at"}).
			AddRow(int64(1), int64(1), "owner_one", *policyMinCount, time.Now(), time.Now())
		mock.ExpectQuery("SELECT").WillReturnRows(policyRows)

		// 4. Approval count
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(approvalsDone))
	} else {
		mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)
	}
}

func TestAdvance_GateNotMet_NoForce_ReturnsError(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// spec stage: no spec document → gate not met
	rows := sqlmock.NewRows(changeColumns).
		AddRow(int64(1), int64(1), "Test Change", "spec", "in_progress", "feature", time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mockGateQueries(mock, true, false, 0, nil)

	stage, err := svc.Advance(ctx, 1, 1, 1, false)
	require.ErrorIs(t, err, ErrGateNotMet)
	assert.Equal(t, "", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdvance_GateNotMet_Force_Succeeds(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// spec stage: gate not met, but force=true skips gate queries entirely
	rows := sqlmock.NewRows(changeColumns).
		AddRow(int64(1), int64(1), "Test Change", "spec", "in_progress", "feature", time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	stage, err := svc.Advance(ctx, 1, 1, 1, true)
	require.NoError(t, err)
	assert.Equal(t, "approved", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdvance_GateMet_NoForce_Succeeds(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// draft stage: proposal exists → gate met
	rows := sqlmock.NewRows(changeColumns).
		AddRow(int64(1), int64(1), "Test Change", "draft", "in_progress", "feature", time.Now(), time.Now(), nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mockGateQueries(mock, true, false, 0, nil)
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	stage, err := svc.Advance(ctx, 1, 1, 1, false)
	require.NoError(t, err)
	assert.Equal(t, "spec", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}
