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
	ErrEmailAlreadyExists  = errors.New("이미 사용 중인 이메일입니다")
	ErrInvalidCredentials  = errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
	ErrInvalidRefreshToken = errors.New("유효하지 않거나 만료된 리프레시 토큰입니다")
)

type Service struct {
	db         *bun.DB
	jwtManager *JWTManager
}

func NewService(db *bun.DB, jwtManager *JWTManager) *Service {
	return &Service{db: db, jwtManager: jwtManager}
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

	// Create organization
	orgName := req.OrganizationName
	if orgName == "" {
		orgName = fmt.Sprintf("%s's Workspace", req.Name)
	}
	orgSlug := generateOrgSlug(orgName)
	org := &models.Organization{
		Name: orgName,
		Slug: orgSlug,
	}
	if _, err := tx.NewInsert().Model(org).Exec(ctx); err != nil {
		return nil, err
	}

	orgMember := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.OrgRoleOwner,
	}
	if _, err := tx.NewInsert().Model(orgMember).Exec(ctx); err != nil {
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

	return s.createSession(ctx, user, org.ID)
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

	// Get user's first organization
	orgID, err := s.getDefaultOrgID(ctx, user.ID)
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

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	session := new(models.Session)
	err := s.db.NewSelect().Model(session).
		Relation("User").
		Where("s.refresh_token = ?", refreshToken).
		Where("s.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Delete old session
	if _, err := s.db.NewDelete().Model(session).WherePK().Exec(ctx); err != nil {
		return nil, err
	}

	orgID, err := s.getDefaultOrgID(ctx, session.User.ID)
	if err != nil {
		return nil, err
	}

	return s.createSession(ctx, session.User, orgID)
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
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(RefreshTokenDuration),
	}

	if _, err := s.db.NewInsert().Model(session).Exec(ctx); err != nil {
		return nil, err
	}

	return tokenPair, nil
}
