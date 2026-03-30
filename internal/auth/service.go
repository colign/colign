package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"strings"

	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"

	"github.com/gobenpark/colign/internal/models"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already in use")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrUserNotFound        = errors.New("user not found")
)

// OrgJoiner handles auto-joining organizations on registration.
type OrgJoiner interface {
	// AutoJoinOrgs joins the user to orgs matching their email domain or pending invitations.
	// Returns the first org ID joined, or 0 if none. Use only for new user registration.
	AutoJoinOrgs(ctx context.Context, userID int64, email string) (int64, error)
	// AcceptPendingInvitations accepts pending invitations without domain-based auto-join.
	// Use on every login to accept invitations without re-adding removed members.
	AcceptPendingInvitations(ctx context.Context, userID int64, email string) (int64, error)
}

type Service struct {
	db                *bun.DB
	jwtManager        *JWTManager
	orgJoiner         OrgJoiner
	apiTokenValidator APITokenValidator
}

func NewService(db *bun.DB, jwtManager *JWTManager) *Service {
	return &Service{db: db, jwtManager: jwtManager}
}

// SetOrgJoiner sets the OrgJoiner after construction to avoid circular dependency.
func (s *Service) SetOrgJoiner(oj OrgJoiner) {
	s.orgJoiner = oj
}

// SetAPITokenValidator sets the API token validator to support MCP OAuth tokens.
func (s *Service) SetAPITokenValidator(v APITokenValidator) {
	s.apiTokenValidator = v
}

