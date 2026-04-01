package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/gobenpark/colign/internal/auth"
)

type Config struct {
	Port                string
	Debug               bool
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	ClaudeAPIKey        string
	GitHubClientID      string
	GitHubClientSecret  string
	GoogleClientID      string
	GoogleClientSecret  string
	RedirectBaseURL     string
	FrontendURL         string
	MigrationsPath      string
	Edition             string // "ce" (Community) or "ee" (Enterprise)
	HocuspocusURL       string
	HocuspocusAPISecret string
	CookieDomain        string
	CookieSecure        bool
	AIEncryptionKey     string
	VAPIDPublicKey      string
	VAPIDPrivateKey     string
	VAPIDSubject        string
}

func Load() (*Config, error) {
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	redirectBaseURL := getEnv("REDIRECT_BASE_URL", "http://localhost:8080")

	debug := getEnv("DEBUG", "") == "true" || getEnv("DEBUG", "") == "1"
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		if debug {
			jwtSecret = "dev-secret-change-in-production"
		} else {
			return nil, fmt.Errorf("JWT_SECRET environment variable is required in production")
		}
	}

	aiEncryptionKey := getEnv("AI_ENCRYPTION_KEY", "")

	return &Config{
		Port:                getEnv("PORT", "8080"),
		Debug:               debug,
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/colign?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:           jwtSecret,
		ClaudeAPIKey:        getEnv("CLAUDE_API_KEY", ""),
		GitHubClientID:      getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:  getEnv("GITHUB_CLIENT_SECRET", ""),
		GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  getEnv("GOOGLE_CLIENT_SECRET", ""),
		RedirectBaseURL:     redirectBaseURL,
		FrontendURL:         frontendURL,
		MigrationsPath:      getEnv("MIGRATIONS_PATH", "migrations"),
		Edition:             getEnv("COLIGN_EDITION", "ce"),
		HocuspocusURL:       getEnv("HOCUSPOCUS_URL", ""),
		HocuspocusAPISecret: getEnv("HOCUSPOCUS_API_SECRET", ""),
		CookieDomain:        getEnv("AUTH_COOKIE_DOMAIN", deriveCookieDomain(frontendURL, redirectBaseURL)),
		CookieSecure:        deriveCookieSecure(frontendURL, redirectBaseURL),
		AIEncryptionKey:     aiEncryptionKey,
		VAPIDPublicKey:      getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:     getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:        getEnv("VAPID_SUBJECT", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func deriveCookieDomain(urls ...string) string {
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		if domain := auth.DeriveCookieDomain(parsed.Hostname()); domain != "" {
			return domain
		}
	}
	return ""
}

func deriveCookieSecure(urls ...string) bool {
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if parsed.Scheme == "https" {
			return true
		}
	}
	return false
}
