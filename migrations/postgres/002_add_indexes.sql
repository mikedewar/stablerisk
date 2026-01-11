-- Additional indexes and optimizations for StableRisk

-- Composite indexes for common query patterns

-- Users - find active users by role
CREATE INDEX IF NOT EXISTS idx_users_role_active ON users(role, is_active) WHERE is_active = true;

-- API Keys - find active keys for a user
CREATE INDEX IF NOT EXISTS idx_api_keys_user_active ON api_keys(user_id, is_active) WHERE is_active = true;

-- Audit logs - time-based queries with filtering
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp_action ON audit_logs(timestamp DESC, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC);

-- Outliers - common filtering combinations
CREATE INDEX IF NOT EXISTS idx_outliers_type_severity ON outliers(type, severity);
CREATE INDEX IF NOT EXISTS idx_outliers_detected_type ON outliers(detected_at DESC, type);
CREATE INDEX IF NOT EXISTS idx_outliers_address_detected ON outliers(address, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_outliers_unacknowledged ON outliers(detected_at DESC) WHERE acknowledged = false;

-- Refresh tokens - cleanup expired tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_cleanup ON refresh_tokens(expires_at, revoked) WHERE revoked = false;

-- Partial indexes for performance

-- Active API keys only
CREATE INDEX IF NOT EXISTS idx_api_keys_active_only ON api_keys(user_id, created_at DESC) WHERE is_active = true;

-- Non-expired refresh tokens
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_valid ON refresh_tokens(user_id, expires_at DESC) WHERE revoked = false AND expires_at > NOW();

-- High severity outliers
CREATE INDEX IF NOT EXISTS idx_outliers_high_severity ON outliers(detected_at DESC) WHERE severity IN ('high', 'critical');

-- Create materialized view for outlier statistics (optional, for dashboard performance)
CREATE MATERIALIZED VIEW IF NOT EXISTS outlier_stats AS
SELECT
    date_trunc('hour', detected_at) as time_bucket,
    type,
    severity,
    COUNT(*) as count,
    AVG(amount) as avg_amount,
    MAX(amount) as max_amount,
    MIN(amount) as min_amount
FROM outliers
GROUP BY date_trunc('hour', detected_at), type, severity
ORDER BY time_bucket DESC;

-- Index on materialized view
CREATE INDEX IF NOT EXISTS idx_outlier_stats_time ON outlier_stats(time_bucket DESC);
CREATE INDEX IF NOT EXISTS idx_outlier_stats_type ON outlier_stats(type);

-- Function to refresh outlier stats (call periodically)
CREATE OR REPLACE FUNCTION refresh_outlier_stats()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY outlier_stats;
END;
$$ LANGUAGE plpgsql;

-- Log the migration
INSERT INTO audit_logs (action, resource, details, signature, user_id)
VALUES (
    'migration',
    'database',
    '{"migration": "002_add_indexes", "description": "Additional indexes and materialized views for performance"}',
    encode(digest('002_add_indexes', 'sha256'), 'hex'),
    'system'
);
