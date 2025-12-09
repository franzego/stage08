-- Rollback api_keys table
DROP TRIGGER IF EXISTS enforce_max_active_keys ON api_keys;
DROP FUNCTION IF EXISTS check_max_active_keys();
DROP INDEX IF EXISTS idx_api_keys_expires_at;
DROP INDEX IF EXISTS idx_api_keys_user_active;
DROP INDEX IF EXISTS idx_api_keys_key_hash;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS api_keys;
