package task

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrTaskNotFound  = errors.New("task not found")
	ErrInvalidChange = errors.New("task does not belong to specified change")
)

type ReorderItem struct {
	ID         int64
	Status     string
	OrderIndex int
}

// ArchiveEvaluator is implemented by the archive service to trigger auto-archive
// evaluation when a task transitions to done.
type ArchiveEvaluator interface {
	EvaluateAutoArchive(ctx context.Context, changeID int64) (bool, error)
}

// Option configures a Service.
type Option func(*Service)

// WithArchiveEvaluator injects an ArchiveEvaluator into the service.
func WithArchiveEvaluator(ae ArchiveEvaluator) Option {
	return func(s *Service) { s.archiveEvaluator = ae }
}

type Service struct {
	db               *bun.DB
	archiveEvaluator ArchiveEvaluator
}

func NewService(db *bun.DB, opts ...Option) *Service {
	s := &Service{db: db}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) List(ctx context.Context, changeID, projectID, orgID int64) ([]models.Task, error) {
	tasks := make([]models.Task, 0)
	err := s.db.NewSelect().Model(&tasks).
		Relation("Assignee").
		Relation("Creator").
		Join("JOIN changes AS c ON c.id = t.change_id").
		Join("JOIN projects AS p ON p.id = c.project_id").
		Where("t.change_id = ?", changeID).
		Where("c.project_id = ?", projectID).
		Where("p.organization_id = ?", orgID).
		OrderExpr("t.status ASC, t.order_index ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Service) Create(ctx context.Context, task *models.Task, projectID, orgID int64) error {
	exists, err := s.db.NewSelect().
		TableExpr("changes c").
		Join("JOIN projects p ON p.id = c.project_id").
		ColumnExpr("1").
		Where("c.id = ?", task.ChangeID).
		Where("c.project_id = ?", projectID).
		Where("p.organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidChange
	}

	var maxOrderIndex int
	err = s.db.NewSelect().
		TableExpr("tasks").
		ColumnExpr("COALESCE(MAX(order_index), -1)").
		Where("change_id = ?", task.ChangeID).
		Where("status = ?", task.Status).
		Scan(ctx, &maxOrderIndex)
	if err != nil {
		return err
	}

	task.OrderIndex = maxOrderIndex + 1
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if _, err := s.db.NewInsert().Model(task).Exec(ctx); err != nil {
		return err
	}

	// Reload with relations
	if err := s.db.NewSelect().Model(task).
		Relation("Assignee").
		Relation("Creator").
		WherePK().
		Scan(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) Update(ctx context.Context, id int64, title *string, description *string, status *string, specRef *string, assigneeID *int64, clearAssignee bool, orgID int64) (*models.Task, error) {
	task := new(models.Task)
	err := s.db.NewSelect().Model(task).
		Join("JOIN changes AS c ON c.id = t.change_id").
		Join("JOIN projects AS p ON p.id = c.project_id").
		Where("t.id = ?", id).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	columns := []string{"updated_at"}
	task.UpdatedAt = time.Now()

	if title != nil {
		task.Title = *title
		columns = append(columns, "title")
	}
	if description != nil {
		task.Description = *description
		columns = append(columns, "description")
	}
	if status != nil {
		task.Status = models.TaskStatus(*status)
		columns = append(columns, "status")
	}
	if specRef != nil {
		task.SpecRef = *specRef
		columns = append(columns, "spec_ref")
	}
	if clearAssignee {
		task.AssigneeID = nil
		columns = append(columns, "assignee_id")
	} else if assigneeID != nil {
		task.AssigneeID = assigneeID
		columns = append(columns, "assignee_id")
	}

	if _, err := s.db.NewUpdate().Model(task).
		Column(columns...).
		WherePK().
		Exec(ctx); err != nil {
		return nil, err
	}

	// Reload with relations
	if err := s.db.NewSelect().Model(task).
		Relation("Assignee").
		Relation("Creator").
		WherePK().
		Scan(ctx); err != nil {
		return nil, err
	}

	if status != nil && models.TaskStatus(*status) == models.TaskDone && s.archiveEvaluator != nil {
		if _, err := s.archiveEvaluator.EvaluateAutoArchive(ctx, task.ChangeID); err != nil {
			slog.Error("auto-archive evaluation failed", "error", err, "change_id", task.ChangeID)
		}
	}

	return task, nil
}

func (s *Service) Delete(ctx context.Context, id int64, orgID int64) error {
	res, err := s.db.NewDelete().
		TableExpr("tasks").
		Where("id = ?", id).
		Where("change_id IN (SELECT c.id FROM changes c JOIN projects p ON p.id = c.project_id WHERE p.organization_id = ?)", orgID).
		Exec(ctx)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (s *Service) Reorder(ctx context.Context, changeID, projectID int64, items []ReorderItem, orgID int64) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Verify change belongs to org
		exists, err := tx.NewSelect().
			TableExpr("changes c").
			Join("JOIN projects p ON p.id = c.project_id").
			ColumnExpr("1").
			Where("c.id = ?", changeID).
			Where("c.project_id = ?", projectID).
			Where("p.organization_id = ?", orgID).
			Exists(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return ErrTaskNotFound
		}

		if len(items) == 0 {
			return nil
		}

		ids := make([]int64, len(items))
		for i, item := range items {
			ids[i] = item.ID
		}

		var count int
		err = tx.NewSelect().
			TableExpr("tasks").
			ColumnExpr("COUNT(*)").
			Where("id IN (?)", bun.List(ids)).
			Where("change_id = ?", changeID).
			Scan(ctx, &count)
		if err != nil {
			return err
		}
		if count != len(items) {
			return ErrInvalidChange
		}

		for _, item := range items {
			if _, err := tx.NewUpdate().
				TableExpr("tasks").
				Set("status = ?", item.Status).
				Set("order_index = ?", item.OrderIndex).
				Set("updated_at = NOW()").
				Where("id = ?", item.ID).
				Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}
