package omg_test

import (
	"os"
	"strings"
	"testing"

	"github.com/demetere/omg/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMigrationFromChanges_NoChanges(t *testing.T) {
	changes := []omg.ModelChange{}

	_, err := omg.GenerateMigrationFromChanges(changes, "test", "migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no changes detected")
}

func TestGenerateMigrationFromChanges_AddType(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "add_type",
			TypeName: "document",
			Details:  "New type 'document' with 0 relations",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "add_document", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	// Read generated file
	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify basic structure
	assert.Contains(t, code, "package main")
	assert.Contains(t, code, "func main()")
	assert.Contains(t, code, "add_document")

	// Verify up function
	assert.Contains(t, code, "func up(")
	assert.Contains(t, code, "Add type: document")
	assert.Contains(t, code, "AddTypeToModel")

	// Verify down function
	assert.Contains(t, code, "func down(")
}

func TestGenerateMigrationFromChanges_AddRelation(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:         "add_relation",
			TypeName:     "document",
			RelationName: "owner",
			NewValue:     `{"this":{}}`,
			Details:      "Added relation 'document.owner'",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "add_owner_relation", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify relation addition code
	assert.Contains(t, code, "Add relation: document.owner")
	assert.Contains(t, code, "AddRelationToType")
	assert.Contains(t, code, `"document"`)
	assert.Contains(t, code, `"owner"`)
}

func TestGenerateMigrationFromChanges_RemoveType(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "remove_type",
			TypeName: "deprecated",
			Details:  "Type 'deprecated' removed",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "remove_deprecated", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify remove type code
	assert.Contains(t, code, "Remove type: deprecated")
	assert.Contains(t, code, "ReadAllTuples")
	assert.Contains(t, code, "DeleteTuplesBatch")
	assert.Contains(t, code, "RemoveTypeFromModel")
}

func TestGenerateMigrationFromChanges_RenameType_HighConfidence(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:       "rename_type",
			TypeName:   "team",
			OldValue:   "team",
			NewValue:   "teams",
			Confidence: "high",
			Details:    "Rename detected: 'team' -> 'teams' (high confidence: 80% name, 100% relations)",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "rename_team", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify high confidence rename code
	assert.Contains(t, code, "Rename type: team -> teams")
	assert.Contains(t, code, "high confidence")
	assert.Contains(t, code, "RenameType")
	assert.Contains(t, code, `"team"`)
	assert.Contains(t, code, `"teams"`)

	// Should NOT have manual review warnings
	assert.NotContains(t, code, "⚠️  MANUAL REVIEW REQUIRED")
	assert.NotContains(t, code, "OPTION 1")
}

func TestGenerateMigrationFromChanges_RenameType_MediumConfidence(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:       "rename_type",
			TypeName:   "team",
			OldValue:   "team",
			NewValue:   "organization",
			Confidence: "medium",
			Details:    "Possible rename: 'team' -> 'organization' (medium confidence - review required)",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "rename_team", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify medium confidence rename code
	assert.Contains(t, code, "⚠️  REVIEW REQUIRED")
	assert.Contains(t, code, "team -> organization")
	assert.Contains(t, code, "similarity analysis")
	assert.Contains(t, code, "RenameType")

	// Should have review warning but not multiple options
	assert.NotContains(t, code, "OPTION 1")
	assert.NotContains(t, code, "OPTION 2")
}

func TestGenerateMigrationFromChanges_RenameType_LowConfidence(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:       "rename_type",
			TypeName:   "grp",
			OldValue:   "grp",
			NewValue:   "team",
			Confidence: "low",
			Details:    "Potential rename: 'grp' -> 'team' (low confidence - verify before using)",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "rename_grp", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify low confidence rename code
	assert.Contains(t, code, "⚠️  MANUAL REVIEW REQUIRED")
	assert.Contains(t, code, "low confidence")
	assert.Contains(t, code, "OPTION 1: If this IS a rename")
	assert.Contains(t, code, "OPTION 2: If these are separate types")

	// Rename code should be commented out
	assert.Contains(t, code, "// if err := omg.RenameType")

	// Default safe option should be active (delete old type)
	assert.Contains(t, code, "tuples, err := omg.ReadAllTuples")
	assert.Contains(t, code, "DeleteTuplesBatch")
}

