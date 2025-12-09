-- Create wallets table
-- Each user has ONE wallet
-- Balance stored in KOBO (smallest unit - multiply by 100 from Naira)
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_number VARCHAR(20) UNIQUE NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for fast lookups
CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_wallets_wallet_number ON wallets(wallet_number);

-- Function to generate unique wallet number (13 digits)
CREATE OR REPLACE FUNCTION generate_wallet_number() RETURNS VARCHAR(20) AS $$
DECLARE
    new_number VARCHAR(20);
    done BOOL;
BEGIN
    done := false;
    WHILE NOT done LOOP
        -- Generate 13-digit number
        new_number := LPAD(FLOOR(RANDOM() * 10000000000000)::TEXT, 13, '0');
        -- Check if it exists
        done := NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_number = new_number);
    END LOOP;
    RETURN new_number;
END;
$$ LANGUAGE plpgsql;
