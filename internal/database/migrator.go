package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migrator struct {
	db *sql.DB
}

func NewMigrator(dbURL string) (*Migrator, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Migrator{db: db}, nil
}

func (m *Migrator) Run() error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		migrationName := entry.Name()
		
		// Check if migration already applied
		applied, err := m.isMigrationApplied(migrationName)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if applied {
			log.Printf("Migration %s already applied, skipping", migrationName)
			continue
		}

		// Read and execute migration
		migrationSQL, err := migrationsFS.ReadFile("migrations/" + migrationName)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migrationName, err)
		}

		log.Printf("Applying migration: %s", migrationName)
		
		// Execute migration in a transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.Exec(string(migrationSQL)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
		}

		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (name, applied_at) VALUES ($1, NOW())",
			migrationName,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migrationName, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migrationName, err)
		}

		log.Printf("Successfully applied migration: %s", migrationName)
	}

	return nil
}

func (m *Migrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) isMigrationApplied(name string) (bool, error) {
	var count int
	err := m.db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE name = $1",
		name,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *Migrator) Close() error {
	return m.db.Close()
}

