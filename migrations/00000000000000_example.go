package main

// Migration: example_migration
// Version: 00000000000000
//
// This is an example migration file showing common patterns
// This file is for reference only - it won't be executed unless you explicitly run it

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

func up(ctx context.Context, client *omg.Client) error {
	// ============================================================================
	// EXAMPLE 1: Rename relation with tuple migration
	// ============================================================================
	// Use case: You want to completely replace old relation name with new one
	/*
		// Step 1: Copy tuples from old to new relation
		if err := omg.CopyRelation(ctx, client, "team", "can_manage_members", "can_manage"); err != nil {
			return err
		}

		// Step 2: Wait for application to deploy with new relation name support

		// Step 3: Delete old tuples (run this in a separate migration after deploy)
		if err := omg.DeleteRelation(ctx, client, "team", "can_manage_members"); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 2: Rename object type
	// ============================================================================
	// Use case: Renaming team -> organization
	/*
		// Step 1: Migrate all tuples from old type to new type
		if err := omg.RenameType(ctx, client, "team", "organization"); err != nil {
			return err
		}

		// Note: This creates new tuples with "organization:" prefix
		// and deletes old "team:" tuples atomically
	*/

	// ============================================================================
	// EXAMPLE 3: Custom tuple transformation
	// ============================================================================
	// Use case: Change user ID format (user:123 -> user:uuid-123)
	/*
		transform := func(tuple openfgaSdk.TupleKey) (*openfgaSdk.TupleKey, error) {
			// Extract the numeric ID
			parts := strings.Split(tuple.User, ":")
			if len(parts) != 2 {
				return &tuple, nil // Skip non-standard format
			}

			// Transform to new format
			newUser := fmt.Sprintf("user:uuid-%s", parts[1])
			tuple.User = newUser
			return &tuple, nil
		}

		if err := omg.MigrateRelationWithTransform(ctx, client, "document", "viewer", "viewer", transform); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 4: Add new type with relations (MODEL OPERATIONS)
	// ============================================================================
	// Use case: Adding a new type to the authorization model
	/*
		relationMap := map[string]string{
			"owner":  "[user]",
			"editor": "[user] or owner",
			"viewer": "[user] or editor",
		}

		if err := omg.AddType(ctx, client, "document", relationMap); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 5: Add relation to existing type
	// ============================================================================
	/*
		if err := omg.AddRelation(ctx, client, "document", "admin", "[user]"); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 6: Update relation definition
	// ============================================================================
	/*
		// Change from "[user]" to "[user] or owner"
		if err := omg.UpdateRelation(ctx, client, "document", "viewer", "[user] or owner"); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 7: Remove relation (with tuple cleanup)
	// ============================================================================
	/*
		if err := omg.RemoveRelation(ctx, client, "document", "legacy_permission", true); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 8: Remove type (with tuple cleanup)
	// ============================================================================
	/*
		if err := omg.RemoveType(ctx, client, "deprecated_type", true); err != nil {
			return err
		}
	*/

	// ============================================================================
	// EXAMPLE 9: Backup before risky operation
	// ============================================================================
	/*
		// Backup all tuples
		tuples, err := omg.BackupTuples(ctx, client)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Printf("Backed up %d tuples\n", len(tuples))

		// Do risky operation
		if err := omg.RenameType(ctx, client, "team", "organization"); err != nil {
			// Restore if failed
			fmt.Println("Operation failed, restoring backup...")
			if restoreErr := omg.RestoreTuples(ctx, client, tuples); restoreErr != nil {
				return fmt.Errorf("operation failed and restore failed: %w, restore error: %w", err, restoreErr)
			}
			return err
		}
	*/

	return nil
}

func down(ctx context.Context, client *omg.Client) error {
	// Rollback operations
	// Reverse the operations from the up() function

	// Example: If up() renamed team -> organization, down() reverses it
	/*
		if err := omg.RenameType(ctx, client, "organization", "team"); err != nil {
			return err
		}
	*/

	return nil
}
