package omg

import (
	"context"
	"fmt"
	"time"
)

// Tracker manages migration state using OpenFGA tuples
// Migrations are tracked as tuples: system:migration-tracker applied migration:<version>
type Tracker struct {
	client *Client
}

// NewTracker creates a new migration tracker
func NewTracker(client *Client) *Tracker {
	return &Tracker{client: client}
}

// MigrationInfo contains metadata about an applied migration
type MigrationInfo struct {
	Version   string
	AppliedAt time.Time
}

// GetApplied returns all applied migrations
func (t *Tracker) GetApplied(ctx context.Context) (map[string]MigrationInfo, error) {
	tuples, err := t.client.ReadAllTuples(ctx, ReadTuplesRequest{
		User:   "system:migration-tracker",
		Object: "migration:",
	})
	if err != nil {
		// If no tuples found, return empty map (not an error)
		return make(map[string]MigrationInfo), nil
	}

	applied := make(map[string]MigrationInfo)
	for _, tuple := range tuples {
		// Extract version from "migration:<version>"
		if len(tuple.Object) > 10 && tuple.Object[:10] == "migration:" {
			version := tuple.Object[10:]
			applied[version] = MigrationInfo{
				Version:   version,
				AppliedAt: time.Now(), // Note: OpenFGA doesn't store timestamps
			}
		}
	}

	return applied, nil
}

// Record marks a migration as applied
func (t *Tracker) Record(ctx context.Context, version, name string) error {
	tuple := Tuple{
		User:     "system:migration-tracker",
		Relation: "applied",
		Object:   fmt.Sprintf("migration:%s", version),
	}

	return t.client.WriteTuple(ctx, tuple)
}

// Remove removes a migration record (used for rollback)
func (t *Tracker) Remove(ctx context.Context, version string) error {
	tuple := Tuple{
		User:     "system:migration-tracker",
		Relation: "applied",
		Object:   fmt.Sprintf("migration:%s", version),
	}

	return t.client.DeleteTuple(ctx, tuple)
}
