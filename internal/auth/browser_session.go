package auth

import (
	"net/http"
	"strings"
	"time"
)

const (
	BrowserAccessCookieName  = "colign_access_token"
	BrowserRefreshCookieName = "colign_refresh_token"
)

type BrowserSessionOptions struct {
	Domain string
	Secure bool
}

func SetBrowserSessionCookies(w http.ResponseWriter, tokenPair *TokenPair, opts BrowserSessionOptions) {
	for _, cookie := range browserSessionCookies(tokenPair, opts) {
		http.SetCookie(w, cookie)
	}
}

func AppendBrowserSessionCookies(header http.Header, tokenPair *TokenPair, opts BrowserSessionOptions) {
	for _, cookie := range browserSessionCookies(tokenPair, opts) {
		header.Add("Set-Cookie", cookie.String())
	}
}

func ClearBrowserSessionCookies(w http.ResponseWriter, opts BrowserSessionOptions) {
	for _, cookie := range clearBrowserSessionCookies(opts) {
		http.SetCookie(w, cookie)
	}
}

func AppendClearBrowserSessionCookies(header http.Header, opts BrowserSessionOptions) {
	for _, cookie := range clearBrowserSessionCookies(opts) {
		header.Add("Set-Cookie", cookie.String())
	}
}

func BrowserSessionFromRequest(r *http.Request) (accessToken, refreshToken string) {
	if accessCookie, err := r.Cookie(BrowserAccessCookieName); err == nil {
		accessToken = accessCookie.Value
	}
	if refreshCookie, err := r.Cookie(BrowserRefreshCookieName); err == nil {
		refreshToken = refreshCookie.Value
	}
	return accessToken, refreshToken
}

func browserSessionCookies(tokenPair *TokenPair, opts BrowserSessionOptions) []*http.Cookie {
	return []*http.Cookie{
		{
			Name:     BrowserAccessCookieName,
			Value:    tokenPair.AccessToken,
			Path:     "/",
			Domain:   opts.Domain,
			HttpOnly: false,
			Secure:   opts.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(AccessTokenDuration / time.Second),
		},
		{
			Name:     BrowserRefreshCookieName,
			Value:    tokenPair.RefreshToken,
			Path:     "/",
			Domain:   opts.Domain,
			HttpOnly: true,
			Secure:   opts.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(RefreshTokenDuration / time.Second),
		},
	}
}

func clearBrowserSessionCookies(opts BrowserSessionOptions) []*http.Cookie {
	return []*http.Cookie{
		{
			Name:     BrowserAccessCookieName,
			Value:    "",
			Path:     "/",
			Domain:   opts.Domain,
			HttpOnly: false,
			Secure:   opts.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		},
		{
			Name:     BrowserRefreshCookieName,
			Value:    "",
			Path:     "/",
			Domain:   opts.Domain,
			HttpOnly: true,
			Secure:   opts.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		},
	}
}

func DeriveCookieDomain(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Split(host, ":")[0]
	if host == "" || host == "localhost" {
		return ""
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + strings.Join(parts[len(parts)-2:], ".")
}
