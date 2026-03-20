package models

import (
	"time"

	"github.com/uptrace/bun"
)

type TaskStatus string

const (
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
)

type Task struct {
	bun.BaseModel `bun:"table:tasks,alias:t"`

	ID          int64      `bun:"id,pk,autoincrement"`
	ChangeID    int64      `bun:"change_id,notnull"`
	Title       string     `bun:"title,notnull"`
	Description string     `bun:"description"`
	Status      TaskStatus `bun:"status,notnull,default:'todo'"`
	OrderIndex  int        `bun:"order_index,notnull,default:0"`
	SpecRef     string     `bun:"spec_ref"` // reference to spec requirement
	AssigneeID  *int64     `bun:"assignee_id"`
	CreatorID   *int64     `bun:"creator_id"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`

	Change   *Change `bun:"rel:belongs-to,join:change_id=id"`
	Assignee *User   `bun:"rel:belongs-to,join:assignee_id=id"`
	Creator  *User   `bun:"rel:belongs-to,join:creator_id=id"`
}
