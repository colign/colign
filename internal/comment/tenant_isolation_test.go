package comment

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestResolve_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	result, err := svc.Resolve(ctx, 1, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrCommentNotFound)
	require.Nil(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	err := svc.Delete(ctx, 1, 1, wrongOrgID)
	require.ErrorIs(t, err, ErrCommentNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateReply_CrossTenantBlocked(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	wrongOrgID := int64(999)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	result, err := svc.CreateReply(ctx, 1, "body", 1, wrongOrgID, nil)
	require.ErrorIs(t, err, ErrCommentNotFound)
	require.Nil(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}
