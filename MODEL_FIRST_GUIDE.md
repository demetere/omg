# Model-First Migration Guide

## Overview

OMG (OpenFGA Migration Tool) now supports a **model-first workflow** where you edit your authorization model in `model.fga` and the tool automatically generates migration files based on the changes detected.

This approach is similar to how Prisma, Entity Framework, or other modern ORMs work - you focus on your schema/model, and migrations are generated automatically.

## How It Works

### 1. Model State Tracking

OMG queries OpenFGA directly to understand your current model state:
- **Desired state**: Your `model.fga` file (what you want)
- **Current state**: Live model in OpenFGA database (what exists)
- **No cache files**: Queries the API for truth

### 2. Change Detection

When you run `omg diff` or `omg generate`, the tool:
1. Queries OpenFGA API for the current authorization model
2. Reads your desired model from `model.fga` file
3. Compares the two and detects all changes
4. Uses heuristics to detect potential renames

### 3. Migration Generation

The `omg generate` command creates a complete migration file with:
- Model changes (using helper functions)
- Tuple migrations (when needed for renames/removals)
- Proper up and down functions

## Quick Start

### Initial Setup

1. **Create your first model file:**
```bash
cat > model.fga << 'EOF'
model
  schema 1.1

type user

type file
  relations
    define owner: [user]
    define can_read: [user] or owner
    define can_write: [user] or owner
EOF
```

2. **Generate initial migration:**
```bash
omg generate initial_model
```

3. **Apply the migration:**
```bash
omg up
```

### Making Changes

Let's say you want to add a `folder` type and a `can_delete` relation to files:

1. **Edit `model.fga`:**
```fga
model
  schema 1.1

type user

type folder
  relations
    define owner: [user]
    define can_read: [user] or owner

type file
  relations
    define owner: [user]
    define can_read: [user] or owner
    define can_write: [user] or owner
    define can_delete: owner
```

2. **Check what changed:**
```bash
omg diff
```

Output:
```
Detecting model changes...

Detected 2 change(s):

+ New type 'folder' with 2 relations
+ Added relation 'file.can_delete'

Run 'omg generate <name>' to create a migration for these changes
```

3. **Generate migration:**
```bash
omg generate add_folders
```

Output:
```
Detecting model changes...

Detected 2 change(s):
  1. New type 'folder' with 2 relations
  2. Added relation 'file.can_delete'

Generating migration...

✓ Migration created: migrations/20241128120000_add_folders.go

Next steps:
  1. Review the generated migration file
  2. Edit if needed (especially for renames)
  3. Run 'omg up' to apply the migration
```

4. **Review and apply:**
```bash
# Review the generated file
cat migrations/20241128120000_add_folders.go

# Apply it
omg up
```

## Supported Changes

### ✅ Automatically Handled

| Change | Detection | Migration Generated |
|--------|-----------|---------------------|
| Add Type | ✅ Automatic | `AddTypeToModel()` |
| Add Relation | ✅ Automatic | `AddRelationToType()` |
| Remove Relation | ✅ Automatic | `RemoveRelationFromType()` + `DeleteRelation()` |
| Remove Type | ✅ Automatic | `RemoveTypeFromModel()` + tuple cleanup |
| Update Relation Definition | ✅ Automatic | `UpdateRelationDefinition()` |

### ⚠️ Requires Manual Review

| Change | Detection | Migration Generated | Note |
|--------|-----------|---------------------|------|
| Rename Type | ⚠️ Heuristic | Add + Remove | Edit to use `RenameType()` |
| Rename Relation | ⚠️ Heuristic | Add + Remove | Edit to use `RenameRelation()` |

## Rename Detection

OMG uses similarity heuristics to detect potential renames:

```bash
# Before: model.fga
type team
  relations
    define member: [user]

# After: model.fga
type organization
  relations
    define member: [user]
```

Running `omg diff`:
```
→ Possible rename: 'team' -> 'organization' (requires confirmation)
    Old: team
    New: organization
```

Running `omg generate`:
```
⚠ Detected potential rename: team -> organization
   This will be handled as remove + add operations.
   Edit the generated migration to use RenameType/RenameRelation if needed.
```

**Generated migration:**
```go
// Step 1: Remove type team
if err := omg.RemoveTypeFromModel(ctx, client, "team"); err != nil {
    return err
}

// Step 2: Add type organization
if err := omg.AddTypeToModel(ctx, client, "organization", map[string]string{
    "member": "[user]",
}); err != nil {
    return err
}
```

**Edit it to:**
```go
// Rename team -> organization (preserves tuples)
if err := omg.RenameType(ctx, client, "team", "organization"); err != nil {
    return err
}
```

## Common Patterns

### Pattern 1: Adding a New Type

**Edit `model.fga`:**
```fga
type document
  relations
    define owner: [user]
    define viewer: [user] or owner
```

**Generate & Apply:**
```bash
omg generate add_documents
omg up
```

### Pattern 2: Adding Relations to Existing Type

**Edit `model.fga`:**
```fga
type file
  relations
    define owner: [user]
    define can_read: [user] or owner
    define can_write: [user] or owner
    define can_share: [user] or owner  # New!
```

**Generate & Apply:**
```bash
omg diff  # See the change
omg generate add_share_permission
omg up
```

### Pattern 3: Removing a Relation

**Edit `model.fga`** (remove a relation):
```fga
type file
  relations
    define owner: [user]
    # Removed: define can_write: [user] or owner
```

