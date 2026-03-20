-- Drop old comments tables (no data, schema change)
DROP TABLE IF EXISTS comment_replies;
DROP TABLE IF EXISTS comments;

CREATE TABLE comments (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    quoted_text TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_change_doc ON comments(change_id, document_type);

CREATE TABLE comment_replies (
    id BIGSERIAL PRIMARY KEY,
    comment_id BIGINT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comment_replies_comment ON comment_replies(comment_id);
