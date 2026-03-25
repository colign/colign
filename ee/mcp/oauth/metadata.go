package oauth

import (
	"encoding/json"
	"net/http"
)

// ProtectedResourceMetadata serves RFC 9728 Protected Resource Metadata.
func ProtectedResourceMetadata(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resource":                 baseURL + "/mcp",
			"authorization_servers":    []string{baseURL},
			"bearer_methods_supported": []string{"header"},
		})
	}
}

// AuthorizationServerMetadata serves OAuth 2.0 Authorization Server Metadata (RFC 8414).
func AuthorizationServerMetadata(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                baseURL,
			"authorization_endpoint":                baseURL + "/oauth/authorize",
			"token_endpoint":                        baseURL + "/oauth/token",
			"registration_endpoint":                 baseURL + "/oauth/register",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	}
}