**The generated migration will:**
1. Remove the relation from the model
2. Delete all tuples with that relation

**Review carefully!** This is destructive.

### Pattern 4: Updating Relation Logic

**Edit `model.fga`:**
```fga
type file
  relations
    define owner: [user]
    # Changed from: define can_write: [user] or owner
    define can_write: [user, group#member] or owner  # Now supports groups!
```

**Generate:**
```bash
omg generate add_group_support
```

**Generated migration** uses `UpdateRelationDefinition()` - no tuple migration needed unless you're changing the relation name.

### Pattern 5: Renaming (Manual)

**For a relation rename** (e.g., `can_write` → `can_modify`):

**Edit `model.fga`:**
```fga
type file
  relations
    define owner: [user]
    define can_modify: [user] or owner  # Renamed from can_write
```

**Generate & Edit:**
```bash
omg generate rename_write_to_modify
```

**Edit the generated file** to use the rename helpers:
```go
func up_XXX(ctx context.Context, client *omg.Client) error {
    // Instead of remove + add, use rename (preserves tuples):
    if err := omg.RenameRelation(ctx, client, "file", "can_write", "can_modify"); err != nil {
        return err
    }
    return nil
}

func down_XXX(ctx context.Context, client *omg.Client) error {
    // Rename back
    return omg.RenameRelation(ctx, client, "file", "can_modify", "can_write")
}
```

## File Structure

```
your-project/
├── model.fga                          # Your desired state (source of truth)
├── migrations/
│   ├── migrations.go                  # Package file
│   ├── 20241127120000_initial.go      # Your migrations
│   └── 20241128130000_add_folders.go
├── .env                               # OpenFGA connection
└── go.mod
```

## CLI Commands

### Model-First Workflow

| Command | Purpose |
|---------|---------|
| `omg diff` | Show changes between `model.fga` and current state |
| `omg generate [name]` | Auto-generate migration from detected changes |
| `omg up` | Apply pending migrations (syncs state automatically) |
| `omg down` | Rollback last migration |
| `omg status` | Show migration status |

### Utilities

| Command | Purpose |
|---------|---------|
| `omg show-model` | Display current model in OpenFGA |
| `omg list-tuples [type]` | List tuples (optionally filtered) |
| `omg create <name>` | Create blank migration (manual mode) |

### Store Management

| Command | Purpose |
|---------|---------|
| `omg init <name>` | Create new OpenFGA store |
| `omg list-stores` | List all stores |

## Best Practices

### 1. Always Run `diff` First
```bash
# Before generating
omg diff

# Review the changes
# Then generate
omg generate my_feature
```

### 2. Review Generated Migrations

Generated migrations are a starting point. Always review them:
- Check relation definitions are correct
- Verify destructive operations (removes/deletes)
- Convert add+remove to renames when appropriate

### 3. Test in Development First

```bash
# Dev environment
export OPENFGA_STORE_ID=dev-store-id
omg up

# Verify it worked
omg show-model
omg list-tuples

# Then apply to staging/prod
export OPENFGA_STORE_ID=prod-store-id
omg up
```

### 4. Keep `model.fga` in Sync

Your `model.fga` should always match what's deployed. After applying migrations:
```bash
# Verify sync
omg diff  # Should show "No changes detected"
```

### 5. Use Descriptive Migration Names

```bash
# Good
omg generate add_document_sharing
omg generate remove_deprecated_viewer_role
omg generate add_team_hierarchy

# Bad
omg generate changes
omg generate update
```

## Migrating from Manual Migrations

If you have an existing project with manual migrations:

1. **Capture current state:**
```bash
# Apply all existing migrations
omg up

# This automatically syncs the model state
```

2. **Create `model.fga`** matching your current model:
```bash
omg show-model > model.fga
```

3. **Make changes** to `model.fga` and use the model-first workflow going forward.

## Troubleshooting

### "failed to load model.fga"

**Cause:** No `model.fga` file exists.

**Solution:** Create one:
```bash
touch model.fga
# Add your model content
```

### "failed to load current model from OpenFGA"

**Cause:** OpenFGA is not running or not accessible, or the store doesn't have a model yet.

**Solution:**
- Ensure OpenFGA is running and accessible
- Check your `OPENFGA_API_URL` and `OPENFGA_STORE_ID` settings
- If this is a new store, apply your first migration with `omg up`

### Generated migration has wrong relation definition

**Cause:** Simplified DSL parser can't handle complex definitions yet.

**Solution:** Edit the generated migration to use the correct definition string.

### Changes not detected

**Cause:** `.omg/model_state.json` is out of sync.

**Solution:**
```bash
# Re-sync by running migrations
omg up

# Or manually delete and regenerate
rm -rf .omg/
omg up
```

## Advanced: Custom Transformations

For complex migrations, you can edit generated files to add custom logic:

```go
func up_XXX(ctx context.Context, client *omg.Client) error {
    // Generated code
    if err := omg.AddRelationToType(ctx, client, "file", "can_delete", "owner"); err != nil {
        return err
    }

    // Add custom logic
    // Example: Migrate existing owner tuples to can_delete
    if err := omg.CopyRelation(ctx, client, "file", "owner", "can_delete"); err != nil {
        return err
    }

    return nil
}
```

## Next Steps

- Read [CONTRIBUTING.md](CONTRIBUTING.md) for development guide
- Read [TESTING.md](TESTING.md) for testing guide
- Check out [README.md](README.md) for complete feature list

## Examples

See the `migrations/` directory for example migrations showing all patterns.
