-- Create transactions table
-- Records all wallet activities
CREATE TYPE transaction_type AS ENUM ('deposit', 'transfer_in', 'transfer_out');
CREATE TYPE transaction_status AS ENUM ('pending', 'success', 'failed');

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    type transaction_type NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status transaction_status NOT NULL DEFAULT 'pending',
    reference VARCHAR(255) UNIQUE, -- Paystack reference or transfer ID
    description TEXT,
    metadata JSONB, -- Store additional info (recipient, sender, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for queries
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_reference ON transactions(reference);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);

-- Composite index for user transaction history
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at DESC);
