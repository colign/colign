package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Comment struct {
	bun.BaseModel `bun:"table:comments,alias:c"`

	ID           int64     `bun:"id,pk,autoincrement"`
	ChangeID     int64     `bun:"change_id,notnull"`
	DocumentType string    `bun:"document_type,notnull"`
	QuotedText   string    `bun:"quoted_text,notnull,default:''"`
	Body         string    `bun:"body,notnull"`
	UserID       int64     `bun:"user_id,notnull"`
	Resolved     bool      `bun:"resolved,notnull,default:false"`
	ResolvedBy   *int64    `bun:"resolved_by"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `bun:"updated_at,notnull,default:current_timestamp"`

	Change  *Change        `bun:"rel:belongs-to,join:change_id=id"`
	User    *User          `bun:"rel:belongs-to,join:user_id=id"`
	Replies []CommentReply `bun:"rel:has-many,join:id=comment_id"`
}

type CommentReply struct {
	bun.BaseModel `bun:"table:comment_replies,alias:cr"`

	ID        int64     `bun:"id,pk,autoincrement"`
	CommentID int64     `bun:"comment_id,notnull"`
	Body      string    `bun:"body,notnull"`
	UserID    int64     `bun:"user_id,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`

	Comment *Comment `bun:"rel:belongs-to,join:comment_id=id"`
	User    *User    `bun:"rel:belongs-to,join:user_id=id"`
}

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:n"`

	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    int64     `bun:"user_id,notnull"`
	Type      string    `bun:"type,notnull"` // comment, review, stage_change
	Title     string    `bun:"title,notnull"`
	Body      string    `bun:"body"`
	Link      string    `bun:"link"`
	Read      bool      `bun:"read,notnull,default:false"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
