package omg

import (
	"context"
	"fmt"
	"strings"

	openfgaSdk "github.com/openfga/go-sdk"
)

const batchSize = 100

// TransformFunc is a function that transforms a tuple
type TransformFunc func(tuple Tuple) (Tuple, error)

// MODEL OPERATIONS

// GetCurrentModel retrieves the current authorization model as DSL string
func GetCurrentModel(ctx context.Context, client *Client) (string, error) {
	return client.GetCurrentModel(ctx)
}

// ApplyModelFromDSL parses and applies an authorization model from DSL string
// Example DSL:
//
//	model
//	  schema 1.1
//
//	type user
//
//	type file
//	  relations
//	    define owner: [user]
//	    define can_modify: [user] or owner
func ApplyModelFromDSL(ctx context.Context, client *Client, dsl string) error {
	fmt.Println("Applying authorization model from DSL...")
	model, err := parseDSLToModel(dsl)
	if err != nil {
		return fmt.Errorf("failed to parse DSL: %w", err)
	}

	if err := client.WriteAuthorizationModel(ctx, model); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Println("Model applied successfully")
	return nil
}

// ApplyModelFromFile reads a DSL file and applies the authorization model
func ApplyModelFromFile(ctx context.Context, client *Client, filePath string) error {
	fmt.Printf("Loading model from file: %s\n", filePath)

	content, err := readFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return ApplyModelFromDSL(ctx, client, string(content))
}

// CompareModels compares two models and returns a description of differences
func CompareModels(ctx context.Context, client *Client, newDSL string) (string, error) {
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current model: %w", err)
	}

	// Parse both models
	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return "", fmt.Errorf("failed to parse current model: %w", err)
	}

	newModel, err := parseDSLToModel(newDSL)
	if err != nil {
		return "", fmt.Errorf("failed to parse new model: %w", err)
	}

	// Generate diff
	diff := generateModelDiff(currentModel, newModel)
	return diff, nil
}

// TUPLE OPERATIONS

// RenameRelation renames a relation on all tuples of a specific object type
// Example: RenameRelation(ctx, client, "team", "can_manage_members", "can_manage")
func RenameRelation(ctx context.Context, client *Client, objectType, oldRelation, newRelation string) error {
	fmt.Printf("Renaming relation %s -> %s on type %s\n", oldRelation, newRelation, objectType)

	// Read all tuples with old relation
	tuples, err := ReadAllTuples(ctx, client, objectType, oldRelation)
	if err != nil {
		return fmt.Errorf("failed to read tuples: %w", err)
	}

	if len(tuples) == 0 {
		fmt.Println("No tuples found to rename")
		return nil
	}

	fmt.Printf("Found %d tuples to rename\n", len(tuples))

	// Create new tuples with new relation
	var newTuples []Tuple
	for _, t := range tuples {
		newTuples = append(newTuples, Tuple{
			User:     t.User,
			Relation: newRelation,
			Object:   t.Object,
		})
	}

	// Write new tuples
	if err := WriteTuplesBatch(ctx, client, newTuples); err != nil {
		return fmt.Errorf("failed to write new tuples: %w", err)
	}

	// Delete old tuples
	if err := DeleteTuplesBatch(ctx, client, tuples); err != nil {
		return fmt.Errorf("failed to delete old tuples: %w", err)
	}

	fmt.Println("Relation rename completed")
	return nil
}

