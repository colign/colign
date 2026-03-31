package models

import (
	"time"

	"github.com/uptrace/bun"
)

type ChangeStage string

const (
	StageDraft    ChangeStage = "draft"
	StageSpec     ChangeStage = "spec"
	StageApproved ChangeStage = "approved"
)

func StageOrder() []ChangeStage {
	return []ChangeStage{StageDraft, StageSpec, StageApproved}
}

type SubStatus string

const (
	SubStatusInProgress SubStatus = "in_progress"
	SubStatusReady      SubStatus = "ready"
)

func (s SubStatus) IsValid() bool {
	return s == SubStatusInProgress || s == SubStatusReady
}

type ChangeType string

const (
	ChangeFeature  ChangeType = "feature"
	ChangeBugfix   ChangeType = "bugfix"
	ChangeRefactor ChangeType = "refactor"
)

type Change struct {
	bun.BaseModel `bun:"table:changes,alias:ch"`

	ID         int64       `bun:"id,pk,autoincrement"`
	ProjectID  int64       `bun:"project_id,notnull"`
	Number     int         `bun:"number,notnull"`
	Name       string      `bun:"name,notnull"`
	Stage      ChangeStage `bun:"stage,notnull,default:'draft'"`
	SubStatus  SubStatus   `bun:"sub_status,notnull,default:'in_progress'"`
	ChangeType ChangeType  `bun:"change_type,notnull,default:'feature'"`
	CreatedAt  time.Time   `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt  time.Time   `bun:"updated_at,notnull,default:current_timestamp"`
	ArchivedAt *time.Time  `bun:"archived_at"`

	Project *Project       `bun:"rel:belongs-to,join:project_id=id"`
	Labels  []ProjectLabel `bun:"m2m:change_label_assignments,join:Change=Label"`
}

type ChangeLabelAssignment struct {
	bun.BaseModel `bun:"table:change_label_assignments"`

	ChangeID int64         `bun:"change_id,pk"`
	LabelID  int64         `bun:"label_id,pk"`
	Change   *Change       `bun:"rel:belongs-to,join:change_id=id"`
	Label    *ProjectLabel `bun:"rel:belongs-to,join:label_id=id"`
}
