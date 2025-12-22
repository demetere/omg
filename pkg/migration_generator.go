package omg

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// GenerateMigrationFromChanges generates a migration file from detected model changes
func GenerateMigrationFromChanges(changes []ModelChange, name string, migrationsDir string) (string, error) {
	if len(changes) == 0 {
		return "", fmt.Errorf("no changes detected")
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s/%s_%s.go", migrationsDir, timestamp, sanitizeName(name))

	// Generate migration code
	code := generateMigrationCode(timestamp, name, changes)

	// Write to file
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write migration file: %w", err)
	}

	return filename, nil
}

// generateMigrationCode generates the Go code for a migration
func generateMigrationCode(version, name string, changes []ModelChange) string {
	var builder strings.Builder

	// Package and imports for standalone executable
	builder.WriteString(`package main

// Migration: ` + sanitizeName(name) + `
// Version: ` + version + `
// Auto-generated migration

import (
	"context"
	"fmt"
	"os"

	omg "github.com/demetere/omg"
)

func main() {
	// Get connection info from environment
	client, err := omg.NewClient(omg.Config{
		ApiURL:     os.Getenv("OPENFGA_API_URL"),
		StoreID:    os.Getenv("OPENFGA_STORE_ID"),
		AuthMethod: getAuthMethod(),
		APIToken:   os.Getenv("OPENFGA_API_TOKEN"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Check if we should run Up or Down
	if len(os.Args) > 1 && os.Args[1] == "down" {
		if err := down(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := up(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func getAuthMethod() string {
	if token := os.Getenv("OPENFGA_API_TOKEN"); token != "" {
		return "token"
	}
	return "none"
}

`)

	// Up function
	builder.WriteString("func up(ctx context.Context, client *omg.Client) error {\n")
	builder.WriteString("\t// Auto-generated migration\n")
	builder.WriteString("\t// Changes detected:\n")

	for _, change := range changes {
		builder.WriteString(fmt.Sprintf("\t// - %s\n", change.Details))
	}

	builder.WriteString("\n")

	// Generate up migration code
	builder.WriteString(generateUpMigration(changes))

	builder.WriteString("\n\treturn nil\n")
	builder.WriteString("}\n\n")

	// Down function
	builder.WriteString("func down(ctx context.Context, client *omg.Client) error {\n")
	builder.WriteString("\t// Rollback operations\n\n")

	// Generate down migration code
	builder.WriteString(generateDownMigration(changes))

	builder.WriteString("\n\treturn nil\n")
	builder.WriteString("}\n")

	return builder.String()
}

// generateUpMigration generates the up migration code
func generateUpMigration(changes []ModelChange) string {
	var builder strings.Builder

	// Process changes in order:
	// 1. Add types
	// 2. Add relations
	// 3. Update relations
	// 4. Rename relations (with tuple migration)
	// 5. Remove relations (with tuple cleanup)
	// 6. Rename types (with tuple migration)
	// 7. Remove types (with tuple cleanup)

	orderedChanges := orderChangesForUp(changes)

	for _, change := range orderedChanges {
		switch change.Type {
		case ChangeTypeAddType:
			builder.WriteString(generateAddType(change))

		case ChangeTypeAddRelation:
			builder.WriteString(generateAddRelation(change))

		case ChangeTypeUpdateRelation:
			builder.WriteString(generateUpdateRelation(change))

		case ChangeTypeRenameRelation:
			builder.WriteString(generateRenameRelation(change))

		case ChangeTypeRemoveRelation:
			builder.WriteString(generateRemoveRelation(change))

		case ChangeTypeRenameType:
			builder.WriteString(generateRenameType(change))

		case ChangeTypeRemoveType:
			builder.WriteString(generateRemoveType(change))
		}
	}

	return builder.String()
}