// RenameType renames an object type on all tuples
// Example: RenameType(ctx, client, "team", "organization")
func RenameType(ctx context.Context, client *Client, oldType, newType string) error {
	fmt.Printf("Renaming type %s -> %s\n", oldType, newType)

	// Read all tuples for old type
	tuples, err := ReadAllTuples(ctx, client, oldType, "")
	if err != nil {
		return fmt.Errorf("failed to read tuples: %w", err)
	}

	if len(tuples) == 0 {
		fmt.Println("No tuples found to rename")
		return nil
	}

	fmt.Printf("Found %d tuples to rename\n", len(tuples))

	// Create new tuples with new type
	var newTuples []Tuple
	for _, t := range tuples {
		// Replace type in object: "team:123" -> "organization:123"
		newObject := strings.Replace(t.Object, oldType+":", newType+":", 1)
		newTuples = append(newTuples, Tuple{
			User:     t.User,
			Relation: t.Relation,
			Object:   newObject,
		})
	}

	// Write new tuples
	if err := WriteTuplesBatch(ctx, client, newTuples); err != nil {
		return fmt.Errorf("failed to write new tuples: %w", err)
	}

	// Delete old tuples
	if err := DeleteTuplesBatch(ctx, client, tuples); err != nil {
		return fmt.Errorf("failed to delete old tuples: %w", err)
	}

	fmt.Println("Type rename completed")
	return nil
}

// CopyRelation copies tuples from one relation to another
// Example: CopyRelation(ctx, client, "team", "can_manage_members", "can_manage")
func CopyRelation(ctx context.Context, client *Client, objectType, sourceRelation, targetRelation string) error {
	fmt.Printf("Copying relation %s -> %s on type %s\n", sourceRelation, targetRelation, objectType)

	// Read all tuples with source relation
	tuples, err := ReadAllTuples(ctx, client, objectType, sourceRelation)
	if err != nil {
		return fmt.Errorf("failed to read tuples: %w", err)
	}

	if len(tuples) == 0 {
		fmt.Println("No tuples found to copy")
		return nil
	}

	fmt.Printf("Found %d tuples to copy\n", len(tuples))

	// Create new tuples with target relation
	var newTuples []Tuple
	for _, t := range tuples {
		newTuples = append(newTuples, Tuple{
			User:     t.User,
			Relation: targetRelation,
			Object:   t.Object,
		})
	}

	// Write new tuples
	if err := WriteTuplesBatch(ctx, client, newTuples); err != nil {
		return fmt.Errorf("failed to write new tuples: %w", err)
	}

	fmt.Println("Relation copy completed")
	return nil
}

// DeleteRelation deletes all tuples with a specific relation
// Example: DeleteRelation(ctx, client, "team", "can_manage_members")
func DeleteRelation(ctx context.Context, client *Client, objectType, relation string) error {
	fmt.Printf("Deleting relation %s on type %s\n", relation, objectType)

	// Read all tuples with relation
	tuples, err := ReadAllTuples(ctx, client, objectType, relation)
	if err != nil {
		return fmt.Errorf("failed to read tuples: %w", err)
	}

	if len(tuples) == 0 {
		fmt.Println("No tuples found to delete")
		return nil
	}

	fmt.Printf("Found %d tuples to delete\n", len(tuples))

	// Delete tuples
	if err := DeleteTuplesBatch(ctx, client, tuples); err != nil {
		return fmt.Errorf("failed to delete tuples: %w", err)
	}

	fmt.Println("Relation delete completed")
	return nil
}

// MigrateRelationWithTransform migrates tuples with a custom transformation function
// Example: Transform user IDs, change object formats, etc.
func MigrateRelationWithTransform(ctx context.Context, client *Client, objectType, oldRelation, newRelation string, transform TransformFunc) error {
	fmt.Printf("Migrating relation %s -> %s on type %s with custom transform\n", oldRelation, newRelation, objectType)

	// Read all tuples with old relation
	oldTuples, err := ReadAllTuples(ctx, client, objectType, oldRelation)
	if err != nil {
		return fmt.Errorf("failed to read tuples: %w", err)
	}

	if len(oldTuples) == 0 {
		fmt.Println("No tuples found to migrate")
		return nil
	}

	fmt.Printf("Found %d tuples to migrate\n", len(oldTuples))

	// Transform tuples
	var newTuples []Tuple
	for _, t := range oldTuples {
		transformed, err := transform(t)
		if err != nil {
			return fmt.Errorf("transform failed for tuple %v: %w", t, err)
		}

		// Use new relation if specified
		if newRelation != "" {
			transformed.Relation = newRelation
		}

		newTuples = append(newTuples, transformed)
	}

	// Write new tuples
	if err := WriteTuplesBatch(ctx, client, newTuples); err != nil {
		return fmt.Errorf("failed to write new tuples: %w", err)
	}

	// Delete old tuples if relation changed
	if oldRelation != newRelation && newRelation != "" {
		if err := DeleteTuplesBatch(ctx, client, oldTuples); err != nil {
			return fmt.Errorf("failed to delete old tuples: %w", err)
		}
	}

	fmt.Println("Migration with transform completed")
	return nil
}

