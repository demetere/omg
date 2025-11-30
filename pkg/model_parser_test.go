package omg_test

import (
	"testing"

	"github.com/demetere/omg/pkg"
	openfgaSdk "github.com/openfga/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDSLToModel_BasicTypes(t *testing.T) {
	dsl := `
type user
type document
type folder
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Equal(t, "1.1", model.SchemaVersion)
	assert.Len(t, model.TypeDefinitions, 3)

	// Check types are present
	typeNames := make(map[string]bool)
	for _, typeDef := range model.TypeDefinitions {
		typeNames[typeDef.Type] = true
	}

	assert.True(t, typeNames["user"])
	assert.True(t, typeNames["document"])
	assert.True(t, typeNames["folder"])
}

func TestParseDSLToModel_SchemaVersion(t *testing.T) {
	dsl := `
schema 1.2

type user
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Equal(t, "1.2", model.SchemaVersion)
	assert.Len(t, model.TypeDefinitions, 1)
}

func TestParseDSLToModel_DirectRelations(t *testing.T) {
	dsl := `
type user

type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user]
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	assert.Len(t, relations, 3)
	assert.Contains(t, relations, "owner")
	assert.Contains(t, relations, "editor")
	assert.Contains(t, relations, "viewer")

	// Check that relations are direct (have "this")
	assert.NotNil(t, relations["owner"].This)
	assert.NotNil(t, relations["editor"].This)
}

func TestParseDSLToModel_MultipleTypeRestrictions(t *testing.T) {
	dsl := `
type user
type group

type document
  relations
    define viewer: [user, group#member]
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	assert.Contains(t, relations, "viewer")
	// Should have direct relation
	assert.NotNil(t, relations["viewer"].This)
}

func TestParseDSLToModel_ComputedRelations(t *testing.T) {
	dsl := `
type user

type document
  relations
    define owner: [user]
    define viewer: owner
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// owner should be direct
	assert.NotNil(t, relations["owner"].This)

	// viewer should be computed from owner
	assert.NotNil(t, relations["viewer"].ComputedUserset)
	assert.Equal(t, "owner", *relations["viewer"].ComputedUserset.Relation)
}

func TestParseDSLToModel_UnionRelations(t *testing.T) {
	dsl := `
type user

type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: owner or editor
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// viewer should be a union
	assert.NotNil(t, relations["viewer"].Union)
	assert.Len(t, relations["viewer"].Union.Child, 2)
}

func TestParseDSLToModel_IntersectionRelations(t *testing.T) {
	dsl := `
type user

type document
  relations
    define allowed: [user]
    define approved: [user]
    define viewer: allowed and approved
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// viewer should be an intersection
	assert.NotNil(t, relations["viewer"].Intersection)
	assert.Len(t, relations["viewer"].Intersection.Child, 2)
}

func TestParseDSLToModel_DifferenceRelations(t *testing.T) {
	dsl := `
type user

type document
  relations
    define allowed: [user]
    define blocked: [user]
    define viewer: allowed but not blocked
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// viewer should be a difference
	assert.NotNil(t, relations["viewer"].Difference)
	assert.NotNil(t, relations["viewer"].Difference.Base)
	assert.NotNil(t, relations["viewer"].Difference.Subtract)
}

func TestParseDSLToModel_TupleToUserset(t *testing.T) {
	dsl := `
type user

type folder
  relations
    define owner: [user]

type document
  relations
    define parent: [folder]
    define viewer: parent->owner
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// viewer should be tuple-to-userset
	assert.NotNil(t, relations["viewer"].TupleToUserset)
	assert.Equal(t, "parent", *relations["viewer"].TupleToUserset.Tupleset.Relation)
	assert.Equal(t, "owner", *relations["viewer"].TupleToUserset.ComputedUserset.Relation)
}

func TestParseDSLToModel_TupleToUsersetFromSyntax(t *testing.T) {
	dsl := `
type user

type team
  relations
    define member: [user]

type resource
  relations
    define team: [team]
    define owner: member from team
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find resource type
	var resourceType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "resource" {
			resourceType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, resourceType)
	relations := resourceType.GetRelations()

	// owner should be tuple-to-userset
	// "owner: member from team" means team->member
	assert.NotNil(t, relations["owner"].TupleToUserset)
	assert.Equal(t, "team", *relations["owner"].TupleToUserset.Tupleset.Relation)
	assert.Equal(t, "member", *relations["owner"].TupleToUserset.ComputedUserset.Relation)
}

func TestParseDSLToModel_ComplexRealWorld(t *testing.T) {
	dsl := `
schema 1.1

type user

type organization
  relations
    define member: [user]
    define admin: [user]
    define owner: [user]

type folder
  relations
    define parent: [organization]
    define owner: [user] or parent->owner
    define editor: [user] or owner
    define viewer: [user] or editor

type document
  relations
    define parent: [folder]
    define owner: [user] or parent->owner
    define editor: [user] or owner
    define viewer: [user] or editor
    define can_delete: owner
    define can_edit: editor
    define can_view: viewer
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Equal(t, "1.1", model.SchemaVersion)
	assert.Len(t, model.TypeDefinitions, 4)

	// Verify all types are present
	typeNames := make(map[string]bool)
	for _, typeDef := range model.TypeDefinitions {
		typeNames[typeDef.Type] = true
	}

	assert.True(t, typeNames["user"])
	assert.True(t, typeNames["organization"])
	assert.True(t, typeNames["folder"])
	assert.True(t, typeNames["document"])

	// Check document type has all relations
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	assert.Len(t, relations, 7)
	assert.Contains(t, relations, "parent")
	assert.Contains(t, relations, "owner")
	assert.Contains(t, relations, "editor")
	assert.Contains(t, relations, "viewer")
	assert.Contains(t, relations, "can_delete")
	assert.Contains(t, relations, "can_edit")
	assert.Contains(t, relations, "can_view")
}

func TestParseDSLToModel_EmptyModel(t *testing.T) {
	dsl := ``
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Equal(t, "1.1", model.SchemaVersion)
	assert.Len(t, model.TypeDefinitions, 0)
}

func TestParseDSLToModel_CommentsAndWhitespace(t *testing.T) {
	dsl := `
# This is a comment
schema 1.1

# User type
type user

# Document type with relations
type document
  relations
    # Owner relation
    define owner: [user]
    define viewer: owner
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Equal(t, "1.1", model.SchemaVersion)
	assert.Len(t, model.TypeDefinitions, 2)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()
	assert.Len(t, relations, 2)
}

// Note: The simplified parser doesn't support inline comments after definitions
func TestParseDSLToModel_InlineCommentsNotSupported(t *testing.T) {
	dsl := `
type user
type document
  relations
    define owner: [user]  # inline comment causes parse error
`
	_, err := omg.ParseDSLToModel(dsl)
	// This is expected to fail with the simplified parser
	assert.Error(t, err)
}

func TestParseDSLToModel_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		dsl         string
		expectError bool
		errorMsg    string
	}{
		{
			name: "relation without type",
			dsl: `
relations
  define owner: [user]
`,
			expectError: true,
			errorMsg:    "relation defined outside of type",
		},
		{
			name: "invalid relation definition",
			dsl: `
type document
  relations
    define owner
`,
			expectError: true,
			errorMsg:    "invalid relation definition",
		},
		{
			name: "invalid but not syntax",
			dsl: `
type user
type document
  relations
    define viewer: [user] but not blocked but not admin
`,
			expectError: true,
			errorMsg:    "invalid 'but not' syntax",
		},
		{
			name: "invalid tuple-to-userset syntax",
			dsl: `
type document
  relations
    define viewer: parent->owner->admin
`,
			expectError: true,
			errorMsg:    "invalid tuple-to-userset format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := omg.ParseDSLToModel(tt.dsl)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseDSLToModel_TypesWithNoRelations(t *testing.T) {
	dsl := `
type user

type group
  relations
    define member: [user]

type admin
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	assert.Len(t, model.TypeDefinitions, 3)

	// Find types
	types := make(map[string]*openfgaSdk.TypeDefinition)
	for i, typeDef := range model.TypeDefinitions {
		types[typeDef.Type] = &model.TypeDefinitions[i]
	}

	// user and admin should have no relations
	assert.Len(t, types["user"].GetRelations(), 0)
	assert.Len(t, types["admin"].GetRelations(), 0)

	// group should have 1 relation
	assert.Len(t, types["group"].GetRelations(), 1)
}

func TestParseDSLToModel_ComplexUnionsAndIntersections(t *testing.T) {
	dsl := `
type user

type document
  relations
    define owner: [user]
    define editor: [user]
    define approved: [user]
    define can_view: owner or editor
    define can_edit: owner and approved
`
	model, err := omg.ParseDSLToModel(dsl)
	require.NoError(t, err)

	// Find document type
	var docType *openfgaSdk.TypeDefinition
	for i, typeDef := range model.TypeDefinitions {
		if typeDef.Type == "document" {
			docType = &model.TypeDefinitions[i]
			break
		}
	}

	require.NotNil(t, docType)
	relations := docType.GetRelations()

	// Verify basic union/intersection patterns work
	assert.Contains(t, relations, "can_view")
	assert.Contains(t, relations, "can_edit")
	assert.NotNil(t, relations["can_view"].Union)
	assert.NotNil(t, relations["can_edit"].Intersection)
}

// Note: The simplified parser doesn't support parentheses for grouping
func TestParseDSLToModel_ParenthesesNotSupported(t *testing.T) {
	dsl := `
type user
type document
  relations
    define owner: [user]
    define editor: [user]
    define can_view: (owner or editor)
`
	_, err := omg.ParseDSLToModel(dsl)
	// This is expected to fail with the simplified parser
	assert.Error(t, err)
}
