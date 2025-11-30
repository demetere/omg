package omg

import (
	"context"
	"sort"
)

// Migration represents a single OpenFGA migration with Up and Down functions
type Migration struct {
	Version string
	Name    string
	Up      func(ctx context.Context, client *Client) error
	Down    func(ctx context.Context, client *Client) error
}

var migrations []Migration

// Register adds a migration to the global registry
// This should be called in init() functions of migration files
func Register(m Migration) {
	migrations = append(migrations, m)
}

// GetAll returns all registered migrations sorted by version (timestamp)
func GetAll() []Migration {
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})
	return sorted
}

// Reset clears all registered migrations (used for testing)
func Reset() {
	migrations = []Migration{}
}
