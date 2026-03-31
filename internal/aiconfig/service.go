package aiconfig

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
)

// Service handles CRUD operations for AI configuration records.
type Service struct {
	db            *bun.DB
	encryptionKey []byte
}

// UpsertInput holds the fields required to create or update an AIConfig.
type UpsertInput struct {
	Provider              string
	Model                 string
	APIKey                string // empty = keep existing encrypted key
	IncludeProjectContext bool
}

// NewService creates a new Service.
// encryptionKey must be 32 bytes for AES-256.
func NewService(db *bun.DB, encryptionKey []byte) *Service {
	return &Service{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// Upsert creates or updates an AIConfig for the given project.
// If input.APIKey is empty and a record already exists, the existing
// api_key_encrypted value is preserved.
func (s *Service) Upsert(ctx context.Context, projectID int64, input UpsertInput) (*AIConfig, error) {
	var encryptedKey []byte

	if input.APIKey == "" {
		// Attempt to load the existing record so we can preserve its key.
		existing, err := s.GetByProjectID(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			encryptedKey = existing.APIKeyEncrypted
		}
		// If there is no existing record and APIKey is empty, we proceed with an
		// empty slice; the DB constraint will surface the error at insert time.
	} else {
		var err error
		encryptedKey, err = Encrypt(input.APIKey, s.encryptionKey, 1)
		if err != nil {
			return nil, err
		}
	}

	cfg := &AIConfig{
		ProjectID:             projectID,
		Provider:              input.Provider,
		Model:                 input.Model,
		APIKeyEncrypted:       encryptedKey,
		KeyVersion:            1,
		IncludeProjectContext: input.IncludeProjectContext,
		UpdatedAt:             time.Now(),
	}

	_, err := s.db.NewInsert().
		Model(cfg).
		On("CONFLICT (project_id) DO UPDATE").
		Set("provider = EXCLUDED.provider").
		Set("model = EXCLUDED.model").
		Set("api_key_encrypted = EXCLUDED.api_key_encrypted").
		Set("key_version = EXCLUDED.key_version").
		Set("include_project_context = EXCLUDED.include_project_context").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id, created_at, updated_at").
		Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: upsert failed",
			slog.Int64("project_id", projectID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return cfg, nil
}

// GetByProjectID returns the AIConfig for the given project.
// Returns nil, nil when no record is found.
func (s *Service) GetByProjectID(ctx context.Context, projectID int64) (*AIConfig, error) {
	cfg := new(AIConfig)
	err := s.db.NewSelect().
		Model(cfg).
		Where("project_id = ?", projectID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

// Delete removes the AIConfig for the given project.
func (s *Service) Delete(ctx context.Context, projectID int64) error {
	_, err := s.db.NewDelete().
		Model((*AIConfig)(nil)).
		Where("project_id = ?", projectID).
		Exec(ctx)
	return err
}

// DecryptAPIKey decrypts the API key stored in cfg.
func (s *Service) DecryptAPIKey(cfg *AIConfig) (string, error) {
	return Decrypt(cfg.APIKeyEncrypted, s.encryptionKey)
}

// OrgUpsertInput holds the fields for creating or updating an OrgAIConfig.
type OrgUpsertInput struct {
	Provider string
	Model    string
	APIKey   string // empty = keep existing encrypted key
}

// UpsertOrg creates or updates an OrgAIConfig for the given organization.
func (s *Service) UpsertOrg(ctx context.Context, orgID int64, input OrgUpsertInput) (*OrgAIConfig, error) {
	var encryptedKey []byte

	if input.APIKey == "" {
		existing, err := s.GetByOrgID(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			encryptedKey = existing.APIKeyEncrypted
		} else {
			return nil, errors.New("aiconfig: api key is required for first-time configuration")
		}
	} else {
		var err error
		encryptedKey, err = Encrypt(input.APIKey, s.encryptionKey, 1)
		if err != nil {
			return nil, err
		}
	}

	cfg := &OrgAIConfig{
		OrgID:           orgID,
		Provider:        input.Provider,
		Model:           input.Model,
		APIKeyEncrypted: encryptedKey,
		KeyVersion:      1,
		UpdatedAt:       time.Now(),
	}

	_, err := s.db.NewInsert().
		Model(cfg).
		On("CONFLICT (org_id) DO UPDATE").
		Set("provider = EXCLUDED.provider").
		Set("model = EXCLUDED.model").
		Set("api_key_encrypted = EXCLUDED.api_key_encrypted").
		Set("key_version = EXCLUDED.key_version").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id, created_at, updated_at").
		Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "aiconfig: org upsert failed",
			slog.Int64("org_id", orgID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return cfg, nil
}

// GetByOrgID returns the OrgAIConfig for the given organization.
// Returns nil, nil when no record is found.
func (s *Service) GetByOrgID(ctx context.Context, orgID int64) (*OrgAIConfig, error) {
	cfg := new(OrgAIConfig)
	err := s.db.NewSelect().
		Model(cfg).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

// DeleteOrg removes the OrgAIConfig for the given organization.
func (s *Service) DeleteOrg(ctx context.Context, orgID int64) error {
	_, err := s.db.NewDelete().
		Model((*OrgAIConfig)(nil)).
		Where("org_id = ?", orgID).
		Exec(ctx)
	return err
}

// DecryptOrgAPIKey decrypts the API key stored in an OrgAIConfig.
func (s *Service) DecryptOrgAPIKey(cfg *OrgAIConfig) (string, error) {
	return Decrypt(cfg.APIKeyEncrypted, s.encryptionKey)
}