// READ OPERATIONS

// ReadAllTuples reads all tuples matching the criteria
// Use empty string for objectType or relation to match all
func ReadAllTuples(ctx context.Context, client *Client, objectType, relation string) ([]Tuple, error) {
	req := ReadTuplesRequest{}

	if objectType != "" {
		req.Object = objectType + ":"
	}

	if relation != "" {
		req.Relation = relation
	}

	return client.ReadAllTuples(ctx, req)
}

// CountTuples counts tuples matching the criteria
func CountTuples(ctx context.Context, client *Client, objectType, relation string) (int, error) {
	tuples, err := ReadAllTuples(ctx, client, objectType, relation)
	if err != nil {
		return 0, err
	}
	return len(tuples), nil
}

// BATCH OPERATIONS

// WriteTuplesBatch writes tuples in batches to avoid overwhelming the API
func WriteTuplesBatch(ctx context.Context, client *Client, tuples []Tuple) error {
	total := len(tuples)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := tuples[i:end]
		fmt.Printf("Writing batch %d-%d of %d tuples\n", i+1, end, total)

		if err := client.WriteTuples(ctx, batch); err != nil {
			return fmt.Errorf("failed to write batch %d-%d: %w", i+1, end, err)
		}
	}

	return nil
}

// DeleteTuplesBatch deletes tuples in batches
func DeleteTuplesBatch(ctx context.Context, client *Client, tuples []Tuple) error {
	total := len(tuples)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := tuples[i:end]
		fmt.Printf("Deleting batch %d-%d of %d tuples\n", i+1, end, total)

		if err := client.DeleteTuples(ctx, batch); err != nil {
			return fmt.Errorf("failed to delete batch %d-%d: %w", i+1, end, err)
		}
	}

	return nil
}

// UTILITY FUNCTIONS

// BackupTuples exports all tuples to a backup (for safety before migrations)
func BackupTuples(ctx context.Context, client *Client) ([]Tuple, error) {
	fmt.Println("Backing up all tuples...")
	tuples, err := ReadAllTuples(ctx, client, "", "")
	if err != nil {
		return nil, err
	}
	fmt.Printf("Backed up %d tuples\n", len(tuples))
	return tuples, nil
}

// RestoreTuples restores tuples from a backup
func RestoreTuples(ctx context.Context, client *Client, tuples []Tuple) error {
	fmt.Printf("Restoring %d tuples...\n", len(tuples))
	return WriteTuplesBatch(ctx, client, tuples)
}

// ADVANCED MODEL OPERATIONS