// generateDownMigration generates the down migration code (reverse order)
func generateDownMigration(changes []ModelChange) string {
	var builder strings.Builder

	// Reverse the order for down migration
	orderedChanges := orderChangesForDown(changes)

	for _, change := range orderedChanges {
		switch change.Type {
		case ChangeTypeAddType:
			// Reverse: remove type
			builder.WriteString(generateRemoveType(ModelChange{
				TypeName: change.TypeName,
			}))

		case ChangeTypeRemoveType:
			builder.WriteString("\t// NOTE: Cannot automatically restore removed type\n")
			builder.WriteString(fmt.Sprintf("\t// You need to manually add type '%s' back\n\n", change.TypeName))

		case ChangeTypeAddRelation:
			// Reverse: remove relation
			builder.WriteString(generateRemoveRelation(change))

		case ChangeTypeRemoveRelation:
			builder.WriteString("\t// NOTE: Cannot automatically restore removed relation\n")
			builder.WriteString(fmt.Sprintf("\t// You need to manually add relation '%s.%s' back\n\n", change.TypeName, change.RelationName))

		case ChangeTypeUpdateRelation:
			// Reverse: update back to old definition
			builder.WriteString(generateUpdateRelation(ModelChange{
				TypeName:     change.TypeName,
				RelationName: change.RelationName,
				NewValue:     change.OldValue, // Swap old and new
			}))

		case ChangeTypeRenameRelation:
			// Reverse: rename back
			builder.WriteString(generateRenameRelation(ModelChange{
				TypeName:     change.TypeName,
				RelationName: change.NewValue,
				OldValue:     change.NewValue,
				NewValue:     change.OldValue,
			}))

		case ChangeTypeRenameType:
			// Reverse: rename back
			builder.WriteString(generateRenameType(ModelChange{
				TypeName: change.NewValue,
				OldValue: change.NewValue,
				NewValue: change.OldValue,
			}))
		}
	}

	return builder.String()
}

// Code generators for each change type

func generateAddType(change ModelChange) string {
	return fmt.Sprintf(`	// Add type: %s
	// TODO: Define relations for this type
	if err := omg.AddTypeToModel(ctx, client, "%s", map[string]string{
		// Add your relations here
		// "owner": "[user]",
	}); err != nil {
		return fmt.Errorf("failed to add type %s: %%w", err)
	}

`, change.TypeName, change.TypeName, change.TypeName)
}

func generateAddRelation(change ModelChange) string {
	// Try to extract a readable definition (simplified)
	def := extractRelationDefinition(change.NewValue)

	return fmt.Sprintf(`	// Add relation: %s.%s
	if err := omg.AddRelationToType(ctx, client, "%s", "%s", "%s"); err != nil {
		return fmt.Errorf("failed to add relation: %%w", err)
	}

`, change.TypeName, change.RelationName, change.TypeName, change.RelationName, def)
}

func generateUpdateRelation(change ModelChange) string {
	def := extractRelationDefinition(change.NewValue)

	return fmt.Sprintf(`	// Update relation: %s.%s
	if err := omg.UpdateRelationDefinition(ctx, client, "%s", "%s", "%s"); err != nil {
		return fmt.Errorf("failed to update relation: %%w", err)
	}

`, change.TypeName, change.RelationName, change.TypeName, change.RelationName, def)
}

func generateRenameRelation(change ModelChange) string {
	switch change.Confidence {
	case ConfidenceHigh:
		// High confidence: straightforward rename
		return fmt.Sprintf(`	// Rename relation: %s.%s -> %s.%s (high confidence)
	// This will copy all tuples from the old relation to the new one
	if err := omg.RenameRelation(ctx, client, "%s", "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename relation: %%w", err)
	}

`, change.TypeName, change.OldValue, change.TypeName, change.NewValue,
			change.TypeName, change.OldValue, change.NewValue)

	case ConfidenceMedium:
		// Medium confidence: generate rename with review notice
		return fmt.Sprintf(`	// ⚠️  REVIEW REQUIRED: Possible relation rename
	// Detected: %s.%s -> %s.%s
	//
	// This appears to be a rename. Review and confirm before applying.
	// If correct, this will preserve all existing tuples.
	//
	if err := omg.RenameRelation(ctx, client, "%s", "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename relation: %%w", err)
	}

`, change.TypeName, change.OldValue, change.TypeName, change.NewValue,
			change.TypeName, change.OldValue, change.NewValue)

	case ConfidenceLow:
		// Low confidence: offer both options
		return fmt.Sprintf(`	// ⚠️  MANUAL REVIEW REQUIRED ⚠️
	// Potential relation rename: %s.%s -> %s.%s (low confidence)
	//
	// OPTION 1: If this IS a rename (preserve tuples), uncomment:
	// if err := omg.RenameRelation(ctx, client, "%s", "%s", "%s"); err != nil {
	// 	return fmt.Errorf("failed to rename relation: %%w", err)
	// }
	//
	// OPTION 2: If these are separate relations (default, safe):

	// Remove old relation (new relation already in model.fga)
	tuples, err := omg.ReadAllTuples(ctx, client, "%s", "%s")
	if err != nil {
		return fmt.Errorf("failed to read tuples: %%w", err)
	}
	if len(tuples) > 0 {
		fmt.Printf("⚠️  Deleting %%d tuples from relation '%s.%s'\n", len(tuples))
		if err := omg.DeleteTuplesBatch(ctx, client, tuples); err != nil {
			return fmt.Errorf("failed to delete tuples: %%w", err)
		}
	}
	if err := omg.RemoveRelationFromType(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to remove old relation: %%w", err)
	}

`, change.TypeName, change.OldValue, change.TypeName, change.NewValue,
			change.TypeName, change.OldValue, change.NewValue,
			change.TypeName, change.OldValue,
			change.TypeName, change.OldValue,
			change.TypeName, change.OldValue)

	default:
		// Fallback
		return fmt.Sprintf(`	// Rename relation: %s.%s -> %s.%s
	if err := omg.RenameRelation(ctx, client, "%s", "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename relation: %%w", err)
	}

`, change.TypeName, change.OldValue, change.TypeName, change.NewValue,
			change.TypeName, change.OldValue, change.NewValue)
	}
}

