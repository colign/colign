package models

import (
	"time"

	"github.com/uptrace/bun"
)

type NotificationType string

const (
	NotifReviewRequest NotificationType = "review_request"
	NotifComment       NotificationType = "comment"
	NotifMention       NotificationType = "mention"
	NotifStageChange   NotificationType = "stage_change"
	NotifInvite        NotificationType = "invite"
)

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:n"`

	ID               int64            `bun:"id,pk,autoincrement"`
	UserID           int64            `bun:"user_id,notnull"`
	Type             NotificationType `bun:"type,notnull"`
	Read             bool             `bun:"read,notnull,default:false"`
	ActorID          int64            `bun:"actor_id"`
	ChangeID         int64            `bun:"change_id"`
	ProjectID        int64            `bun:"project_id"`
	Stage            string           `bun:"stage"`
	CommentPreview   string           `bun:"comment_preview"`
	MentionedUserIDs []int64          `bun:"mentioned_user_ids,array"`
	CreatedAt        time.Time        `bun:"created_at,notnull,default:current_timestamp"`

	User    *User    `bun:"rel:belongs-to,join:user_id=id"`
	Actor   *User    `bun:"rel:belongs-to,join:actor_id=id"`
	Change  *Change  `bun:"rel:belongs-to,join:change_id=id"`
	Project *Project `bun:"rel:belongs-to,join:project_id=id"`
}
