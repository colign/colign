package aiconfig

import (
	"time"

	"github.com/uptrace/bun"
)

type AIConfig struct {
	bun.BaseModel         `bun:"table:ai_configs,alias:aic"`
	ID                    int64     `bun:"id,pk,autoincrement"`
	ProjectID             int64     `bun:"project_id,notnull,unique"`
	Provider              string    `bun:"provider,notnull"`
	Model                 string    `bun:"model,notnull"`
	APIKeyEncrypted       []byte    `bun:"api_key_encrypted,notnull"`
	KeyVersion            int16     `bun:"key_version,notnull,default:1"`
	IncludeProjectContext bool      `bun:"include_project_context,notnull"`
	CreatedAt             time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt             time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

// OrgAIConfig stores organization-level AI provider configuration.
type OrgAIConfig struct {
	bun.BaseModel   `bun:"table:org_ai_configs,alias:oaic"`
	ID              int64     `bun:"id,pk,autoincrement"`
	OrgID           int64     `bun:"org_id,notnull,unique"`
	Provider        string    `bun:"provider,notnull"`
	Model           string    `bun:"model,notnull"`
	APIKeyEncrypted []byte    `bun:"api_key_encrypted,notnull"`
	KeyVersion      int16     `bun:"key_version,notnull,default:1"`
	CreatedAt       time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt       time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
