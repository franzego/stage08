package database

import (
	"fmt"
	"log"
	"os"

	"github.com/franzego/stage08/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Connect establishes a connection to PostgreSQL using sqlx
func Connect(cfg *config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := cfg.GetDSN()

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Database connection established")
	return db, nil
}

// RunMigrations executes SQL migration files
func RunMigrations(db *sqlx.DB) error {
	migrations := []string{
		"migrations/001_create_users_table.up.sql",
		"migrations/002_create_wallets_table.up.sql",
		"migrations/003_create_transactions_table.up.sql",
		"migrations/004_create_api_keys_table.up.sql",
	}

	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration)
		content, err := readMigrationFile(migration)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		if _, err := db.Exec(content); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
	}

	log.Println("✅ All migrations completed successfully")
	return nil
}

func readMigrationFile(path string) (string, error) {
	// use golang migrate
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
