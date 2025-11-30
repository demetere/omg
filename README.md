# OpenFGA Migration Tool (OMG)

A **model-first migration tool** for OpenFGA authorization models and relationship tuples. Inspired by database migration tools like Prisma and Entity Framework, OMG automatically generates migrations from your authorization model changes.

Think "Prisma for OpenFGA" - edit your model, generate migrations, apply changes.

## ✨ Features

### Model-First Workflow
- **Automatic migration generation** from `model.fga` changes
- **Intelligent rename detection** with confidence levels (High/Medium/Low)
- **Multi-factor analysis** - considers both name and relation structure similarity
- **Context-aware templates** - generated code reflects confidence in changes
- **Safe by default** - prevents accidental data loss with smart defaults

### Migration Management
- **Version-controlled migrations** - Track changes to authorization models and tuples
- **Up/Down migrations** - Apply and rollback changes safely
- **Migration tracking via tuples** - Uses OpenFGA itself to track migration history
- **Direct API queries** - Queries OpenFGA for current model state (no cache files)

### Developer Experience
- **Rich helper functions** - Common operations like rename, copy, transform
- **Batch operations** - Efficiently handle large numbers of tuples
- **Comprehensive testing** - Unit and integration tests with testcontainers
- **CLI interface** - Simple commands: `diff`, `generate`, `up`, `down`

## 🚀 Quick Start (Model-First)

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/demetere/omg.git
cd omg

# Build the CLI
go build -o omg ./cmd/omg

# Optional: Install globally
go install ./cmd/omg
```

### 2. Configure Environment

Create a `.env` file:

```env
OPENFGA_API_URL=http://localhost:8080
OPENFGA_STORE_ID=your-store-id
OPENFGA_AUTH_METHOD=none
```

### 3. Initialize Your Store

```bash
./omg init my-store-name
```

This creates:
- `.omg/` directory for tracking state
- `model.fga.example` as a starting point

### 4. Create Your Authorization Model

Create or edit `model.fga`:

```
model
  schema 1.1

type user

type document
  relations
    define owner: [user]
    define editor: [user]
    define viewer: [user] or editor
```

### 5. See What Changed

```bash
./omg diff
```

Output:
```
Comparing model.fga with current state...

Detected 2 change(s):

+ New type 'document' with 3 relations
+ New type 'user' with 0 relations

Run 'omg generate <name>' to create a migration for these changes
```

### 6. Generate Migration

```bash
./omg generate initial_model
```

Output:
```
Generating migration...

✓ Migration created: migrations/20241128150000_initial_model.go

Next steps:
  1. Review the generated migration file
  2. Edit if needed
  3. Run 'omg up' to apply the migration
```

### 7. Review & Apply

```bash
# Review the generated code
cat migrations/20241128150000_initial_model.go

# Apply the migration
./omg up
```

## 🎯 Model-First Workflow Examples

### Example 1: High Confidence Rename

**Scenario:** Rename `document` → `documents` (very similar names)

```bash
# Edit model.fga: change 'type document' to 'type documents'
vim model.fga

# Check changes
./omg diff
```

Output:
```
→ Rename detected: 'document' -> 'documents' (high confidence: 88% name, 100% relations)
    Old: document
    New: documents
```

```bash
# Generate migration
./omg generate pluralize_documents
```

**Generated code** (clean rename):
```go
// Rename type: document -> documents (high confidence rename detected)
// This will migrate all existing tuples to the new type name
if err := omg.RenameType(ctx, client, "document", "documents"); err != nil {
    return fmt.Errorf("failed to rename type: %w", err)
}
```

✅ **High confidence = clean code, usually safe to apply**

---

### Example 2: Medium Confidence Rename

**Scenario:** Rename `document` → `file` with same relations

```bash
# Edit model.fga: change 'type document' to 'type file'
vim model.fga

./omg diff
```

Output:
```
→ Possible rename: 'document' -> 'file' (medium confidence - review required)
    Old: document
    New: file
