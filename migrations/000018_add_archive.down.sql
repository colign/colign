DROP INDEX IF EXISTS idx_changes_active;
ALTER TABLE changes DROP COLUMN IF EXISTS archived_at;
DROP TABLE IF EXISTS archive_policies;
