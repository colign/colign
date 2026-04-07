package auth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"golang.org/x/crypto/bcrypt"

	"github.com/gobenpark/colign/internal/auth"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupService(t *testing.T) (*auth.Service, sqlmock.Sqlmock, *auth.JWTManager) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	bunDB := bun.NewDB(db, pgdialect.New())
	jwtMgr := auth.NewJWTManager("test-secret-key-for-service-tests")
	svc := auth.NewService(bunDB, jwtMgr)
	return svc, mock, jwtMgr
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

// ---------------------------------------------------------------------------
// 1. Login
// ---------------------------------------------------------------------------

func TestService_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()
		passwordHash := mustHashPassword(t, "password123")

		// 1) Select user by email
		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "email_verified", "created_at", "updated_at"}).
			AddRow(int64(1), "alice@example.com", passwordHash, "Alice", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT").
			WillReturnRows(userRows)

		// 2) getDefaultOrgID: select organization_members
		orgMemberRows := sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role", "created_at"}).
			AddRow(int64(1), int64(10), int64(1), "owner", time.Now())
		mock.ExpectQuery("SELECT").
			WillReturnRows(orgMemberRows)

		// 3) createSession: insert session
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "alice@example.com",
			Password: "password123",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Greater(t, result.ExpiresAt, int64(0))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()
		passwordHash := mustHashPassword(t, "correct-password")

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "email_verified", "created_at", "updated_at"}).
			AddRow(int64(1), "alice@example.com", passwordHash, "Alice", false, time.Now(), time.Now())
		mock.ExpectQuery("SELECT").
			WillReturnRows(userRows)

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "alice@example.com",
			Password: "wrong-password",
		})

		assert.Nil(t, result)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()

		mock.ExpectQuery("SELECT").
			WillReturnError(sql.ErrNoRows)

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		})

		assert.Nil(t, result)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// 2. RefreshToken
// ---------------------------------------------------------------------------

