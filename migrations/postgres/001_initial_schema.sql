-- StableRisk Initial Schema
-- This migration creates the core database schema for audit logs, users, API keys, and outliers

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'analyst', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT username_length CHECK (char_length(username) >= 3),
    CONSTRAINT role_not_empty CHECK (role != '')
);

-- Index for faster user lookups
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_used TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT name_not_empty CHECK (name != '')
);

-- Indexes for API keys
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);

-- Audit Logs table (tamper-proof, append-only)
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id TEXT,
    username TEXT,
    action TEXT NOT NULL,
    resource TEXT,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    signature TEXT NOT NULL,
    CONSTRAINT action_not_empty CHECK (action != '')
);

-- Prevent updates and deletes on audit_logs (append-only)
CREATE OR REPLACE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE OR REPLACE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

-- Indexes for audit logs
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource);
CREATE INDEX idx_audit_logs_ip_address ON audit_logs(ip_address);
CREATE INDEX idx_audit_logs_details ON audit_logs USING GIN(details);

-- Outliers cache table
CREATE TABLE IF NOT EXISTS outliers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type TEXT NOT NULL CHECK (type IN ('zscore', 'iqr', 'pattern_circulation', 'pattern_fanout', 'pattern_fanin', 'pattern_dormant', 'pattern_velocity')),
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    address TEXT NOT NULL,
    transaction_hash TEXT,
    amount NUMERIC(30, 6),
    z_score NUMERIC(10, 4),
    details JSONB NOT NULL,
    acknowledged BOOLEAN NOT NULL DEFAULT false,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    notes TEXT,
    CONSTRAINT type_not_empty CHECK (type != ''),
    CONSTRAINT severity_not_empty CHECK (severity != ''),
    CONSTRAINT address_not_empty CHECK (address != '')
);

-- Indexes for outliers
CREATE INDEX idx_outliers_detected_at ON outliers(detected_at DESC);
CREATE INDEX idx_outliers_address ON outliers(address);
CREATE INDEX idx_outliers_type ON outliers(type);
CREATE INDEX idx_outliers_severity ON outliers(severity);
CREATE INDEX idx_outliers_acknowledged ON outliers(acknowledged);
CREATE INDEX idx_outliers_transaction_hash ON outliers(transaction_hash);
CREATE INDEX idx_outliers_details ON outliers USING GIN(details);

-- Refresh tokens table (for JWT refresh token storage)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT expires_after_creation CHECK (expires_at > created_at)
);

-- Indexes for refresh tokens
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(revoked);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at on users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default admin user (password: admin123456 - CHANGE IN PRODUCTION)
-- Password hash generated with bcrypt cost 12
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES (
    'admin',
    'admin@stablerisk.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIjKYqKy4G',  -- admin123456
    'admin',
    true
) ON CONFLICT (username) DO NOTHING;

-- Insert default analyst user (password: analyst123456 - CHANGE IN PRODUCTION)
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES (
    'analyst',
    'analyst@stablerisk.local',
    '$2a$12$8p2TQ0QmVc.WxKq0Y3YxfuGxqLc5VnE9qTqNv6fV8cVLzJvPxGsYe',  -- analyst123456
    'analyst',
    true
) ON CONFLICT (username) DO NOTHING;

-- Insert default viewer user (password: viewer123456 - CHANGE IN PRODUCTION)
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES (
    'viewer',
    'viewer@stablerisk.local',
    '$2a$12$9q3UR1RnWd/YyLr1Z4ZygOHyrMd6WoF0FsUsRw7gW9dWmKwQyHtHu',  -- viewer123456
    'viewer',
    true
) ON CONFLICT (username) DO NOTHING;

-- Log the migration
INSERT INTO audit_logs (action, resource, details, signature, user_id)
VALUES (
    'migration',
    'database',
    '{"migration": "001_initial_schema", "description": "Initial database schema creation"}',
    encode(digest('001_initial_schema', 'sha256'), 'hex'),
    'system'
);
