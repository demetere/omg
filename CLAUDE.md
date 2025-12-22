# OMG Development Notes

This document contains essential development information for contributors working on the OpenFGA Migration Tool (OMG).

## Project Overview

**OpenFGA Migration Tool (OMG)** - A model-first migration tool for OpenFGA authorization models and relationship tuples, inspired by Prisma and Entity Framework.

**Current Status:** ✅ Production Ready
**Test Coverage:** ~78% (72 tests)
**Latest Update:** December 2, 2025

## Architecture

### Core Design Principles

1. **Model-First with Escape Hatches** - Primary workflow is schema-driven with auto-generated migrations, but manual migrations are fully supported for complex operations
2. **Single Source of Truth** - OpenFGA API is queried directly for current state; no local cache files
3. **Confidence-Based Intelligence** - Multi-factor similarity analysis (name + relation structure) guides rename detection
4. **Runtime Migration Loading** - Migrations are standalone Go programs executed with `go run`, not compiled into the binary

### Package Structure

```
pkg/
├── client.go              # OpenFGA SDK wrapper with three-tier filtering
├── migration.go           # Migration registry (for tests)
├── tracker.go             # Migration tracking via OpenFGA tuples
├── helpers.go             # Tuple & model operation helpers
├── model_parser.go        # DSL parser (simplified, not full OpenFGA spec)
├── model_tracker.go       # Change detection & confidence scoring
└── migration_generator.go # Code generation with dependency ordering
```

### Migration File Format

Migrations are `package main` executables:

```go
package main

import (
    "context"
    "os"
    omg "github.com/demetere/omg"
)

func main() {
    client, _ := omg.NewClient(omg.Config{
        ApiURL:  os.Getenv("OPENFGA_API_URL"),
        StoreID: os.Getenv("OPENFGA_STORE_ID"),
        // ...
    })

    ctx := context.Background()
    if len(os.Args) > 1 && os.Args[1] == "down" {
        down(ctx, client)
    } else {
        up(ctx, client)
    }
}

func up(ctx context.Context, client *omg.Client) error { /* ... */ }
func down(ctx context.Context, client *omg.Client) error { /* ... */ }
```

**Benefits:**
- `-dir` flag works at runtime
- No recompilation needed after creating migrations
- Each migration is independent
- Works like standard migration tools