func TestService_RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()

		refreshToken := "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344"
		now := time.Now()

		// 1) Select session (bun selects all columns: id, user_id, refresh_token, user_agent, ip, expires_at, created_at)
		sessionRows := sqlmock.NewRows([]string{"id", "user_id", "refresh_token", "user_agent", "ip", "expires_at", "created_at"}).
			AddRow(int64(5), int64(1), refreshToken, "", "", now.Add(7*24*time.Hour), now)
		mock.ExpectQuery("SELECT").WillReturnRows(sessionRows)

		// 2) bun Relation("User") — separate SELECT for the user
		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar_url", "email_verified", "created_at", "updated_at"}).
			AddRow(int64(1), "alice@example.com", "hash", "Alice", "", true, now, now)
		mock.ExpectQuery("SELECT").WillReturnRows(userRows)

		// 3) Delete old session
		mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))

		// 4) getOrCreateDefaultOrg → getDefaultOrgID
		orgMemberRows := sqlmock.NewRows([]string{"id", "organization_id", "user_id", "role", "created_at"}).
			AddRow(int64(1), int64(10), int64(1), "owner", time.Now())
		mock.ExpectQuery("SELECT").WillReturnRows(orgMemberRows)

		// 5) Insert new session
		mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(6)))

		result, err := svc.RefreshToken(ctx, refreshToken)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.NotEqual(t, refreshToken, result.RefreshToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("expired or invalid token", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()

		mock.ExpectQuery("SELECT").
			WillReturnError(sql.ErrNoRows)

		result, err := svc.RefreshToken(ctx, "nonexistent-refresh-token")

		assert.Nil(t, result)
		require.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// 3. VerifyEmail
// ---------------------------------------------------------------------------

func TestService_VerifyEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()

		verificationToken := "aabbccddee112233aabbccddee112233aabbccddee112233aabbccddee112233"

		// 1) Select email verification by token
		verificationRows := sqlmock.NewRows([]string{"id", "user_id", "token", "expires_at", "created_at"}).
			AddRow(int64(1), int64(42), verificationToken, time.Now().Add(24*time.Hour), time.Now())
		mock.ExpectQuery("SELECT").
			WillReturnRows(verificationRows)

		// 2) Update user email_verified
		mock.ExpectExec("UPDATE").
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 3) Delete verification record
		mock.ExpectExec("DELETE").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := svc.VerifyEmail(ctx, verificationToken)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid token", func(t *testing.T) {
		svc, mock, _ := setupService(t)
		ctx := context.Background()

		mock.ExpectQuery("SELECT").
			WillReturnError(sql.ErrNoRows)

		err := svc.VerifyEmail(ctx, "bad-token")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired verification token")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// 4. Me
// ---------------------------------------------------------------------------

func TestService_Me(t *testing.T) {
	t.Run("success with valid JWT header", func(t *testing.T) {
		svc, mock, jwtMgr := setupService(t)
		ctx := context.Background()

		accessToken, err := jwtMgr.GenerateAccessToken(7, "me@example.com", "MeUser", 20)
		require.NoError(t, err)

		userRows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "email_verified", "created_at", "updated_at"}).
			AddRow(int64(7), "me@example.com", "hash", "MeUser", true, time.Now(), time.Now())
		mock.ExpectQuery("SELECT").
			WillReturnRows(userRows)

		user, orgID, err := svc.Me(ctx, "Bearer "+accessToken)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(7), user.ID)
		assert.Equal(t, "me@example.com", user.Email)
		assert.Equal(t, "MeUser", user.Name)
		assert.Equal(t, int64(20), orgID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid authorization header", func(t *testing.T) {
		svc, _, _ := setupService(t)
		ctx := context.Background()

		user, orgID, err := svc.Me(ctx, "InvalidHeader")

		assert.Nil(t, user)
		assert.Equal(t, int64(0), orgID)
		require.Error(t, err)
		assert.EqualError(t, err, "invalid authorization header")
	})

	t.Run("user not found in DB", func(t *testing.T) {
		svc, mock, jwtMgr := setupService(t)
		ctx := context.Background()

		accessToken, err := jwtMgr.GenerateAccessToken(999, "ghost@example.com", "Ghost", 1)
		require.NoError(t, err)

		mock.ExpectQuery("SELECT").
			WillReturnError(sql.ErrNoRows)

		user, orgID, err := svc.Me(ctx, "Bearer "+accessToken)

		assert.Nil(t, user)
		assert.Equal(t, int64(0), orgID)
		require.ErrorIs(t, err, auth.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// 5. generateOrgSlug (via internal test in slug_internal_test.go)
// ---------------------------------------------------------------------------

// generateOrgSlug is unexported, so thorough property tests live in
// slug_internal_test.go (package auth). Here we verify that Register
// exercises the slug generation path end-to-end without errors.

func TestGenerateOrgSlug_ViaRegister(t *testing.T) {
	t.Run("Register successfully generates an org slug", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		bunDB := bun.NewDB(db, pgdialect.New())
		jwtMgr := auth.NewJWTManager("slug-test-secret")
		svc := auth.NewService(bunDB, jwtMgr)
		ctx := context.Background()

		// 1) Check email existence — return false
		mock.ExpectQuery("SELECT").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// 2) Begin TX
		mock.ExpectBegin()

		// 3) Insert user (inside TX)
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

		// 4) Insert email verification (inside TX)
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

		// 5) Commit TX
		mock.ExpectCommit()

		// 6) Insert organization (outside TX, orgJoiner is nil so orgID=0)
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

		// 7) Insert org member
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

		// 8) createSession: insert session
		mock.ExpectQuery("INSERT").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

		result, err := svc.Register(ctx, auth.RegisterRequest{
			Email:            "slug@example.com",
			Password:         "password123",
			Name:             "Test User",
			OrganizationName: "My Cool Organization",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
