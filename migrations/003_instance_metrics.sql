-- E-SP-Line2 Database Migration
-- Version: 003
-- Description: Instance-level runtime metrics (heartbeat, message/error counters)
--
-- The monitoring page reads these three columns off an instance. Before this
-- migration they existed on AdapterSession/AdapterConnection only, so the page
-- permanently showed 0 messages, 0 errors and no heartbeat.
--
-- On startup GORM AutoMigrate applies the same three columns, so a fresh or
-- SQLite deployment is covered without this file. Run it by hand only when the
-- schema is managed outside GORM (PostgreSQL production):
--
--   psql "$DSN" -f migrations/003_instance_metrics.sql
--
-- Postgres supports ADD COLUMN IF NOT EXISTS, which makes the script re-runnable;
-- SQLite has no such clause, hence the AutoMigrate note above.

ALTER TABLE adapter_instances
    ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMP;

ALTER TABLE adapter_instances
    ADD COLUMN IF NOT EXISTS message_count BIGINT NOT NULL DEFAULT 0;

ALTER TABLE adapter_instances
    ADD COLUMN IF NOT EXISTS error_count BIGINT NOT NULL DEFAULT 0;

-- The monitoring list orders/serves every instance; the counter columns are
-- read wholesale, so index the only column used in a predicate (liveness).
CREATE INDEX IF NOT EXISTS idx_adapter_instances_last_heartbeat
    ON adapter_instances(last_heartbeat);

-- Rollback:
--   DROP INDEX IF EXISTS idx_adapter_instances_last_heartbeat;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS last_heartbeat;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS message_count;
--   ALTER TABLE adapter_instances DROP COLUMN IF EXISTS error_count;
