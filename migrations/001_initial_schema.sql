-- E-SP-Line2 Database Migration
-- Version: 001
-- Description: Initial schema setup

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Platforms table
CREATE TABLE IF NOT EXISTS platforms (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Adapter packages table
CREATE TABLE IF NOT EXISTS adapter_packages (
    id VARCHAR(36) PRIMARY KEY,
    platform_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    version VARCHAR(20) NOT NULL,
    runtime_type VARCHAR(20),
    protocol_version VARCHAR(10),
    status VARCHAR(20) DEFAULT 'active',
    manifest TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (platform_id) REFERENCES platforms(id)
);

-- Adapter capabilities table
CREATE TABLE IF NOT EXISTS adapter_capabilities (
    id VARCHAR(36) PRIMARY KEY,
    adapter_id VARCHAR(36) NOT NULL,
    capability VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (adapter_id) REFERENCES adapter_packages(id)
);

-- Adapter instances table
CREATE TABLE IF NOT EXISTS adapter_instances (
    id VARCHAR(36) PRIMARY KEY,
    adapter_id VARCHAR(36) NOT NULL,
    platform_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    config TEXT,
    status VARCHAR(20) DEFAULT 'stopped',
    user_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (adapter_id) REFERENCES adapter_packages(id),
    FOREIGN KEY (platform_id) REFERENCES platforms(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Adapter sessions table
CREATE TABLE IF NOT EXISTS adapter_sessions (
    id VARCHAR(36) PRIMARY KEY,
    instance_id VARCHAR(36) NOT NULL UNIQUE,
    worker_id VARCHAR(100),
    lease_expiry TIMESTAMP,
    connected_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    status VARCHAR(20),
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- Inbound events table
CREATE TABLE IF NOT EXISTS inbound_events (
    id VARCHAR(36) PRIMARY KEY,
    platform_id VARCHAR(36) NOT NULL,
    instance_id VARCHAR(36) NOT NULL,
    conversation_id VARCHAR(100),
    sender_id VARCHAR(100),
    sender_name VARCHAR(100),
    message_type VARCHAR(20),
    message_content TEXT,
    raw_message TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    trace_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'received',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (platform_id) REFERENCES platforms(id),
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- Outbound commands table
CREATE TABLE IF NOT EXISTS outbound_commands (
    id VARCHAR(36) PRIMARY KEY,
    instance_id VARCHAR(36) NOT NULL,
    command_type VARCHAR(50) NOT NULL,
    payload TEXT,
    status VARCHAR(20) DEFAULT 'created',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    trace_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- Route rules table
CREATE TABLE IF NOT EXISTS route_rules (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    platform_id VARCHAR(36),
    instance_id VARCHAR(36),
    priority INTEGER DEFAULT 0,
    conditions TEXT,
    target_type VARCHAR(20),
    target_id VARCHAR(100),
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (platform_id) REFERENCES platforms(id),
    FOREIGN KEY (instance_id) REFERENCES adapter_instances(id)
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    resource_id VARCHAR(36),
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_inbound_events_platform ON inbound_events(platform_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_instance ON inbound_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_inbound_events_status ON inbound_events(status);
CREATE INDEX IF NOT EXISTS idx_inbound_events_created ON inbound_events(created_at);

CREATE INDEX IF NOT EXISTS idx_outbound_commands_instance ON outbound_commands(instance_id);
CREATE INDEX IF NOT EXISTS idx_outbound_commands_status ON outbound_commands(status);

CREATE INDEX IF NOT EXISTS idx_route_rules_platform ON route_rules(platform_id);
CREATE INDEX IF NOT EXISTS idx_route_rules_enabled ON route_rules(enabled);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);

-- Insert default platforms
INSERT INTO platforms (id, name, code, description, status) VALUES
('platform-taobao', '淘宝', 'taobao', '淘宝电商平台', 'active'),
('platform-xianyu', '闲鱼', 'xianyu', '闲鱼二手交易平台', 'active')
ON CONFLICT (code) DO NOTHING;