// AddTypeToModel adds a new type to the current model
// Example: AddTypeToModel(ctx, client, "document", map[string]string{"owner": "[user]", "viewer": "[user] or owner"})
func AddTypeToModel(ctx context.Context, client *Client, typeName string, relations map[string]string) error {
	fmt.Printf("Adding type '%s' to model\n", typeName)

	// Get current model
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current model: %w", err)
	}

	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return fmt.Errorf("failed to parse current model: %w", err)
	}

	// Check if type already exists
	for _, t := range currentModel.TypeDefinitions {
		if t.Type == typeName {
			return fmt.Errorf("type '%s' already exists", typeName)
		}
	}

	// Create new type definition
	relationMap := make(map[string]openfgaSdk.Userset)

	// Parse relations
	for relName, relDef := range relations {
		userset, err := parseRelationDefinition(relDef)
		if err != nil {
			return fmt.Errorf("failed to parse relation '%s': %w", relName, err)
		}
		relationMap[relName] = userset
	}

	newType := openfgaSdk.TypeDefinition{
		Type:      typeName,
		Relations: &relationMap,
	}

	// Add to model
	currentModel.TypeDefinitions = append(currentModel.TypeDefinitions, newType)

	// Apply updated model
	if err := client.WriteAuthorizationModel(ctx, currentModel); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Printf("Type '%s' added successfully\n", typeName)
	return nil
}

// AddRelationToType adds a new relation to an existing type
// Example: AddRelationToType(ctx, client, "file", "can_delete", "[user] or owner")
func AddRelationToType(ctx context.Context, client *Client, typeName, relationName, relationDef string) error {
	fmt.Printf("Adding relation '%s' to type '%s'\n", relationName, typeName)

	// Get current model
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current model: %w", err)
	}

	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return fmt.Errorf("failed to parse current model: %w", err)
	}

	// Find the type
	typeFound := false
	for i, t := range currentModel.TypeDefinitions {
		if t.Type == typeName {
			typeFound = true

			relations := t.GetRelations()
			// Check if relation already exists
			if _, exists := relations[relationName]; exists {
				return fmt.Errorf("relation '%s' already exists on type '%s'", relationName, typeName)
			}

			// Parse relation definition
			userset, err := parseRelationDefinition(relationDef)
			if err != nil {
				return fmt.Errorf("failed to parse relation definition: %w", err)
			}

			// Add relation
			relations[relationName] = userset
			currentModel.TypeDefinitions[i].Relations = &relations

			// Extract and set type restrictions in metadata
			// Only set metadata for direct relations (with brackets), not for computed relations
			typeRestrictions := extractTypeRestrictions(relationDef)
			fmt.Printf("DEBUG: Adding %s.%s with def='%s', extracted types=%+v\n", typeName, relationName, relationDef, typeRestrictions)
			// Only set DirectlyRelatedUserTypes for direct relations (those with [...])
			// Tuple-to-userset relations (with 'from' or '->') should NOT have this metadata
			isDirect := strings.Contains(relationDef, "[")
			if len(typeRestrictions) > 0 && isDirect {
				// Ensure metadata exists
				if currentModel.TypeDefinitions[i].Metadata == nil {
					currentModel.TypeDefinitions[i].Metadata = &openfgaSdk.Metadata{}
				}

				// Ensure relations metadata map exists
				if currentModel.TypeDefinitions[i].Metadata.Relations == nil {
					relationsMetadata := make(map[string]openfgaSdk.RelationMetadata)
					currentModel.TypeDefinitions[i].Metadata.Relations = &relationsMetadata
				}

				// Set directly related user types for this relation
				relationsMetadata := *currentModel.TypeDefinitions[i].Metadata.Relations
				relationsMetadata[relationName] = openfgaSdk.RelationMetadata{
					DirectlyRelatedUserTypes: &typeRestrictions,
				}
				currentModel.TypeDefinitions[i].Metadata.Relations = &relationsMetadata
			}

			break
		}
	}

	if !typeFound {
		return fmt.Errorf("type '%s' not found", typeName)
	}

	// Apply updated model
	if err := client.WriteAuthorizationModel(ctx, currentModel); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Printf("Relation '%s' added to type '%s' successfully\n", relationName, typeName)
	return nil
}

