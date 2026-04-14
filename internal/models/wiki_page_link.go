package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WikiPageLink struct {
	bun.BaseModel `bun:"table:wiki_page_links,alias:wpl"`

	ID           int64     `bun:"id,pk,autoincrement"`
	SourcePageID uuid.UUID `bun:"source_page_id,type:uuid,notnull"`
	TargetPageID uuid.UUID `bun:"target_page_id,type:uuid,notnull"`
	ProjectID    int64     `bun:"project_id,notnull"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:current_timestamp"`

	SourcePage *WikiPage `bun:"rel:belongs-to,join:source_page_id=id"`
	TargetPage *WikiPage `bun:"rel:belongs-to,join:target_page_id=id"`
}
