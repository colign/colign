package comment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
	"github.com/gobenpark/colign/internal/notification"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrNotAuthorized   = errors.New("not authorized")
)

type Service struct {
	db            *bun.DB
	notifications *notification.Service
}

func NewService(db *bun.DB, notifications *notification.Service) *Service {
	return &Service{db: db, notifications: notifications}
}

func (s *Service) Create(ctx context.Context, changeID int64, documentType, quotedText, body string, userID int64, mentionedUserIDs []int64) (*models.Comment, error) {
	// Look up the project_id from the change instead of trusting the client
	var projectID int64
	err := s.db.NewSelect().
		Model((*models.Change)(nil)).
		Column("project_id").
		Where("id = ?", changeID).
		Scan(ctx, &projectID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

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

	s.notifyMentions(ctx, mentionNotificationInput{
		ProjectID:        projectID,
		ChangeID:         changeID,
		ActorID:          userID,
		MentionedUserIDs: mentionedUserIDs,
		CommentPreview:   body,
	})

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

func (s *Service) Resolve(ctx context.Context, commentID int64, userID int64, orgID int64) (*models.Comment, error) {
	comment := new(models.Comment)
	err := s.db.NewSelect().Model(comment).
		Join("JOIN changes AS ch ON ch.id = c.change_id").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("c.id = ?", commentID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
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

func (s *Service) Delete(ctx context.Context, commentID int64, userID int64, orgID int64) error {
	comment := new(models.Comment)
	err := s.db.NewSelect().Model(comment).
		Join("JOIN changes AS ch ON ch.id = c.change_id").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("c.id = ?", commentID).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
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

func (s *Service) CreateReply(ctx context.Context, commentID int64, body string, userID int64, orgID int64, mentionedUserIDs []int64) (*models.CommentReply, error) {
	parentComment := struct {
		ID        int64 `bun:"id"`
		ChangeID  int64 `bun:"change_id"`
		ProjectID int64 `bun:"project_id"`
	}{}
	err := s.db.NewSelect().
		Model((*models.Comment)(nil)).
		Join("JOIN changes AS ch ON ch.id = c.change_id").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("c.id = ?", commentID).
		Where("p.organization_id = ?", orgID).
		Column("c.id", "c.change_id").
		ColumnExpr("ch.project_id AS project_id").
		Scan(ctx, &parentComment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
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

	s.notifyMentions(ctx, mentionNotificationInput{
		ProjectID:        parentComment.ProjectID,
		ChangeID:         parentComment.ChangeID,
		ActorID:          userID,
		MentionedUserIDs: mentionedUserIDs,
		CommentPreview:   body,
	})

	return reply, nil
}

type mentionNotificationInput struct {
	ProjectID        int64
	ChangeID         int64
	ActorID          int64
	MentionedUserIDs []int64
	CommentPreview   string
}

func (s *Service) notifyMentions(ctx context.Context, input mentionNotificationInput) {
	if s.notifications == nil || len(input.MentionedUserIDs) == 0 || input.ProjectID == 0 {
		return
	}

	uniqueIDs := make([]int64, 0, len(input.MentionedUserIDs))
	seen := make(map[int64]struct{}, len(input.MentionedUserIDs))
	for _, userID := range input.MentionedUserIDs {
		if userID == 0 || userID == input.ActorID {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		uniqueIDs = append(uniqueIDs, userID)
	}
	if len(uniqueIDs) == 0 {
		return
	}

	var memberIDs []int64
	if err := s.db.NewSelect().
		Model((*models.ProjectMember)(nil)).
		Column("user_id").
		Where("project_id = ?", input.ProjectID).
		Where("user_id IN (?)", bun.In(uniqueIDs)).
		Scan(ctx, &memberIDs); err != nil {
		slog.Warn("failed to query project members for mention notification", "error", err, "project_id", input.ProjectID)
		return
	}

	preview := strings.TrimSpace(input.CommentPreview)
	if len(preview) > 140 {
		preview = preview[:140]
	}

	for _, userID := range memberIDs {
		if _, err := s.notifications.Create(ctx, notification.CreateInput{
			UserID:         userID,
			Type:           models.NotifMention,
			ActorID:        input.ActorID,
			ChangeID:       input.ChangeID,
			ProjectID:      input.ProjectID,
			CommentPreview: preview,
		}); err != nil {
			slog.Warn("failed to create mention notification", "error", err, "user_id", userID, "change_id", input.ChangeID)
		}
	}
}
