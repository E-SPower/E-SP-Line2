-- E-SP-Line2 Database Migration
-- Version: 003
-- Description: Instance-level monitoring counters read by the Monitoring page
--
-- AdapterInstance carries the three metrics the Monitoring UI reads. Before
-- this migration the page mapped instance.last_heartbeat / message_count /
-- error_count while models.AdapterInstance had none of them, so every number
-- was a hard-coded 0. The counters are NOT NULL with a 0 default so the JSON
-- contract is a number, never null; last_heartbeat stays nullable because
-- "no heartbeat yet" and "heartbeat at the epoch" must be distinguishable.
--
-- Dialect note: `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` is PostgreSQL-only;
-- SQLite rejects it even at 3.53. The statements below are the plain form so
-- the same file runs on both, which means a re-run errors on the ADD COLUMN
-- lines after the first application. That is harmless in practice: GORM
-- AutoMigrate creates the same columns on every startup, so SQLite dev boxes
-- need not run this file at all, and PostgreSQL operators can prefix
-- `IF NOT EXISTS` if they script it.

ALTER TABLE adapter_instances ADD COLUMN last_heartbeat TIMESTAMP;
ALTER TABLE adapter_instances ADD COLUMN message_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE adapter_instances ADD COLUMN error_count BIGINT NOT NULL DEFAULT 0;

-- The Monitoring page lists instances by recent activity.
CREATE INDEX IF NOT EXISTS idx_adapter_instances_last_heartbeat
    ON adapter_instances (last_heartbeat);

-- Backfill existing rows from the session table where a session already knows
-- the last heartbeat. message_count / error_count start at 0: there is no
-- historical source for them.
UPDATE adapter_instances
SET last_heartbeat = (
    SELECT s.last_heartbeat
    FROM adapter_sessions s
    WHERE s.instance_id = adapter_instances.id
)
WHERE last_heartbeat IS NULL
  AND EXISTS (
    SELECT 1 FROM adapter_sessions s
    WHERE s.instance_id = adapter_instances.id
      AND s.last_heartbeat IS NOT NULL
  );

-- Verify:
--   PRAGMA table_info(adapter_instances);            -- SQLite
--   \d adapter_instances                             -- PostgreSQL

-- Rollback:
--   DROP INDEX IF EXISTS idx_adapter_instances_last_heartbeat;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS error_count;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS message_count;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS last_heartbeat;
