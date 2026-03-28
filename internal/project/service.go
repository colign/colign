package project

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrNotAuthorized      = errors.New("not authorized")
	slugRegexp            = regexp.MustCompile(`[^a-z0-9-]+`)
	identifierStripRegexp = regexp.MustCompile(`[^A-Za-z0-9 -]+`)
)

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

// GenerateIdentifier creates a short uppercase identifier from a project name.
// Multi-word names use first letters (My Cool Project → MCP).
// Single words use the first 3 characters (Colign → COL).
func GenerateIdentifier(name string) string {
	// Strip special chars, keep letters/digits/spaces/hyphens
	cleaned := identifierStripRegexp.ReplaceAllString(name, "")
	cleaned = strings.TrimSpace(cleaned)

	// Split into words (by space or hyphen)
	words := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == ' ' || r == '-'
	})

	var id string
	if len(words) > 1 {
		// Multi-word: take first letter of each word (up to 5)
		for _, w := range words {
			if len(id) >= 5 {
				break
			}
			if w != "" {
				id += string([]rune(strings.ToUpper(w))[0])
			}
		}
	} else if len(words) == 1 {
		// Single word: take first 3 chars
		upper := strings.ToUpper(words[0])
		runes := []rune(upper)
		if len(runes) > 3 {
			runes = runes[:3]
		}
		id = string(runes)
	}

	// Fallback: use slug first 3 chars
	if id == "" {
		slug := GenerateSlug(name)
		upper := strings.ToUpper(slug)
		runes := []rune(upper)
		if len(runes) > 3 {
			runes = runes[:3]
		}
		id = string(runes)
	}

	// Final fallback
	if id == "" {
		id = "PRJ"
	}

	return id
}

func GenerateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRegexp.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	// Collapse multiple dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

func (s *Service) ensureUniqueIdentifier(ctx context.Context, identifier string, orgID int64, excludeProjectID int64) (string, error) {
	base := strings.TrimSpace(identifier)
	if base == "" {
		return "", fmt.Errorf("identifier must not be empty")
	}
	if len(base) > 5 {
		base = base[:5]
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			suffix := fmt.Sprintf("%d", i+1)
			// Truncate base to leave room for the suffix within 5-char limit
			maxBase := 5 - len(suffix)
			if maxBase < 1 {
				maxBase = 1
			}
			truncated := base
			if len(truncated) > maxBase {
				truncated = truncated[:maxBase]
			}
			candidate = truncated + suffix
		}
		q := s.db.NewSelect().Model((*models.Project)(nil)).
			Where("identifier = ?", candidate).
			Where("organization_id = ?", orgID)
		if excludeProjectID > 0 {
			q = q.Where("id != ?", excludeProjectID)
		}
		exists, err := q.Exists(ctx)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not generate unique identifier for %q", identifier)
}

// nextChangeNumber returns the next sequential number for a change within a project.
// Locks the project row with FOR UPDATE to prevent concurrent number collisions.
func (s *Service) nextChangeNumber(ctx context.Context, tx bun.Tx, projectID int64) (int, error) {
	// Lock the project row to serialize concurrent change creations
	var dummy int64
	if err := tx.NewSelect().
		Model((*models.Project)(nil)).
		Column("id").
		Where("id = ?", projectID).
		For("UPDATE").
		Scan(ctx, &dummy); err != nil {
		return 0, err
	}

	var maxNum int
	err := tx.NewSelect().
		Model((*models.Change)(nil)).
		ColumnExpr("COALESCE(MAX(number), 0)").
		Where("project_id = ?", projectID).
		Scan(ctx, &maxNum)
	if err != nil {
		return 0, err
	}
	return maxNum + 1, nil
}

func (s *Service) ensureUniqueSlug(ctx context.Context, slug string, orgID int64) (string, error) {
	return s.ensureUniqueSlugExcluding(ctx, slug, orgID, 0)
}

