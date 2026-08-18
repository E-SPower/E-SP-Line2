-- E-SP-Line2 Database Migration
-- Version: 002
-- Description: Adapter gateway (接入器) entities and connections

-- Adapter (接入器) entities table
CREATE TABLE IF NOT EXISTS adapters (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    mode VARCHAR(20) DEFAULT 'server',      -- server / client
    listen_path VARCHAR(200),               -- server mode: listen path
    ws_url VARCHAR(500),                    -- client mode: target WebSocket URL
    key VARCHAR(200) NOT NULL,              -- user-defined, editable access key
    platform VARCHAR(50),                   -- taobao / xianyu / "" (all)
    scope VARCHAR(20) DEFAULT 'read+write', -- read / write / read+write
    status VARCHAR(20) DEFAULT 'active',    -- active / disabled
    enabled BOOLEAN DEFAULT TRUE,
    last_connected_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(36)
);

-- Adapter connections table
CREATE TABLE IF NOT EXISTS adapter_connections (
    id VARCHAR(36) PRIMARY KEY,
    adapter_id VARCHAR(36) NOT NULL,
    adapter_name VARCHAR(100),
    mode VARCHAR(20),
    platform VARCHAR(50),
    remote_addr VARCHAR(100),
    status VARCHAR(20) DEFAULT 'connected',
    connected_at TIMESTAMP,
    disconnected_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    message_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (adapter_id) REFERENCES adapters(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_adapters_status ON adapters(status);
CREATE INDEX IF NOT EXISTS idx_adapters_enabled ON adapters(enabled);
CREATE INDEX IF NOT EXISTS idx_adapter_connections_adapter ON adapter_connections(adapter_id);
CREATE INDEX IF NOT EXISTS idx_adapter_connections_status ON adapter_connections(status);
CREATE INDEX IF NOT EXISTS idx_adapter_connections_created ON adapter_connections(created_at);
