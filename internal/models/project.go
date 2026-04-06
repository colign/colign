package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type ProjectStatus string

const (
	ProjectStatusBacklog   ProjectStatus = "backlog"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
)

type ProjectPriority string

const (
	ProjectPriorityUrgent ProjectPriority = "urgent"
	ProjectPriorityHigh   ProjectPriority = "high"
	ProjectPriorityMedium ProjectPriority = "medium"
	ProjectPriorityLow    ProjectPriority = "low"
	ProjectPriorityNone   ProjectPriority = "none"
)

type ProjectHealth string

const (
	ProjectHealthOnTrack  ProjectHealth = "on_track"
	ProjectHealthAtRisk   ProjectHealth = "at_risk"
	ProjectHealthOffTrack ProjectHealth = "off_track"
)

type Project struct {
	bun.BaseModel `bun:"table:projects,alias:p"`

	ID             int64           `bun:"id,pk,autoincrement"`
	OrganizationID int64           `bun:"organization_id"`
	Name           string          `bun:"name,notnull"`
	Slug           string          `bun:"slug,notnull"`
	Identifier     string          `bun:"identifier,notnull"`
	Description    string          `bun:"description"`
	Status         ProjectStatus   `bun:"status,notnull,default:'backlog'"`
	Priority       ProjectPriority `bun:"priority,notnull,default:'none'"`
	Health         ProjectHealth   `bun:"health,notnull,default:'on_track'"`
	LeadID         *int64          `bun:"lead_id"`
	StartDate      *time.Time      `bun:"start_date,type:date"`
	TargetDate     *time.Time      `bun:"target_date,type:date"`
	Icon           string          `bun:"icon,notnull,default:'layers'"`
	Color          string          `bun:"color,notnull,default:'#7C3AED'"`
	CreatedAt      time.Time       `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time       `bun:"updated_at,notnull,default:current_timestamp"`

	Organization *Organization   `bun:"rel:belongs-to,join:organization_id=id"`
	Members      []ProjectMember `bun:"rel:has-many,join:id=project_id"`
	Lead         *User           `bun:"rel:belongs-to,join:lead_id=id"`
	Labels       []ProjectLabel  `bun:"m2m:project_label_assignments,join:Project=Label"`
}

type ProjectMember struct {
	bun.BaseModel `bun:"table:project_members,alias:pm"`

	ID        int64     `bun:"id,pk,autoincrement"`
	ProjectID int64     `bun:"project_id,notnull"`
	UserID    int64     `bun:"user_id,notnull"`
	Role      Role      `bun:"role,notnull,default:'viewer'"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`

	Project *Project `bun:"rel:belongs-to,join:project_id=id"`
	User    *User    `bun:"rel:belongs-to,join:user_id=id"`
}

type ProjectLabel struct {
	bun.BaseModel `bun:"table:project_labels,alias:pl"`

	ID             int64     `bun:"id,pk,autoincrement"`
	OrganizationID int64     `bun:"organization_id,notnull"`
	Name           string    `bun:"name,notnull"`
	Color          string    `bun:"color,notnull,default:'#6B7280'"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type ProjectLabelAssignment struct {
	bun.BaseModel `bun:"table:project_label_assignments"`

	ProjectID int64         `bun:"project_id,pk"`
	LabelID   int64         `bun:"label_id,pk"`
	Project   *Project      `bun:"rel:belongs-to,join:project_id=id"`
	Label     *ProjectLabel `bun:"rel:belongs-to,join:label_id=id"`
}
