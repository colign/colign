CREATE TABLE approval_policies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id BIGINT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    policy TEXT NOT NULL DEFAULT 'owner_one' CHECK (policy IN ('owner_one', 'any_one', 'all')),
    min_count INTEGER NOT NULL DEFAULT 1 CHECK (min_count > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE approvals (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('approved', 'rejected', 'pending')),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(change_id, user_id)
);

CREATE INDEX idx_approvals_change_id ON approvals(change_id);
CREATE INDEX idx_approvals_user_id ON approvals(user_id);

CREATE TABLE workflow_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    from_stage TEXT NOT NULL,
    to_stage TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_events_change_id ON workflow_events(change_id);
CREATE INDEX idx_workflow_events_user_id ON workflow_events(user_id);
CREATE INDEX idx_workflow_events_created_at ON workflow_events(created_at);

CREATE TABLE acceptance_criteria (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    scenario TEXT NOT NULL DEFAULT '',
    steps JSONB NOT NULL DEFAULT '[]',
    met BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_acceptance_criteria_change_id ON acceptance_criteria(change_id);
