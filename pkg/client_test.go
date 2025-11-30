package omg_test

import (
	"context"
	"testing"

	"github.com/demetere/omg/internal/testhelpers"
	"github.com/demetere/omg/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_WriteTuple(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
type folder
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`)
	defer container.Terminate(ctx)

	tuple := omg.Tuple{
		User:     "user:alice",
		Relation: "owner",
		Object:   "document:readme",
	}

	err := client.WriteTuple(ctx, tuple)
	require.NoError(t, err)

	// Verify tuple was written
	tuples, err := client.ReadAllTuples(ctx, omg.ReadTuplesRequest{
		User: "user:alice",
	})
	require.NoError(t, err)
	assert.Len(t, tuples, 1)
	assert.Equal(t, tuple.User, tuples[0].User)
	assert.Equal(t, tuple.Relation, tuples[0].Relation)
	assert.Equal(t, tuple.Object, tuples[0].Object)
}

func TestClient_WriteTuples(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
type folder
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`)
	defer container.Terminate(ctx)

	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "document:readme"},
		{User: "user:bob", Relation: "viewer", Object: "document:readme"},
		{User: "user:charlie", Relation: "editor", Object: "document:readme"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Verify tuples were written
	allTuples, err := client.ReadAllTuples(ctx, omg.ReadTuplesRequest{
		Object: "document:readme",
	})
	require.NoError(t, err)
	assert.Len(t, allTuples, 3)
}

func TestClient_DeleteTuple(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
type folder
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`)
	defer container.Terminate(ctx)

	tuple := omg.Tuple{
		User:     "user:alice",
		Relation: "owner",
		Object:   "document:readme",
	}

	// Write tuple
	err := client.WriteTuple(ctx, tuple)
	require.NoError(t, err)

	// Delete tuple
	err = client.DeleteTuple(ctx, tuple)
	require.NoError(t, err)

	// Verify tuple was deleted
	tuples, err := client.ReadAllTuples(ctx, omg.ReadTuplesRequest{
		User: "user:alice",
	})
	require.NoError(t, err)
	assert.Len(t, tuples, 0)
}

func TestClient_DeleteTuples(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
type folder
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`)
	defer container.Terminate(ctx)

	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "document:readme"},
		{User: "user:bob", Relation: "viewer", Object: "document:readme"},
	}

	// Write tuples
	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	// Delete tuples
	err = client.DeleteTuples(ctx, tuples)
	require.NoError(t, err)

	// Verify tuples were deleted
	allTuples, err := client.ReadAllTuples(ctx, omg.ReadTuplesRequest{
		Object: "document:readme",
	})
	require.NoError(t, err)
	assert.Len(t, allTuples, 0)
}

func TestClient_ReadAllTuples(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
type folder
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`)
	defer container.Terminate(ctx)

	// Setup test data
	tuples := []omg.Tuple{
		{User: "user:alice", Relation: "owner", Object: "document:doc1"},
		{User: "user:bob", Relation: "viewer", Object: "document:doc1"},
		{User: "user:alice", Relation: "owner", Object: "document:doc2"},
		{User: "user:charlie", Relation: "editor", Object: "folder:folder1"},
	}

	err := client.WriteTuples(ctx, tuples)
	require.NoError(t, err)

	tests := []struct {
		name     string
		request  omg.ReadTuplesRequest
		expected int
	}{
		{
			name:     "read all tuples",
			request:  omg.ReadTuplesRequest{},
			expected: 4,
		},
		{
			name:     "filter by user",
			request:  omg.ReadTuplesRequest{User: "user:alice"},
			expected: 2,
		},
		{
			name:     "filter by object",
			request:  omg.ReadTuplesRequest{Object: "document:doc1"},
			expected: 2,
		},
		{
			name:     "filter by relation",
			request:  omg.ReadTuplesRequest{Relation: "owner"},
			expected: 2,
		},
		{
			name:     "filter by object type",
			request:  omg.ReadTuplesRequest{Object: "document:"},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ReadAllTuples(ctx, tt.request)
			require.NoError(t, err)
			assert.Len(t, result, tt.expected)
		})
	}
}

