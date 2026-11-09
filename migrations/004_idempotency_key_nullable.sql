-- E-SP-Line2 Database Migration
-- Version: 004
-- Description: idempotency key becomes nullable, with a generated fallback

-- Why
--
-- `inbound_events.idempotency_key` was `UNIQUE` while the Go field was a plain
-- string, so "no key supplied" was stored as ''. SQLite's UNIQUE treats NULLs as
-- distinct but permits only one ''. The second keyless message therefore failed
-- with `UNIQUE constraint failed: inbound_events.idempotency_key`. That was
-- latent until the ingest route existed; once it did, message #2 onwards 500ed.
--
-- MessageService.Create now writes a generated key whenever the caller supplies
-- none (models.GenerateIdempotencyKey), and the field is a *string so a NULL row
-- is still valid. Both halves are needed: the pointer fixes the schema, the
-- fallback keeps the unique index meaningful for deduplication instead of
-- letting every keyless row dedupe to the same NULL.
--
-- Re-runnable: SQLite cannot ALTER a column, so the table is rebuilt. The
-- column is already nullable when GORM's AutoMigrate ran first, in which case
-- the copies below are behaviour-preserving no-ops.

PRAGMA foreign_keys = OFF;

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
    -- Nullable on purpose: only one '' can exist under a UNIQUE index.
    idempotency_key VARCHAR(255) UNIQUE,
    trace_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'received',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (platform_id) REFERENCES platforms(id),
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- Carry existing rows over. A stored '' becomes NULL so it stops blocking the
-- next keyless insert; ordering is preserved by copying created_at.
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
FROM inbound_events
WHERE true
ON CONFLICT DO NOTHING;

DROP TABLE inbound_events;

ALTER TABLE inbound_events_new RENAME TO inbound_events;

-- Indexes the queries need (the unique index on idempotency_key travels with the
-- column definition). These mirror the GORM tags on the model.
CREATE INDEX IF NOT EXISTS idx_inbound_events_platform ON inbound_events(platform_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_instance ON inbound_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_conversation ON inbound_events(conversation_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_trace ON inbound_events(trace_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_status ON inbound_events(status);

PRAGMA foreign_keys = ON;