func generateRemoveRelation(change ModelChange) string {
	return fmt.Sprintf(`	// Remove relation: %s.%s
	// Step 1: Remove from model
	if err := omg.RemoveRelationFromType(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to remove relation from model: %%w", err)
	}

	// Step 2: Delete all tuples with this relation
	if err := omg.DeleteRelation(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to delete tuples: %%w", err)
	}

`, change.TypeName, change.RelationName, change.TypeName, change.RelationName,
		change.TypeName, change.RelationName)
}

func generateRenameType(change ModelChange) string {
	switch change.Confidence {
	case ConfidenceHigh:
		// High confidence: generate rename with minimal comments
		return fmt.Sprintf(`	// Rename type: %s -> %s (high confidence rename detected)
	// This will migrate all existing tuples to the new type name
	if err := omg.RenameType(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename type: %%w", err)
	}

`, change.OldValue, change.NewValue, change.OldValue, change.NewValue)

	case ConfidenceMedium:
		// Medium confidence: generate rename but warn user to review
		return fmt.Sprintf(`	// ⚠️  REVIEW REQUIRED: Possible rename detected
	// Detected: %s -> %s
	//
	// This appears to be a rename based on similarity analysis.
	// If this IS a rename (preserving tuples), keep the code below.
	// If these are separate types, replace with AddType + DeleteType operations.
	//
	if err := omg.RenameType(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename type: %%w", err)
	}

`, change.OldValue, change.NewValue, change.OldValue, change.NewValue)

	case ConfidenceLow:
		// Low confidence: generate commented-out rename with add+remove as default
		return fmt.Sprintf(`	// ⚠️  MANUAL REVIEW REQUIRED ⚠️
	// Detected potential rename: %s -> %s (low confidence)
	//
	// OPTION 1: If this IS a rename (preserve tuples), uncomment:
	// if err := omg.RenameType(ctx, client, "%s", "%s"); err != nil {
	// 	return fmt.Errorf("failed to rename type: %%w", err)
	// }
	//
	// OPTION 2: If these are separate types (default, safe option):

	// Add new type (already in model.fga)
	// The new model definition is already applied

	// Delete old type and its tuples
	tuples, err := omg.ReadAllTuples(ctx, client, "%s", "")
	if err != nil {
		return fmt.Errorf("failed to read tuples: %%w", err)
	}
	if len(tuples) > 0 {
		fmt.Printf("⚠️  Deleting %%d tuples of type '%s'\n", len(tuples))
		if err := omg.DeleteTuplesBatch(ctx, client, tuples); err != nil {
			return fmt.Errorf("failed to delete tuples: %%w", err)
		}
	}
	if err := omg.RemoveTypeFromModel(ctx, client, "%s"); err != nil {
		return fmt.Errorf("failed to remove old type: %%w", err)
	}

`, change.OldValue, change.NewValue, change.OldValue, change.NewValue,
			change.OldValue, change.OldValue, change.OldValue)

	default:
		// Fallback to simple rename
		return fmt.Sprintf(`	// Rename type: %s -> %s
	if err := omg.RenameType(ctx, client, "%s", "%s"); err != nil {
		return fmt.Errorf("failed to rename type: %%w", err)
	}

`, change.OldValue, change.NewValue, change.OldValue, change.NewValue)
	}
}

