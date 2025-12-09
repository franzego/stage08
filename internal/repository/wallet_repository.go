package repository

import (
	"database/sql"
	"fmt"

	"github.com/franzego/stage08/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type WalletRepository struct {
	db *sqlx.DB
}

func NewWalletRepository(db *sqlx.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// FindByUserID finds a wallet by user ID
func (r *WalletRepository) FindByUserID(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	query := `SELECT * FROM wallets WHERE user_id = $1`

	err := r.db.Get(&wallet, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find wallet: %w", err)
	}

	return &wallet, nil
}

// FindByWalletNumber finds a wallet by wallet number
func (r *WalletRepository) FindByWalletNumber(walletNumber string) (*models.Wallet, error) {
	var wallet models.Wallet
	query := `SELECT * FROM wallets WHERE wallet_number = $1`

	err := r.db.Get(&wallet, query, walletNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find wallet: %w", err)
	}

	return &wallet, nil
}

// UpdateBalance updates wallet balance (use with caution - prefer transactions)
func (r *WalletRepository) UpdateBalance(walletID uuid.UUID, newBalance int64) error {
	query := `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, newBalance, walletID)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	return nil
}

// Credit adds money to a wallet (atomic operation)
func (r *WalletRepository) Credit(walletID uuid.UUID, amount int64) error {
	query := `
		UPDATE wallets 
		SET balance = balance + $1, updated_at = NOW() 
		WHERE id = $2
	`
	result, err := r.db.Exec(query, amount, walletID)
	if err != nil {
		return fmt.Errorf("failed to credit wallet: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("wallet not found")
	}

	return nil
}

// Debit removes money from a wallet (atomic operation with balance check)
func (r *WalletRepository) Debit(walletID uuid.UUID, amount int64) error {
	query := `
		UPDATE wallets 
		SET balance = balance - $1, updated_at = NOW() 
		WHERE id = $2 AND balance >= $1
	`
	result, err := r.db.Exec(query, amount, walletID)
	if err != nil {
		return fmt.Errorf("failed to debit wallet: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient balance or wallet not found")
	}

	return nil
}
