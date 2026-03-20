package acceptance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var ErrNotFound = errors.New("acceptance criteria not found")

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, ac *models.AcceptanceCriteria) error {
	_, err := s.db.NewInsert().Model(ac).Exec(ctx)
	return err
}

func (s *Service) List(ctx context.Context, changeID int64) ([]models.AcceptanceCriteria, error) {
	var items []models.AcceptanceCriteria
	err := s.db.NewSelect().Model(&items).
		Where("change_id = ?", changeID).
		OrderExpr("sort_order ASC, id ASC").
		Scan(ctx)
	return items, err
}

func (s *Service) Update(ctx context.Context, id int64, scenario string, steps []models.ACStep, sortOrder int) (*models.AcceptanceCriteria, error) {
	ac := new(models.AcceptanceCriteria)
	err := s.db.NewSelect().Model(ac).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	ac.Scenario = scenario
	ac.Steps = steps
	ac.SortOrder = sortOrder
	ac.UpdatedAt = time.Now()

	_, err = s.db.NewUpdate().Model(ac).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return ac, nil
}

func (s *Service) Toggle(ctx context.Context, id int64, met bool) (*models.AcceptanceCriteria, error) {
	ac := new(models.AcceptanceCriteria)
	err := s.db.NewSelect().Model(ac).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	ac.Met = met
	ac.UpdatedAt = time.Now()

	_, err = s.db.NewUpdate().Model(ac).Column("met", "updated_at").WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return ac, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	_, err := s.db.NewDelete().Model((*models.AcceptanceCriteria)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}
