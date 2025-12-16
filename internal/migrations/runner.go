// Package migrations handles database schema migrations
package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner handles database migrations
type Runner struct {
	pool *pgxpool.Pool
	fs   embed.FS
}

// NewRunner creates a new migration runner
func NewRunner(pool *pgxpool.Pool, files embed.FS) *Runner {
	return &Runner{
		pool: pool,
		fs:   files,
	}
}

// Run executes all pending migrations
func (r *Runner) Run(ctx context.Context) error {
	// Ensure schema_migrations table exists
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get list of migration files
	files, err := r.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Run pending migrations
	for _, file := range files {
		if applied[file] {
			log.Printf("[MIGRATE] Skipping %s (already applied)", file)
			continue
		}

		log.Printf("[MIGRATE] Running %s...", file)
		if err := r.runMigration(ctx, file); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", file, err)
		}
		log.Printf("[MIGRATE] Completed %s", file)
	}

	return nil
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func (r *Runner) ensureMigrationsTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	return err
}

// getAppliedMigrations returns a map of already-applied migration filenames
func (r *Runner) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := r.pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, nil
}

// getMigrationFiles returns sorted list of .sql files
func (r *Runner) getMigrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(r.fs, ".")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".sql") && !entry.IsDir() {
			files = append(files, name)
		}
	}

	// Sort by filename (assumes NNN_ prefix for ordering)
	sort.Strings(files)
	return files, nil
}

// runMigration executes a single migration file
func (r *Runner) runMigration(ctx context.Context, filename string) error {
	// Read migration file
	content, err := fs.ReadFile(r.fs, filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	sql := string(content)
	if strings.TrimSpace(sql) == "" {
		log.Printf("[MIGRATE] Skipping empty migration %s", filename)
		// Still record it as applied
		_, err = r.pool.Exec(ctx,
			"INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)",
			filename, time.Now())
		return err
	}

	// Execute migration in a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute the SQL
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	// Record the migration
	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)",
		filename, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

// Status returns the current migration status
func (r *Runner) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}

	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	files, err := r.getMigrationFiles()
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, file := range files {
		statuses = append(statuses, MigrationStatus{
			Version: file,
			Applied: applied[file],
		})
	}

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version string `json:"version"`
	Applied bool   `json:"applied"`
}
