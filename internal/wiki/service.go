package wiki

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrPageNotFound  = errors.New("wiki page not found")
	ErrImageNotFound = errors.New("wiki image not found")
	ErrImageTooLarge = errors.New("image exceeds 5MB limit")
)

const maxImageSize = 5 * 1024 * 1024 // 5MB

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

// CreatePage creates a new wiki page in the given project.
func (s *Service) CreatePage(ctx context.Context, projectID int64, parentID *uuid.UUID, title string, userID int64) (*models.WikiPage, error) {
	// Determine sort_order: append after last sibling
	var maxSort int
	err := s.db.NewSelect().
		Model((*models.WikiPage)(nil)).
		ColumnExpr("COALESCE(MAX(sort_order), -1)").
		Where("project_id = ?", projectID).
		Where("deleted_at IS NULL").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if parentID != nil {
				return q.Where("parent_id = ?", *parentID)
			}
			return q.Where("parent_id IS NULL")
		}).
		Scan(ctx, &maxSort)
	if err != nil {
		return nil, err
	}

	page := &models.WikiPage{
		ID:        uuid.New(),
		ProjectID: projectID,
		ParentID:  parentID,
		Title:     title,
		SortOrder: maxSort + 1,
		CreatedBy: userID,
	}

	if _, err := s.db.NewInsert().Model(page).Exec(ctx); err != nil {
		return nil, err
	}

	return page, nil
}

// GetPage returns a single wiki page by ID.
func (s *Service) GetPage(ctx context.Context, projectID int64, pageID uuid.UUID) (*models.WikiPage, error) {
	page := new(models.WikiPage)
	err := s.db.NewSelect().Model(page).
		Where("wp.id = ?", pageID).
		Where("wp.project_id = ?", projectID).
		Where("wp.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPageNotFound
		}
		return nil, err
	}
	return page, nil
}

// ListPages returns all non-deleted wiki pages for a project as a flat list.
// The caller is responsible for building the tree structure.
func (s *Service) ListPages(ctx context.Context, projectID int64) ([]*models.WikiPage, error) {
	var pages []*models.WikiPage
	err := s.db.NewSelect().Model(&pages).
		Relation("Creator").
		Where("wp.project_id = ?", projectID).
		Where("wp.deleted_at IS NULL").
		OrderExpr("wp.sort_order ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return pages, nil
}

// UpdatePage updates the title, icon, and/or content of a wiki page.
func (s *Service) UpdatePage(ctx context.Context, projectID int64, pageID uuid.UUID, title, icon, contentJSON, contentText *string) (*models.WikiPage, error) {
	page, err := s.GetPage(ctx, projectID, pageID)
	if err != nil {
		return nil, err
	}

	columns := []string{"updated_at"}
	if title != nil {
		page.Title = *title
		columns = append(columns, "title")
	}
	if icon != nil {
		page.Icon = *icon
		columns = append(columns, "icon")
	}
	if contentJSON != nil {
		page.ContentJSON = *contentJSON
		columns = append(columns, "content_json")
	}
	if contentText != nil {
		page.ContentText = *contentText
		columns = append(columns, "content_text")
	}
	page.UpdatedAt = time.Now()

	if _, err := s.db.NewUpdate().Model(page).
		Column(columns...).
		WherePK().
		Exec(ctx); err != nil {
		return nil, err
	}

	return page, nil
}

// DeletePage soft-deletes a wiki page and all its descendants.
func (s *Service) DeletePage(ctx context.Context, projectID int64, pageID uuid.UUID) error {
	now := time.Now()

	// Soft-delete the page and all descendants using recursive CTE
	_, err := s.db.NewRaw(`
		WITH RECURSIVE descendants AS (
			SELECT id FROM wiki_pages WHERE id = ? AND project_id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT wp.id FROM wiki_pages wp JOIN descendants d ON wp.parent_id = d.id WHERE wp.deleted_at IS NULL
		)
		UPDATE wiki_pages SET deleted_at = ? WHERE id IN (SELECT id FROM descendants)
	`, pageID, projectID, now).Exec(ctx)

	return err
}

// RestorePage restores a soft-deleted wiki page and all its descendants.
func (s *Service) RestorePage(ctx context.Context, projectID int64, pageID uuid.UUID) (*models.WikiPage, error) {
	// Restore the page and all descendants
	_, err := s.db.NewRaw(`
		WITH RECURSIVE descendants AS (
			SELECT id FROM wiki_pages WHERE id = ? AND project_id = ?
			UNION ALL
			SELECT wp.id FROM wiki_pages wp JOIN descendants d ON wp.parent_id = d.id
		)
		UPDATE wiki_pages SET deleted_at = NULL WHERE id IN (SELECT id FROM descendants)
	`, pageID, projectID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return s.GetPage(ctx, projectID, pageID)
}

// ReorderPage moves a page to a new parent and/or sort position,
// renumbering all siblings to maintain consistent ordering.
func (s *Service) ReorderPage(ctx context.Context, projectID int64, pageID uuid.UUID, parentID *uuid.UUID, sortOrder int) error {
	if _, err := s.GetPage(ctx, projectID, pageID); err != nil {
		return err
	}

	// Get all siblings under the target parent (excluding the moved page)
	var siblings []*models.WikiPage
	q := s.db.NewSelect().Model(&siblings).
		Where("wp.project_id = ?", projectID).
		Where("wp.deleted_at IS NULL").
		Where("wp.id != ?", pageID).
		OrderExpr("wp.sort_order ASC")

	if parentID != nil {
		q = q.Where("wp.parent_id = ?", *parentID)
	} else {
		q = q.Where("wp.parent_id IS NULL")
	}

	if err := q.Scan(ctx); err != nil {
		return err
	}

	// Insert the moved page at the target position
	now := time.Now()
	ordered := make([]*models.WikiPage, 0, len(siblings)+1)
	for i, s := range siblings {
		if i == sortOrder {
			ordered = append(ordered, &models.WikiPage{ID: pageID})
		}
		ordered = append(ordered, s)
	}
	if len(ordered) <= sortOrder {
		ordered = append(ordered, &models.WikiPage{ID: pageID})
	}

	// Update all sort_orders in a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i, p := range ordered {
		update := tx.NewUpdate().Model((*models.WikiPage)(nil)).
			Set("sort_order = ?", i).
			Set("updated_at = ?", now).
			Where("id = ?", p.ID)

		// Also update parent_id for the moved page
		if p.ID == pageID {
			if parentID != nil {
				update = update.Set("parent_id = ?", *parentID)
			} else {
				update = update.Set("parent_id = NULL")
			}
		}

		if _, err := update.Exec(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UploadImage stores an image in the database and returns its ID.
func (s *Service) UploadImage(ctx context.Context, projectID int64, pageID uuid.UUID, filename, contentType string, data []byte, userID int64) (*models.WikiImage, error) {
	if len(data) > maxImageSize {
		return nil, ErrImageTooLarge
	}

	img := &models.WikiImage{
		ProjectID:   projectID,
		PageID:      pageID,
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
		Size:        len(data),
		CreatedBy:   userID,
	}

	if _, err := s.db.NewInsert().Model(img).Exec(ctx); err != nil {
		return nil, err
	}

	return img, nil
}

// GetImage returns an image by ID.
func (s *Service) GetImage(ctx context.Context, imageID int64) (*models.WikiImage, error) {
	img := new(models.WikiImage)
	err := s.db.NewSelect().Model(img).
		Where("wi.id = ?", imageID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, err
	}
	return img, nil
}
