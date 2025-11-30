package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/demetere/omg/pkg"
	"github.com/joho/godotenv"
)

var (
	migrationsDir string
	dbURL         string
	modelPath     string
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Define flags
	flag.StringVar(&migrationsDir, "dir", "migrations", "directory with migration files")
	flag.StringVar(&dbURL, "dburl", "", "OpenFGA database URL (openfga://store_id@host:port?auth=...)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Parse flags from remaining arguments
	flagSet := flag.NewFlagSet(command, flag.ExitOnError)
	flagSet.StringVar(&migrationsDir, "dir", "migrations", "directory with migration files")
	flagSet.StringVar(&dbURL, "dburl", os.Getenv("OPENFGA_DATABASE_URL"), "OpenFGA database URL")
	flagSet.StringVar(&modelPath, "model", "model.fga", "path to authorization model file")
	flagSet.Parse(os.Args[2:])

	ctx := context.Background()

	// Commands that don't need OpenFGA client
	switch command {
	case "create":
		args := flagSet.Args()
		if len(args) < 1 {
			fmt.Println("Usage: omg create <migration_name>")
			os.Exit(1)
		}
		if err := createMigration(args[0]); err != nil {
			fmt.Printf("Error: Failed to create migration: %v\n", err)
			os.Exit(1)
		}
		return
	case "generate":
		args := flagSet.Args()
		name := "auto_migration"
		if len(args) >= 1 {
			name = args[0]
		}
		if err := generateMigration(name); err != nil {
			fmt.Printf("Error: Failed to generate migration: %v\n", err)
			os.Exit(1)
		}
		return
	case "diff":
		if err := showDiff(); err != nil {
			fmt.Printf("Error: Failed to show diff: %v\n", err)
			os.Exit(1)
		}
		return
	case "init":
		args := flagSet.Args()
		if len(args) < 1 {
			fmt.Println("Usage: omg init <store_name>")
			os.Exit(1)
		}
		if err := initStore(args[0]); err != nil {
			fmt.Printf("Error: Failed to initialize store: %v\n", err)
			os.Exit(1)
		}
		return
	case "list-stores":
		if err := listStores(); err != nil {
			fmt.Printf("Error: Failed to list stores: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Initialize OpenFGA client for other commands
	client, err := initOpenFGAClient()
	if err != nil {
		fmt.Printf("Error: Failed to initialize OpenFGA client: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "up":
		if err := runUp(ctx, client); err != nil {
			fmt.Printf("Error: Migration up failed: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := runDown(ctx, client); err != nil {
			fmt.Printf("Error: Migration down failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := showStatus(ctx, client); err != nil {
			fmt.Printf("Error: Failed to show status: %v\n", err)
			os.Exit(1)
		}
	case "list-tuples":
		filter := ""
		args := flagSet.Args()
		if len(args) >= 1 {
			filter = args[0]
		}
		if err := listTuples(ctx, client, filter); err != nil {
			fmt.Printf("Error: Failed to list tuples: %v\n", err)
			os.Exit(1)
		}
	case "show-model":
		if err := showModel(ctx, client); err != nil {
			fmt.Printf("Error: Failed to show model: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Error: Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("OpenFGA Migration Tool - Model-First Migrations")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  omg [options] <command>")
	fmt.Println("")
	fmt.Println("Model-First Workflow:")
	fmt.Println("  diff                Show changes between model.fga and current state")
	fmt.Println("  generate [name]     Auto-generate migration from model.fga changes")
	fmt.Println("  up                  Apply pending migrations")
	fmt.Println("  down                Rollback last migration")
	fmt.Println("  status              Show migration status")
	fmt.Println("")
	fmt.Println("Manual Migration Commands:")
	fmt.Println("  create <name>       Create blank migration file")
	fmt.Println("")
	fmt.Println("Store Management:")
	fmt.Println("  init <name>         Create a new OpenFGA store")
	fmt.Println("  list-stores         List all OpenFGA stores")
	fmt.Println("")
	fmt.Println("Utilities:")
	fmt.Println("  show-model          Show current authorization model")
	fmt.Println("  list-tuples [type]  List all tuples (optionally filtered)")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -dir string         Directory with migration files (default: migrations)")
	fmt.Println("  -dburl string       OpenFGA database URL")
	fmt.Println("  -model string       Path to authorization model file (default: model.fga)")
	fmt.Println("")
	fmt.Println("Database URL format:")
	fmt.Println("  openfga://store_id@host:port")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  OPENFGA_DATABASE_URL   - Database URL (alternative to -dburl)")
	fmt.Println("  OPENFGA_API_URL        - OpenFGA API URL (alternative)")
	fmt.Println("  OPENFGA_STORE_ID       - OpenFGA Store ID (alternative)")
	fmt.Println("  OPENFGA_AUTH_METHOD    - Auth method: none, token, client_credentials")
	fmt.Println("  OPENFGA_API_TOKEN      - API token (if auth_method=token)")
	fmt.Println("  OPENFGA_CLIENT_ID      - Client ID (if auth_method=client_credentials)")
	fmt.Println("  OPENFGA_CLIENT_SECRET  - Client secret (if auth_method=client_credentials)")
	fmt.Println("  OPENFGA_TOKEN_ISSUER   - Token issuer (optional)")
	fmt.Println("  OPENFGA_TOKEN_AUDIENCE - Token audience (optional)")
	fmt.Println("")
	fmt.Println("Typical Workflow:")
	fmt.Println("  1. Edit model.fga with your changes")
	fmt.Println("  2. Run 'omg diff' to see what changed")
	fmt.Println("  3. Run 'omg generate my_feature' to create migration")
	fmt.Println("  4. Review and edit the generated migration if needed")
	fmt.Println("  5. Run 'omg up' to apply migrations")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  omg diff                               # See model changes")
	fmt.Println("  omg diff -model custom/auth.fga        # Use custom model file")
	fmt.Println("  omg generate add_files                 # Generate migration from model changes")
	fmt.Println("  omg generate -model custom/auth.fga    # Generate with custom model")
	fmt.Println("  omg up                                 # Apply migrations")
	fmt.Println("  omg down                               # Rollback last migration")
	fmt.Println("  omg status                             # Check migration status")
}

func initOpenFGAClient() (*omg.Client, error) {
	var cfg omg.Config

	// Try to parse database URL first
	if dbURL != "" {
		parsedCfg, err := parseDBURL(dbURL)
		if err != nil {
			return nil, fmt.Errorf("invalid database URL: %w", err)
		}
		cfg = parsedCfg
	} else {
		// Fall back to environment variables
		cfg = omg.Config{
			ApiURL:        os.Getenv("OPENFGA_API_URL"),
			StoreID:       os.Getenv("OPENFGA_STORE_ID"),
			AuthMethod:    os.Getenv("OPENFGA_AUTH_METHOD"),
			APIToken:      os.Getenv("OPENFGA_API_TOKEN"),
			ClientID:      os.Getenv("OPENFGA_CLIENT_ID"),
			ClientSecret:  os.Getenv("OPENFGA_CLIENT_SECRET"),
			TokenIssuer:   os.Getenv("OPENFGA_TOKEN_ISSUER"),
			TokenAudience: os.Getenv("OPENFGA_TOKEN_AUDIENCE"),
		}
	}

	return omg.NewClient(cfg)
}

// parseDBURL parses a database URL in the format:
// openfga://store_id@host:port
func parseDBURL(dburl string) (omg.Config, error) {
	u, err := url.Parse(dburl)
	if err != nil {
		return omg.Config{}, err
	}

	if u.Scheme != "openfga" {
		return omg.Config{}, fmt.Errorf("invalid scheme: expected 'openfga', got '%s'", u.Scheme)
	}

	storeID := u.User.Username()
	if storeID == "" {
		return omg.Config{}, fmt.Errorf("store ID is required")
	}

	host := u.Host
	if host == "" {
		return omg.Config{}, fmt.Errorf("host is required")
	}

	// Build API URL
	scheme := "https"
	if u.Query().Get("tls") == "false" {
		scheme = "http"
	}
	apiURL := fmt.Sprintf("%s://%s", scheme, host)

	cfg := omg.Config{
		ApiURL:  apiURL,
		StoreID: storeID,
	}

	// Parse query parameters for auth
	query := u.Query()
	if authMethod := query.Get("auth"); authMethod != "" {
		cfg.AuthMethod = authMethod
	}
	if token := query.Get("token"); token != "" {
		cfg.APIToken = token
	}
	if clientID := query.Get("client_id"); clientID != "" {
		cfg.ClientID = clientID
	}
	if clientSecret := query.Get("client_secret"); clientSecret != "" {
		cfg.ClientSecret = clientSecret
	}
	if issuer := query.Get("issuer"); issuer != "" {
		cfg.TokenIssuer = issuer
	}
	if audience := query.Get("audience"); audience != "" {
		cfg.TokenAudience = audience
	}

	return cfg, nil
}

func runUp(ctx context.Context, client *omg.Client) error {
	tracker := omg.NewTracker(client)

	// Find all migration files in directory
	pattern := filepath.Join(migrationsDir, "*_*.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Filter out non-migration files
	var migrationFiles []string
	for _, file := range files {
		base := filepath.Base(file)
		// Skip migrations.go and example files
		if base == "migrations.go" || strings.Contains(base, "example") {
			continue
		}
		migrationFiles = append(migrationFiles, file)
	}

	// Sort by version (timestamp in filename)
	sort.Strings(migrationFiles)

	applied, err := tracker.GetApplied(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, file := range migrationFiles {
		version := extractVersionFromFilename(file)
		name := extractNameFromFilename(file)

		if _, exists := applied[version]; exists {
			continue
		}

		fmt.Printf("OK  %s  %s\n", version, name)

		// Run the migration file with 'go run'
		cmd := exec.Command("go", "run", file, "up")
		cmd.Env = os.Environ() // Pass through all environment variables
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("migration %s failed: %w", version, err)
		}

		if err := tracker.Record(ctx, version, name); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		count++
	}

	if count == 0 {
		fmt.Println("No migrations to run. Current version: up to date")
	} else {
		fmt.Println("\n✓ All migrations applied successfully")
	}

	return nil
}

// extractVersionFromFilename extracts the version (timestamp) from a migration filename
// Example: "migrations/20251130123456_add_feature.go" -> "20251130123456"
func extractVersionFromFilename(filename string) string {
	base := filepath.Base(filename)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// extractNameFromFilename extracts the name from a migration filename
// Example: "migrations/20251130123456_add_feature.go" -> "add_feature"
func extractNameFromFilename(filename string) string {
	base := filepath.Base(filename)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return ""
	}
	name := strings.TrimSuffix(parts[1], ".go")
	return name
}

func runDown(ctx context.Context, client *omg.Client) error {
	tracker := omg.NewTracker(client)
	applied, err := tracker.GetApplied(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("No migrations to roll back")
		return nil
	}

	// Find all migration files in directory
	pattern := filepath.Join(migrationsDir, "*_*.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Filter out non-migration files
	var migrationFiles []string
	for _, file := range files {
		base := filepath.Base(file)
		if base == "migrations.go" || strings.Contains(base, "example") {
			continue
		}
		migrationFiles = append(migrationFiles, file)
	}

	// Sort by version (timestamp in filename) in reverse order
	sort.Sort(sort.Reverse(sort.StringSlice(migrationFiles)))

	// Find the last applied migration
	var lastMigrationFile string
	var lastVersion string
	var lastName string

	for _, file := range migrationFiles {
		version := extractVersionFromFilename(file)
		if _, exists := applied[version]; exists {
			lastMigrationFile = file
			lastVersion = version
			lastName = extractNameFromFilename(file)
			break
		}
	}

	if lastMigrationFile == "" {
		fmt.Println("No migrations to roll back")
		return nil
	}

	fmt.Printf("OK  %s  %s\n", lastVersion, lastName)

	// Run the migration file with 'go run' and 'down' argument
	cmd := exec.Command("go", "run", lastMigrationFile, "down")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rollback %s failed: %w", lastVersion, err)
	}

	if err := tracker.Remove(ctx, lastVersion); err != nil {
		return fmt.Errorf("failed to remove migration record %s: %w", lastVersion, err)
	}

	return nil
}

func showStatus(ctx context.Context, client *omg.Client) error {
	tracker := omg.NewTracker(client)

	// Find all migration files in directory
	pattern := filepath.Join(migrationsDir, "*_*.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// Filter out non-migration files
	var migrationFiles []string
	for _, file := range files {
		base := filepath.Base(file)
		if base == "migrations.go" || strings.Contains(base, "example") {
			continue
		}
		migrationFiles = append(migrationFiles, file)
	}

	// Sort by version (timestamp in filename)
	sort.Strings(migrationFiles)

	applied, err := tracker.GetApplied(ctx)
	if err != nil {
		return err
	}

	if len(migrationFiles) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	fmt.Printf("Migration status for directory '%s'\n", migrationsDir)
	for _, file := range migrationFiles {
		version := extractVersionFromFilename(file)
		name := extractNameFromFilename(file)

		status := "Pending"
		if info, exists := applied[version]; exists {
			status = fmt.Sprintf("Applied At: %s", info.AppliedAt.Format("Mon Jan  2 15:04:05 2006"))
		}
		fmt.Printf("    %-15s  %-40s  %s\n", version, name, status)
	}

	return nil
}

func createMigration(name string) error {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s/%s_%s.go", migrationsDir, timestamp, name)

	template := fmt.Sprintf(`package main

// Migration: %s
// Version: %s

import (
	"context"
	"fmt"
	"os"

	"github.com/demetere/omg/pkg"
)

func main() {
	// Get connection info from environment
	client, err := omg.NewClient(
		os.Getenv("OPENFGA_API_URL"),
		os.Getenv("OPENFGA_STORE_ID"),
		os.Getenv("OPENFGA_API_TOKEN"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Check if we should run Up or Down
	if len(os.Args) > 1 && os.Args[1] == "down" {
		if err := down(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "Migration down failed: %%v\n", err)
			os.Exit(1)
		}
	} else {
		if err := up(ctx, client); err != nil {
			fmt.Fprintf(os.Stderr, "Migration up failed: %%v\n", err)
			os.Exit(1)
		}
	}
}

func up(ctx context.Context, client *omg.Client) error {
	// TODO: Implement migration
	//
	// Available omg functions:
	//
	// MODEL OPERATIONS:
	// - omg.GetCurrentModel(ctx, client) - Get current model as DSL string
	//
	// TUPLE OPERATIONS:
	// - omg.RenameRelation(ctx, client, objectType, oldRel, newRel) - Rename relation on all tuples
	// - omg.RenameType(ctx, client, oldType, newType) - Rename object type on all tuples
	// - omg.CopyRelation(ctx, client, objectType, sourceRel, targetRel) - Copy tuples to new relation
	// - omg.DeleteRelation(ctx, client, objectType, relation) - Delete all tuples with relation
	// - omg.MigrateRelationWithTransform(ctx, client, objectType, oldRel, newRel, transform) - Custom transform
	//
	// READ OPERATIONS:
	// - omg.ReadAllTuples(ctx, client, objectType, relation) - Read tuples by type/relation
	// - omg.CountTuples(ctx, client, objectType, relation) - Count matching tuples
	//
	// BATCH OPERATIONS:
	// - omg.WriteTuplesBatch(ctx, client, tuples) - Write tuples in batches
	// - omg.DeleteTuplesBatch(ctx, client, tuples) - Delete tuples in batches
	//
	// UTILITY:
	// - omg.BackupTuples(ctx, client) - Backup all tuples before migration
	// - omg.RestoreTuples(ctx, client, tuples) - Restore tuples from backup

	return nil
}

func down(ctx context.Context, client *omg.Client) error {
	// TODO: Implement rollback
	// Reverse the operations from Up

	return nil
}
`, name, timestamp)

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
		return err
	}

	fmt.Printf("Created migration file: %s\n", filename)
	return nil
}

func listTuples(ctx context.Context, client *omg.Client, filter string) error {
	req := omg.ReadTuplesRequest{}

	if filter != "" {
		req.Object = filter + ":"
	}

	tuples, err := client.ReadAllTuples(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d tuples:\n\n", len(tuples))

	for _, tuple := range tuples {
		fmt.Printf("%s  %s  %s\n", tuple.User, tuple.Relation, tuple.Object)
	}

	return nil
}

func showModel(ctx context.Context, client *omg.Client) error {
	model, err := client.GetCurrentModel(ctx)
	if err != nil {
		return err
	}

	fmt.Println(model)

	return nil
}

func initStore(storeName string) error {
	// Get API URL from environment or dbURL
	apiURL := os.Getenv("OPENFGA_API_URL")
	if apiURL == "" && dbURL != "" {
		cfg, err := parseDBURL(dbURL)
		if err != nil {
			return fmt.Errorf("invalid database URL: %w", err)
		}
		apiURL = cfg.ApiURL
	}

	if apiURL == "" {
		return fmt.Errorf("OPENFGA_API_URL or -dburl is required")
	}

	fmt.Printf("Creating OpenFGA store '%s'...\n", storeName)

	storeID, err := omg.CreateStore(apiURL, storeName)
	if err != nil {
		return err
	}

	fmt.Printf("\nStore created successfully!\n\n")
	fmt.Printf("Store ID: %s\n", storeID)
	fmt.Printf("Store Name: %s\n\n", storeName)
	fmt.Println("Add to your environment:")
	fmt.Printf("  export OPENFGA_STORE_ID=%s\n", storeID)
	fmt.Printf("  export OPENFGA_DATABASE_URL=openfga://%s@%s\n", storeID, strings.TrimPrefix(apiURL, "http://"))
	fmt.Println("")
	fmt.Println("Or use with -dburl flag:")
	fmt.Printf("  omg -dburl openfga://%s@%s up\n", storeID, strings.TrimPrefix(apiURL, "http://"))

	return nil
}

func listStores() error {
	// Get API URL from environment or dbURL
	apiURL := os.Getenv("OPENFGA_API_URL")
	if apiURL == "" && dbURL != "" {
		cfg, err := parseDBURL(dbURL)
		if err != nil {
			return fmt.Errorf("invalid database URL: %w", err)
		}
		apiURL = cfg.ApiURL
	}

	if apiURL == "" {
		return fmt.Errorf("OPENFGA_API_URL or -dburl is required")
	}

	stores, err := omg.ListStores(apiURL)
	if err != nil {
		return err
	}

	if len(stores) == 0 {
		fmt.Println("No stores found")
		return nil
	}

	fmt.Printf("Found %d store(s):\n\n", len(stores))
	for _, store := range stores {
		fmt.Printf("  ID:   %s\n", store.ID)
		fmt.Printf("  Name: %s\n\n", store.Name)
	}

	return nil
}

func generateMigration(name string) error {
	fmt.Println("Detecting model changes...")

	// Create client to query OpenFGA
	client, err := initOpenFGAClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	ctx := context.Background()

	// Load current state from OpenFGA
	fmt.Println("Querying OpenFGA for current model...")
	oldState, err := omg.LoadModelStateFromOpenFGA(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to load current model from OpenFGA: %w\nMake sure OpenFGA is running and accessible", err)
	}

	// Load desired model from file
	newModelDSL, err := omg.LoadCurrentModelFromPath(modelPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", modelPath, err)
	}

	// Parse desired model
	newModel, err := omg.ParseDSLToModel(newModelDSL)
	if err != nil {
		return fmt.Errorf("failed to parse model.fga: %w", err)
	}

	// Build desired state
	newState := omg.BuildModelState(newModel)

	// Detect changes
	changes := omg.DetectChanges(oldState, newState)
	if len(changes) == 0 {
		fmt.Println("No changes detected")
		return nil
	}

	// Detect potential renames
	changes = omg.DetectPotentialRenames(changes, oldState, newState)

	// Print detected changes
	fmt.Printf("\nDetected %d change(s):\n", len(changes))
	for i, change := range changes {
		fmt.Printf("  %d. %s\n", i+1, change.Details)
	}

	// Ask for confirmation on potential renames
	confirmedChanges, err := confirmChanges(changes)
	if err != nil {
		return err
	}

	// Generate migration
	fmt.Println("\nGenerating migration...")
	filename, err := omg.GenerateMigrationFromChanges(confirmedChanges, name, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	fmt.Printf("\n✓ Migration created: %s\n", filename)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated migration file")
	fmt.Println("  2. Edit if needed (especially for renames)")
	fmt.Println("  3. Run 'omg up' to apply the migration")

	return nil
}

func showDiff() error {
	fmt.Printf("Comparing %s with OpenFGA...\n", modelPath)

	// Create client to query OpenFGA
	client, err := initOpenFGAClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	ctx := context.Background()

	// Load current state from OpenFGA
	oldState, err := omg.LoadModelStateFromOpenFGA(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to load current model from OpenFGA: %w\nMake sure OpenFGA is running and accessible", err)
	}

	// Load desired model from file
	newModelDSL, err := omg.LoadCurrentModelFromPath(modelPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", modelPath, err)
	}

	// Parse desired model
	newModel, err := omg.ParseDSLToModel(newModelDSL)
	if err != nil {
		return fmt.Errorf("failed to parse model.fga: %w", err)
	}

	// Build desired state
	newState := omg.BuildModelState(newModel)

	// Detect changes
	changes := omg.DetectChanges(oldState, newState)
	if len(changes) == 0 {
		fmt.Println("\n✓ No changes detected - model.fga matches current state")
		return nil
	}

	// Detect potential renames
	changes = omg.DetectPotentialRenames(changes, oldState, newState)

	// Print changes
	fmt.Printf("\nDetected %d change(s):\n\n", len(changes))
	for _, change := range changes {
		symbol := getChangeSymbol(change.Type)
		fmt.Printf("%s %s\n", symbol, change.Details)

		switch change.Type {
		case omg.ChangeTypeRenameType, omg.ChangeTypeRenameRelation:
			fmt.Printf("    Old: %s\n", change.OldValue)
			fmt.Printf("    New: %s\n", change.NewValue)
		}
	}

	fmt.Println("\nRun 'omg generate <name>' to create a migration for these changes")
	return nil
}

func getChangeSymbol(changeType omg.ChangeType) string {
	switch changeType {
	case omg.ChangeTypeAddType, omg.ChangeTypeAddRelation:
		return "+"
	case omg.ChangeTypeRemoveType, omg.ChangeTypeRemoveRelation:
		return "-"
	case omg.ChangeTypeUpdateRelation:
		return "~"
	case omg.ChangeTypeRenameType, omg.ChangeTypeRenameRelation:
		return "→"
	default:
		return "•"
	}
}

func confirmChanges(changes []omg.ModelChange) ([]omg.ModelChange, error) {
	// Process changes with confidence-aware handling
	var confirmed []omg.ModelChange
	for _, change := range changes {
		if change.Type == omg.ChangeTypeRenameType || change.Type == omg.ChangeTypeRenameRelation {
			// Handle based on confidence level
			switch change.Confidence {
			case omg.ConfidenceHigh:
				// High confidence: keep as rename, inform user
				fmt.Printf("\n✓ Rename detected: %s -> %s (high confidence)\n", change.OldValue, change.NewValue)
				fmt.Println("   Will generate rename migration that preserves tuples.")
				confirmed = append(confirmed, change)

			case omg.ConfidenceMedium:
				// Medium confidence: keep as rename but warn user to review
				fmt.Printf("\n⚠  Possible rename: %s -> %s (medium confidence - review required)\n", change.OldValue, change.NewValue)
				fmt.Println("   Will generate rename migration - review before applying.")
				confirmed = append(confirmed, change)

			case omg.ConfidenceLow:
				// Low confidence: keep as rename, generator will create commented code
				fmt.Printf("\n⚠  Potential rename: %s -> %s (low confidence)\n", change.OldValue, change.NewValue)
				fmt.Println("   Will generate both options - uncomment the rename if confirmed.")
				confirmed = append(confirmed, change)

			default:
				// No confidence info (legacy): treat conservatively
				fmt.Printf("\n⚠  Detected potential rename: %s -> %s\n", change.OldValue, change.NewValue)
				fmt.Println("   Will generate rename migration - review carefully.")
				confirmed = append(confirmed, change)
			}
		} else {
			confirmed = append(confirmed, change)
		}
	}

	return confirmed, nil
}
