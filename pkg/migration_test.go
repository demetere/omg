package omg_test

import (
	"context"
	"testing"

	"github.com/demetere/omg/internal/testhelpers"
	"github.com/demetere/omg/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_RegisterAndGetAll(t *testing.T) {
	// Reset registry before test
	omg.Reset()

	migrations := []omg.Migration{
		{
			Version: "20240101000000",
			Name:    "first_migration",
			Up:      func(ctx context.Context, client *omg.Client) error { return nil },
			Down:    func(ctx context.Context, client *omg.Client) error { return nil },
		},
		{
			Version: "20240102000000",
			Name:    "second_migration",
			Up:      func(ctx context.Context, client *omg.Client) error { return nil },
			Down:    func(ctx context.Context, client *omg.Client) error { return nil },
		},
		{
			Version: "20240103000000",
			Name:    "third_migration",
			Up:      func(ctx context.Context, client *omg.Client) error { return nil },
			Down:    func(ctx context.Context, client *omg.Client) error { return nil },
		},
	}

	// Register migrations out of order
	omg.Register(migrations[1])
	omg.Register(migrations[0])
	omg.Register(migrations[2])

	// Get all should return sorted
	all := omg.GetAll()

	require.Len(t, all, 3)
	assert.Equal(t, "20240101000000", all[0].Version)
	assert.Equal(t, "20240102000000", all[1].Version)
	assert.Equal(t, "20240103000000", all[2].Version)
}

func TestTracker_RecordAndGetApplied(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type migration
  relations
    define applied: [user]
`)
	defer container.Terminate(ctx)

	tracker := omg.NewTracker(client)

	// Initially no migrations applied
	applied, err := tracker.GetApplied(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 0)

	// Record a migration
	err = tracker.Record(ctx, "20240101000000", "test_migration")
	require.NoError(t, err)

	// Check it was recorded
	applied, err = tracker.GetApplied(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.Contains(t, applied, "20240101000000")

	// Record another migration
	err = tracker.Record(ctx, "20240102000000", "second_migration")
	require.NoError(t, err)

	applied, err = tracker.GetApplied(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 2)
}

func TestTracker_Remove(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type migration
  relations
    define applied: [user]
`)
	defer container.Terminate(ctx)

	tracker := omg.NewTracker(client)

	// Record migrations
	err := tracker.Record(ctx, "20240101000000", "first")
	require.NoError(t, err)
	err = tracker.Record(ctx, "20240102000000", "second")
	require.NoError(t, err)

	// Verify both recorded
	applied, err := tracker.GetApplied(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 2)

	// Remove one migration
	err = tracker.Remove(ctx, "20240102000000")
	require.NoError(t, err)

	// Verify only one remains
	applied, err = tracker.GetApplied(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.Contains(t, applied, "20240101000000")
	assert.NotContains(t, applied, "20240102000000")
}