```

```bash
./omg generate rename_to_file
```

**Generated code** (with review warning):
```go
// ⚠️  REVIEW REQUIRED: Possible rename detected
// Detected: document -> file
//
// This appears to be a rename based on similarity analysis.
// If this IS a rename (preserving tuples), keep the code below.
// If these are separate types, replace with AddType + DeleteType operations.
//
if err := omg.RenameType(ctx, client, "document", "file"); err != nil {
    return fmt.Errorf("failed to rename type: %w", err)
}
```

⚠️ **Medium confidence = review required, likely correct**

---

### Example 3: Low Confidence or No Detection

**Scenario:** Very different names or no relation similarity

```bash
./omg diff
```

Output:
```
- Type 'document' removed
+ New type 'asset' with 3 relations
```

```bash
./omg generate separate_types
```

**Generated code** (safe add+remove):
```go
// Add new type (already in model.fga)
if err := omg.AddTypeToModel(ctx, client, "asset", relations); err != nil {
    return fmt.Errorf("failed to add type: %w", err)
}

// Remove old type and tuples
tuples, err := omg.ReadAllTuples(ctx, client, "document", "")
if err != nil {
    return fmt.Errorf("failed to read tuples: %w", err)
}
// ... deletion code
```

✅ **Low/no confidence = safe operations, prevents data loss**

## 📋 CLI Commands

### Model-First Commands

#### `diff`
Compare `model.fga` with current state:
```bash
./omg diff
```

#### `generate <name>`
Generate migration from detected changes:
```bash
./omg generate add_folders
```

#### `init <store-name>`
Initialize tracking for a store:
```bash
./omg init my-store
```

### Migration Commands

#### `up`
Apply all pending migrations:
```bash
./omg up
```

#### `down`
Rollback the last migration:
```bash
./omg down
```

#### `status`
Show migration status:
```bash
./omg status
```

Output:
```
Migration Status:
================

20241128150000  initial_model                     APPLIED
20241128151000  add_folders                       PENDING

Total: 2 migrations (1 applied, 1 pending)
```

### Utility Commands

#### `show-model`
Display current authorization model:
```bash
./omg show-model
```

#### `list-tuples [type]`
List tuples, optionally filtered by type:
```bash
./omg list-tuples
./omg list-tuples document
```

#### `list-stores`
List available OpenFGA stores:
```bash
./omg list-stores
```

## 🔧 Advanced: Manual Migrations

For complex data operations that can't be auto-generated, create manual migrations:

### Create Manual Migration

```bash
./omg create custom_data_migration
```

### Edit Migration

```go
package migrations

import (
    "context"
    "fmt"
    "strings"

    "github.com/demetere/omg/pkg"
)

func init() {
    omg.Register(omg.Migration{
        Version: "20241128160000",
        Name:    "custom_data_migration",
        Up:      up_20241128160000,
        Down:    down_20241128160000,
    })
}

func up_20241128160000(ctx context.Context, client *omg.Client) error {
    // Custom transformation: migrate user ID format
    transform := func(tuple openfgaSdk.Tuple) (openfgaSdk.Tuple, error) {
        // Change user:123 -> user:uuid-123
        if strings.HasPrefix(tuple.User, "user:") {
            id := strings.TrimPrefix(tuple.User, "user:")
            tuple.User = "user:uuid-" + id
        }
        return tuple, nil
    }

    return omg.TransformAllTuples(ctx, client, "document", "viewer", transform)
}

