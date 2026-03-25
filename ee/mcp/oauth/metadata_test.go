package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtectedResourceMetadata(t *testing.T) {
	baseURL := "https://api.colign.co"
	handler := ProtectedResourceMetadata(baseURL)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, baseURL+"/mcp", body["resource"])

	authServers, ok := body["authorization_servers"].([]any)
	require.True(t, ok, "authorization_servers should be an array")
	require.Len(t, authServers, 1)
	assert.Equal(t, baseURL, authServers[0])

	bearerMethods, ok := body["bearer_methods_supported"].([]any)
	require.True(t, ok, "bearer_methods_supported should be an array")
	require.Len(t, bearerMethods, 1)
	assert.Equal(t, "header", bearerMethods[0])
}

func TestAuthorizationServerMetadata(t *testing.T) {
	baseURL := "https://api.colign.co"
	handler := AuthorizationServerMetadata(baseURL)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, baseURL, body["issuer"])
	assert.Equal(t, baseURL+"/oauth/authorize", body["authorization_endpoint"])
	assert.Equal(t, baseURL+"/oauth/token", body["token_endpoint"])
	assert.Equal(t, baseURL+"/oauth/register", body["registration_endpoint"])

	responseTypes, ok := body["response_types_supported"].([]any)
	require.True(t, ok, "response_types_supported should be an array")
	require.Len(t, responseTypes, 1)
	assert.Equal(t, "code", responseTypes[0])

	grantTypes, ok := body["grant_types_supported"].([]any)
	require.True(t, ok, "grant_types_supported should be an array")
	require.Len(t, grantTypes, 2)
	assert.Equal(t, "authorization_code", grantTypes[0])
	assert.Equal(t, "refresh_token", grantTypes[1])

	codeMethods, ok := body["code_challenge_methods_supported"].([]any)
	require.True(t, ok, "code_challenge_methods_supported should be an array")
	require.Len(t, codeMethods, 1)
	assert.Equal(t, "S256", codeMethods[0])

	tokenAuth, ok := body["token_endpoint_auth_methods_supported"].([]any)
	require.True(t, ok, "token_endpoint_auth_methods_supported should be an array")
	require.Len(t, tokenAuth, 1)
	assert.Equal(t, "none", tokenAuth[0])
}
