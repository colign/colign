DROP INDEX IF EXISTS idx_tasks_assignee;
ALTER TABLE tasks DROP COLUMN IF EXISTS creator_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS assignee_id;
