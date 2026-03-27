package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/uptrace/bun"

	"github.com/gobenpark/colign/internal/apitoken"
)

type TokenHandler struct {
	db              *bun.DB
	apiTokenService *apitoken.Service
}

const (
	oauthAccessTokenTTL  = 30 * 24 * time.Hour
	oauthRefreshTokenTTL = 30 * 24 * time.Hour
)

func NewTokenHandler(db *bun.DB, apiTokenService *apitoken.Service) *TokenHandler {
	return &TokenHandler{db: db, apiTokenService: apiTokenService}
}

// ServeHTTP handles POST /oauth/token — exchanges authorization code for access token.
func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	grantType := r.FormValue("grant_type")
	slog.Info("oauth token request", "grant_type", grantType, "client_id", r.FormValue("client_id"))
	switch grantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r)
	default:
		writeTokenError(w, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
	}
}

func (h *TokenHandler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")
	if code == "" || codeVerifier == "" {
		writeTokenError(w, "invalid_request", "code and code_verifier are required")
		return
	}

	// Look up authorization code
	authCode := new(OAuthAuthorizationCode)
	err := h.db.NewSelect().Model(authCode).
		Where("oac.code = ?", code).
		Where("oac.used = ?", false).
		Where("oac.expires_at > ?", time.Now()).
		Scan(r.Context())
	if err != nil {
		writeTokenError(w, "invalid_grant", "invalid or expired authorization code")
		return
	}

	// Verify PKCE
	if !verifyPKCE(codeVerifier, authCode.CodeChallenge) {
		writeTokenError(w, "invalid_grant", "code_verifier does not match code_challenge")
		return
	}

	// Mark code as used
	authCode.Used = true
	if _, err := h.db.NewUpdate().Model(authCode).WherePK().Column("used").Exec(r.Context()); err != nil {
		writeTokenError(w, "server_error", "internal error")
		return
	}

	accessToken, refreshToken, err := h.issueTokens(r.Context(), authCode.UserID, authCode.OrgID, authCode.ClientID)
	if err != nil {
		slog.Error("oauth token issue failed", "grant_type", "authorization_code", "error", err)
		writeTokenError(w, "server_error", "failed to create oauth tokens")
		return
	}

	slog.Info("oauth token issued", "grant_type", "authorization_code", "user_id", authCode.UserID, "client_id", authCode.ClientID)
	h.writeTokenResponse(w, accessToken, refreshToken)
}

func (h *TokenHandler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	rawRefreshToken := r.FormValue("refresh_token")
	if rawRefreshToken == "" {
		writeTokenError(w, "invalid_request", "refresh_token is required")
		return
	}

	refreshToken := new(OAuthRefreshToken)
	err := h.db.NewSelect().Model(refreshToken).
		Where("ort.token_hash = ?", hashOAuthToken(rawRefreshToken)).
		Where("ort.used = ?", false).
		Where("ort.expires_at > ?", time.Now()).
		Scan(r.Context())
	if err != nil {
		slog.Warn("oauth refresh token lookup failed", "error", err)
		writeTokenError(w, "invalid_grant", "invalid or expired refresh token")
		return
	}

	clientID := r.FormValue("client_id")
	if clientID != "" && clientID != refreshToken.ClientID {
		slog.Warn("oauth refresh token client mismatch", "expected", refreshToken.ClientID, "got", clientID)
		writeTokenError(w, "invalid_grant", "refresh token does not match client")
		return
	}

	refreshToken.Used = true
	if _, err := h.db.NewUpdate().Model(refreshToken).WherePK().Column("used").Exec(r.Context()); err != nil {
		writeTokenError(w, "server_error", "internal error")
		return
	}

	accessToken, newRefreshToken, err := h.issueTokens(r.Context(), refreshToken.UserID, refreshToken.OrgID, refreshToken.ClientID)
	if err != nil {
		slog.Error("oauth token issue failed", "grant_type", "refresh_token", "error", err)
		writeTokenError(w, "server_error", "failed to refresh oauth tokens")
		return
	}

	slog.Info("oauth token refreshed", "user_id", refreshToken.UserID, "client_id", refreshToken.ClientID)
	h.writeTokenResponse(w, accessToken, newRefreshToken)
}

func (h *TokenHandler) issueTokens(ctx context.Context, userID, orgID int64, clientID string) (string, string, error) {
	// Create an OAuth access token scoped to this MCP client.
	_, rawAccessToken, err := h.apiTokenService.CreateOAuth(ctx, userID, orgID, clientID, "MCP OAuth")
	if err != nil {
		return "", "", err
	}

	// Replace existing refresh tokens for the same MCP client.
	if _, err := h.db.NewDelete().Model((*OAuthRefreshToken)(nil)).
		Where("user_id = ?", userID).
		Where("org_id = ?", orgID).
		Where("client_id = ?", clientID).
		Exec(ctx); err != nil {
		slog.Warn("oauth old refresh token cleanup failed", "user_id", userID, "error", err)
	}

	rawRefreshToken, err := generateOAuthRefreshToken()
	if err != nil {
		return "", "", err
	}

	refreshToken := &OAuthRefreshToken{
		UserID:    userID,
		OrgID:     orgID,
		ClientID:  clientID,
		TokenHash: hashOAuthToken(rawRefreshToken),
		ExpiresAt: time.Now().Add(oauthRefreshTokenTTL),
	}
	if _, err := h.db.NewInsert().Model(refreshToken).Exec(ctx); err != nil {
		return "", "", err
	}

	return rawAccessToken, rawRefreshToken, nil
}

func (h *TokenHandler) writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"token_type":    "bearer",
		"expires_in":    int(oauthAccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
	})
}

// verifyPKCE checks that SHA256(code_verifier) == code_challenge.
func verifyPKCE(verifier, challenge string) bool {
	h := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return computed == challenge
}

func generateOAuthRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "colrt_" + hex.EncodeToString(b), nil
}

func hashOAuthToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func writeTokenError(w http.ResponseWriter, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
