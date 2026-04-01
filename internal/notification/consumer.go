package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/events"
	"github.com/gobenpark/colign/internal/models"
	"github.com/gobenpark/colign/internal/push"
)

// Consumer listens for NotificationEvents and creates notifications + push.
type Consumer struct {
	db      *bun.DB
	service *Service
	push    *push.Service
	hub     *events.Hub
	ch      chan events.NotificationEvent
}

func NewConsumer(db *bun.DB, service *Service, pushService *push.Service, hub *events.Hub) *Consumer {
	return &Consumer{
		db:      db,
		service: service,
		push:    pushService,
		hub:     hub,
		ch:      make(chan events.NotificationEvent, 64),
	}
}

// Start begins consuming notification events in a goroutine.
func (c *Consumer) Start(ctx context.Context) {
	c.hub.SubscribeNotifications(c.ch)
	go c.run(ctx)
}

func (c *Consumer) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-c.ch:
			c.handle(ctx, evt)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, evt events.NotificationEvent) {
	// Resolve change_id from comment_id for reply events
	if evt.ChangeID == 0 {
		if commentID, ok := evt.Metadata["comment_id"]; ok {
			if cid, _ := commentID.(int64); cid > 0 {
				evt.ChangeID = c.changeIDByComment(ctx, cid)
			} else if cid, _ := commentID.(float64); cid > 0 {
				evt.ChangeID = c.changeIDByComment(ctx, int64(cid))
			}
		}
	}

	targetIDs, err := c.resolveTargets(ctx, evt)
	if err != nil {
		slog.WarnContext(ctx, "notification consumer: resolve targets failed",
			slog.String("type", evt.Type),
			slog.String("error", err.Error()))
		return
	}

	for _, userID := range targetIDs {
		if userID == evt.ActorID {
			continue // don't notify the actor
		}

		notifType := mapEventType(evt.Type)
		n, err := c.service.Create(ctx, CreateInput{
			UserID:         userID,
			Type:           notifType,
			ActorID:        evt.ActorID,
			ChangeID:       evt.ChangeID,
			ProjectID:      evt.ProjectID,
			Stage:          getMetaString(evt.Metadata, "new_stage"),
			CommentPreview: getMetaString(evt.Metadata, "preview"),
		})
		if err != nil {
			slog.WarnContext(ctx, "notification consumer: create failed",
				slog.String("error", err.Error()))
			continue
		}

		// Send push notification
		if c.push != nil {
			changeName := c.lookupChangeName(ctx, evt.ChangeID)
			actorName := c.lookupUserName(ctx, evt.ActorID)
			c.push.SendToUser(ctx, userID, push.Payload{
				Title: formatPushTitle(evt.Type, actorName),
				Body:  formatPushBody(evt, changeName),
				URL:   fmt.Sprintf("/projects/_/changes/%d", n.ChangeID),
				Tag:   fmt.Sprintf("change-%d", n.ChangeID),
			})
		}
	}
}

func (c *Consumer) resolveTargets(ctx context.Context, evt events.NotificationEvent) ([]int64, error) {
	switch evt.Type {
	case "mention":
		return getMetaInt64Slice(evt.Metadata, "mentioned_user_ids"), nil

	case "comment":
		// Mentions + project members related to the change
		targets := getMetaInt64Slice(evt.Metadata, "mentioned_user_ids")
		members, err := c.projectMembersByChange(ctx, evt.ChangeID)
		if err != nil {
			return targets, err
		}
		return dedup(append(targets, members...)), nil

	case "approve", "reject":
		// Notify change creator
		creatorID, err := c.changeCreator(ctx, evt.ChangeID)
		if err != nil {
			return nil, err
		}
		return []int64{creatorID}, nil

	case "stage_change":
		return c.projectMembersByChange(ctx, evt.ChangeID)

	default:
		return nil, nil
	}
}

func (c *Consumer) projectMembersByChange(ctx context.Context, changeID int64) ([]int64, error) {
	var userIDs []int64
	err := c.db.NewSelect().
		TableExpr("project_members pm").
		ColumnExpr("pm.user_id").
		Join("JOIN changes ch ON ch.project_id = pm.project_id").
		Where("ch.id = ?", changeID).
		Scan(ctx, &userIDs)
	return userIDs, err
}

func (c *Consumer) changeCreator(ctx context.Context, changeID int64) (int64, error) {
	// Changes don't have a creator_id yet, so we use the first workflow event actor
	var userID int64
	err := c.db.NewSelect().
		TableExpr("workflow_events").
		ColumnExpr("user_id").
		Where("change_id = ?", changeID).
		Where("user_id > 0").
		OrderExpr("created_at ASC").
		Limit(1).
		Scan(ctx, &userID)
	return userID, err
}

func (c *Consumer) changeIDByComment(ctx context.Context, commentID int64) int64 {
	var changeID int64
	if err := c.db.NewSelect().
		TableExpr("comments").
		ColumnExpr("change_id").
		Where("id = ?", commentID).
		Scan(ctx, &changeID); err != nil {
		return 0
	}
	return changeID
}

func (c *Consumer) lookupChangeName(ctx context.Context, changeID int64) string {
	var name string
	if err := c.db.NewSelect().
		TableExpr("changes").
		ColumnExpr("name").
		Where("id = ?", changeID).
		Scan(ctx, &name); err != nil {
		return ""
	}
	return name
}

func (c *Consumer) lookupUserName(ctx context.Context, userID int64) string {
	var name string
	if err := c.db.NewSelect().
		TableExpr("users").
		ColumnExpr("name").
		Where("id = ?", userID).
		Scan(ctx, &name); err != nil {
		return ""
	}
	return name
}

func mapEventType(t string) models.NotificationType {
	switch t {
	case "stage_change":
		return models.NotifStageChange
	case "comment":
		return models.NotifComment
	case "mention":
		return models.NotifMention
	case "approve":
		return models.NotifReviewRequest
	case "reject":
		return models.NotifReviewRequest
	default:
		return models.NotifComment
	}
}

func formatPushTitle(eventType, actorName string) string {
	if actorName == "" {
		actorName = "Someone"
	}
	switch eventType {
	case "stage_change":
		return actorName + " changed the stage"
	case "comment":
		return actorName + " commented"
	case "mention":
		return actorName + " mentioned you"
	case "approve":
		return actorName + " approved"
	case "reject":
		return actorName + " requested changes"
	default:
		return actorName + " updated"
	}
}

func formatPushBody(evt events.NotificationEvent, changeName string) string {
	preview := getMetaString(evt.Metadata, "preview")
	if preview != "" {
		return preview
	}
	if changeName != "" {
		return changeName
	}
	return ""
}

func getMetaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getMetaInt64Slice(m map[string]any, key string) []int64 {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch ids := v.(type) {
	case []int64:
		return ids
	case []any:
		result := make([]int64, 0, len(ids))
		for _, id := range ids {
			switch n := id.(type) {
			case int64:
				result = append(result, n)
			case float64:
				result = append(result, int64(n))
			}
		}
		return result
	}
	return nil
}

func dedup(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}
