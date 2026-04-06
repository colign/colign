package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WikiImage struct {
	bun.BaseModel `bun:"table:wiki_images,alias:wi"`

	ID          int64     `bun:"id,pk,autoincrement"`
	ProjectID   int64     `bun:"project_id,notnull"`
	PageID      uuid.UUID `bun:"page_id,notnull,type:uuid"`
	Filename    string    `bun:"filename,notnull"`
	ContentType string    `bun:"content_type,notnull"`
	Data        []byte    `bun:"data,notnull,type:bytea"`
	Size        int       `bun:"size,notnull"`
	CreatedBy   int64     `bun:"created_by,notnull"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`

	Project *Project  `bun:"rel:belongs-to,join:project_id=id"`
	Page    *WikiPage `bun:"rel:belongs-to,join:page_id=id"`
	Creator *User     `bun:"rel:belongs-to,join:created_by=id"`
}
