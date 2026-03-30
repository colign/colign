package organization

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrOrgNotFound        = errors.New("organization not found")
	ErrInvitationNotFound = errors.New("invitation not found or expired")
	ErrAlreadyMember      = errors.New("user is already a member of this organization")
	ErrNotOwner           = errors.New("only organization owners can perform this action")
	ErrLastOrganization   = errors.New("cannot delete your only organization")
)

type Service struct {
	db *bun.DB
}

func NewService(db *bun.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, userID int64, name string) (*models.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("organization name is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	org := &models.Organization{
		Name: name,
		Slug: generateOrgSlug(name),
	}
	if _, err := tx.NewInsert().Model(org).Exec(ctx); err != nil {
		return nil, err
	}

	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.OrgRoleOwner,
	}
	if _, err := tx.NewInsert().Model(member).Exec(ctx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]models.Organization, error) {
	var orgs []models.Organization
	err := s.db.NewSelect().Model(&orgs).
		Join("JOIN organization_members AS om ON om.organization_id = o.id").
		Where("om.user_id = ?", userID).
		OrderExpr("o.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*models.Organization, error) {
	org := new(models.Organization)
	err := s.db.NewSelect().Model(org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return org, nil
}

func (s *Service) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	return s.db.NewSelect().Model((*models.OrganizationMember)(nil)).
		Where("organization_id = ?", orgID).
		Where("user_id = ?", userID).
		Exists(ctx)
}

func (s *Service) ListMembers(ctx context.Context, orgID int64) ([]models.OrganizationMember, error) {
	var members []models.OrganizationMember
	err := s.db.NewSelect().Model(&members).
		Relation("User").
		Where("om.organization_id = ?", orgID).
		OrderExpr("om.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// InviteMember creates an invitation token for the given email.
// If the user already exists and is already a member, returns an error.
// If the user exists but is not a member, they can accept the invitation to join.
func (s *Service) InviteMember(ctx context.Context, orgID, invitedBy int64, email string, role models.OrgRole) (*models.OrgInvitation, error) {
	// Check if already a member
	user := new(models.User)
	err := s.db.NewSelect().Model(user).Where("email = ?", email).Scan(ctx)
	if err == nil {
		isMember, _ := s.IsMember(ctx, orgID, user.ID)
		if isMember {
			return nil, ErrAlreadyMember
		}
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	invitation := &models.OrgInvitation{
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		Token:          hex.EncodeToString(tokenBytes),
		InvitedBy:      invitedBy,
		Status:         models.InvitationStatusPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
	}

	if _, err := s.db.NewInsert().Model(invitation).
		On("CONFLICT (organization_id, email) DO UPDATE").
		Set("token = EXCLUDED.token").
		Set("role = EXCLUDED.role").
		Set("invited_by = EXCLUDED.invited_by").
		Set("status = EXCLUDED.status").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx); err != nil {
		return nil, err
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation by token and adds the user to the organization.
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID int64) (*models.OrganizationMember, error) {
	invitation := new(models.OrgInvitation)
	err := s.db.NewSelect().Model(invitation).
		Where("oi.token = ?", token).
		Where("oi.status = ?", models.InvitationStatusPending).
		Where("oi.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	member := &models.OrganizationMember{
		OrganizationID: invitation.OrganizationID,
		UserID:         userID,
		Role:           invitation.Role,
	}
	if _, err := tx.NewInsert().Model(member).
		On("CONFLICT (organization_id, user_id) DO UPDATE").
		Set("role = EXCLUDED.role").
		Exec(ctx); err != nil {
		return nil, err
	}

	invitation.Status = models.InvitationStatusAccepted
	if _, err := tx.NewUpdate().Model(invitation).
		Set("status = ?", models.InvitationStatusAccepted).
		WherePK().
		Exec(ctx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return member, nil
}

// GetInvitationByToken returns the invitation with organization details.
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*models.OrgInvitation, error) {
	invitation := new(models.OrgInvitation)
	err := s.db.NewSelect().Model(invitation).
		Relation("Organization").
		Where("oi.token = ?", token).
		Where("oi.status = ?", models.InvitationStatusPending).
		Where("oi.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return invitation, nil
}

// ListInvitations returns pending invitations for an organization.
func (s *Service) ListInvitations(ctx context.Context, orgID int64) ([]models.OrgInvitation, error) {
	var invitations []models.OrgInvitation
	err := s.db.NewSelect().Model(&invitations).
		Where("oi.organization_id = ?", orgID).
		Where("oi.status = ?", models.InvitationStatusPending).
		OrderExpr("oi.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

// RevokeInvitation deletes a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, orgID, invitationID int64) error {
	_, err := s.db.NewDelete().Model((*models.OrgInvitation)(nil)).
		Where("id = ?", invitationID).
		Where("organization_id = ?", orgID).
		Where("status = ?", models.InvitationStatusPending).
		Exec(ctx)
	return err
}

// SetAllowedDomains updates the allowed email domains for an organization.
func (s *Service) SetAllowedDomains(ctx context.Context, orgID int64, domains []string) (*models.Organization, error) {
	// Normalize domains (lowercase, trim)
	for i, d := range domains {
		domains[i] = strings.ToLower(strings.TrimSpace(d))
	}

	org := new(models.Organization)
	_, err := s.db.NewUpdate().Model(org).
		Set("allowed_domains = ?", domains).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", orgID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// FindOrgsByEmailDomain finds organizations that allow the given email domain.
func (s *Service) FindOrgsByEmailDomain(ctx context.Context, email string) ([]models.Organization, error) {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email: %s", email)
	}
	domain := strings.ToLower(parts[1])

	var orgs []models.Organization
	err := s.db.NewSelect().Model(&orgs).
		Where("? = ANY(allowed_domains)", domain).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

// FindPendingInvitationsByEmail finds all pending invitations for a given email.
func (s *Service) FindPendingInvitationsByEmail(ctx context.Context, email string) ([]models.OrgInvitation, error) {
	var invitations []models.OrgInvitation
	err := s.db.NewSelect().Model(&invitations).
		Where("oi.email = ?", email).
		Where("oi.status = ?", models.InvitationStatusPending).
		Where("oi.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

// AcceptPendingInvitations accepts pending invitations without domain-based auto-join.
// Use on every login to accept invitations without re-adding removed members.
func (s *Service) AcceptPendingInvitations(ctx context.Context, userID int64, email string) (int64, error) {
	var firstOrgID int64

	invitations, err := s.FindPendingInvitationsByEmail(ctx, email)
	if err != nil {
		return 0, nil
	}
	for _, inv := range invitations {
		member := &models.OrganizationMember{
			OrganizationID: inv.OrganizationID,
			UserID:         userID,
			Role:           inv.Role,
		}
		if _, err := s.db.NewInsert().Model(member).
			On("CONFLICT (organization_id, user_id) DO UPDATE").
			Set("role = EXCLUDED.role").
			Exec(ctx); err != nil {
			continue
		}
		_, _ = s.db.NewUpdate().Model(&inv).
			Set("status = ?", models.InvitationStatusAccepted).
			WherePK().
			Exec(ctx)

		if firstOrgID == 0 {
			firstOrgID = inv.OrganizationID
		}
	}

	return firstOrgID, nil
}

// AutoJoinOrgs handles domain-based auto-join and pending invitation acceptance for a newly registered user.
// Returns the org ID the user should default to (first joined org, or 0 if none).
func (s *Service) AutoJoinOrgs(ctx context.Context, userID int64, email string) (int64, error) {
	var firstOrgID int64

	// 1. Domain-based auto-join
	orgs, err := s.FindOrgsByEmailDomain(ctx, email)
	if err != nil {
		return 0, err
	}
	for _, org := range orgs {
		member := &models.OrganizationMember{
			OrganizationID: org.ID,
			UserID:         userID,
			Role:           models.OrgRoleMember,
		}
		if _, err := s.db.NewInsert().Model(member).
			On("CONFLICT (organization_id, user_id) DO NOTHING").
			Exec(ctx); err != nil {
			continue
		}
		if firstOrgID == 0 {
			firstOrgID = org.ID
		}
	}

	// 2. Accept pending invitations
	invitations, err := s.FindPendingInvitationsByEmail(ctx, email)
	if err != nil {
		return firstOrgID, nil
	}
	for _, inv := range invitations {
		member := &models.OrganizationMember{
			OrganizationID: inv.OrganizationID,
			UserID:         userID,
			Role:           inv.Role,
		}
		if _, err := s.db.NewInsert().Model(member).
			On("CONFLICT (organization_id, user_id) DO UPDATE").
			Set("role = EXCLUDED.role").
			Exec(ctx); err != nil {
			continue
		}
		// Mark invitation as accepted
		_, _ = s.db.NewUpdate().Model(&inv).
			Set("status = ?", models.InvitationStatusAccepted).
			WherePK().
			Exec(ctx)

		if firstOrgID == 0 {
			firstOrgID = inv.OrganizationID
		}
	}

	return firstOrgID, nil
}

func (s *Service) RemoveMember(ctx context.Context, orgID, userID int64) error {
	_, err := s.db.NewDelete().Model((*models.OrganizationMember)(nil)).
		Where("organization_id = ?", orgID).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID int64, role models.OrgRole) (*models.OrganizationMember, error) {
	member := new(models.OrganizationMember)
	_, err := s.db.NewUpdate().Model(member).
		Set("role = ?", role).
		Where("organization_id = ?", orgID).
		Where("user_id = ?", userID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return member, nil
}

func (s *Service) Update(ctx context.Context, id int64, name string) (*models.Organization, error) {
	org := new(models.Organization)
	err := s.db.NewSelect().Model(org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	org.Name = name
	org.UpdatedAt = time.Now()

	if _, err := s.db.NewUpdate().Model(org).WherePK().Exec(ctx); err != nil {
		return nil, err
	}
	return org, nil
}

// Delete permanently removes an organization and all its data (CASCADE).
// Only an owner of the target org can delete it, and the user must belong to at least one other org.
func (s *Service) Delete(ctx context.Context, orgID, userID int64) (int64, error) {
	// Verify the user is an owner of this organization
	member := new(models.OrganizationMember)
	err := s.db.NewSelect().Model(member).
		Where("organization_id = ?", orgID).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrOrgNotFound
		}
		return 0, err
	}
	if models.OrgRole(member.Role) != models.OrgRoleOwner {
		return 0, ErrNotOwner
	}

	// Count total orgs the user belongs to
	count, err := s.db.NewSelect().Model((*models.OrganizationMember)(nil)).
		Where("user_id = ?", userID).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	if count <= 1 {
		return 0, ErrLastOrganization
	}

	// Find the next org to switch to
	nextOrg := new(models.OrganizationMember)
	err = s.db.NewSelect().Model(nextOrg).
		Where("user_id = ?", userID).
		Where("organization_id != ?", orgID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return 0, err
	}

	// Delete the organization (CASCADE handles related data)
	_, err = s.db.NewDelete().Model((*models.Organization)(nil)).
		Where("id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return nextOrg.OrganizationID, nil
}

func generateOrgSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(b))
}
