package repository

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/franzego/stage08/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create generates and stores a new API key
func (r *APIKeyRepository) Create(userID uuid.UUID, name string, permissions []string, expiresAt time.Time) (*models.APIKey, string, error) {
	// Generate raw API key
	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	keyHash := hashAPIKey(rawKey)
	keyPrefix := rawKey[:12] // Store prefix for identification (e.g., sk_live_xxx)

	// Create API key record
	apiKey := &models.APIKey{
		UserID:      userID,
		Name:        name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Permissions: permissions,
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}

	query := `
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowx(query,
		apiKey.UserID,
		apiKey.Name,
		apiKey.KeyHash,
		apiKey.KeyPrefix,
		pq.Array(apiKey.Permissions),
		apiKey.IsActive,
		apiKey.ExpiresAt,
	).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, rawKey, nil
}

// FindByKey finds an API key by its raw key value
func (r *APIKeyRepository) FindByKey(rawKey string) (*models.APIKey, error) {
	keyHash := hashAPIKey(rawKey)

	var apiKey models.APIKey
	query := `SELECT * FROM api_keys WHERE key_hash = $1`

	err := r.db.Get(&apiKey, query, keyHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find API key: %w", err)
	}

	return &apiKey, nil
}

// CountActiveByUser counts active API keys for a user
func (r *APIKeyRepository) CountActiveByUser(userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND is_active = true`

	err := r.db.Get(&count, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	return count, nil
}

// ListByUser lists all API keys for a user
func (r *APIKeyRepository) ListByUser(userID uuid.UUID) ([]models.APIKey, error) {
	var keys []models.APIKey
	query := `SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`

	err := r.db.Select(&keys, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return keys, nil
}

// FindByID finds an API key by ID
func (r *APIKeyRepository) FindByID(id uuid.UUID) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := `SELECT * FROM api_keys WHERE id = $1`

	err := r.db.Get(&apiKey, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find API key: %w", err)
	}

	return &apiKey, nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *APIKeyRepository) UpdateLastUsed(id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// Revoke deactivates an API key
func (r *APIKeyRepository) Revoke(id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// generateAPIKey generates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_live_" + base64.URLEncoding.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA256 hash of the API key
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.StdEncoding.EncodeToString(hash[:])
}