func generateRemoveType(change ModelChange) string {
	return fmt.Sprintf(`	// Remove type: %s
	// Step 1: Delete all tuples of this type
	{
		tuples, err := omg.ReadAllTuples(ctx, client, "%s", "")
		if err != nil {
			return fmt.Errorf("failed to read tuples: %%w", err)
		}

		if len(tuples) > 0 {
			fmt.Printf("Deleting %%d tuples of type %s\n", len(tuples))
			if err := omg.DeleteTuplesBatch(ctx, client, tuples); err != nil {
				return fmt.Errorf("failed to delete tuples: %%w", err)
			}
		}
	}

	// Step 2: Remove type from model
	if err := omg.RemoveTypeFromModel(ctx, client, "%s"); err != nil {
		return fmt.Errorf("failed to remove type from model: %%w", err)
	}

`, change.TypeName, change.TypeName, change.TypeName, change.TypeName)
}

// Helper functions

func sanitizeName(name string) string {
	// Convert to valid Go identifier
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	// Remove invalid characters
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func extractRelationDefinition(serialized string) string {
	// The serialized string IS the DSL definition stored in ModelChange.NewValue
	// It's already in the correct format from model_tracker.go
	if serialized == "" {
		return "[user]" // fallback
	}
	// Escape quotes for embedding in Go string literal
	return strings.ReplaceAll(serialized, `"`, `\"`)
}

func orderChangesForUp(changes []ModelChange) []ModelChange {
	var ordered []ModelChange

	// Order: add types, add relations (sorted by dependency), update relations, renames, removes
	order := []ChangeType{
		ChangeTypeAddType,
		ChangeTypeAddRelation,
		ChangeTypeUpdateRelation,
		ChangeTypeRenameRelation,
		ChangeTypeRenameType,
		ChangeTypeRemoveRelation,
		ChangeTypeRemoveType,
	}

	for _, changeType := range order {
		if changeType == ChangeTypeAddRelation {
			// Special handling for AddRelation - sort by dependencies across all types
			var relationChanges []ModelChange
			for _, change := range changes {
				if change.Type == ChangeTypeAddRelation {
					relationChanges = append(relationChanges, change)
				}
			}
			sorted := sortRelationChanges(relationChanges)
			ordered = append(ordered, sorted...)
		} else {
			for _, change := range changes {
				if change.Type == changeType {
					ordered = append(ordered, change)
				}
			}
		}
	}

	return ordered
}

// sortRelationChanges sorts relation changes by dependency across all types
func sortRelationChanges(changes []ModelChange) []ModelChange {
	// Build a map of all relations for dependency analysis
	relations := make(map[string]string) // "type.relation" -> definition
	changeMap := make(map[string]ModelChange) // "type.relation" -> change

	for _, change := range changes {
		key := fmt.Sprintf("%s.%s", change.TypeName, change.RelationName)
		relations[key] = change.NewValue
		changeMap[key] = change
	}

	// Build dependency graph
	deps := make(map[string][]string) // relation -> dependencies

	for key, relDef := range relations {
		parts := strings.Split(key, ".")
		typeName := parts[0]
		relName := parts[1]

		// Extract dependencies
		depNames := extractRelationDependenciesFromDef(relDef, relName)

		// Convert to full keys (type.relation)
		var fullDeps []string

		// Special handling for tuple-to-userset (e.g., "member from team")
		if strings.Contains(relDef, " from ") {
			parts := strings.Split(relDef, " from ")
			if len(parts) == 2 {
				computedUserset := strings.TrimSpace(parts[0])
				tuplesetType := strings.TrimSpace(parts[1])
				// The dependency is on tuplesetType.computedUserset
				depKey := fmt.Sprintf("%s.%s", tuplesetType, computedUserset)
				if _, exists := relations[depKey]; exists {
					fullDeps = append(fullDeps, depKey)
				}
			}
		} else if strings.Contains(relDef, "->") {
			// Handle arrow syntax: "parent->owner"
			parts := strings.Split(relDef, "->")
			if len(parts) == 2 {
				_ = strings.TrimSpace(parts[0]) // tuplesetRel - not used for dependency
				computedUserset := strings.TrimSpace(parts[1])
				// The dependency is on typeName.computedUserset
				depKey := fmt.Sprintf("%s.%s", typeName, computedUserset)
				if _, exists := relations[depKey]; exists {
					fullDeps = append(fullDeps, depKey)
				}
			}
		} else {
			// Regular dependencies (e.g., "owner", "[user] or owner")
			for _, depName := range depNames {
				// Check if it exists in same type first
				sameTypeKey := fmt.Sprintf("%s.%s", typeName, depName)
				if _, exists := relations[sameTypeKey]; exists {
					fullDeps = append(fullDeps, sameTypeKey)
				} else {
					// Could be a cross-type reference, search all types
					for otherKey := range relations {
						if strings.HasSuffix(otherKey, "."+depName) {
							fullDeps = append(fullDeps, otherKey)
						}
					}
				}
			}
		}
		deps[key] = fullDeps
	}

	// Topological sort
	sorted := topologicalSort(deps, relations)

	// Convert back to changes
	var result []ModelChange
	for _, key := range sorted {
		if change, exists := changeMap[key]; exists {
			result = append(result, change)
		}
	}

	return result
}

// topologicalSort performs topological sort on the dependency graph
func topologicalSort(deps map[string][]string, allNodes map[string]string) []string {
	inDegree := make(map[string]int)
	for node := range allNodes {
		inDegree[node] = 0
	}

	// Count incoming edges: if node X depends on Y, then X has an incoming edge
	for node, dependencies := range deps {
		inDegree[node] = len(dependencies)
	}

	// Find nodes with no incoming edges
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// Reduce in-degree for dependents
		for dependent, dependencies := range deps {
			for _, dep := range dependencies {
				if dep == current {
					inDegree[dependent]--
					if inDegree[dependent] == 0 {
						queue = append(queue, dependent)
					}
				}
			}
		}
	}

	// If we couldn't sort everything (cycle), append remaining
	if len(sorted) < len(allNodes) {
		for node := range allNodes {
			found := false
			for _, s := range sorted {
				if s == node {
					found = true
					break
				}
			}
			if !found {
				sorted = append(sorted, node)
			}
		}
	}

	return sorted
}

