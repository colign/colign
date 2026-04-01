package models

import (
	"time"

	"github.com/uptrace/bun"
)

type PushSubscription struct {
	bun.BaseModel `bun:"table:push_subscriptions,alias:ps"`

	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    int64     `bun:"user_id,notnull"`
	Endpoint  string    `bun:"endpoint,notnull"`
	P256dh    string    `bun:"p256dh,notnull"`
	Auth      string    `bun:"auth,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`

	User *User `bun:"rel:belongs-to,join:user_id=id"`
}
