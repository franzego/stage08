-- Rollback transactions table
DROP INDEX IF EXISTS idx_transactions_user_created;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_reference;
DROP INDEX IF EXISTS idx_transactions_wallet_id;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS transaction_type;