// extractRelationDependenciesFromDef extracts relation names from a definition
// Returns relation names that this definition depends on
// For "member from team", returns ["member"] (the computed userset, not the tupleset)
// The tupleset ("team") is handled separately as it's just a type reference
func extractRelationDependenciesFromDef(relDef, currentRel string) []string {
	var deps []string

	// Handle tuple-to-userset with 'from' syntax: "member from team"
	// This depends on the "member" relation (computedUserset), not "team" (which is the tupleset type)
	// The actual dependency will be resolved later as "team.member"
	if strings.Contains(relDef, " from ") {
		parts := strings.Split(relDef, " from ")
		if len(parts) == 2 {
			computedUserset := strings.TrimSpace(parts[0])
			// The computed userset is the relation we depend on
			deps = append(deps, computedUserset)
			return deps
		}
	}

	// Handle arrow syntax: "parent->owner"
	// This depends on the "owner" relation
	if strings.Contains(relDef, "->") {
		parts := strings.Split(relDef, "->")
		if len(parts) == 2 {
			computedUserset := strings.TrimSpace(parts[1])
			deps = append(deps, computedUserset)
			return deps
		}
	}

	// Extract words that could be relation names
	words := strings.FieldsFunc(relDef, func(r rune) bool {
		return r == ' ' || r == ',' || r == ':' || r == '#'
	})

	for _, word := range words {
		// Skip keywords and the relation itself
		if word == "or" || word == "and" || word == "but" || word == "not" ||
			word == "from" || word == "define" || word == currentRel ||
			strings.HasPrefix(word, "[") || strings.HasSuffix(word, "]") ||
			strings.Contains(word, "[") || strings.Contains(word, "]") {
			continue
		}

		// This might be a relation dependency
		deps = append(deps, word)
	}

	return deps
}

func orderChangesForDown(changes []ModelChange) []ModelChange {
	// Reverse order for down migration
	ordered := orderChangesForUp(changes)

	// Reverse the slice
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}

	return ordered
}
