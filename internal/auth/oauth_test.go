package auth

import "testing"

func TestOAuthServiceEnabledProviders(t *testing.T) {
	service := NewOAuthService(nil, nil, OAuthConfig{
		GitHubClientID:     "github-id",
		GitHubClientSecret: "github-secret",
	}, nil)

	status := service.EnabledProviders()

	if !status.GitHub {
		t.Fatal("expected github provider to be enabled")
	}
	if status.Google {
		t.Fatal("expected google provider to be disabled")
	}
}

func TestOAuthServiceGetAuthURLRejectsDisabledProvider(t *testing.T) {
	service := NewOAuthService(nil, nil, OAuthConfig{}, nil)

	if _, err := service.GetAuthURL("github", "state"); err == nil {
		t.Fatal("expected disabled github provider to be rejected")
	}
}
