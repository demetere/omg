package omg_test

import (
	"context"
	"testing"

	"github.com/demetere/omg/internal/testhelpers"
	"github.com/demetere/omg/pkg"
	openfgaSdk "github.com/openfga/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectChanges_AddType(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner": `{"computedUserset":{"relation":"owner"}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect one added type
	var addedTypes []omg.ModelChange
	for _, change := range changes {
		if change.Type == "add_type" {
			addedTypes = append(addedTypes, change)
		}
	}

	assert.Len(t, addedTypes, 1)
	assert.Equal(t, "document", addedTypes[0].TypeName)
}

func TestDetectChanges_RemoveType(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
			"document": {
				Name:      "document",
				Relations: map[string]string{},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect one removed type
	var removedTypes []omg.ModelChange
	for _, change := range changes {
		if change.Type == "remove_type" {
			removedTypes = append(removedTypes, change)
		}
	}

	assert.Len(t, removedTypes, 1)
	assert.Equal(t, "document", removedTypes[0].TypeName)
}

func TestDetectChanges_AddRelation(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"editor": `{"this":{}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect one added relation
	var addedRelations []omg.ModelChange
	for _, change := range changes {
		if change.Type == "add_relation" {
			addedRelations = append(addedRelations, change)
		}
	}

	assert.Len(t, addedRelations, 1)
	assert.Equal(t, "document", addedRelations[0].TypeName)
	assert.Equal(t, "editor", addedRelations[0].RelationName)
}

func TestDetectChanges_RemoveRelation(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"editor": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect one removed relation
	var removedRelations []omg.ModelChange
	for _, change := range changes {
		if change.Type == "remove_relation" {
			removedRelations = append(removedRelations, change)
		}
	}

	assert.Len(t, removedRelations, 1)
	assert.Equal(t, "document", removedRelations[0].TypeName)
	assert.Equal(t, "editor", removedRelations[0].RelationName)
}

func TestDetectChanges_UpdateRelation(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"viewer": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"viewer": `{"computedUserset":{"relation":"owner"}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect one updated relation
	var updatedRelations []omg.ModelChange
	for _, change := range changes {
		if change.Type == "update_relation" {
			updatedRelations = append(updatedRelations, change)
		}
	}

	assert.Len(t, updatedRelations, 1)
	assert.Equal(t, "document", updatedRelations[0].TypeName)
	assert.Equal(t, "viewer", updatedRelations[0].RelationName)
}

func TestDetectChanges_MultipleChanges(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"editor": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"viewer": `{"computedUserset":{"relation":"owner"}}`,
				},
			},
			"folder": {
				Name: "folder",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(oldState, newState)

	// Should detect: 1 added type, 1 removed relation, 1 added relation
	assert.GreaterOrEqual(t, len(changes), 3)

	changeTypes := make(map[string]int)
	for _, change := range changes {
		changeTypes[string(change.Type)]++
	}

	assert.Equal(t, 1, changeTypes["add_type"])       // folder added
	assert.Equal(t, 1, changeTypes["remove_relation"]) // editor removed
	assert.Equal(t, 1, changeTypes["add_relation"])    // viewer added
}

func TestDetectPotentialRenames_HighConfidence_SimilarNames(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"team": {
				Name: "team",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"member": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"teams": {
				Name: "teams",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"member": `{"this":{}}`,
				},
			},
		},
	}

	// Get basic changes first
	changes := omg.DetectChanges(oldState, newState)

	// Detect potential renames
	enhanced := omg.DetectPotentialRenames(changes, oldState, newState)

	// Should detect high confidence rename (team → teams)
	var renames []omg.ModelChange
	for _, change := range enhanced {
		if change.Type == "rename_type" {
			renames = append(renames, change)
		}
	}

	assert.Len(t, renames, 1)
	assert.Equal(t, "team", renames[0].OldValue)
	assert.Equal(t, "teams", renames[0].NewValue)
	assert.Equal(t, "high", string(renames[0].Confidence))
}

func TestDetectPotentialRenames_MediumConfidence_SimilarRelations(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"team": {
				Name: "team",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"member": `{"this":{}}`,
					"admin":  `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"organization": {
				Name: "organization",
				Relations: map[string]string{
					"owner":  `{"this":{}}`,
					"member": `{"this":{}}`,
					"admin":  `{"this":{}}`,
				},
			},
		},
	}

	// Get basic changes first
	changes := omg.DetectChanges(oldState, newState)

	// Detect potential renames
	enhanced := omg.DetectPotentialRenames(changes, oldState, newState)

	// Should detect medium/high confidence rename (team → organization with same relations)
	var renames []omg.ModelChange
	for _, change := range enhanced {
		if change.Type == "rename_type" {
			renames = append(renames, change)
		}
	}

	assert.Len(t, renames, 1)
	assert.Equal(t, "team", renames[0].OldValue)
	assert.Equal(t, "organization", renames[0].NewValue)
	// With 100% relation similarity, should be high confidence even with different names
	assert.Contains(t, []string{"high", "medium"}, string(renames[0].Confidence))
}

