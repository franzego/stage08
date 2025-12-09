-- Create api_keys table
-- For service-to-service wallet access
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL, -- e.g., "wallet-service"
    key_hash VARCHAR(255) UNIQUE NOT NULL, -- SHA256 hash of the actual key
    key_prefix VARCHAR(20) NOT NULL, -- First few chars for identification (sk_live_xxx)
    permissions TEXT[] NOT NULL, -- Array: ['deposit', 'transfer', 'read']
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure max 5 active keys per user
    CONSTRAINT check_permissions CHECK (
        array_length(permissions, 1) > 0 AND
        permissions <@ ARRAY['deposit', 'transfer', 'read']::TEXT[]
    )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_active ON api_keys(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);

-- Function to enforce max 5 active keys per user
CREATE OR REPLACE FUNCTION check_max_active_keys() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_active = true THEN
        IF (SELECT COUNT(*) FROM api_keys 
            WHERE user_id = NEW.user_id 
            AND is_active = true 
            AND id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::UUID)) >= 5 THEN
            RAISE EXCEPTION 'Maximum 5 active API keys allowed per user';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_max_active_keys ON api_keys;
CREATE TRIGGER enforce_max_active_keys
    BEFORE INSERT OR UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION check_max_active_keys();
