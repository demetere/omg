package omg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	openfgaSdk "github.com/openfga/go-sdk"
)

const (
	modelFile = "model.fga"
)

// ModelState represents the state of an authorization model
// This is built from either OpenFGA (current state) or model.fga (desired state)
type ModelState struct {
	Types map[string]TypeState
}

// TypeState represents the state of a single type
type TypeState struct {
	Name      string            `json:"name"`
	Relations map[string]string `json:"relations"` // relation name -> definition
}

// ModelChange represents a detected change in the model
type ModelChange struct {
	Type         ChangeType
	TypeName     string
	RelationName string
	OldValue     string
	NewValue     string
	Details      string
	Confidence   ConfidenceLevel // For renames: high = auto-apply, medium = needs review, low = suggest only
}

// ChangeType represents the kind of change detected
type ChangeType string

const (
	ChangeTypeAddType         ChangeType = "add_type"
	ChangeTypeRemoveType      ChangeType = "remove_type"
	ChangeTypeRenameType      ChangeType = "rename_type"      // Requires user confirmation
	ChangeTypeAddRelation     ChangeType = "add_relation"
	ChangeTypeRemoveRelation  ChangeType = "remove_relation"
	ChangeTypeRenameRelation  ChangeType = "rename_relation"  // Requires user confirmation
	ChangeTypeUpdateRelation  ChangeType = "update_relation"
)

// ConfidenceLevel represents how confident we are about a rename detection
type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"   // Very likely a rename (e.g., team_member → teamMember)
	ConfidenceMedium ConfidenceLevel = "medium" // Possibly a rename, needs review (e.g., team → organization)
	ConfidenceLow    ConfidenceLevel = "low"    // Unlikely, but suggest as option
	ConfidenceNone   ConfidenceLevel = ""       // Not applicable (for non-rename changes)
)

// LoadModelStateFromOpenFGA loads the current model state from OpenFGA
func LoadModelStateFromOpenFGA(ctx context.Context, client *Client) (*ModelState, error) {
	model, err := client.GetCurrentAuthorizationModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current model from OpenFGA: %w", err)
	}

	return BuildModelStateFromAuthorizationModel(model), nil
}

// BuildModelStateFromAuthorizationModel converts an OpenFGA authorization model to ModelState
func BuildModelStateFromAuthorizationModel(model openfgaSdk.AuthorizationModel) *ModelState {
	state := &ModelState{
		Types: make(map[string]TypeState),
	}

	for _, typeDef := range model.GetTypeDefinitions() {
		typeName := typeDef.GetType()
		typeState := TypeState{
			Name:      typeName,
			Relations: make(map[string]string),
		}

		relations := typeDef.GetRelations()
		for relName, relDef := range relations {
			typeState.Relations[relName] = serializeUserset(relDef)
		}

		state.Types[typeName] = typeState
	}

	return state
}

// LoadCurrentModel loads the current model from model.fga file
func LoadCurrentModel() (string, error) {
	return LoadCurrentModelFromPath(modelFile)
}

// LoadCurrentModelFromPath loads the current model from a specified file path
func LoadCurrentModelFromPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	return string(data), nil
}


// BuildModelState builds a ModelState from a parsed model (from model.fga)
func BuildModelState(model openfgaSdk.AuthorizationModel) *ModelState {
	state := &ModelState{
		Types: make(map[string]TypeState),
	}

	for _, typeDef := range model.TypeDefinitions {
		typeState := TypeState{
			Name:      typeDef.Type,
			Relations: make(map[string]string),
		}

		relations := typeDef.GetRelations()
		for relName, relDef := range relations {
			// Convert relation definition to string for comparison
			typeState.Relations[relName] = serializeUserset(relDef)
		}

		state.Types[typeDef.Type] = typeState
	}

	return state
}

// serializeUserset converts a Userset to a string representation
func serializeUserset(userset openfgaSdk.Userset) string {
	// Simplified serialization - good enough for comparison
	data, _ := json.Marshal(userset)
	return string(data)
}

