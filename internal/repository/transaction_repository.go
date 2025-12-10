package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/franzego/stage08/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TransactionRepository struct {
	db *sqlx.DB
}

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create creates a new transaction
func (r *TransactionRepository) Create(tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (user_id, wallet_id, type, amount, status, reference, description, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	// Handle nil metadata - pass NULL to database
	var metadata interface{}
	if tx.Metadata != nil && len(tx.Metadata) > 0 {
		metadata = tx.Metadata
	} else {
		metadata = nil
	}

	err := r.db.QueryRowx(query,
		tx.UserID,
		tx.WalletID,
		tx.Type,
		tx.Amount,
		tx.Status,
		tx.Reference,
		tx.Description,
		metadata,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// FindByReference finds a transaction by reference
func (r *TransactionRepository) FindByReference(reference string) (*models.Transaction, error) {
	var tx models.Transaction
	query := `SELECT * FROM transactions WHERE reference = $1`

	err := r.db.Get(&tx, query, reference)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	return &tx, nil
}

// UpdateStatus updates transaction status
func (r *TransactionRepository) UpdateStatus(id uuid.UUID, status models.TransactionStatus) error {
	query := `UPDATE transactions SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}
	return nil
}

// ListByUser lists all transactions for a user
func (r *TransactionRepository) ListByUser(userID uuid.UUID, limit, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := `
		SELECT * FROM transactions 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`

	err := r.db.Select(&transactions, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return transactions, nil
}

// Helper to create metadata JSON
func CreateMetadata(data map[string]interface{}) ([]byte, error) {
	return json.Marshal(data)
}
