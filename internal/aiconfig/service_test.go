package aiconfig

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// newTestDB creates a bun.DB backed by sqlmock.
func newTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	t.Cleanup(func() { _ = sqlDB.Close() })
	return bunDB, mock
}

var svcKey = []byte("svckey0000000000svckey0000000000") // 32 bytes

// ---------------------------------------------------------------------------
// NewService
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	bunDB, _ := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	require.NotNil(t, svc)
	assert.Equal(t, bunDB, svc.db)
	assert.Equal(t, svcKey, svc.encryptionKey)
}

// ---------------------------------------------------------------------------
// Upsert — Create (new record)
// ---------------------------------------------------------------------------

func TestUpsert_Create(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	// bun INSERT … ON CONFLICT with RETURNING uses Query (not Exec).
	mock.ExpectQuery(`INSERT INTO "ai_configs"`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, time.Now(), time.Now()),
		)

	cfg, err := svc.Upsert(context.Background(), 42, UpsertInput{
		Provider:              "anthropic",
		Model:                 "claude-3-5-sonnet",
		APIKey:                "sk-ant-test-key-1234567890abcdef",
		IncludeProjectContext: true,
	})

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, int64(1), cfg.ID)
	assert.Equal(t, int64(42), cfg.ProjectID)
	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "claude-3-5-sonnet", cfg.Model)
	assert.True(t, cfg.IncludeProjectContext)
	// Verify that the API key was encrypted (non-empty bytes stored).
	assert.NotEmpty(t, cfg.APIKeyEncrypted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Upsert — Update keeping existing key when APIKey is empty
// ---------------------------------------------------------------------------

func TestUpsert_Update_KeepKey(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	// Pre-encrypt a key to simulate an existing record.
	existingEncrypted, err := Encrypt("sk-existing-key", svcKey, 1)
	require.NoError(t, err)

	// First call: SELECT to fetch existing record.
	mock.ExpectQuery(`SELECT .+ FROM "ai_configs"`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "project_id", "provider", "model", "api_key_encrypted", "key_version", "include_project_context", "created_at", "updated_at"}).
				AddRow(1, 42, "openai", "gpt-4", existingEncrypted, 1, false, time.Now(), time.Now()),
		)

	// Second call: INSERT … ON CONFLICT DO UPDATE (RETURNING).
	mock.ExpectQuery(`INSERT INTO "ai_configs"`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(1, time.Now(), time.Now()),
		)

	cfg, err := svc.Upsert(context.Background(), 42, UpsertInput{
		Provider:              "openai",
		Model:                 "gpt-4o",
		APIKey:                "", // empty → keep existing
		IncludeProjectContext: false,
	})

	require.NoError(t, err)
	require.NotNil(t, cfg)
	// The stored encrypted key must be the original one (unchanged).
	assert.Equal(t, existingEncrypted, cfg.APIKeyEncrypted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByProjectID — Found
// ---------------------------------------------------------------------------

func TestGetByProjectID_Found(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	existingEncrypted, err := Encrypt("sk-secret", svcKey, 1)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT .+ FROM "ai_configs"`).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "project_id", "provider", "model", "api_key_encrypted", "key_version", "include_project_context", "created_at", "updated_at"}).
				AddRow(5, 99, "anthropic", "claude-3-haiku", existingEncrypted, 1, true, time.Now(), time.Now()),
		)

	cfg, err := svc.GetByProjectID(context.Background(), 99)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, int64(5), cfg.ID)
	assert.Equal(t, int64(99), cfg.ProjectID)
	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "claude-3-haiku", cfg.Model)
	assert.True(t, cfg.IncludeProjectContext)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByProjectID — Not Found (returns nil, nil)
// ---------------------------------------------------------------------------

func TestGetByProjectID_NotFound(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	mock.ExpectQuery(`SELECT .+ FROM "ai_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "model", "api_key_encrypted", "key_version", "include_project_context", "created_at", "updated_at"}))

	cfg, err := svc.GetByProjectID(context.Background(), 999)

	require.NoError(t, err)
	assert.Nil(t, cfg, "not found should return nil config with no error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	mock.ExpectExec(`DELETE FROM "ai_configs"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.Delete(context.Background(), 42)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_DBError(t *testing.T) {
	bunDB, mock := newTestDB(t)
	svc := NewService(bunDB, svcKey)

	mock.ExpectExec(`DELETE FROM "ai_configs"`).
		WillReturnError(assert.AnError)

	err := svc.Delete(context.Background(), 42)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DecryptAPIKey
// ---------------------------------------------------------------------------

func TestDecryptAPIKey(t *testing.T) {
	svc := &Service{encryptionKey: svcKey}

	original := "sk-ant-test-key-abcdef"
	encrypted, err := Encrypt(original, svcKey, 1)
	require.NoError(t, err)

	cfg := &AIConfig{APIKeyEncrypted: encrypted}
	decrypted, err := svc.DecryptAPIKey(cfg)

	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestDecryptAPIKey_WrongKey(t *testing.T) {
	svc := &Service{encryptionKey: []byte("wrongkey00000000wrongkey00000000")}

	encrypted, err := Encrypt("secret", svcKey, 1)
	require.NoError(t, err)

	cfg := &AIConfig{APIKeyEncrypted: encrypted}
	_, err = svc.DecryptAPIKey(cfg)

	assert.Error(t, err)
}