// DetectChanges compares old and new model states and returns detected changes
func DetectChanges(oldState, newState *ModelState) []ModelChange {
	var changes []ModelChange

	// Detect type changes
	oldTypes := oldState.Types
	newTypes := newState.Types

	// Find added types
	for typeName, typeState := range newTypes {
		if _, exists := oldTypes[typeName]; !exists {
			changes = append(changes, ModelChange{
				Type:     ChangeTypeAddType,
				TypeName: typeName,
				Details:  fmt.Sprintf("New type '%s' with %d relations", typeName, len(typeState.Relations)),
			})
		}
	}

	// Find removed types
	for typeName := range oldTypes {
		if _, exists := newTypes[typeName]; !exists {
			changes = append(changes, ModelChange{
				Type:     ChangeTypeRemoveType,
				TypeName: typeName,
				Details:  fmt.Sprintf("Type '%s' removed", typeName),
			})
		}
	}

	// Find modified types (relation changes)
	for typeName, newTypeState := range newTypes {
		oldTypeState, exists := oldTypes[typeName]
		if !exists {
			continue // Already handled as added type
		}

		// Compare relations
		oldRels := oldTypeState.Relations
		newRels := newTypeState.Relations

		// Added relations
		for relName, relDef := range newRels {
			if _, exists := oldRels[relName]; !exists {
				changes = append(changes, ModelChange{
					Type:         ChangeTypeAddRelation,
					TypeName:     typeName,
					RelationName: relName,
					NewValue:     relDef,
					Details:      fmt.Sprintf("Added relation '%s.%s'", typeName, relName),
				})
			}
		}

		// Removed relations
		for relName, relDef := range oldRels {
			if _, exists := newRels[relName]; !exists {
				changes = append(changes, ModelChange{
					Type:         ChangeTypeRemoveRelation,
					TypeName:     typeName,
					RelationName: relName,
					OldValue:     relDef,
					Details:      fmt.Sprintf("Removed relation '%s.%s'", typeName, relName),
				})
			}
		}

		// Modified relations
		for relName, newRelDef := range newRels {
			oldRelDef, exists := oldRels[relName]
			if exists && oldRelDef != newRelDef {
				changes = append(changes, ModelChange{
					Type:         ChangeTypeUpdateRelation,
					TypeName:     typeName,
					RelationName: relName,
					OldValue:     oldRelDef,
					NewValue:     newRelDef,
					Details:      fmt.Sprintf("Updated relation '%s.%s' definition", typeName, relName),
				})
			}
		}
	}

	return changes
}

