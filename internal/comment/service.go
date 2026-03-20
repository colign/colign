package comment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrNotAuthorized   = errors.New("not authorized")
)

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, changeID int64, documentType, quotedText, body string, userID int64) (*models.Comment, error) {
	comment := &models.Comment{
		ChangeID:     changeID,
		DocumentType: documentType,
		QuotedText:   quotedText,
		Body:         body,
		UserID:       userID,
	}

	if _, err := s.db.NewInsert().Model(comment).Exec(ctx); err != nil {
		return nil, err
	}

	// Reload with user relation
	if err := s.db.NewSelect().Model(comment).
		Relation("User").
		Where("c.id = ?", comment.ID).
		Scan(ctx); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *Service) List(ctx context.Context, changeID int64, documentType string) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.NewSelect().Model(&comments).
		Relation("User").
		Relation("Replies", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Relation("User").OrderExpr("cr.created_at ASC")
		}).
		Where("c.change_id = ?", changeID).
		Where("c.document_type = ?", documentType).
		OrderExpr("c.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (s *Service) Resolve(ctx context.Context, commentID int64, userID int64) (*models.Comment, error) {
	comment := new(models.Comment)
	err := s.db.NewSelect().Model(comment).Where("id = ?", commentID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	comment.Resolved = true
	comment.ResolvedBy = &userID
	comment.UpdatedAt = time.Now()

	if _, err := s.db.NewUpdate().Model(comment).
		Column("resolved", "resolved_by", "updated_at").
		WherePK().
		Exec(ctx); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *Service) Delete(ctx context.Context, commentID int64, userID int64) error {
	comment := new(models.Comment)
	err := s.db.NewSelect().Model(comment).Where("id = ?", commentID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCommentNotFound
		}
		return err
	}

	if comment.UserID != userID {
		return ErrNotAuthorized
	}

	if _, err := s.db.NewDelete().Model(comment).WherePK().Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateReply(ctx context.Context, commentID int64, body string, userID int64) (*models.CommentReply, error) {
	// Verify comment exists
	exists, err := s.db.NewSelect().Model((*models.Comment)(nil)).Where("id = ?", commentID).Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCommentNotFound
	}

	reply := &models.CommentReply{
		CommentID: commentID,
		Body:      body,
		UserID:    userID,
	}

	if _, err := s.db.NewInsert().Model(reply).Exec(ctx); err != nil {
		return nil, err
	}

	// Reload with user
	if err := s.db.NewSelect().Model(reply).
		Relation("User").
		Where("cr.id = ?", reply.ID).
		Scan(ctx); err != nil {
		return nil, err
	}

	return reply, nil
}
