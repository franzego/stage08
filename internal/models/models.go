package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	GoogleID  string    `db:"google_id" json:"google_id"`
	Email     string    `db:"email" json:"email"`
	Name      string    `db:"name" json:"name"`
	Picture   *string   `db:"picture" json:"picture,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Wallet represents a user's wallet
type Wallet struct {
	ID           uuid.UUID `db:"id" json:"id"`
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	WalletNumber string    `db:"wallet_number" json:"wallet_number"`
	Balance      int64     `db:"balance" json:"balance"` // in kobo
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// Transaction types
type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "deposit"
	TransactionTypeTransferIn  TransactionType = "transfer_in"
	TransactionTypeTransferOut TransactionType = "transfer_out"
)

// Transaction statuses
type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

// Transaction represents a wallet transaction
type Transaction struct {
	ID          uuid.UUID         `db:"id" json:"id"`
	UserID      uuid.UUID         `db:"user_id" json:"user_id"`
	WalletID    uuid.UUID         `db:"wallet_id" json:"wallet_id"`
	Type        TransactionType   `db:"type" json:"type"`
	Amount      int64             `db:"amount" json:"amount"`
	Status      TransactionStatus `db:"status" json:"status"`
	Reference   *string           `db:"reference" json:"reference,omitempty"`
	Description *string           `db:"description" json:"description,omitempty"`
	Metadata    []byte            `db:"metadata" json:"metadata,omitempty"` // JSONB
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}

// APIKey represents an API key for service-to-service access
type APIKey struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	Name        string     `db:"name" json:"name"`
	KeyHash     string     `db:"key_hash" json:"-"` // Never expose hash
	KeyPrefix   string     `db:"key_prefix" json:"key_prefix"`
	Permissions []string   `db:"permissions" json:"permissions"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	ExpiresAt   time.Time  `db:"expires_at" json:"expires_at"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// IsExpired checks if the API key has expired
func (a *APIKey) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

// HasPermission checks if the API key has a specific permission
func (a *APIKey) HasPermission(permission string) bool {
	for _, p := range a.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
