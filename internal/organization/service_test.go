package organization

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

func TestDelete_OwnerSuccess(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	orgID := int64(1)
	userID := int64(10)
	nextOrgID := int64(2)

	// 1. SELECT member — user is owner
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role"}).
			AddRow(1, orgID, userID, "owner"))

	// 2. COUNT user's orgs — 2 orgs
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3. SELECT next org
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role"}).
			AddRow(2, nextOrgID, userID, "member"))

	// 4. DELETE organization
	mock.ExpectExec("DELETE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := svc.Delete(ctx, orgID, userID)
	require.NoError(t, err)
	assert.Equal(t, nextOrgID, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_NonOwnerRejected(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// SELECT member — user is admin, not owner
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role"}).
			AddRow(1, 1, 10, "admin"))

	_, err := svc.Delete(ctx, 1, 10)
	require.ErrorIs(t, err, ErrNotOwner)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_SingleOrgRejected(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// SELECT member — user is owner
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role"}).
			AddRow(1, 1, 10, "owner"))

	// COUNT user's orgs — only 1
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	_, err := svc.Delete(ctx, 1, 10)
	require.ErrorIs(t, err, ErrLastOrganization)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_NotMember(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// SELECT member — no rows
	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)

	_, err := svc.Delete(ctx, 1, 10)
	require.ErrorIs(t, err, ErrOrgNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
