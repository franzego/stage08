-- Rollback wallets table
DROP FUNCTION IF EXISTS generate_wallet_number();
DROP INDEX IF EXISTS idx_wallets_wallet_number;
DROP INDEX IF EXISTS idx_wallets_user_id;
DROP TABLE IF EXISTS wallets;