type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=8"`
	Name             string `json:"name" binding:"required"`
	OrganizationName string `json:"organization_name"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*TokenPair, error) {
	exists, err := s.db.NewSelect().Model((*models.User)(nil)).Where("email = ?", req.Email).Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
	}

	if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
		return nil, err
	}

	// Generate verification token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	verification := &models.EmailVerification{
		UserID:    user.ID,
		Token:     hex.EncodeToString(tokenBytes),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if _, err := tx.NewInsert().Model(verification).Exec(ctx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// TODO: send verification email

	// Try auto-joining existing orgs via domain or invitation
	var orgID int64
	if s.orgJoiner != nil {
		orgID, _ = s.orgJoiner.AutoJoinOrgs(ctx, user.ID, req.Email)
	}

	// If no org was joined, create a personal workspace
	if orgID == 0 {
		orgName := req.OrganizationName
		if orgName == "" {
			orgName = fmt.Sprintf("%s's Workspace", req.Name)
		}
		org := &models.Organization{
			Name: orgName,
			Slug: generateOrgSlug(orgName),
		}
		if _, err := s.db.NewInsert().Model(org).Exec(ctx); err != nil {
			return nil, err
		}
		orgMember := &models.OrganizationMember{
			OrganizationID: org.ID,
			UserID:         user.ID,
			Role:           models.OrgRoleOwner,
		}
		if _, err := s.db.NewInsert().Model(orgMember).Exec(ctx); err != nil {
			return nil, err
		}
		orgID = org.ID
	}

	return s.createSession(ctx, user, orgID)
}

func generateOrgSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(b))
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenPair, error) {
	user := new(models.User)
	err := s.db.NewSelect().Model(user).Where("email = ?", req.Email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Accept pending invitations on every login (without domain-based auto-join)
	if s.orgJoiner != nil {
		s.orgJoiner.AcceptPendingInvitations(ctx, user.ID, req.Email)
	}

	// Get user's first organization, or create one if none exists
	orgID, err := s.getOrCreateDefaultOrg(ctx, user)
	if err != nil {
		return nil, err
	}

	return s.createSession(ctx, user, orgID)
}

func (s *Service) getDefaultOrgID(ctx context.Context, userID int64) (int64, error) {
	om := new(models.OrganizationMember)
	err := s.db.NewSelect().Model(om).
		Where("user_id = ?", userID).
		OrderExpr("created_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return om.OrganizationID, nil
}

// getOrCreateDefaultOrg returns the user's default org, creating one if none exists.
// This handles users created before the organization feature was added.
func (s *Service) getOrCreateDefaultOrg(ctx context.Context, user *models.User) (int64, error) {
	orgID, err := s.getDefaultOrgID(ctx, user.ID)
	if err != nil {
		return 0, err
	}
	if orgID != 0 {
		return orgID, nil
	}

	// No org exists — create one for the user
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	orgName := fmt.Sprintf("%s's Workspace", user.Name)
	org := &models.Organization{
		Name: orgName,
		Slug: generateOrgSlug(orgName),
	}
	if _, err := tx.NewInsert().Model(org).Exec(ctx); err != nil {
		return 0, err
	}

	orgMember := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrgRoleOwner,
	}
	if _, err := tx.NewInsert().Model(orgMember).Exec(ctx); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return org.ID, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	session := new(models.Session)
	err := s.db.NewSelect().Model(session).
		Where("s.refresh_token = ?", refreshToken).
		Where("s.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	user := new(models.User)
	if err := s.db.NewSelect().Model(user).Where("id = ?", session.UserID).Scan(ctx); err != nil {
		return nil, err
	}

	// Delete old session
	if _, err := s.db.NewDelete().Model(session).WherePK().Exec(ctx); err != nil {
		return nil, err
	}

	// Use the org stored in session; fall back to default if unset
	orgID := session.OrgID
	if orgID == 0 {
		orgID, err = s.getOrCreateDefaultOrg(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return s.createSession(ctx, user, orgID)
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	verification := new(models.EmailVerification)
	err := s.db.NewSelect().Model(verification).
		Where("token = ?", token).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token")
	}

	_, err = s.db.NewUpdate().Model((*models.User)(nil)).
		Set("email_verified = ?", true).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", verification.UserID).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = s.db.NewDelete().Model(verification).WherePK().Exec(ctx)
	return err
}

func (s *Service) createSession(ctx context.Context, user *models.User, orgID int64) (*TokenPair, error) {
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Name, orgID)
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		UserID:       user.ID,
		OrgID:        orgID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(RefreshTokenDuration),
	}

	if _, err := s.db.NewInsert().Model(session).Exec(ctx); err != nil {
		return nil, err
	}

	return tokenPair, nil
}

// SwitchOrg updates the current session's org and returns a new token pair.
func (s *Service) SwitchOrg(ctx context.Context, userID int64, email, name string, newOrgID int64) (*TokenPair, error) {
	// Delete all existing sessions for this user (they'll get a fresh one)
	if _, err := s.db.NewDelete().Model((*models.Session)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx); err != nil {
		return nil, err
	}

	user := &models.User{ID: userID, Email: email, Name: name}
	return s.createSession(ctx, user, newOrgID)
}

func (s *Service) Me(ctx context.Context, authHeader string) (*models.User, int64, error) {
	claims, err := ResolveFromHeader(s.jwtManager, s.apiTokenValidator, ctx, authHeader)
	if err != nil {
		return nil, 0, err
	}

	user := new(models.User)
	err = s.db.NewSelect().Model(user).Where("id = ?", claims.UserID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrUserNotFound
		}
		return nil, 0, err
	}

	return user, claims.OrgID, nil
}

func (s *Service) UpdateProfile(ctx context.Context, authHeader, name, avatarURL string) (*models.User, int64, error) {
	claims, err := ResolveFromHeader(s.jwtManager, s.apiTokenValidator, ctx, authHeader)
	if err != nil {
		return nil, 0, err
	}

	user := new(models.User)
	if err := s.db.NewSelect().Model(user).Where("id = ?", claims.UserID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrUserNotFound
		}
		return nil, 0, err
	}

	user.Name = name
	user.AvatarURL = avatarURL
	user.UpdatedAt = time.Now()

	if _, err := s.db.NewUpdate().Model(user).WherePK().Exec(ctx); err != nil {
		return nil, 0, err
	}

	return user, claims.OrgID, nil
}
