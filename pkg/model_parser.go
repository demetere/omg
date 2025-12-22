package omg

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	openfgaSdk "github.com/openfga/go-sdk"
)

// ParseDSLToModel parses OpenFGA DSL format to an AuthorizationModel
// This is a simplified parser - for production, consider using the official FGA parser
func ParseDSLToModel(dsl string) (openfgaSdk.AuthorizationModel, error) {
	return parseDSLToModel(dsl)
}

// parseDSLToModel parses OpenFGA DSL format to an AuthorizationModel (internal)
func parseDSLToModel(dsl string) (openfgaSdk.AuthorizationModel, error) {
	model := openfgaSdk.AuthorizationModel{
		SchemaVersion:   "1.1",
		TypeDefinitions: []openfgaSdk.TypeDefinition{},
	}

	lines := strings.Split(dsl, "\n")
	var currentType *openfgaSdk.TypeDefinition
	var inRelations bool

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse schema version
		if strings.HasPrefix(line, "schema ") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "schema"))
			model.SchemaVersion = version
			continue
		}

		// Skip "model" keyword
		if line == "model" {
			continue
		}

		// Parse type definition
		if strings.HasPrefix(line, "type ") {
			// Save previous type if exists
			if currentType != nil {
				model.TypeDefinitions = append(model.TypeDefinitions, *currentType)
			}

			typeName := strings.TrimSpace(strings.TrimPrefix(line, "type"))
			relationMap := make(map[string]openfgaSdk.Userset)
			currentType = &openfgaSdk.TypeDefinition{
				Type:      typeName,
				Relations: &relationMap,
			}
			inRelations = false
			continue
		}

		// Parse relations section
		if line == "relations" {
			inRelations = true
			continue
		}

		// Parse relation definition
		if inRelations && strings.HasPrefix(line, "define ") {
			if currentType == nil {
				return model, fmt.Errorf("relation defined outside of type at line %d", i+1)
			}

			// Parse: define relation_name: [user, group#member] or owner
			defineStr := strings.TrimSpace(strings.TrimPrefix(line, "define"))
			parts := strings.SplitN(defineStr, ":", 2)
			if len(parts) != 2 {
				return model, fmt.Errorf("invalid relation definition at line %d: %s", i+1, line)
			}

			relationName := strings.TrimSpace(parts[0])
			relationDef := strings.TrimSpace(parts[1])

			userset, err := parseRelationDefinition(relationDef)
			if err != nil {
				return model, fmt.Errorf("failed to parse relation '%s' at line %d: %w", relationName, i+1, err)
			}

			relations := currentType.GetRelations()
			relations[relationName] = userset
			currentType.Relations = &relations

			// Extract and set type restrictions in metadata
			// Only set metadata for direct relations (with brackets), not for computed relations
			typeRestrictions := extractTypeRestrictions(relationDef)
			// Only set DirectlyRelatedUserTypes for direct relations (those with [...])
			// Tuple-to-userset relations (with 'from' or '->') should NOT have this metadata
			isDirect := strings.Contains(relationDef, "[")
			if len(typeRestrictions) > 0 && isDirect {
				// Ensure metadata exists
				if currentType.Metadata == nil {
					currentType.Metadata = &openfgaSdk.Metadata{}
				}
				metadata := currentType.GetMetadata()

				// Ensure relations metadata map exists
				if metadata.Relations == nil {
					relationsMetadata := make(map[string]openfgaSdk.RelationMetadata)
					metadata.Relations = &relationsMetadata
				}

				// Set directly related user types for this relation
				relationsMetadata := metadata.GetRelations()
				relationsMetadata[relationName] = openfgaSdk.RelationMetadata{
					DirectlyRelatedUserTypes: &typeRestrictions,
				}
				metadata.Relations = &relationsMetadata
				currentType.Metadata = &metadata
			}
		}
	}

	// Save last type
	if currentType != nil {
		model.TypeDefinitions = append(model.TypeDefinitions, *currentType)
	}

	return model, nil
}

