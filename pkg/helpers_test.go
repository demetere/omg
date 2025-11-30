package omg_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/demetere/omg/internal/testhelpers"
	"github.com/demetere/omg/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameRelation(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "can_manage_members", Object: "team:engineering"},
		{User: "user:bob", Relation: "can_manage_members", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Rename relation
	err = omg.RenameRelation(ctx, client, "team", "can_manage_members", "can_manage")
	require.NoError(t, err)

	// Verify old relation has no tuples
	oldTuples, err := omg.ReadAllTuples(ctx, client, "team", "can_manage_members")
	require.NoError(t, err)
	assert.Len(t, oldTuples, 0)

	// Verify new relation has the tuples
	newTuples, err := omg.ReadAllTuples(ctx, client, "team", "can_manage")
	require.NoError(t, err)
	assert.Len(t, newTuples, 2)

	// Verify tuple content
	for _, tuple := range newTuples {
		assert.Equal(t, "can_manage", tuple.Relation)
	}
}

func TestRenameType(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "team:engineering"},
		{User: "user:bob", Relation: "member", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Rename type
	err = omg.RenameType(ctx, client, "team", "organization")
	require.NoError(t, err)

	// Verify old type has no tuples
	oldTuples, err := omg.ReadAllTuples(ctx, client, "team", "")
	require.NoError(t, err)
	assert.Len(t, oldTuples, 0)

	// Verify new type has the tuples
	newTuples, err := omg.ReadAllTuples(ctx, client, "organization", "")
	require.NoError(t, err)
	assert.Len(t, newTuples, 2)

	// Verify tuple content
	for _, tuple := range newTuples {
		assert.Contains(t, tuple.Object, "organization:")
		assert.NotContains(t, tuple.Object, "team:")
	}
}

func TestCopyRelation(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "admin", Object: "team:engineering"},
		{User: "user:bob", Relation: "admin", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Copy relation
	err = omg.CopyRelation(ctx, client, "team", "admin", "manager")
	require.NoError(t, err)

	// Verify old relation still exists
	oldTuples, err := omg.ReadAllTuples(ctx, client, "team", "admin")
	require.NoError(t, err)
	assert.Len(t, oldTuples, 2)

	// Verify new relation was created
	newTuples, err := omg.ReadAllTuples(ctx, client, "team", "manager")
	require.NoError(t, err)
	assert.Len(t, newTuples, 2)

	// Total tuples should be 4 (2 original + 2 copied)
	allTuples, err := omg.ReadAllTuples(ctx, client, "team", "")
	require.NoError(t, err)
	assert.Len(t, allTuples, 4)
}

func TestDeleteRelation(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "deprecated", Object: "team:engineering"},
		{User: "user:bob", Relation: "deprecated", Object: "team:sales"},
		{User: "user:charlie", Relation: "owner", Object: "team:engineering"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Delete relation
	err = omg.DeleteRelation(ctx, client, "team", "deprecated")
	require.NoError(t, err)

	// Verify deprecated relation has no tuples
	deletedTuples, err := omg.ReadAllTuples(ctx, client, "team", "deprecated")
	require.NoError(t, err)
	assert.Len(t, deletedTuples, 0)

	// Verify other relations still exist
	ownerTuples, err := omg.ReadAllTuples(ctx, client, "team", "owner")
	require.NoError(t, err)
	assert.Len(t, ownerTuples, 1)
}

func TestMigrateRelationWithTransform(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "member", Object: "team:engineering"},
		{User: "user:bob", Relation: "member", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Transform function: change team to organization in object
	transform := func(tuple omg.Tuple) (omg.Tuple, error) {
		tuple.Object = "organization:" + tuple.Object[5:] // Replace "team:" with "organization:"
		return tuple, nil
	}

	// Migrate with transform
	err = omg.MigrateRelationWithTransform(ctx, client, "team", "member", "employee", transform)
	require.NoError(t, err)

	// Verify old relation has no tuples
	oldTuples, err := omg.ReadAllTuples(ctx, client, "team", "member")
	require.NoError(t, err)
	assert.Len(t, oldTuples, 0)

	// Verify new relation has transformed tuples
	newTuples, err := omg.ReadAllTuples(ctx, client, "organization", "employee")
	require.NoError(t, err)
	assert.Len(t, newTuples, 2)

	for _, tuple := range newTuples {
		assert.Contains(t, tuple.Object, "organization:")
		assert.Equal(t, "employee", tuple.Relation)
	}
}

func TestCountTuples(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "team:engineering"},
		{User: "user:bob", Relation: "member", Object: "team:engineering"},
		{User: "user:charlie", Relation: "member", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Count all team tuples
	count, err := omg.CountTuples(ctx, client, "team", "")
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Count member tuples
	count, err = omg.CountTuples(ctx, client, "team", "member")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Count owner tuples
	count, err = omg.CountTuples(ctx, client, "team", "owner")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestBackupAndRestoreTuples(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create test tuples
	originalTuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "team:engineering"},
		{User: "user:bob", Relation: "member", Object: "team:sales"},
	}

	err := client.WriteTuples(ctx, originalTuples)
	require.NoError(t, err)

	// Backup tuples
	backup, err := omg.BackupTuples(ctx, client)
	require.NoError(t, err)
	assert.Len(t, backup, 2)

	// Delete all tuples
	err = client.DeleteTuples(ctx, originalTuples)
	require.NoError(t, err)

	// Verify tuples are gone
	tuples, err := omg.ReadAllTuples(ctx, client, "", "")
	require.NoError(t, err)
	assert.Len(t, tuples, 0)

	// Restore from backup
	err = omg.RestoreTuples(ctx, client, backup)
	require.NoError(t, err)

	// Verify tuples are restored
	restoredTuples, err := omg.ReadAllTuples(ctx, client, "", "")
	require.NoError(t, err)
	assert.Len(t, restoredTuples, 2)
}

func TestWriteAndDeleteTuplesBatch(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type team
type organization
`)
	defer container.Terminate(ctx)

	// Create a large number of tuples (more than batch size)
	var tuples []omg.Tuple
	for i := 0; i < 250; i++ {
		tuples = append(tuples, omg.Tuple{
			User:     "user:alice",
			Relation: "member",
			Object:   fmt.Sprintf("team:team%d", i),
		})
	}

	// Write in batches
	err := omg.WriteTuplesBatch(ctx, client, tuples)
	require.NoError(t, err)

	// Verify all tuples were written
	allTuples, err := omg.ReadAllTuples(ctx, client, "team", "")
	require.NoError(t, err)
	assert.Len(t, allTuples, 250)

	// Delete in batches
	err = omg.DeleteTuplesBatch(ctx, client, tuples)
	require.NoError(t, err)

	// Verify all tuples were deleted
	remainingTuples, err := omg.ReadAllTuples(ctx, client, "team", "")
	require.NoError(t, err)
	assert.Len(t, remainingTuples, 0)
}