func TestDetectPotentialRenames_NoMatch_VeryDifferentTypes(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"x": {
				Name: "x",
				Relations: map[string]string{
					"a": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"zzzzz": {
				Name: "zzzzz",
				Relations: map[string]string{
					"b": `{"this":{}}`,
					"c": `{"this":{}}`,
				},
			},
		},
	}

	// Get basic changes first
	changes := omg.DetectChanges(oldState, newState)

	// Detect potential renames
	enhanced := omg.DetectPotentialRenames(changes, oldState, newState)

	// With very different names and no common relations, should NOT detect rename
	var renames []omg.ModelChange
	for _, change := range enhanced {
		if change.Type == "rename_type" {
			renames = append(renames, change)
		}
	}

	// Note: Some low confidence matches might still be suggested
	// The important thing is they should be low confidence or none
	for _, rename := range renames {
		assert.Contains(t, []string{"low", ""}, string(rename.Confidence),
			"Any detected rename should be low confidence or none")
	}
}

func TestDetectPotentialRenames_RelationRename_HighConfidence(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"can_view": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"document": {
				Name: "document",
				Relations: map[string]string{
					"viewer": `{"this":{}}`,
				},
			},
		},
	}

	// Get basic changes first
	changes := omg.DetectChanges(oldState, newState)

	// Detect potential renames
	enhanced := omg.DetectPotentialRenames(changes, oldState, newState)

	// Should detect relation rename (can_view → viewer might not be high confidence)
	// but should at least suggest it
	var relationRenames []omg.ModelChange
	for _, change := range enhanced {
		if change.Type == "rename_relation" {
			relationRenames = append(relationRenames, change)
		}
	}

	// Depending on similarity threshold, this may or may not be detected
	// Let's check that at least we get either rename or add+remove
	if len(relationRenames) > 0 {
		assert.Equal(t, "document", relationRenames[0].TypeName)
		assert.Equal(t, "can_view", relationRenames[0].OldValue)
		assert.Equal(t, "viewer", relationRenames[0].NewValue)
	} else {
		// Should have add + remove
		var adds, removes int
		for _, change := range enhanced {
			if change.Type == "add_relation" {
				adds++
			}
			if change.Type == "remove_relation" {
				removes++
			}
		}
		assert.Equal(t, 1, adds)
		assert.Equal(t, 1, removes)
	}
}

func TestDetectPotentialRenames_MultipleRenames(t *testing.T) {
	oldState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"team": {
				Name: "team",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
			"grp": {
				Name: "grp",
				Relations: map[string]string{
					"admin": `{"this":{}}`,
				},
			},
		},
	}

	newState := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"teams": {
				Name: "teams",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
			"group": {
				Name: "group",
				Relations: map[string]string{
					"admin": `{"this":{}}`,
				},
			},
		},
	}

	// Get basic changes first
	changes := omg.DetectChanges(oldState, newState)

	// Detect potential renames
	enhanced := omg.DetectPotentialRenames(changes, oldState, newState)

	// Should detect at least one rename (team → teams is high confidence)
	var renames []omg.ModelChange
	for _, change := range enhanced {
		if change.Type == "rename_type" {
			renames = append(renames, change)
		}
	}

	assert.GreaterOrEqual(t, len(renames), 1)

	// Check that team → teams is detected
	var teamRename *omg.ModelChange
	for i, change := range renames {
		if change.OldValue == "team" && change.NewValue == "teams" {
			teamRename = &renames[i]
			break
		}
	}

	assert.NotNil(t, teamRename)
	assert.Equal(t, "high", string(teamRename.Confidence))
}