func down_20241128160000(ctx context.Context, client *omg.Client) error {
    // Reverse transformation
    transform := func(tuple openfgaSdk.Tuple) (openfgaSdk.Tuple, error) {
        if strings.HasPrefix(tuple.User, "user:uuid-") {
            id := strings.TrimPrefix(tuple.User, "user:uuid-")
            tuple.User = "user:" + id
        }
        return tuple, nil
    }

    return omg.TransformAllTuples(ctx, client, "document", "viewer", transform)
}
```

## 🧠 How Confidence Levels Work

OMG uses multi-factor analysis to determine rename confidence:

### Factors Analyzed

1. **Name Similarity** (Levenshtein distance)
   - `team` → `teams`: 80% similar
   - `document` → `file`: 30% similar
   - `team` → `organization`: 8% similar

2. **Relation Structure** (Jaccard coefficient)
   - Same relations: 100% similar
   - Partial overlap: 50-99% similar
   - No overlap: 0% similar

### Confidence Thresholds

| Confidence | Criteria | Action |
|------------|----------|--------|
| **High** | Name ≥70% OR (Name ≥40% AND Relations ≥70%) | Clean rename code |
| **Medium** | Name ≥30% OR Relations ≥70% | Rename with warning |
| **Low** | Name ≥20% OR Relations ≥50% | Both options provided |
| **None** | Below thresholds | Separate add+remove |

### Examples

```
team → teams (80% name, 100% relations)           = High
team → organization (8% name, 100% relations)     = Medium
document → file (30% name, 0% relations)          = Medium
user → person (40% name, 0% relations)            = Low
team → asset (0% name, 0% relations)              = None
```

## 🛠️ Helper Functions

The generated migrations use these helper functions (you can use them in manual migrations too):

### Type Operations

```go
// Rename a type and migrate all tuples
omg.RenameType(ctx, client, "team", "organization")

// Add a type to the model
omg.AddTypeToModel(ctx, client, "folder", relations)

// Remove a type from the model
omg.RemoveTypeFromModel(ctx, client, "team")
```

### Relation Operations

```go
// Rename a relation
omg.RenameRelation(ctx, client, "document", "can_view", "viewer")

// Copy tuples to a new relation
omg.CopyRelation(ctx, client, "document", "editor", "can_edit")

// Delete all tuples of a relation
omg.DeleteRelation(ctx, client, "document", "deprecated_relation")

// Add a relation to a type
omg.AddRelationToType(ctx, client, "document", "commenter", "[user]")

// Remove a relation from a type
omg.RemoveRelationFromType(ctx, client, "document", "commenter")

// Update a relation definition
omg.UpdateRelationDefinition(ctx, client, "document", "viewer", "[user] or editor")
```

### Tuple Operations

```go
// Read all tuples matching criteria
tuples, err := omg.ReadAllTuples(ctx, client, "document", "viewer")

// Count tuples
count, err := omg.CountTuples(ctx, client, "document", "owner")

// Transform tuples with custom function
transform := func(t openfgaSdk.Tuple) (openfgaSdk.Tuple, error) {
    t.Object = "new:" + t.Object
    return t, nil
}
omg.TransformAllTuples(ctx, client, "document", "viewer", transform)

// Batch write tuples (100 per batch)
omg.WriteTuplesBatch(ctx, client, tuples)

// Batch delete tuples
omg.DeleteTuplesBatch(ctx, client, tuples)
```

### Backup & Restore

```go
// Backup all tuples before risky operation
backup, err := omg.BackupTuples(ctx, client)

// Restore if something goes wrong
omg.RestoreTuples(ctx, client, backup)
```

## 📁 Project Structure

```
.
├── cmd/omg/                    # CLI application
│   └── main.go
├── pkg/
│   ├── client.go              # OpenFGA SDK wrapper
│   ├── migration.go           # Migration registry
│   ├── tracker.go             # Migration tracking
│   ├── helpers.go             # Migration helper functions
│   ├── model_parser.go        # DSL parser
│   ├── model_tracker.go       # Change detection & confidence
│   └── migration_generator.go # Code generation
├── migrations/                # Generated/manual migrations
│   ├── migrations.go
│   └── YYYYMMDDHHMMSS_name.go
├── model.fga                  # Your authorization model (desired state)
├── .env                       # Configuration
└── README.md
```

## 🧪 Testing

### Unit Tests (No Docker Required)

```bash
go test -v ./pkg -run Unit
```

### Integration Tests (Requires Docker)

```bash
# Start Docker first
# Then run all tests
go test ./...

# With coverage
go test -cover ./...
```

Integration tests use testcontainers to spin up real OpenFGA instances.

## 📖 Best Practices

### 1. Model-First for Schema Changes

Use the model-first workflow for all schema changes:
```bash
# Edit model.fga
vim model.fga