// RemoveRelationFromType removes a relation from a type
// WARNING: This will NOT delete the tuples - use DeleteRelation() for that
func RemoveRelationFromType(ctx context.Context, client *Client, typeName, relationName string) error {
	fmt.Printf("Removing relation '%s' from type '%s'\n", relationName, typeName)

	// Get current model
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current model: %w", err)
	}

	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return fmt.Errorf("failed to parse current model: %w", err)
	}

	// Find the type and remove relation
	typeFound := false
	relationFound := false
	for i, t := range currentModel.TypeDefinitions {
		if t.Type == typeName {
			typeFound = true

			relations := t.GetRelations()
			if _, exists := relations[relationName]; exists {
				relationFound = true
				delete(relations, relationName)
				currentModel.TypeDefinitions[i].Relations = &relations
			}
			break
		}
	}

	if !typeFound {
		return fmt.Errorf("type '%s' not found", typeName)
	}

	if !relationFound {
		return fmt.Errorf("relation '%s' not found on type '%s'", relationName, typeName)
	}

	// Apply updated model
	if err := client.WriteAuthorizationModel(ctx, currentModel); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Printf("Relation '%s' removed from type '%s' successfully\n", relationName, typeName)
	fmt.Println("NOTE: Existing tuples with this relation are NOT deleted. Run DeleteRelation() to remove them.")
	return nil
}

// RemoveTypeFromModel removes a type from the model
// WARNING: This will NOT delete the tuples - handle tuple cleanup separately
func RemoveTypeFromModel(ctx context.Context, client *Client, typeName string) error {
	fmt.Printf("Removing type '%s' from model\n", typeName)

	// Get current model
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current model: %w", err)
	}

	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return fmt.Errorf("failed to parse current model: %w", err)
	}

	// Find and remove the type
	typeFound := false
	var updatedTypes []openfgaSdk.TypeDefinition
	for _, t := range currentModel.TypeDefinitions {
		if t.Type == typeName {
			typeFound = true
			continue
		}
		updatedTypes = append(updatedTypes, t)
	}

	if !typeFound {
		return fmt.Errorf("type '%s' not found", typeName)
	}

	currentModel.TypeDefinitions = updatedTypes

	// Apply updated model
	if err := client.WriteAuthorizationModel(ctx, currentModel); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Printf("Type '%s' removed successfully\n", typeName)
	fmt.Println("NOTE: Existing tuples of this type are NOT deleted. Handle tuple cleanup separately.")
	return nil
}

// UpdateRelationDefinition updates the definition of an existing relation
// Example: UpdateRelationDefinition(ctx, client, "file", "can_modify", "[user, group#member] or owner")
func UpdateRelationDefinition(ctx context.Context, client *Client, typeName, relationName, newDefinition string) error {
	fmt.Printf("Updating relation '%s' on type '%s'\n", relationName, typeName)

	// Get current model
	currentDSL, err := client.GetCurrentModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current model: %w", err)
	}

	currentModel, err := parseDSLToModel(currentDSL)
	if err != nil {
		return fmt.Errorf("failed to parse current model: %w", err)
	}

	// Find the type and update relation
	typeFound := false
	relationFound := false
	for i, t := range currentModel.TypeDefinitions {
		if t.Type == typeName {
			typeFound = true

			relations := t.GetRelations()
			if _, exists := relations[relationName]; exists {
				relationFound = true

				// Parse new definition
				userset, err := parseRelationDefinition(newDefinition)
				if err != nil {
					return fmt.Errorf("failed to parse new relation definition: %w", err)
				}

				relations[relationName] = userset
				currentModel.TypeDefinitions[i].Relations = &relations
			}
			break
		}
	}

	if !typeFound {
		return fmt.Errorf("type '%s' not found", typeName)
	}

	if !relationFound {
		return fmt.Errorf("relation '%s' not found on type '%s'", relationName, typeName)
	}

	// Apply updated model
	if err := client.WriteAuthorizationModel(ctx, currentModel); err != nil {
		return fmt.Errorf("failed to write model: %w", err)
	}

	fmt.Printf("Relation '%s' on type '%s' updated successfully\n", relationName, typeName)
	return nil
}