func TestGenerateMigrationFromChanges_RenameRelation_HighConfidence(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:         "rename_relation",
			TypeName:     "document",
			RelationName: "can_view",
			OldValue:     "can_view",
			NewValue:     "viewer",
			Confidence:   "high",
			Details:      "Rename detected: 'document.can_view' -> 'document.viewer' (high confidence: 70%)",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "rename_relation", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify high confidence relation rename
	assert.Contains(t, code, "Rename relation: document.can_view -> document.viewer")
	assert.Contains(t, code, "high confidence")
	assert.Contains(t, code, "RenameRelation")
}

func TestGenerateMigrationFromChanges_UpdateRelation(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:         "update_relation",
			TypeName:     "document",
			RelationName: "viewer",
			OldValue:     `{"this":{}}`,
			NewValue:     `{"computedUserset":{"relation":"owner"}}`,
			Details:      "Updated relation 'document.viewer' definition",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "update_viewer", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify update relation code
	assert.Contains(t, code, "Update relation: document.viewer")
	assert.Contains(t, code, "UpdateRelationDefinition")
}

func TestGenerateMigrationFromChanges_RemoveRelation(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:         "remove_relation",
			TypeName:     "document",
			RelationName: "deprecated",
			Details:      "Removed relation 'document.deprecated'",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "remove_deprecated_rel", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify remove relation code
	assert.Contains(t, code, "Remove relation: document.deprecated")
	assert.Contains(t, code, "RemoveRelationFromType")
	assert.Contains(t, code, "DeleteRelation")
}

func TestGenerateMigrationFromChanges_MultipleChanges(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "add_type",
			TypeName: "folder",
			Details:  "New type 'folder' with 2 relations",
		},
		{
			Type:         "add_relation",
			TypeName:     "document",
			RelationName: "parent",
			Details:      "Added relation 'document.parent'",
		},
		{
			Type:         "remove_relation",
			TypeName:     "document",
			RelationName: "deprecated",
			Details:      "Removed relation 'document.deprecated'",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "multiple_changes", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify all changes are included
	assert.Contains(t, code, "Add type: folder")
	assert.Contains(t, code, "Add relation: document.parent")
	assert.Contains(t, code, "Remove relation: document.deprecated")

	// Verify comments list all changes
	assert.Contains(t, code, "// - New type 'folder' with 2 relations")
	assert.Contains(t, code, "// - Added relation 'document.parent'")
	assert.Contains(t, code, "// - Removed relation 'document.deprecated'")
}

