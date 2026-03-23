package workflow

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupWorkflowTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db := bun.NewDB(mockDB, pgdialect.New())
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestAdvance_ArchivedChange(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	archivedAt := time.Now()
	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "ready", "feature", time.Now(), time.Now(), archivedAt)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	stage, err := svc.Advance(ctx, 1, 1)
	require.ErrorIs(t, err, ErrChangeArchived)
	require.Equal(t, "", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdvance_NotFound(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	stage, err := svc.Advance(ctx, 999, 1)
	require.ErrorIs(t, err, ErrChangeNotFound)
	require.Equal(t, "", string(stage))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRevert_ArchivedChange(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	archivedAt := time.Now()
	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "ready", "feature", time.Now(), time.Now(), archivedAt)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	err := svc.Revert(ctx, 1, 1, "rollback")
	require.ErrorIs(t, err, ErrChangeArchived)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEvaluateAndAdvance_ArchivedChange(t *testing.T) {
	db, mock := setupWorkflowTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	archivedAt := time.Now()
	rows := sqlmock.NewRows([]string{"id", "project_id", "name", "stage", "change_type", "created_at", "updated_at", "archived_at"}).
		AddRow(int64(1), int64(1), "Test Change", "ready", "feature", time.Now(), time.Now(), archivedAt)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	advanced, err := svc.EvaluateAndAdvance(ctx, 1)
	require.ErrorIs(t, err, ErrChangeArchived)
	require.False(t, advanced)
	require.NoError(t, mock.ExpectationsWereMet())
}