func (s *Service) ensureUniqueSlugExcluding(ctx context.Context, slug string, orgID int64, excludeProjectID int64) (string, error) {
	baseSlug := slug
	for i := 0; ; i++ {
		candidate := baseSlug
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", baseSlug, i+1)
		}
		q := s.db.NewSelect().Model((*models.Project)(nil)).
			Where("slug = ?", candidate).
			Where("organization_id = ?", orgID)
		if excludeProjectID > 0 {
			q = q.Where("id != ?", excludeProjectID)
		}
		exists, err := q.Exists(ctx)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
}

type CreateProjectInput struct {
	Name           string
	Description    string
	UserID         int64
	OrganizationID int64
}

func (s *Service) Create(ctx context.Context, input CreateProjectInput) (*models.Project, error) {
	slug := GenerateSlug(input.Name)
	uniqueSlug, err := s.ensureUniqueSlug(ctx, slug, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	identifier := GenerateIdentifier(input.Name)
	uniqueIdentifier, err := s.ensureUniqueIdentifier(ctx, identifier, input.OrganizationID, 0)
	if err != nil {
		return nil, err
	}

	project := &models.Project{
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		Slug:           uniqueSlug,
		Identifier:     uniqueIdentifier,
		Description:    input.Description,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewInsert().Model(project).Exec(ctx); err != nil {
		return nil, err
	}

	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    input.UserID,
		Role:      models.RoleOwner,
	}
	if _, err := tx.NewInsert().Model(member).Exec(ctx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return project, nil
}

func parseProjectRef(projectRef string) (int64, string, bool) {
	head, _, found := strings.Cut(projectRef, "-")
	if !found {
		if id, err := strconv.ParseInt(projectRef, 10, 64); err == nil && id > 0 {
			return id, "", true
		}
		return 0, "", false
	}
	id, err := strconv.ParseInt(head, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	return id, projectRef, true
}

func (s *Service) resolveProjectByRef(ctx context.Context, projectRef string, orgID int64) (*models.Project, error) {
	project := new(models.Project)
	if projectID, exactSlug, ok := parseProjectRef(projectRef); ok {
		// Prefer an exact slug match first so legacy slug-only URLs like
		// "123-service" don't get misrouted to an unrelated project id.
		if exactSlug != "" {
			err := s.db.NewSelect().Model(project).
				Where("organization_id = ?", orgID).
				Where("slug = ?", exactSlug).
				Scan(ctx)
			if err == nil {
				return project, nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}

		err := s.db.NewSelect().Model(project).
			Where("organization_id = ?", orgID).
			Where("id = ?", projectID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrProjectNotFound
			}
			return nil, err
		}
	} else {
		err := s.db.NewSelect().Model(project).
			Where("organization_id = ?", orgID).
			Where("slug = ?", projectRef).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrProjectNotFound
			}
			return nil, err
		}
	}
	return project, nil
}

func (s *Service) GetBySlug(ctx context.Context, projectRef string, orgID int64) (*models.Project, []models.ProjectMember, []models.ProjectLabel, error) {
	project, err := s.resolveProjectByRef(ctx, projectRef, orgID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Load Lead user
	if project.LeadID != nil {
		lead := new(models.User)
		if err := s.db.NewSelect().Model(lead).Where("id = ?", *project.LeadID).Scan(ctx); err == nil {
			project.Lead = lead
		}
	}

	// Load Labels via junction table
	var labels []models.ProjectLabel
	var assignments []models.ProjectLabelAssignment
	err = s.db.NewSelect().Model(&assignments).
		Relation("Label").
		Where("project_id = ?", project.ID).
		Scan(ctx)
	if err == nil {
		for _, a := range assignments {
			if a.Label != nil {
				labels = append(labels, *a.Label)
			}
		}
	}

	var members []models.ProjectMember
	err = s.db.NewSelect().Model(&members).
		Relation("User").
		Where("pm.project_id = ?", project.ID).
		Scan(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return project, members, labels, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64, orgID int64) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.NewSelect().Model(&projects).
		Join("JOIN project_members AS pm ON pm.project_id = p.id").
		Where("pm.user_id = ?", userID).
		Where("p.organization_id = ?", orgID).
		OrderExpr("p.updated_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	for i := range projects {
		if projects[i].LeadID == nil {
			continue
		}
		lead := new(models.User)
		if err := s.db.NewSelect().Model(lead).Where("id = ?", *projects[i].LeadID).Scan(ctx); err == nil {
			projects[i].Lead = lead
		}
	}

	return projects, nil
}

type UpdateProjectInput struct {
	ID          int64
	Name        string
	Description string
	Identifier  *string
	Status      *string
	Priority    *string
	Health      *string
	LeadID      *int64
	ClearLead   bool
	StartDate   *time.Time
	ClearStart  bool
	TargetDate  *time.Time
	ClearTarget bool
	Icon        *string
	Color       *string
	Readme      *string
}

func (s *Service) Update(ctx context.Context, input UpdateProjectInput, orgID int64) (*models.Project, error) {
	project := new(models.Project)
	err := s.db.NewSelect().Model(project).Where("id = ?", input.ID).Where("organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if input.Name != "" {
		if input.Name != project.Name {
			slug := GenerateSlug(input.Name)
			uniqueSlug, err := s.ensureUniqueSlugExcluding(ctx, slug, orgID, project.ID)
			if err != nil {
				return nil, err
			}
			project.Slug = uniqueSlug
		}
		project.Name = input.Name
	}
	project.Description = input.Description
	if input.Identifier != nil && *input.Identifier != project.Identifier {
		trimmed := strings.TrimSpace(*input.Identifier)
		if trimmed == "" {
			return nil, fmt.Errorf("identifier must not be empty")
		}
		uniqueID, err := s.ensureUniqueIdentifier(ctx, trimmed, orgID, project.ID)
		if err != nil {
			return nil, err
		}
		project.Identifier = uniqueID
	}
	if input.Status != nil {
		project.Status = models.ProjectStatus(*input.Status)
	}
	if input.Priority != nil {
		project.Priority = models.ProjectPriority(*input.Priority)
	}
	if input.Health != nil {
		project.Health = models.ProjectHealth(*input.Health)
	}
	if input.ClearLead {
		project.LeadID = nil
	} else if input.LeadID != nil {
		project.LeadID = input.LeadID
	}
	if input.ClearStart {
		project.StartDate = nil
	} else if input.StartDate != nil {
		project.StartDate = input.StartDate
	}
	if input.ClearTarget {
		project.TargetDate = nil
	} else if input.TargetDate != nil {
		project.TargetDate = input.TargetDate
	}
	if input.Icon != nil {
		project.Icon = *input.Icon
	}
	if input.Color != nil {
		project.Color = *input.Color
	}
	if input.Readme != nil {
		project.Readme = *input.Readme
	}
	project.UpdatedAt = time.Now()

	if _, err := s.db.NewUpdate().Model(project).WherePK().Exec(ctx); err != nil {
		return nil, err
	}

	// Reload with Lead relation
	if project.LeadID != nil {
		lead := new(models.User)
		if err := s.db.NewSelect().Model(lead).Where("id = ?", *project.LeadID).Scan(ctx); err == nil {
			project.Lead = lead
		}
	}

	return project, nil
}

// Label management

func (s *Service) CreateLabel(ctx context.Context, orgID int64, name, color string) (*models.ProjectLabel, error) {
	label := &models.ProjectLabel{
		OrganizationID: orgID,
		Name:           name,
		Color:          color,
	}
	if _, err := s.db.NewInsert().Model(label).Exec(ctx); err != nil {
		return nil, err
	}
	return label, nil
}

func (s *Service) ListLabels(ctx context.Context, orgID int64) ([]models.ProjectLabel, error) {
	var labels []models.ProjectLabel
	err := s.db.NewSelect().Model(&labels).
		Where("organization_id = ?", orgID).
		OrderExpr("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return labels, nil
}

func (s *Service) DeleteLabel(ctx context.Context, id int64, orgID int64) error {
	res, err := s.db.NewDelete().Model((*models.ProjectLabel)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *Service) AssignLabel(ctx context.Context, projectID, labelID int64, orgID int64) error {
	// Verify project belongs to org
	exists, err := s.db.NewSelect().Model((*models.Project)(nil)).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}
	// Verify label belongs to org
	exists, err = s.db.NewSelect().Model((*models.ProjectLabel)(nil)).
		Where("id = ?", labelID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}

	assignment := &models.ProjectLabelAssignment{
		ProjectID: projectID,
		LabelID:   labelID,
	}
	_, err = s.db.NewInsert().Model(assignment).
		On("CONFLICT (project_id, label_id) DO NOTHING").
		Exec(ctx)
	return err
}

func (s *Service) RemoveLabel(ctx context.Context, projectID, labelID int64, orgID int64) error {
	exists, err := s.db.NewSelect().Model((*models.Project)(nil)).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}

	_, err = s.db.NewDelete().Model((*models.ProjectLabelAssignment)(nil)).
		Where("project_id = ?", projectID).
		Where("label_id = ?", labelID).
		Exec(ctx)
	return err
}

func (s *Service) Delete(ctx context.Context, id int64, orgID int64) error {
	res, err := s.db.NewDelete().Model((*models.Project)(nil)).Where("id = ?", id).Where("organization_id = ?", orgID).Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *Service) GetProjectIdentifier(ctx context.Context, projectID int64) (string, error) {
	var identifier string
	err := s.db.NewSelect().Model((*models.Project)(nil)).
		Column("identifier").
		Where("id = ?", projectID).
		Scan(ctx, &identifier)
	if err != nil {
		return "", err
	}
	return identifier, nil
}

func (s *Service) CreateChange(ctx context.Context, projectID int64, name string, orgID int64) (*models.Change, error) {
	exists, err := s.db.NewSelect().Model((*models.Project)(nil)).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	num, err := s.nextChangeNumber(ctx, tx, projectID)
	if err != nil {
		return nil, err
	}

	change := &models.Change{
		ProjectID: projectID,
		Number:    num,
		Name:      name,
		Stage:     models.StageDraft,
	}

	if _, err := tx.NewInsert().Model(change).Exec(ctx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *Service) ListChanges(ctx context.Context, projectID int64, filter string, orgID int64, labelIDs []int64) ([]models.Change, error) {
	exists, err := s.db.NewSelect().Model((*models.Project)(nil)).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	var changes []models.Change
	q := s.db.NewSelect().Model(&changes).
		Where("ch.project_id = ?", projectID).
		OrderExpr("ch.created_at DESC")

	switch filter {
	case "archived":
		q = q.Where("ch.archived_at IS NOT NULL")
	case "all":
		// no filter
	default: // "active" or empty
		q = q.Where("ch.archived_at IS NULL")
	}

	if len(labelIDs) > 0 {
		q = q.Join("JOIN change_label_assignments AS cla ON cla.change_id = ch.id").
			Where("cla.label_id IN (?)", bun.List(labelIDs)).
			GroupExpr("ch.id")
	}

	if err := q.Scan(ctx); err != nil {
		return nil, err
	}

	if err := s.loadChangeLabels(ctx, changes); err != nil {
		return nil, err
	}

	return changes, nil
}

func (s *Service) GetChange(ctx context.Context, id int64, orgID int64) (*models.Change, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", id).
		Where("p.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	changes := []models.Change{*change}
	if err := s.loadChangeLabels(ctx, changes); err != nil {
		return nil, err
	}
	change.Labels = changes[0].Labels

	return change, nil
}

// loadChangeLabels batch-loads labels for a slice of changes (N+1 prevention).
func (s *Service) loadChangeLabels(ctx context.Context, changes []models.Change) error {
	if len(changes) == 0 {
		return nil
	}

	changeIDs := make([]int64, len(changes))
	for i, c := range changes {
		changeIDs[i] = c.ID
	}

	var assignments []struct {
		ChangeID int64 `bun:"change_id"`
		LabelID  int64 `bun:"label_id"`
		Name     string
		Color    string
	}
	err := s.db.NewSelect().
		TableExpr("change_label_assignments AS cla").
		Join("JOIN project_labels AS pl ON pl.id = cla.label_id").
		ColumnExpr("cla.change_id, cla.label_id, pl.name, pl.color").
		Where("cla.change_id IN (?)", bun.List(changeIDs)).
		OrderExpr("pl.name ASC").
		Scan(ctx, &assignments)
	if err != nil {
		return err
	}

	labelMap := make(map[int64][]models.ProjectLabel)
	for _, a := range assignments {
		labelMap[a.ChangeID] = append(labelMap[a.ChangeID], models.ProjectLabel{
			ID:    a.LabelID,
			Name:  a.Name,
			Color: a.Color,
		})
	}

	for i := range changes {
		changes[i].Labels = labelMap[changes[i].ID]
	}
	return nil
}

func (s *Service) AssignChangeLabel(ctx context.Context, changeID, labelID, orgID int64) error {
	// Verify change belongs to an org project
	exists, err := s.db.NewSelect().
		TableExpr("changes AS ch").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}

	// Verify label belongs to org
	exists, err = s.db.NewSelect().Model((*models.ProjectLabel)(nil)).
		Where("id = ?", labelID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}

	assignment := &models.ChangeLabelAssignment{
		ChangeID: changeID,
		LabelID:  labelID,
	}
	_, err = s.db.NewInsert().Model(assignment).
		On("CONFLICT (change_id, label_id) DO NOTHING").
		Exec(ctx)
	return err
}

func (s *Service) RemoveChangeLabel(ctx context.Context, changeID, labelID, orgID int64) error {
	exists, err := s.db.NewSelect().
		TableExpr("changes AS ch").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		Where("ch.id = ?", changeID).
		Where("p.organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}

	_, err = s.db.NewDelete().Model((*models.ChangeLabelAssignment)(nil)).
		Where("change_id = ?", changeID).
		Where("label_id = ?", labelID).
		Exec(ctx)
	return err
}

// GetChangeOG returns minimal change info for Open Graph metadata without auth.
func (s *Service) GetChangeOG(ctx context.Context, projectRef string, changeID int64) (changeName, projectName, stage string, err error) {
	var result struct {
		ChangeName  string `bun:"change_name"`
		ProjectName string `bun:"project_name"`
		Stage       string `bun:"stage"`
	}
	q := s.db.NewSelect().
		TableExpr("changes AS ch").
		Join("JOIN projects AS p ON p.id = ch.project_id").
		ColumnExpr("ch.name AS change_name").
		ColumnExpr("p.name AS project_name").
		ColumnExpr("ch.stage AS stage").
		Where("ch.id = ?", changeID)
	if projectID, exactSlug, ok := parseProjectRef(projectRef); ok {
		if exactSlug != "" {
			q = q.Where("(p.slug = ? OR p.id = ?)", exactSlug, projectID)
		} else {
			q = q.Where("p.id = ?", projectID)
		}
	} else {
		q = q.Where("p.slug = ?", projectRef)
	}
	err = q.Scan(ctx, &result)
	if err != nil {
		return "", "", "", err
	}
	return result.ChangeName, result.ProjectName, result.Stage, nil
}

// GetProjectOG returns minimal project info for Open Graph metadata without auth.
func (s *Service) GetProjectOG(ctx context.Context, projectRef string) (projectName, description string, err error) {
	var result struct {
		Name        string `bun:"name"`
		Description string `bun:"description"`
	}
	q := s.db.NewSelect().
		TableExpr("projects AS p").
		ColumnExpr("p.name").
		ColumnExpr("p.description")
	if projectID, exactSlug, ok := parseProjectRef(projectRef); ok {
		if exactSlug != "" {
			q = q.Where("(p.slug = ? OR p.id = ?)", exactSlug, projectID)
		} else {
			q = q.Where("p.id = ?", projectID)
		}
	} else {
		q = q.Where("p.slug = ?", projectRef)
	}
	err = q.Scan(ctx, &result)
	if err != nil {
		return "", "", err
	}
	return result.Name, result.Description, nil
}

func (s *Service) UpdateChange(ctx context.Context, id int64, name string, orgID int64) (*models.Change, error) {
	change := new(models.Change)
	err := s.db.NewSelect().Model(change).
		Where("ch.id = ?", id).
		Where("ch.project_id IN (SELECT id FROM projects WHERE organization_id = ?)", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	change.Name = name
	change.UpdatedAt = time.Now()
	res, err := s.db.NewUpdate().Model(change).
		Column("name", "updated_at").
		Where("id = ?", id).
		Where("project_id IN (SELECT id FROM projects WHERE organization_id = ?)", orgID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrProjectNotFound
	}
	return change, nil
}

func (s *Service) DeleteChange(ctx context.Context, id int64, orgID int64) error {
	res, err := s.db.NewDelete().Model((*models.Change)(nil)).
		Where("id = ?", id).
		Where("project_id IN (SELECT id FROM projects WHERE organization_id = ?)", orgID).
		Exec(ctx)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

type InviteMemberInput struct {
	ProjectID int64
	Email     string
	Role      models.Role
	OrgID     int64
}

func (s *Service) InviteMember(ctx context.Context, input InviteMemberInput) (*models.ProjectMember, error) {
	// Verify project belongs to org
	exists, err := s.db.NewSelect().Model((*models.Project)(nil)).
		Where("id = ?", input.ProjectID).
		Where("organization_id = ?", input.OrgID).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	// Find user by email
	user := new(models.User)
	err = s.db.NewSelect().Model(user).Where("email = ?", input.Email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User not registered — create pending invitation
			tokenBytes := make([]byte, 32)
			if _, err := rand.Read(tokenBytes); err != nil {
				return nil, err
			}
			invitation := &models.PendingInvitation{
				ProjectID: input.ProjectID,
				Email:     input.Email,
				Role:      input.Role,
				Token:     hex.EncodeToString(tokenBytes),
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			if _, err := s.db.NewInsert().Model(invitation).
				On("CONFLICT (project_id, email) DO UPDATE").
				Set("role = EXCLUDED.role").
				Set("token = EXCLUDED.token").
				Set("expires_at = EXCLUDED.expires_at").
				Exec(ctx); err != nil {
				return nil, err
			}
			// TODO: send invitation email with token
			return nil, nil
		}
		return nil, err
	}

	member := &models.ProjectMember{
		ProjectID: input.ProjectID,
		UserID:    user.ID,
		Role:      input.Role,
	}

	if _, err := s.db.NewInsert().Model(member).
		On("CONFLICT (project_id, user_id) DO UPDATE").
		Set("role = EXCLUDED.role").
		Exec(ctx); err != nil {
		return nil, err
	}

	// Also add to organization if not already a member
	project := new(models.Project)
	if err := s.db.NewSelect().Model(project).Where("id = ?", input.ProjectID).Scan(ctx); err == nil && project.OrganizationID > 0 {
		orgMember := &models.OrganizationMember{
			OrganizationID: project.OrganizationID,
			UserID:         user.ID,
			Role:           "member",
		}
		_, _ = s.db.NewInsert().Model(orgMember).
			On("CONFLICT (organization_id, user_id) DO NOTHING").
			Exec(ctx)
	}

	return member, nil
}

// ProcessPendingInvitations converts pending invitations to project memberships
// after a new user registers. Called during user registration flow.
func (s *Service) ProcessPendingInvitations(ctx context.Context, userID int64, email string) error {
	var invitations []models.PendingInvitation
	err := s.db.NewSelect().Model(&invitations).
		Where("email = ?", email).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return err
	}

	for _, inv := range invitations {
		member := &models.ProjectMember{
			ProjectID: inv.ProjectID,
			UserID:    userID,
			Role:      inv.Role,
		}
		if _, err := s.db.NewInsert().Model(member).
			On("CONFLICT (project_id, user_id) DO NOTHING").
			Exec(ctx); err != nil {
			return err
		}
	}

	// Clean up processed invitations
	if len(invitations) > 0 {
		_, err = s.db.NewDelete().Model((*models.PendingInvitation)(nil)).
			Where("email = ?", email).
			Exec(ctx)
	}

	return err
}

type SearchResult struct {
	Type      string
	ID        int64
	Title     string
	Subtitle  string
	Slug      string
	ProjectID int64
}

func (s *Service) Search(ctx context.Context, query string, userID, orgID int64) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	like := "%" + query + "%"
	var results []SearchResult

	// Search projects
	var projects []models.Project
	if err := s.db.NewSelect().Model(&projects).
		Join("JOIN project_members AS pm ON pm.project_id = p.id").
		Where("pm.user_id = ?", userID).
		Where("p.organization_id = ?", orgID).
		Where("p.name ILIKE ?", like).
		Limit(5).
		Scan(ctx); err == nil {
		for _, p := range projects {
			results = append(results, SearchResult{
				Type:      "project",
				ID:        p.ID,
				Title:     p.Name,
				Subtitle:  string(p.Status),
				Slug:      p.Slug,
				ProjectID: p.ID,
			})
		}
	}

	// Search changes (by name or identifier like COL-3)
	var changes []models.Change
	changeQuery := s.db.NewSelect().Model(&changes).
		Relation("Project").
		Join("JOIN project_members AS pm ON pm.project_id = c.project_id").
		Where("pm.user_id = ?", userID).
		Limit(10)

	// Check if query matches identifier pattern (e.g. COL-3)
	if parts := strings.SplitN(strings.ToUpper(query), "-", 2); len(parts) == 2 {
		if num, err := strconv.Atoi(parts[1]); err == nil && num > 0 {
			changeQuery = changeQuery.Where(
				"(c.name ILIKE ? OR (EXISTS (SELECT 1 FROM projects WHERE id = c.project_id AND UPPER(identifier) = ?) AND c.number = ?))",
				like, parts[0], num,
			)
		} else {
			changeQuery = changeQuery.Where("c.name ILIKE ?", like)
		}
	} else {
		changeQuery = changeQuery.Where("c.name ILIKE ?", like)
	}

	if err := changeQuery.Scan(ctx); err == nil {
		for _, c := range changes {
			subtitle := string(c.Stage)
			slug := ""
			projectIdentifier := ""
			if c.Project != nil {
				slug = c.Project.Slug
				projectIdentifier = c.Project.Identifier
			}
			title := c.Name
			if projectIdentifier != "" && c.Number > 0 {
				title = fmt.Sprintf("%s-%d %s", projectIdentifier, c.Number, c.Name)
			}
			results = append(results, SearchResult{
				Type:      "change",
				ID:        c.ID,
				Title:     title,
				Subtitle:  subtitle,
				Slug:      slug,
				ProjectID: c.ProjectID,
			})
		}
	}

	// Search tasks
	var tasks []models.Task
	if err := s.db.NewSelect().Model(&tasks).
		Relation("Change").
		Relation("Change.Project").
		Join("JOIN changes AS ch ON ch.id = \"task\".change_id").
		Join("JOIN project_members AS pm ON pm.project_id = ch.project_id").
		Where("pm.user_id = ?", userID).
		Where("\"task\".title ILIKE ?", like).
		Limit(5).
		Scan(ctx); err == nil {
		for _, t := range tasks {
			slug := ""
			var projectID int64
			if t.Change != nil {
				projectID = t.Change.ProjectID
				if t.Change.Project != nil {
					slug = t.Change.Project.Slug
				}
			}
			results = append(results, SearchResult{
				Type:      "task",
				ID:        t.ID,
				Title:     t.Title,
				Subtitle:  string(t.Status),
				Slug:      slug,
				ProjectID: projectID,
			})
		}
	}

	return results, nil
}