# See changes
./omg diff

# Generate migration
./omg generate descriptive_name

# Review and apply
./omg up
```

### 2. Manual Migrations for Complex Operations

Use manual migrations for:
- User ID format changes
- Bulk data transformations
- Complex tuple migrations
- Data fixes and corrections

### 3. Trust the Confidence System

- **High confidence**: Usually safe to apply directly
- **Medium confidence**: Review but likely correct
- **Low confidence**: Verify the rename is intentional
- **No detection**: Correctly identified as separate operations

### 4. Always Review Generated Code

Even with high confidence, review migrations before applying:
```bash
# Review the generated file
cat migrations/YYYYMMDDHHMMSS_name.go

# Edit if needed
vim migrations/YYYYMMDDHHMMSS_name.go

# Then apply
./omg up
```

### 5. Test Locally First

```bash
# Test in local environment
OPENFGA_STORE_ID=test-store ./omg up

# Check result
./omg status

# Test rollback
./omg down
```

### 6. Commit Your Model

Commit `model.fga` to version control:
```bash
git add model.fga migrations/
git commit -m "Add folder type to authorization model"
```

This keeps your team in sync with model changes.

### 7. Use Descriptive Migration Names

```bash
# Good
./omg generate add_folder_hierarchy
./omg generate rename_team_to_organization

# Bad
./omg generate update
./omg generate fix
```

## 📚 Migration Patterns

### Pattern 1: Adding New Type

```
1. Edit model.fga - add new type
2. Run: omg generate add_<type>
3. Review generated code
4. Apply: omg up
```

### Pattern 2: Renaming Type/Relation

```
1. Edit model.fga - change name
2. Run: omg diff (see confidence level)
3. Run: omg generate rename_<old>_to_<new>
4. Review:
   - High confidence: usually correct
   - Medium confidence: verify it's a rename
   - Low confidence: check both options
5. Apply: omg up
```

### Pattern 3: Complex Schema + Data Change

```
1. Generate schema migration:
   omg generate update_schema

2. Edit generated file to add data transformations:
   vim migrations/YYYYMMDDHHMMSS_update_schema.go

3. Apply combined migration:
   omg up
```

### Pattern 4: Backward Compatible Change

```
1. Migration 1: Add new relation (copy from old)
2. Deploy app update (use new relation)
3. Migration 2: Remove old relation
```

## 🌐 Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENFGA_API_URL` | Yes | - | OpenFGA API endpoint |
| `OPENFGA_STORE_ID` | Yes | - | OpenFGA store ID |
| `OPENFGA_AUTH_METHOD` | No | `none` | Auth: `none`, `token`, or `client_credentials` |
| `OPENFGA_API_TOKEN` | Conditional | - | API token (if `AUTH_METHOD=token`) |
| `OPENFGA_CLIENT_ID` | Conditional | - | OAuth client ID |
| `OPENFGA_CLIENT_SECRET` | Conditional | - | OAuth client secret |
| `OPENFGA_TOKEN_ISSUER` | No | - | OAuth issuer URL |
| `OPENFGA_TOKEN_AUDIENCE` | No | - | OAuth audience |
| `LOG_LEVEL` | No | `info` | Log level: `debug`, `info`, `warn`, `error` |

## 🔗 Related Documentation

- [MODEL_FIRST_GUIDE.md](MODEL_FIRST_GUIDE.md) - Detailed model-first workflow guide
- [TESTING.md](TESTING.md) - Testing guide and best practices
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development guide
- [CLAUDE.md](CLAUDE.md) - Complete development journal

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details

## 🙏 Acknowledgments

- Inspired by [Prisma](https://www.prisma.io/), [Entity Framework](https://docs.microsoft.com/en-us/ef/), and [goose](https://github.com/pressly/goose)
- Built with [OpenFGA Go SDK](https://github.com/openfga/go-sdk)
- Testing with [testcontainers-go](https://github.com/testcontainers/testcontainers-go)

---

**Made with ❤️ for the OpenFGA community**
