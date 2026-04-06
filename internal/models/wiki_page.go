package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WikiPage struct {
	bun.BaseModel `bun:"table:wiki_pages,alias:wp"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ProjectID   int64      `bun:"project_id,notnull"`
	ParentID    *uuid.UUID `bun:"parent_id,type:uuid"`
	Title       string     `bun:"title,notnull,default:'Untitled'"`
	Icon        string     `bun:"icon,notnull,default:''"`
	SortOrder   int        `bun:"sort_order,notnull,default:0"`
	YjsState    []byte     `bun:"yjs_state,type:bytea"`
	ContentJSON string     `bun:"content_json,type:jsonb"`
	ContentText string     `bun:"content_text,notnull,default:''"`
	CreatedBy   int64      `bun:"created_by,notnull"`
	DeletedAt   *time.Time `bun:"deleted_at,soft_delete"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`

	Project  *Project    `bun:"rel:belongs-to,join:project_id=id"`
	Parent   *WikiPage   `bun:"rel:belongs-to,join:parent_id=id"`
	Children []*WikiPage `bun:"rel:has-many,join:id=parent_id"`
	Creator  *User       `bun:"rel:belongs-to,join:created_by=id"`
}