**Trade-offs:**
- Requires Go installed
- Slightly slower (compiles on-the-fly, though Go's build cache helps)

## Key Technical Challenges & Solutions

### 1. Rename Detection

**Challenge:** Distinguishing renames from separate add+remove operations.

**Solution:** Multi-factor confidence system:

```go
// Name similarity: Levenshtein distance (0.0-1.0)
nameSimilarity := calculateSimilarity("team", "teams") // 0.80

// Relation similarity: Jaccard coefficient (0.0-1.0)
relSimilarity := haveSimilarRelations(oldType, newType) // 1.0

// Confidence determination:
// High:   nameSim ≥70% OR (nameSim ≥40% AND relSim ≥70%)
// Medium: nameSim ≥30% OR relSim ≥70%
// Low:    nameSim ≥20% OR relSim ≥50%
```

Generated code reflects confidence level:
- **High:** Clean rename code
- **Medium:** Rename with review warning
- **Low:** Both options (rename commented out, safe add+remove active)

### 2. OpenFGA SDK Pointer Semantics

**Problem:** Relations use `*map[string]Userset`, not direct maps.

**Solution:** Consistent pointer handling:

```go
// Creating
relationMap := make(map[string]openfgaSdk.Userset)
relationMap[relName] = userset
typeDef.Relations = &relationMap  // Pointer to map

// Accessing
relations := typeDef.GetRelations()  // Returns map (not pointer)
```

### 3. Tuple-to-Userset Metadata

**Problem:** OpenFGA rejects DirectlyRelatedUserTypes metadata on computed relations.

**Solution:** Only set metadata for direct relations:

```go
isDirect := strings.Contains(relationDef, "[")
if len(typeRestrictions) > 0 && isDirect {
    // Set metadata...
}
```

### 4. Cross-Type Dependencies

**Problem:** `membership.member: member from team` depends on `team.member`.

**Solution:** Enhanced dependency extraction with topological sort:

```go
// Parse "member from team" syntax
if strings.Contains(relDef, " from ") {
    parts := strings.Split(relDef, " from ")
    computedUserset := strings.TrimSpace(parts[0])
    tuplesetType := strings.TrimSpace(parts[1])
    depKey := fmt.Sprintf("%s.%s", tuplesetType, computedUserset)
    // Add to dependencies
}

// Topological sort ensures correct order
orderedRelations := topologicalSort(dependencies)
```

### 5. DSL Parser Limitations

**Current Support:**
- ✅ Direct relations: `[user]`, `[user, group#member]`
- ✅ Computed relations: `owner`
- ✅ Operators: `or`, `and`, `but not`
- ✅ Tuple-to-userset: `parent->owner`, `owner from team`
- ❌ Conditions/caveats
- ❌ Inline comments
- ❌ Parenthesized expressions

**Future:** Integrate official OpenFGA language parser for full DSL support.

## Testing Strategy

### Unit Tests (No Docker)
```bash
go test -v ./pkg -run Unit
```
- Fast feedback (<1s)
- Tests pure logic: parsing, change detection, code generation

### Integration Tests (Requires Docker)
```bash
go test ./...
```
- Uses testcontainers to spin up real OpenFGA instances
- Tests actual API interactions
- ~97s execution time (includes container startup)

### Coverage
- Model parser: ~85% (21 tests)
- Change detection: ~80% (16 tests)
- Code generation: ~90% (23 tests)
- Core operations: ~75% (22 tests)
- **Overall: ~78%**

## Development Workflow

### Adding a New Helper Function

```go
// 1. Add to helpers.go with godoc
// TransformUserIDs migrates user IDs from old to new format
func TransformUserIDs(ctx context.Context, client *Client, ...) error {
    // Implementation
}

// 2. Add test in helpers_test.go
func TestTransformUserIDs(t *testing.T) {
    // Setup testcontainer
    // Execute
    // Verify
}

// 3. Update documentation if needed
```

### Adding a New Change Detection Type

```go
// 1. Add to model_tracker.go
const ChangeTypeAddCondition ChangeType = "add_condition"

// 2. Detect in DetectChanges()
for condName := range newConditions {
    if _, exists := oldConditions[condName]; !exists {
        changes = append(changes, ModelChange{
            Type: ChangeTypeAddCondition,
            // ...
        })
    }
}

// 3. Generate code in migration_generator.go
case ChangeTypeAddCondition:
    builder.WriteString(generateAddCondition(change))

// 4. Add tests
```

## Common Development Commands

```bash
# Build
go build ./cmd/omg

# Run locally
./omg diff
./omg generate my_feature
./omg up

# Test
go test ./...                          # All tests
go test -v ./pkg -run TestModelParser  # Specific test
go test -cover ./...                   # With coverage

# Install globally
go install ./cmd/omg
```

## Code Quality Standards

1. **Error Handling:** Always wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
2. **Documentation:** Godoc comments on all exported functions
3. **Testing:** Add tests for new functionality (aim for >75% coverage)
4. **Generated Code:** Include helpful comments and error handling
5. **Backwards Compatibility:** Don't break existing migrations or APIs

## Migration Tracking

Uses OpenFGA itself to track applied migrations:
- Tuple format: `system:migration-tracker applied migration:<version>`
- No external database required
- Query via: `client.Read(ctx, ReadRequest{Object: "system:migration-tracker", Relation: "applied"})`

## Performance Considerations

- DSL parsing (<100 lines): <10ms
- Change detection: <50ms
- Migration generation: <100ms
- Tuple operations: Batched at 100 per write/delete
- Consider streaming for very large tuple migrations

## Security Notes

- Never commit `.env` files
- Use service accounts with minimal permissions
- Test migrations in dev environment first
- Backup production tuples before major changes
- `model.fga` is source code - commit to version control

## Dependencies

**Core:**
- `github.com/openfga/go-sdk` - Official OpenFGA SDK
- `github.com/joho/godotenv` - Environment configuration

**Testing:**
- `github.com/stretchr/testify` - Assertions
- `github.com/testcontainers/testcontainers-go` - Container testing

**Requirements:**
- Go 1.23+
- Docker (for integration tests)

## Future Enhancements

### High Priority
1. **Official FGA Parser** - Replace simplified parser with full OpenFGA language parser
2. **Interactive Prompts** - For medium/low confidence renames
3. **Migration Testing Framework** - Auto-generate tests for migrations
4. **Model Validation** - Syntax checking, circular dependency detection

### Medium Priority
5. **Migration Plan Preview** - Show affected tuples count, potential issues
6. **Dry Run Mode** - Simulate migrations without applying
7. **Model Versioning** - Support multiple model file versions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Update documentation if needed
6. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Additional Documentation

- [README.md](README.md) - User guide and quick start
- [MODEL_FIRST_GUIDE.md](MODEL_FIRST_GUIDE.md) - Detailed workflow guide
- [TESTING.md](TESTING.md) - Testing guide and best practices
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines

---

**Last Updated:** December 2, 2025
**Project Status:** Production Ready
