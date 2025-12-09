package repository

import (
	"database/sql"
	"fmt"

	"github.com/franzego/stage08/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByGoogleID finds a user by their Google ID
func (r *UserRepository) FindByGoogleID(googleID string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE google_id = $1`

	err := r.db.Get(&user, query, googleID)
	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.Get(&user, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.Get(&user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

// Create creates a new user and their wallet
func (r *UserRepository) Create(googleID, email, name string, picture *string) (*models.User, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create user
	user := &models.User{
		GoogleID: googleID,
		Email:    email,
		Name:     name,
		Picture:  picture,
	}

	query := `
		INSERT INTO users (google_id, email, name, picture)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRowx(query, googleID, email, name, picture).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create wallet for the user
	walletQuery := `
		INSERT INTO wallets (user_id, wallet_number)
		VALUES ($1, generate_wallet_number())
	`

	if _, err := tx.Exec(walletQuery, user.ID); err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return user, nil
}