// parseRelationDefinition parses a relation definition into a Userset
// Examples:
//   - [user] -> direct relation to user type
//   - [user, group#member] -> direct relation to multiple types
//   - [user] or owner -> union of direct and computed relation
//   - owner -> computed relation
//   - parent->owner -> tuple-to-userset (arrow syntax)
//   - owner from team -> tuple-to-userset (from syntax)
func parseRelationDefinition(def string) (openfgaSdk.Userset, error) {
	userset := openfgaSdk.Userset{}

	// Handle "or" operator (union)
	if strings.Contains(def, " or ") {
		parts := strings.Split(def, " or ")
		var children []openfgaSdk.Userset
		for _, part := range parts {
			child, err := parseRelationDefinition(strings.TrimSpace(part))
			if err != nil {
				return userset, err
			}
			children = append(children, child)
		}
		userset.Union = &openfgaSdk.Usersets{Child: children}
		return userset, nil
	}

	// Handle "and" operator (intersection)
	if strings.Contains(def, " and ") {
		parts := strings.Split(def, " and ")
		var children []openfgaSdk.Userset
		for _, part := range parts {
			child, err := parseRelationDefinition(strings.TrimSpace(part))
			if err != nil {
				return userset, err
			}
			children = append(children, child)
		}
		userset.Intersection = &openfgaSdk.Usersets{Child: children}
		return userset, nil
	}

	// Handle "but not" operator (difference)
	if strings.Contains(def, " but not ") {
		parts := strings.Split(def, " but not ")
		if len(parts) != 2 {
			return userset, fmt.Errorf("invalid 'but not' syntax: %s", def)
		}
		base, err := parseRelationDefinition(strings.TrimSpace(parts[0]))
		if err != nil {
			return userset, err
		}
		subtract, err := parseRelationDefinition(strings.TrimSpace(parts[1]))
		if err != nil {
			return userset, err
		}
		userset.Difference = &openfgaSdk.Difference{
			Base:     base,
			Subtract: subtract,
		}
		return userset, nil
	}

	// Handle direct relation: [user] or [user, group#member]
	if strings.HasPrefix(def, "[") && strings.HasSuffix(def, "]") {
		typesStr := strings.TrimPrefix(strings.TrimSuffix(def, "]"), "[")
		typeList := strings.Split(typesStr, ",")

		var typeRestrictions []openfgaSdk.RelationReference
		for _, t := range typeList {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			// Parse type#relation format
			if strings.Contains(t, "#") {
				parts := strings.Split(t, "#")
				if len(parts) != 2 {
					return userset, fmt.Errorf("invalid type#relation format: %s", t)
				}
				typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
					Type:     parts[0],
					Relation: openfgaSdk.PtrString(parts[1]),
				})
			} else {
				// Simple type reference
				typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
					Type: t,
				})
			}
		}

		// Direct userset - for simplified parsing, just set this
		// In full implementation, you'd properly handle type restrictions
		thisMap := make(map[string]interface{})
		userset.This = &thisMap
		return userset, nil
	}

	// Handle tuple-to-userset with 'from' syntax: owner from team
	// This is equivalent to: team->owner
	if strings.Contains(def, " from ") {
		parts := strings.Split(def, " from ")
		if len(parts) != 2 {
			return userset, fmt.Errorf("invalid 'from' syntax: %s", def)
		}
		computedUserset := strings.TrimSpace(parts[0])
		tupleset := strings.TrimSpace(parts[1])

		userset.TupleToUserset = &openfgaSdk.TupleToUserset{
			Tupleset: openfgaSdk.ObjectRelation{
				Relation: openfgaSdk.PtrString(tupleset),
			},
			ComputedUserset: openfgaSdk.ObjectRelation{
				Relation: openfgaSdk.PtrString(computedUserset),
			},
		}
		return userset, nil
	}

	// Handle tuple-to-userset with arrow syntax: parent->owner
	if strings.Contains(def, "->") {
		parts := strings.Split(def, "->")
		if len(parts) != 2 {
			return userset, fmt.Errorf("invalid tuple-to-userset format: %s", def)
		}
		userset.TupleToUserset = &openfgaSdk.TupleToUserset{
			Tupleset: openfgaSdk.ObjectRelation{
				Relation: openfgaSdk.PtrString(strings.TrimSpace(parts[0])),
			},
			ComputedUserset: openfgaSdk.ObjectRelation{
				Relation: openfgaSdk.PtrString(strings.TrimSpace(parts[1])),
			},
		}
		return userset, nil
	}

	// Handle computed relation: owner, parent
	if matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, def); matched {
		userset.ComputedUserset = &openfgaSdk.ObjectRelation{
			Relation: openfgaSdk.PtrString(def),
		}
		return userset, nil
	}

	return userset, fmt.Errorf("unable to parse relation definition: %s", def)
}

