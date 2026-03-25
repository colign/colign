CREATE TABLE oauth_clients (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    client_id    TEXT NOT NULL UNIQUE,
    client_name  TEXT NOT NULL,
    redirect_uris TEXT[],
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oauth_authorization_codes (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id         BIGINT NOT NULL,
    client_id      TEXT NOT NULL,
    code           TEXT NOT NULL UNIQUE,
    code_challenge TEXT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    used           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_codes_code ON oauth_authorization_codes(code);

CREATE TABLE oauth_refresh_tokens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id BIGINT NOT NULL,
    client_id TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oauth_refresh_tokens_hash ON oauth_refresh_tokens(token_hash);
CREATE INDEX idx_oauth_refresh_tokens_client ON oauth_refresh_tokens(user_id, org_id, client_id);
