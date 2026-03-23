ALTER TABLE changes ADD COLUMN archived_at TIMESTAMPTZ;
CREATE INDEX idx_changes_active ON changes (project_id) WHERE archived_at IS NULL;

CREATE TABLE archive_policies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id BIGINT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    mode TEXT NOT NULL DEFAULT 'manual' CHECK (mode IN ('manual', 'auto')),
    trigger_type TEXT NOT NULL DEFAULT 'tasks_done' CHECK (trigger_type IN ('tasks_done', 'days_after_ready', 'tasks_done_and_days')),
    days_delay INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
