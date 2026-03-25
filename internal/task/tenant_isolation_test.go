package task

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	db := bun.NewDB(mockDB, pgdialect.New())
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func TestTaskUpdate_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	// SELECT with JOINs returns no rows — wrong org
	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)

	result, err := svc.Update(ctx, 1, nil, nil, nil, nil, nil, false, wrongOrgID)
	assert.Nil(t, result)
	require.ErrorIs(t, err, ErrTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskDelete_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	// DELETE returns 0 rows affected — wrong org
	mock.ExpectExec("DELETE").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.Delete(ctx, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskReorder_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectBegin()

	// bun Exists() returns false — change does not belong to this org
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectRollback()

	err := svc.Reorder(ctx, 1, 1, []ReorderItem{{ID: 1, Status: "todo", OrderIndex: 0}}, wrongOrgID)
	require.ErrorIs(t, err, ErrTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
