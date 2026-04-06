package wiki

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
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

func TestCreatePage(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Expect max sort_order query
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(2))

	// Expect insert (bun uses RETURNING)
	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{}))

	page, err := svc.CreatePage(ctx, 1, nil, "Test Page", 10)
	require.NoError(t, err)
	assert.Equal(t, "Test Page", page.Title)
	assert.Equal(t, int64(1), page.ProjectID)
	assert.Equal(t, int64(10), page.CreatedBy)
	assert.Equal(t, 3, page.SortOrder) // maxSort(2) + 1
	assert.Nil(t, page.ParentID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePage_WithParent(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	parentID := uuid.New()

	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))

	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{})) // bun uses RETURNING

	page, err := svc.CreatePage(ctx, 1, &parentID, "Child Page", 10)
	require.NoError(t, err)
	assert.Equal(t, "Child Page", page.Title)
	require.NotNil(t, page.ParentID)
	assert.Equal(t, parentID, *page.ParentID)
	assert.Equal(t, 1, page.SortOrder)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPage_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	page, err := svc.GetPage(ctx, 1, uuid.New())
	require.ErrorIs(t, err, ErrPageNotFound)
	assert.Nil(t, page)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePage(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	pageID := uuid.New()

	mock.ExpectExec("WITH RECURSIVE").
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := svc.DeletePage(ctx, 1, pageID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRestorePage(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	pageID := uuid.New()

	// Expect recursive restore
	mock.ExpectExec("WITH RECURSIVE").
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Expect GetPage select
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "title", "sort_order", "icon", "content_text", "created_by", "created_at", "updated_at"}).
			AddRow(pageID, 1, "Restored Page", 0, "", "", 10, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"))

	page, err := svc.RestorePage(ctx, 1, pageID)
	require.NoError(t, err)
	assert.Equal(t, "Restored Page", page.Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUploadImage_TooLarge(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	data := make([]byte, maxImageSize+1)

	img, err := svc.UploadImage(ctx, 1, uuid.New(), "big.png", "image/png", data, 10)
	require.ErrorIs(t, err, ErrImageTooLarge)
	assert.Nil(t, img)
}

func TestUploadImage_OK(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	pageID := uuid.New()
	data := []byte("fake image data")

	mock.ExpectQuery("INSERT").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, "2026-01-01T00:00:00Z"))

	img, err := svc.UploadImage(ctx, 1, pageID, "test.png", "image/png", data, 10)
	require.NoError(t, err)
	assert.Equal(t, "test.png", img.Filename)
	assert.Equal(t, "image/png", img.ContentType)
	assert.Equal(t, len(data), img.Size)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetImage_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	img, err := svc.GetImage(ctx, 999)
	require.ErrorIs(t, err, ErrImageNotFound)
	assert.Nil(t, img)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReorderPage_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	err := svc.ReorderPage(ctx, 1, uuid.New(), nil, 5)
	require.ErrorIs(t, err, ErrPageNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
