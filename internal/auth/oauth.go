package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/uptrace/bun"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/gobenpark/colign/internal/models"
)

type OAuthConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
	RedirectBaseURL    string
}

type OAuthService struct {
	db         *bun.DB
	jwtManager *JWTManager
	github     *oauth2.Config
	google     *oauth2.Config
}

func NewOAuthService(db *bun.DB, jwtManager *JWTManager, cfg OAuthConfig) *OAuthService {
	return &OAuthService{
		db:         db,
		jwtManager: jwtManager,
		github: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  cfg.RedirectBaseURL + "/api/auth/github/callback",
			Scopes:       []string{"user:email"},
		},
		google: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  cfg.RedirectBaseURL + "/api/auth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
	}
}

func (s *OAuthService) GetAuthURL(provider, state string) (string, error) {
	cfg, err := s.getConfig(provider)
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state), nil
}

func (s *OAuthService) HandleCallback(ctx context.Context, provider, code string) (*TokenPair, error) {
	cfg, err := s.getConfig(provider)
	if err != nil {
		return nil, err
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange failed: %w", err)
	}

	userInfo, err := s.fetchUserInfo(ctx, provider, token)
	if err != nil {
		return nil, err
	}

	user, orgID, err := s.findOrCreateUser(ctx, provider, userInfo, token)
	if err != nil {
		return nil, err
	}

	return s.createSession(ctx, user, orgID)
}

type oauthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

func (s *OAuthService) fetchUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	switch provider {
	case "github":
		return s.fetchGitHubUser(client)
	case "google":
		return s.fetchGoogleUser(client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *OAuthService) fetchGitHubUser(client *http.Client) (*oauthUserInfo, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	name := data.Name
	if name == "" {
		name = data.Login
	}

	// If email is private, fetch from emails endpoint
	email := data.Email
	if email == "" {
		resp2, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer func() { _ = resp2.Body.Close() }()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if json.NewDecoder(resp2.Body).Decode(&emails) == nil {
				for _, e := range emails {
					if e.Primary {
						email = e.Email
						break
					}
				}
			}
		}
	}

	return &oauthUserInfo{
		ProviderID: fmt.Sprintf("%d", data.ID),
		Email:      email,
		Name:       name,
		AvatarURL:  data.AvatarURL,
	}, nil
}

func (s *OAuthService) fetchGoogleUser(client *http.Client) (*oauthUserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var data struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &oauthUserInfo{
		ProviderID: data.ID,
		Email:      data.Email,
		Name:       data.Name,
		AvatarURL:  data.Picture,
	}, nil
}

func (s *OAuthService) findOrCreateUser(ctx context.Context, provider string, info *oauthUserInfo, token *oauth2.Token) (*models.User, int64, error) {
	// Check if account already exists
	account := new(models.Account)
	err := s.db.NewSelect().Model(account).
		Relation("User").
		Where("a.provider = ?", provider).
		Where("a.provider_account_id = ?", info.ProviderID).
		Scan(ctx)

	if err == nil {
		// Update tokens
		account.AccessToken = token.AccessToken
		account.RefreshToken = token.RefreshToken
		if !token.Expiry.IsZero() {
			account.ExpiresAt = token.Expiry.Unix()
		}
		_, _ = s.db.NewUpdate().Model(account).WherePK().Exec(ctx)
		orgID := s.getDefaultOrgID(ctx, account.User.ID)
		return account.User, orgID, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, 0, err
	}

	// Check if user with same email exists
	user := new(models.User)
	isNewUser := false
	err = s.db.NewSelect().Model(user).Where("email = ?", info.Email).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		isNewUser = true
		// Create new user
		user = &models.User{
			Email:         info.Email,
			Name:          info.Name,
			AvatarURL:     info.AvatarURL,
			EmailVerified: true,
		}
		if _, err := s.db.NewInsert().Model(user).Exec(ctx); err != nil {
			return nil, 0, err
		}
	} else if err != nil {
		return nil, 0, err
	}

	// Link account
	newAccount := &models.Account{
		UserID:            user.ID,
		Provider:          provider,
		ProviderAccountID: info.ProviderID,
		AccessToken:       token.AccessToken,
		RefreshToken:      token.RefreshToken,
	}
	if !token.Expiry.IsZero() {
		newAccount.ExpiresAt = token.Expiry.Unix()
	}
	if _, err := s.db.NewInsert().Model(newAccount).Exec(ctx); err != nil {
		return nil, 0, err
	}

	// Create org for new OAuth users
	var orgID int64
	if isNewUser {
		org := &models.Organization{
			Name: fmt.Sprintf("%s's Workspace", info.Name),
			Slug: fmt.Sprintf("%s-oauth", info.ProviderID),
		}
		if _, err := s.db.NewInsert().Model(org).Exec(ctx); err != nil {
			return nil, 0, err
		}
		om := &models.OrganizationMember{
			OrganizationID: org.ID,
			UserID:         user.ID,
			Role:           models.OrgRoleOwner,
		}
		if _, err := s.db.NewInsert().Model(om).Exec(ctx); err != nil {
			return nil, 0, err
		}
		orgID = org.ID
	} else {
		orgID = s.getDefaultOrgID(ctx, user.ID)
	}

	return user, orgID, nil
}

func (s *OAuthService) getDefaultOrgID(ctx context.Context, userID int64) int64 {
	om := new(models.OrganizationMember)
	err := s.db.NewSelect().Model(om).
		Where("user_id = ?", userID).
		OrderExpr("created_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return 0
	}
	return om.OrganizationID
}

func (s *OAuthService) createSession(ctx context.Context, user *models.User, orgID int64) (*TokenPair, error) {
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

func (s *OAuthService) getConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "github":
		return s.github, nil
	case "google":
		return s.google, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