// DetectPotentialRenames attempts to detect renames by looking for similar type/relation names
// It uses name similarity and relation similarity to determine confidence levels
func DetectPotentialRenames(changes []ModelChange, oldState, newState *ModelState) []ModelChange {
	var enhanced []ModelChange

	// Group changes by type
	var addedTypes []ModelChange
	var removedTypes []ModelChange
	var addedRelations []ModelChange
	var removedRelations []ModelChange

	for _, change := range changes {
		switch change.Type {
		case ChangeTypeAddType:
			addedTypes = append(addedTypes, change)
		case ChangeTypeRemoveType:
			removedTypes = append(removedTypes, change)
		case ChangeTypeAddRelation:
			addedRelations = append(addedRelations, change)
		case ChangeTypeRemoveRelation:
			removedRelations = append(removedRelations, change)
		default:
			enhanced = append(enhanced, change)
		}
	}

	// Detect type renames (1 removed + 1 added = potential rename)
	usedRemovals := make(map[int]bool)
	usedAdditions := make(map[int]bool)

	for i, removed := range removedTypes {
		bestMatch := -1
		bestConfidence := ConfidenceNone
		bestNameSim := 0.0
		bestRelSim := 0.0

		for j, added := range addedTypes {
			if usedAdditions[j] {
				continue
			}

			// Calculate name similarity
			nameSim := calculateSimilarity(removed.TypeName, added.TypeName)

			// Calculate relation similarity (if we have access to type states)
			relSim := 0.0
			if oldState != nil && newState != nil {
				oldTypeState, oldExists := oldState.Types[removed.TypeName]
				newTypeState, newExists := newState.Types[added.TypeName]
				if oldExists && newExists {
					relSim = haveSimilarRelations(oldTypeState, newTypeState)
				}
			}

			// Determine confidence
			confidence := determineRenameConfidence(nameSim, relSim)

			// Track the best match
			if confidence != ConfidenceNone && (bestMatch == -1 || confidence > bestConfidence ||
				(confidence == bestConfidence && nameSim > bestNameSim)) {
				bestMatch = j
				bestConfidence = confidence
				bestNameSim = nameSim
				bestRelSim = relSim
			}
		}

		// If we found a potential rename, add it
		if bestMatch != -1 {
			added := addedTypes[bestMatch]

			detailsMsg := fmt.Sprintf("Rename detected: '%s' -> '%s'", removed.TypeName, added.TypeName)
			switch bestConfidence {
			case ConfidenceHigh:
				detailsMsg = fmt.Sprintf("Rename detected: '%s' -> '%s' (high confidence: %.0f%% name, %.0f%% relations)",
					removed.TypeName, added.TypeName, bestNameSim*100, bestRelSim*100)
			case ConfidenceMedium:
				detailsMsg = fmt.Sprintf("Possible rename: '%s' -> '%s' (medium confidence - review required)",
					removed.TypeName, added.TypeName)
			case ConfidenceLow:
				detailsMsg = fmt.Sprintf("Potential rename: '%s' -> '%s' (low confidence - verify before using)",
					removed.TypeName, added.TypeName)
			}

			enhanced = append(enhanced, ModelChange{
				Type:       ChangeTypeRenameType,
				TypeName:   removed.TypeName,
				OldValue:   removed.TypeName,
				NewValue:   added.TypeName,
				Confidence: bestConfidence,
				Details:    detailsMsg,
			})
			usedRemovals[i] = true
			usedAdditions[bestMatch] = true
		}
	}

	// Add remaining type changes that weren't matched as renames
	for i, change := range removedTypes {
		if !usedRemovals[i] {
			enhanced = append(enhanced, change)
		}
	}
	for i, change := range addedTypes {
		if !usedAdditions[i] {
			enhanced = append(enhanced, change)
		}
	}

	// Similar logic for relation renames within the same type
	relationsByType := make(map[string]struct {
		added   []ModelChange
		removed []ModelChange
	})

	for _, change := range addedRelations {
		entry := relationsByType[change.TypeName]
		entry.added = append(entry.added, change)
		relationsByType[change.TypeName] = entry
	}

	for _, change := range removedRelations {
		entry := relationsByType[change.TypeName]
		entry.removed = append(entry.removed, change)
		relationsByType[change.TypeName] = entry
	}

	for typeName, relations := range relationsByType {
		usedRemovals := make(map[int]bool)
		usedAdditions := make(map[int]bool)

		for i, removed := range relations.removed {
			bestMatch := -1
			bestConfidence := ConfidenceNone
			bestSim := 0.0

			for j, added := range relations.added {
				if usedAdditions[j] {
					continue
				}

				// Calculate relation name similarity
				sim := calculateSimilarity(removed.RelationName, added.RelationName)

				// For relations, we only use name similarity (no sub-structure to compare)
				confidence := determineRenameConfidence(sim, 0.0)

				if confidence != ConfidenceNone && (bestMatch == -1 || confidence > bestConfidence ||
					(confidence == bestConfidence && sim > bestSim)) {
					bestMatch = j
					bestConfidence = confidence
					bestSim = sim
				}
			}

			// If we found a potential rename, add it
			if bestMatch != -1 {
				added := relations.added[bestMatch]

				detailsMsg := ""
				switch bestConfidence {
				case ConfidenceHigh:
					detailsMsg = fmt.Sprintf("Rename detected: '%s.%s' -> '%s.%s' (high confidence: %.0f%%)",
						typeName, removed.RelationName, typeName, added.RelationName, bestSim*100)
				case ConfidenceMedium:
					detailsMsg = fmt.Sprintf("Possible rename: '%s.%s' -> '%s.%s' (medium confidence - review required)",
						typeName, removed.RelationName, typeName, added.RelationName)
				case ConfidenceLow:
					detailsMsg = fmt.Sprintf("Potential rename: '%s.%s' -> '%s.%s' (low confidence - verify before using)",
						typeName, removed.RelationName, typeName, added.RelationName)
				}

				enhanced = append(enhanced, ModelChange{
					Type:         ChangeTypeRenameRelation,
					TypeName:     typeName,
					RelationName: removed.RelationName,
					OldValue:     removed.RelationName,
					NewValue:     added.RelationName,
					Confidence:   bestConfidence,
					Details:      detailsMsg,
				})
				usedRemovals[i] = true
				usedAdditions[bestMatch] = true
			}
		}

		// Add remaining changes
		for i, change := range relations.removed {
			if !usedRemovals[i] {
				enhanced = append(enhanced, change)
			}
		}
		for i, change := range relations.added {
			if !usedAdditions[i] {
				enhanced = append(enhanced, change)
			}
		}
	}

	return enhanced
}

