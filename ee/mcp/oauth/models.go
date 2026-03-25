package oauth

import (
	"time"

	"github.com/uptrace/bun"
)

type OAuthClient struct {
	bun.BaseModel `bun:"table:oauth_clients,alias:oc"`

	ID           int64     `bun:"id,pk,autoincrement"`
	ClientID     string    `bun:"client_id,notnull,unique"`
	ClientName   string    `bun:"client_name,notnull"`
	RedirectURIs []string  `bun:"redirect_uris,array"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type OAuthAuthorizationCode struct {
	bun.BaseModel `bun:"table:oauth_authorization_codes,alias:oac"`

	ID            int64     `bun:"id,pk,autoincrement"`
	UserID        int64     `bun:"user_id,notnull"`
	OrgID         int64     `bun:"org_id,notnull"`
	ClientID      string    `bun:"client_id,notnull"`
	Code          string    `bun:"code,notnull,unique"`
	CodeChallenge string    `bun:"code_challenge,notnull"`
	RedirectURI   string    `bun:"redirect_uri,notnull"`
	ExpiresAt     time.Time `bun:"expires_at,notnull"`
	Used          bool      `bun:"used,notnull,default:false"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type OAuthRefreshToken struct {
	bun.BaseModel `bun:"table:oauth_refresh_tokens,alias:ort"`

	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    int64     `bun:"user_id,notnull"`
	OrgID     int64     `bun:"org_id,notnull"`
	ClientID  string    `bun:"client_id,notnull"`
	TokenHash string    `bun:"token_hash,notnull,unique"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	Used      bool      `bun:"used,notnull,default:false"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}
