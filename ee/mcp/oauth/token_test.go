package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- verifyPKCE tests ---

func TestVerifyPKCE_ValidPair(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	assert.True(t, verifyPKCE(verifier, challenge))
}

func TestVerifyPKCE_InvalidPair(t *testing.T) {
	verifier := "correct-verifier"
	h := sha256.Sum256([]byte("wrong-verifier"))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	assert.False(t, verifyPKCE(verifier, challenge))
}

func TestVerifyPKCE_EmptyVerifier(t *testing.T) {
	// SHA256 of empty string is a specific hash, so it won't match a non-empty challenge.
	challenge := "some-challenge-value"
	assert.False(t, verifyPKCE("", challenge))
}

func TestVerifyPKCE_EmptyChallenge(t *testing.T) {
	verifier := "some-verifier"
	assert.False(t, verifyPKCE(verifier, ""))
}

// --- TokenHandler.ServeHTTP tests ---

func TestTokenHandler_GETMethodNotAllowed(t *testing.T) {
	handler := &TokenHandler{}

	req := httptest.NewRequest(http.MethodGet, "/oauth/token", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestTokenHandler_WrongGrantType(t *testing.T) {
	handler := &TokenHandler{}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token",
		strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unsupported_grant_type", body["error"])
	assert.Equal(t, "only authorization_code and refresh_token are supported", body["error_description"])
}

func TestTokenHandler_MissingCode(t *testing.T) {
	handler := &TokenHandler{}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token",
		strings.NewReader("grant_type=authorization_code&code_verifier=some-verifier"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", body["error"])
	assert.Equal(t, "code and code_verifier are required", body["error_description"])
}

func TestTokenHandler_MissingCodeVerifier(t *testing.T) {
	handler := &TokenHandler{}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token",
		strings.NewReader("grant_type=authorization_code&code=some-code"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", body["error"])
	assert.Equal(t, "code and code_verifier are required", body["error_description"])
}

func TestTokenHandler_MissingRefreshToken(t *testing.T) {
	handler := &TokenHandler{}

	req := httptest.NewRequest(http.MethodPost, "/oauth/token",
		strings.NewReader("grant_type=refresh_token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]string
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", body["error"])
	assert.Equal(t, "refresh_token is required", body["error_description"])
}