// areSimilar checks if two names are similar (basic heuristic)
func areSimilar(name1, name2 string) bool {
	// Convert to lowercase for comparison
	n1 := strings.ToLower(name1)
	n2 := strings.ToLower(name2)

	// Check if one contains the other
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return true
	}

	// Check Levenshtein distance (simplified)
	distance := levenshteinDistance(n1, n2)
	maxLen := len(n1)
	if len(n2) > maxLen {
		maxLen = len(n2)
	}

	// Similar if distance is less than 40% of max length
	return float64(distance)/float64(maxLen) < 0.4
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// calculateSimilarity returns a similarity score between 0.0 (completely different) and 1.0 (identical)
func calculateSimilarity(name1, name2 string) float64 {
	n1 := strings.ToLower(name1)
	n2 := strings.ToLower(name2)

	// Identical names
	if n1 == n2 {
		return 1.0
	}

	// Check if one contains the other (very high similarity)
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		shorter := len(n1)
		if len(n2) < shorter {
			shorter = len(n2)
		}
		longer := len(n1)
		if len(n2) > longer {
			longer = len(n2)
		}
		return float64(shorter) / float64(longer) // e.g., "team" in "team_member" = 0.73
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(n1, n2)
	maxLen := len(n1)
	if len(n2) > maxLen {
		maxLen = len(n2)
	}

	// Convert distance to similarity (1.0 - normalized distance)
	return 1.0 - (float64(distance) / float64(maxLen))
}

// determineRenameConfidence determines confidence level based on similarity score and relation similarity
func determineRenameConfidence(nameSimilarity float64, relationSimilarity float64) ConfidenceLevel {
	// High confidence: very similar names OR similar relations with decent name match
	if nameSimilarity >= 0.7 || (nameSimilarity >= 0.4 && relationSimilarity >= 0.7) {
		return ConfidenceHigh
	}

	// Medium confidence: somewhat similar names OR very similar relations
	if nameSimilarity >= 0.3 || relationSimilarity >= 0.7 {
		return ConfidenceMedium
	}

	// Low confidence: might be worth suggesting
	if nameSimilarity >= 0.2 || relationSimilarity >= 0.5 {
		return ConfidenceLow
	}

	// Too dissimilar - not a rename
	return ConfidenceNone
}

// haveSimilarRelations checks if two types have similar relation structures
func haveSimilarRelations(removedTypeState, addedTypeState TypeState) float64 {
	if len(removedTypeState.Relations) == 0 && len(addedTypeState.Relations) == 0 {
		return 0.0 // No relations to compare
	}

	if len(removedTypeState.Relations) == 0 || len(addedTypeState.Relations) == 0 {
		return 0.0 // One has relations, other doesn't
	}

	// Count matching relation names
	matchingRelations := 0
	for oldRelName := range removedTypeState.Relations {
		if _, exists := addedTypeState.Relations[oldRelName]; exists {
			matchingRelations++
		}
	}

	// Calculate Jaccard similarity: intersection / union
	totalRelations := len(removedTypeState.Relations) + len(addedTypeState.Relations) - matchingRelations
	if totalRelations == 0 {
		return 0.0
	}

	return float64(matchingRelations) / float64(totalRelations)
}