func TestGenerateMigrationFromChanges_OperationOrdering(t *testing.T) {
	changes := []omg.ModelChange{
		{Type: "remove_type", TypeName: "old", Details: "Remove old"},
		{Type: "add_type", TypeName: "new", Details: "Add new"},
		{Type: "remove_relation", TypeName: "doc", RelationName: "old_rel", Details: "Remove old rel"},
		{Type: "add_relation", TypeName: "doc", RelationName: "new_rel", Details: "Add new rel"},
		{Type: "update_relation", TypeName: "doc", RelationName: "viewer", Details: "Update viewer"},
		{Type: "rename_type", OldValue: "team", NewValue: "teams", Confidence: "high", Details: "Rename team"},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "ordered", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Find positions of each operation in the up function
	upStart := strings.Index(code, "func up(")
	downStart := strings.Index(code, "func down(")

	upCode := code[upStart:downStart]

	addTypePos := strings.Index(upCode, "Add type: new")
	addRelPos := strings.Index(upCode, "Add relation: doc.new_rel")
	updateRelPos := strings.Index(upCode, "Update relation: doc.viewer")
	renameTypePos := strings.Index(upCode, "Rename type: team -> teams")
	removeRelPos := strings.Index(upCode, "Remove relation: doc.old_rel")
	removeTypePos := strings.Index(upCode, "Remove type: old")

	// Verify ordering: add types, add relations, update relations, renames, removes
	assert.True(t, addTypePos < addRelPos, "Add type should come before add relation")
	assert.True(t, addRelPos < updateRelPos, "Add relation should come before update relation")
	assert.True(t, updateRelPos < renameTypePos, "Update relation should come before rename type")
	assert.True(t, renameTypePos < removeRelPos, "Rename type should come before remove relation")
	assert.True(t, removeRelPos < removeTypePos, "Remove relation should come before remove type")
}

func TestGenerateMigrationFromChanges_DownMigration(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "add_type",
			TypeName: "folder",
			Details:  "New type 'folder'",
		},
		{
			Type:         "add_relation",
			TypeName:     "document",
			RelationName: "parent",
			Details:      "Added relation 'document.parent'",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "test_down", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Extract down function
	downStart := strings.Index(code, "func down(")
	downCode := code[downStart:]

	// Verify down migration reverses operations
	// Add type -> Remove type in down
	assert.Contains(t, downCode, "Remove type: folder")

	// Add relation -> Remove relation in down
	assert.Contains(t, downCode, "Remove relation: document.parent")
}

func TestGenerateMigrationFromChanges_DownMigration_RenameReversal(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:       "rename_type",
			TypeName:   "team",
			OldValue:   "team",
			NewValue:   "teams",
			Confidence: "high",
			Details:    "Rename team -> teams",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "test_rename_down", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Extract down function
	downStart := strings.Index(code, "func down(")
	downCode := code[downStart:]

	// Verify down migration reverses rename (teams -> team)
	assert.Contains(t, downCode, "RenameType")
	assert.Contains(t, downCode, `"teams"`)
	assert.Contains(t, downCode, `"team"`)
}

func TestGenerateMigrationFromChanges_ValidGoSyntax(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "add_type",
			TypeName: "document",
			Details:  "Add document type",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "syntax_test", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Basic syntax checks for new standalone format
	assert.Contains(t, code, "package main")
	assert.Contains(t, code, "import (")
	assert.Contains(t, code, "func main()")
	assert.Contains(t, code, "func up(")
	assert.Contains(t, code, "func down(")

	// Check for balanced braces (basic check)
	openBraces := strings.Count(code, "{")
	closeBraces := strings.Count(code, "}")
	assert.Equal(t, openBraces, closeBraces, "Braces should be balanced")

	// Check return statements
	assert.Contains(t, code, "return nil")
	assert.Contains(t, code, "return fmt.Errorf")
}

func TestGenerateMigrationFromChanges_NameSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "spaces to underscores",
			input:    "add user type",
			expected: "add_user_type",
		},
		{
			name:     "hyphens to underscores",
			input:    "add-user-type",
			expected: "add_user_type",
		},
		{
			name:     "mixed case to lowercase",
			input:    "AddUserType",
			expected: "addusertype",
		},
		{
			name:     "special characters removed",
			input:    "add@user#type!",
			expected: "addusertype",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := []omg.ModelChange{
				{Type: "add_type", TypeName: "test", Details: "test"},
			}

			filename, err := omg.GenerateMigrationFromChanges(changes, tt.input, "migrations")
			require.NoError(t, err)
			defer os.Remove(filename)

			// Check filename contains sanitized name
			assert.Contains(t, filename, tt.expected)

			// Check file content uses sanitized name
			content, err := os.ReadFile(filename)
			require.NoError(t, err)
			assert.Contains(t, string(content), tt.expected)
		})
	}
}

func TestGenerateMigrationFromChanges_ErrorHandlingInGenerated(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:     "add_type",
			TypeName: "document",
			Details:  "Add document",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "error_test", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify error handling is present
	assert.Contains(t, code, "if err :=")
	assert.Contains(t, code, "return fmt.Errorf")
	assert.Contains(t, code, "%w")

	// Verify each operation has error handling
	opCount := strings.Count(code, "if err := omg.")
	errReturnCount := strings.Count(code, "return fmt.Errorf")

	assert.Greater(t, opCount, 0, "Should have at least one operation")
	assert.Greater(t, errReturnCount, 0, "Should have error returns")
}

func TestGenerateMigrationFromChanges_FileCreation(t *testing.T) {
	changes := []omg.ModelChange{
		{Type: "add_type", TypeName: "test", Details: "test"},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "file_test", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	// Verify file exists
	_, err = os.Stat(filename)
	assert.NoError(t, err, "File should exist")

	// Verify file is in migrations directory
	assert.Contains(t, filename, "migrations/")

	// Verify filename format (timestamp_name.go)
	assert.Contains(t, filename, "_file_test.go")
	assert.Regexp(t, `migrations/\d{14}_file_test\.go`, filename)
}

func TestGenerateMigrationFromChanges_CommentsInGenerated(t *testing.T) {
	changes := []omg.ModelChange{
		{
			Type:       "rename_type",
			OldValue:   "team",
			NewValue:   "organization",
			Confidence: "medium",
			Details:    "Possible rename: team -> organization",
		},
	}

	filename, err := omg.GenerateMigrationFromChanges(changes, "comments_test", "migrations")
	require.NoError(t, err)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	code := string(content)

	// Verify helpful comments are generated
	assert.Contains(t, code, "// Auto-generated migration")
	assert.Contains(t, code, "// Changes detected:")
	assert.Contains(t, code, "// Rollback operations")

	// For medium confidence rename, should have review comments
	assert.Contains(t, code, "// ⚠️  REVIEW REQUIRED")
	assert.Contains(t, code, "// This appears to be a rename")
}