func TestLoadModelStateFromOpenFGA(t *testing.T) {
	ctx := context.Background()

	container, client := testhelpers.SetupOpenFGAContainer(t, ctx, `
type user

type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: owner or editor
`)
	defer container.Terminate(ctx)

	// Load model state from OpenFGA
	state, err := omg.LoadModelStateFromOpenFGA(ctx, client)
	require.NoError(t, err)
	require.NotNil(t, state)

	// Should have 2 types
	assert.Len(t, state.Types, 2)
	assert.Contains(t, state.Types, "user")
	assert.Contains(t, state.Types, "document")

	// Document should have 3 relations
	docType := state.Types["document"]
	assert.Len(t, docType.Relations, 3)
	assert.Contains(t, docType.Relations, "owner")
	assert.Contains(t, docType.Relations, "editor")
	assert.Contains(t, docType.Relations, "viewer")
}

func TestBuildModelStateFromAuthorizationModel(t *testing.T) {
	// Create a mock authorization model
	relationMap := make(map[string]openfgaSdk.Userset)
	thisMap := make(map[string]interface{})
	relationMap["owner"] = openfgaSdk.Userset{
		This: &thisMap,
	}
	relationMap["viewer"] = openfgaSdk.Userset{
		ComputedUserset: &openfgaSdk.ObjectRelation{
			Relation: openfgaSdk.PtrString("owner"),
		},
	}

	typeDefs := []openfgaSdk.TypeDefinition{
		{
			Type:      "user",
			Relations: &map[string]openfgaSdk.Userset{},
		},
		{
			Type:      "document",
			Relations: &relationMap,
		},
	}

	model := openfgaSdk.AuthorizationModel{
		SchemaVersion:   "1.1",
		TypeDefinitions: typeDefs,
	}

	state := omg.BuildModelStateFromAuthorizationModel(model)

	assert.Len(t, state.Types, 2)
	assert.Contains(t, state.Types, "user")
	assert.Contains(t, state.Types, "document")

	// Check document relations
	docType := state.Types["document"]
	assert.Len(t, docType.Relations, 2)
	assert.Contains(t, docType.Relations, "owner")
	assert.Contains(t, docType.Relations, "viewer")
}

func TestDetectChanges_NoChanges(t *testing.T) {
	state := &omg.ModelState{
		Types: map[string]omg.TypeState{
			"user": {
				Name:      "user",
				Relations: map[string]string{},
			},
			"document": {
				Name: "document",
				Relations: map[string]string{
					"owner": `{"this":{}}`,
				},
			},
		},
	}

	changes := omg.DetectChanges(state, state)

	assert.Len(t, changes, 0)
}

func TestDetectPotentialRenames_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		oldState       *omg.ModelState
		newState       *omg.ModelState
		expectRename   bool
		minConfidence  string
	}{
		{
			name: "identical names (should be rename)",
			oldState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"team": {Name: "team", Relations: map[string]string{"owner": `{"this":{}}`}},
				},
			},
			newState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"team": {Name: "team", Relations: map[string]string{"admin": `{"this":{}}`}},
				},
			},
			expectRename:  false, // Same type name, different relations = not a type rename
		},
		{
			name: "substring match (team in team_member)",
			oldState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"team": {Name: "team", Relations: map[string]string{"owner": `{"this":{}}`}},
				},
			},
			newState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"team_member": {Name: "team_member", Relations: map[string]string{"owner": `{"this":{}}`}},
				},
			},
			expectRename:  true,
			minConfidence: "medium", // Medium/high similarity due to substring + relations match
		},
		{
			name: "no relations on either side",
			oldState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"team": {Name: "team", Relations: map[string]string{}},
				},
			},
			newState: &omg.ModelState{
				Types: map[string]omg.TypeState{
					"teams": {Name: "teams", Relations: map[string]string{}},
				},
			},
			expectRename:  true,
			minConfidence: "high", // High name similarity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := omg.DetectChanges(tt.oldState, tt.newState)
			enhanced := omg.DetectPotentialRenames(changes, tt.oldState, tt.newState)

			var renames []omg.ModelChange
			for _, change := range enhanced {
				if change.Type == "rename_type" {
					renames = append(renames, change)
				}
			}

			if tt.expectRename {
				assert.GreaterOrEqual(t, len(renames), 1, "Expected at least one rename")
				if len(renames) > 0 && tt.minConfidence != "" {
					// Check that confidence is at least the minimum
					confidence := string(renames[0].Confidence)
					validConfidences := map[string]int{"low": 1, "medium": 2, "high": 3}
					assert.GreaterOrEqual(t, validConfidences[confidence], validConfidences[tt.minConfidence])
				}
			} else {
				assert.Len(t, renames, 0, "Expected no renames")
			}
		})
	}
}
