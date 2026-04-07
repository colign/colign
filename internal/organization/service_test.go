package organization

import (
	"context"
	"database/sql"
	"errors"
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
	svc := NewService(db, nil)
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
	svc := NewService(db, nil)
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
	svc := NewService(db, nil)
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
	svc := NewService(db, nil)
	ctx := context.Background()

	// SELECT member — no rows
	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)

	_, err := svc.Delete(ctx, 1, 10)
	require.ErrorIs(t, err, ErrOrgNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// AcceptPendingInvitations
// ---------------------------------------------------------------------------

func TestAcceptPendingInvitations_NoPendingInvitations(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	// FindPendingInvitationsByEmail returns empty set
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "email", "role", "token", "invited_by", "status", "expires_at", "created_at"}))

	orgID, err := svc.AcceptPendingInvitations(ctx, 1, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), orgID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptPendingInvitations_AcceptsSingleInvitation(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	// FindPendingInvitationsByEmail returns one invitation
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "email", "role", "token", "invited_by", "status", "expires_at", "created_at"}).
			AddRow(int64(100), int64(5), "alice@example.com", "member", "tok-abc", int64(2), "pending", "2099-01-01 00:00:00", "2025-01-01 00:00:00"))

	// Insert organization member (ON CONFLICT DO UPDATE)
	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	// Update invitation status to accepted
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	orgID, err := svc.AcceptPendingInvitations(ctx, 1, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(5), orgID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptPendingInvitations_AcceptsMultipleInvitations(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	// FindPendingInvitationsByEmail returns two invitations
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "email", "role", "token", "invited_by", "status", "expires_at", "created_at"}).
			AddRow(int64(100), int64(5), "alice@example.com", "member", "tok-1", int64(2), "pending", "2099-01-01 00:00:00", "2025-01-01 00:00:00").
			AddRow(int64(101), int64(8), "alice@example.com", "admin", "tok-2", int64(3), "pending", "2099-01-01 00:00:00", "2025-01-01 00:00:00"))

	// First invitation: insert member + update status
	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Second invitation: insert member + update status
	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	orgID, err := svc.AcceptPendingInvitations(ctx, 1, "alice@example.com")
	require.NoError(t, err)
	// Returns the first org ID joined
	assert.Equal(t, int64(5), orgID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptPendingInvitations_DBErrorOnFindInvitations(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	// FindPendingInvitationsByEmail returns a DB error
	dbErr := errors.New("connection refused")
	mock.ExpectQuery("SELECT").
		WillReturnError(dbErr)

	orgID, err := svc.AcceptPendingInvitations(ctx, 1, "alice@example.com")
	require.Error(t, err)
	assert.Equal(t, int64(0), orgID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptPendingInvitations_DoesNotDoDomainAutoJoin(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db, nil)
	ctx := context.Background()

	// AcceptPendingInvitations should ONLY query org_invitations, never organizations.
	// With no pending invitations, only one SELECT should fire.
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "email", "role", "token", "invited_by", "status", "expires_at", "created_at"}))

	orgID, err := svc.AcceptPendingInvitations(ctx, 1, "alice@corp.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), orgID)

	// If domain auto-join were running, there would be a second SELECT
	// against organizations with allowed_domains. Verify only 1 query ran.
	require.NoError(t, mock.ExpectationsWereMet())
}