// extractTypeRestrictions extracts type restrictions from a relation definition
// For example: "[user]" returns [{Type: "user"}], "[user, group#member]" returns [{Type: "user"}, {Type: "group", Relation: "member"}]
// For tuple-to-userset like "admin from team", returns [{Type: "team"}]
func extractTypeRestrictions(def string) []openfgaSdk.RelationReference {
	var typeRestrictions []openfgaSdk.RelationReference

	// Handle tuple-to-userset with 'from' syntax: admin from team
	// The tupleset is the type that should be in DirectlyRelatedUserTypes
	if strings.Contains(def, " from ") {
		parts := strings.Split(def, " from ")
		if len(parts) == 2 {
			tupleset := strings.TrimSpace(parts[1])
			typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
				Type: tupleset,
			})
			return typeRestrictions
		}
	}

	// Handle arrow syntax: parent->owner (tupleset is parent)
	if strings.Contains(def, "->") {
		parts := strings.Split(def, "->")
		if len(parts) == 2 {
			tupleset := strings.TrimSpace(parts[0])
			typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
				Type: tupleset,
			})
			return typeRestrictions
		}
	}

	// Check if definition contains direct type restrictions [...]
	if !strings.Contains(def, "[") || !strings.Contains(def, "]") {
		return typeRestrictions
	}

	// Extract content between []
	start := strings.Index(def, "[")
	end := strings.Index(def, "]")
	if start == -1 || end == -1 || end <= start {
		return typeRestrictions
	}

	typesStr := def[start+1 : end]
	typeList := strings.Split(typesStr, ",")

	for _, t := range typeList {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		// Parse type#relation format
		if strings.Contains(t, "#") {
			parts := strings.Split(t, "#")
			if len(parts) == 2 {
				typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
					Type:     parts[0],
					Relation: openfgaSdk.PtrString(parts[1]),
				})
			}
		} else {
			// Simple type reference
			typeRestrictions = append(typeRestrictions, openfgaSdk.RelationReference{
				Type: t,
			})
		}
	}

	return typeRestrictions
}

// generateModelDiff generates a human-readable diff between two models
func generateModelDiff(current, new openfgaSdk.AuthorizationModel) string {
	var diff strings.Builder

	diff.WriteString("Model Changes:\n")
	diff.WriteString("==============\n\n")

	// Build maps for easier comparison
	currentTypes := make(map[string]openfgaSdk.TypeDefinition)
	for _, t := range current.TypeDefinitions {
		currentTypes[t.Type] = t
	}

	newTypes := make(map[string]openfgaSdk.TypeDefinition)
	for _, t := range new.TypeDefinitions {
		newTypes[t.Type] = t
	}

	// Check for new types
	for typeName, typeDef := range newTypes {
		if _, exists := currentTypes[typeName]; !exists {
			diff.WriteString(fmt.Sprintf("+ Added type: %s\n", typeName))
			relations := typeDef.GetRelations()
			for relName := range relations {
				diff.WriteString(fmt.Sprintf("  + Added relation: %s\n", relName))
			}
			diff.WriteString("\n")
		}
	}

	// Check for removed types
	for typeName := range currentTypes {
		if _, exists := newTypes[typeName]; !exists {
			diff.WriteString(fmt.Sprintf("- Removed type: %s\n", typeName))
			diff.WriteString("\n")
		}
	}

	// Check for modified types
	for typeName, currentType := range currentTypes {
		if newType, exists := newTypes[typeName]; exists {
			changes := false

			currentRels := currentType.GetRelations()
			newRels := newType.GetRelations()

			// Check for new relations
			for relName := range newRels {
				if _, exists := currentRels[relName]; !exists {
					if !changes {
						diff.WriteString(fmt.Sprintf("~ Modified type: %s\n", typeName))
						changes = true
					}
					diff.WriteString(fmt.Sprintf("  + Added relation: %s\n", relName))
				}
			}

			// Check for removed relations
			for relName := range currentRels {
				if _, exists := newRels[relName]; !exists {
					if !changes {
						diff.WriteString(fmt.Sprintf("~ Modified type: %s\n", typeName))
						changes = true
					}
					diff.WriteString(fmt.Sprintf("  - Removed relation: %s\n", relName))
				}
			}

			if changes {
				diff.WriteString("\n")
			}
		}
	}

	result := diff.String()
	if result == "Model Changes:\n==============\n\n" {
		return "No changes detected"
	}

	return result
}

// readFile reads file content
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
