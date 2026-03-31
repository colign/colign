CREATE TABLE org_ai_configs (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    org_id BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    api_key_encrypted BYTEA NOT NULL DEFAULT '',
    key_version SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_org_ai_configs_org_id ON org_ai_configs (org_id);
