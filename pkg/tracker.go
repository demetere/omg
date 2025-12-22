package omg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Tracker manages migration state using a database table
// Migrations are tracked in: omg_migrations table
type Tracker struct {
	db *sql.DB
}

// NewTracker creates a new migration tracker with database connection
func NewTracker(db *sql.DB) (*Tracker, error) {
	tracker := &Tracker{db: db}

	// Ensure the migrations table exists
	if err := tracker.ensureTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	return tracker, nil
}

// MigrationInfo contains metadata about an applied migration
type MigrationInfo struct {
	Version   string
	Name      string
	AppliedAt time.Time
}

// ensureTable creates the migrations table if it doesn't exist
func (t *Tracker) ensureTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS omg_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := t.db.ExecContext(ctx, query)
	return err
}

// GetApplied returns all applied migrations
func (t *Tracker) GetApplied(ctx context.Context) (map[string]MigrationInfo, error) {
	query := `SELECT version, name, applied_at FROM omg_migrations ORDER BY version`

	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]MigrationInfo)
	for rows.Next() {
		var info MigrationInfo
		if err := rows.Scan(&info.Version, &info.Name, &info.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[info.Version] = info
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migrations: %w", err)
	}

	return applied, nil
}

// Record marks a migration as applied
func (t *Tracker) Record(ctx context.Context, version, name string) error {
	query := `INSERT INTO omg_migrations (version, name, applied_at) VALUES ($1, $2, $3)`

	_, err := t.db.ExecContext(ctx, query, version, name, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// Remove removes a migration record (used for rollback)
func (t *Tracker) Remove(ctx context.Context, version string) error {
	query := `DELETE FROM omg_migrations WHERE version = $1`

	_, err := t.db.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration: %w", err)
	}

	return nil
}

// Close closes the database connection
func (t *Tracker) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}
