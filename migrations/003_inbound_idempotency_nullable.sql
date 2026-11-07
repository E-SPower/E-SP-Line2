-- E-SP-Line2 Database Migration
-- Version: 003
-- Description: Make inbound_events.idempotency_key safely unique for empty keys
--
-- Why: `idempotency_key VARCHAR(255) UNIQUE` fails on the second row without a
-- key. SQLite's UNIQUE allows many NULLs but only one empty string, and Go's
-- zero value for the column used to be "". Once the HTTP/WebSocket ingest path
-- is live this reads as "every message after the first is a 500".
--
-- Shape after this migration: the column is nullable, every row that had an
-- empty key becomes NULL, and the unique index is created with IF NOT EXISTS so
-- the script is safe to re-run. MessageService.Create still fills a generated
-- fallback key for every message it persists, so new rows stay non-empty; the
-- NULL allowance is the safety net for backfills and direct inserts.

-- 1) Drop the inline UNIQUE (SQLite can only do this by rebuilding the table).
CREATE TABLE IF NOT EXISTS inbound_events_new (
    id VARCHAR(36) PRIMARY KEY,
    platform_id VARCHAR(36) NOT NULL,
    instance_id VARCHAR(36) NOT NULL,
    conversation_id VARCHAR(100),
    sender_id VARCHAR(100),
    sender_name VARCHAR(100),
    message_type VARCHAR(20),
    message_content TEXT,
    raw_message TEXT,
    idempotency_key VARCHAR(255),
    trace_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'received',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (platform_id) REFERENCES platforms(id),
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- 2) Copy the data, turning empty keys into NULL so several rows may share it.
INSERT INTO inbound_events_new (
    id, platform_id, instance_id, conversation_id, sender_id, sender_name,
    message_type, message_content, raw_message, idempotency_key, trace_id,
    status, created_at, updated_at
)
SELECT
    id, platform_id, instance_id, conversation_id, sender_id, sender_name,
    message_type, message_content, raw_message,
    NULLIF(idempotency_key, ''), trace_id,
    status, created_at, updated_at
FROM inbound_events;

DROP TABLE inbound_events;
ALTER TABLE inbound_events_new RENAME TO inbound_events;

-- 3) Recreate the indexes the queries need, plus the unique key index.
CREATE INDEX IF NOT EXISTS idx_inbound_events_platform ON inbound_events(platform_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_instance ON inbound_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_status ON inbound_events(status);
CREATE INDEX IF NOT EXISTS idx_inbound_events_created ON inbound_events(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inbound_events_idempotency ON inbound_events(idempotency_key);
