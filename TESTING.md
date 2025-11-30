# Testing Guide

This document explains how to run tests for the OpenFGA Migration Tool.

## Test Types

The project has two types of tests:

1. **Unit Tests** - Fast tests that don't require external dependencies
2. **Integration Tests** - Tests that use testcontainers to spin up real OpenFGA instances

## Running Unit Tests

Unit tests can run without Docker and are fast:

```bash
# Using just
just test-unit

# Using go test directly
go test -v ./pkg/omg -run "TestRegister|TestGetAll|TestReset|TestGetAllReturnsCopy"
```

**Output:**
```
Running unit tests...
=== RUN   TestRegister
--- PASS: TestRegister (0.00s)
=== RUN   TestGetAllSortsVersions
--- PASS: TestGetAllSortsVersions (0.00s)
=== RUN   TestReset
--- PASS: TestReset (0.00s)
=== RUN   TestGetAllReturnsCopy
--- PASS: TestGetAllReturnsCopy (0.00s)
PASS
```

## Running Integration Tests

Integration tests require Docker to be running because they use testcontainers to create real OpenFGA instances.

### Prerequisites

1. **Install Docker**
   - macOS: [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/)
   - Linux: [Docker Engine](https://docs.docker.com/engine/install/)
   - Windows: [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)

2. **Start Docker**
   - macOS: Open Docker Desktop application
   - Linux: `sudo systemctl start docker`
   - Verify: `docker ps` should work without errors

### Running Integration Tests

```bash
# Check if Docker is running
just check-docker

# Run integration tests only
just test-integration

# Run all tests (unit + integration)
just test

# Run with verbose output
just test-verbose

# Run with coverage
just test-coverage
```

### What the Integration Tests Do

The integration tests:
1. Download the OpenFGA Docker image (first run only)
2. Start an OpenFGA container
3. Create a test store and authorization model
4. Run tests against the real OpenFGA instance
5. Clean up containers after tests complete

**First run may take 1-2 minutes** to download the Docker image. Subsequent runs are much faster.

## Common Issues

### Docker Not Running

**Error:**
```
panic: rootless Docker not found
```
or
```
Cannot connect to the Docker daemon at unix:///var/run/docker.sock
```

**Solution:**
Start Docker before running integration tests:
- macOS: Open Docker Desktop
- Linux: `sudo systemctl start docker`

### Docker Permission Issues (Linux)

**Error:**
```
permission denied while trying to connect to the Docker daemon socket
```

**Solution:**
```bash
# Add your user to the docker group
sudo usermod -aG docker $USER

# Log out and log back in, then verify
docker ps
```

### Port Already in Use

**Error:**
```
port 8080 is already allocated
```

**Solution:**
Stop any running OpenFGA containers:
```bash
docker ps
docker stop <container-id>
```

## Test Structure

### Unit Tests

Location: `pkg/omg/migration_unit_test.go`

Tests:
- `TestRegister` - Tests migration registration
- `TestGetAllSortsVersions` - Tests migration sorting
- `TestReset` - Tests registry reset
- `TestGetAllReturnsCopy` - Tests immutability

### Integration Tests

Locations:
- `pkg/omg/client_test.go` - OpenFGA client wrapper tests
- `pkg/omg/migration_test.go` - Migration tracker tests
- `pkg/omg/helpers_test.go` - Helper function tests

Each integration test:
1. Sets up an OpenFGA container
2. Creates a test store with a model
3. Runs test operations
4. Cleans up resources

### Test Helper

Location: `internal/testhelpers/testomg.go`

Provides:
- `SetupOpenFGAContainer()` - Creates OpenFGA container with model
- Model parsing utilities
- Shared test infrastructure

## Writing New Tests

### Unit Test Example

```go
func TestMyFeature(t *testing.T) {
    Reset() // Clear state if needed

    // Your test logic
    result := MyFunction()

    assert.Equal(t, expected, result)
}
```

### Integration Test Example

```go
func TestMyIntegration(t *testing.T) {
    ctx := context.Background()

    // Setup OpenFGA container
    container, client := testomg.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
`)
    defer container.Terminate(ctx)

    // Your test logic
    err := client.WriteTuple(ctx, openfga.Tuple{
        User:     "user:alice",
        Relation: "owner",
        Object:   "document:readme",
    })
    require.NoError(t, err)

    // Verify
    tuples, err := client.ReadAllTuples(ctx, openfga.ReadTuplesRequest{
        User: "user:alice",
    })
    require.NoError(t, err)
    assert.Len(t, tuples, 1)
}
```

## Continuous Integration

For CI/CD pipelines, ensure Docker is available:

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      # Unit tests (no Docker needed)
      - name: Run unit tests
        run: go test -v ./pkg/omg -run "Unit"

      # Integration tests (Docker available by default)
      - name: Run integration tests
        run: go test -v ./...
```

## Performance

### Test Execution Times

- **Unit tests**: < 1 second
- **Integration tests (first run)**: 1-2 minutes (Docker image download)
- **Integration tests (subsequent)**: 10-30 seconds
- **Full test suite**: ~30 seconds (after initial Docker image download)

### Speeding Up Tests

1. **Run unit tests first** during development:
   ```bash
   just test-unit
   ```

2. **Use test caching**:
   ```bash
   go test -count=1 ./...  # Disable cache for fresh run
   go test ./...           # Use cache when possible
   ```

3. **Run specific tests**:
   ```bash
   go test -v ./pkg/omg -run TestClient_WriteTuple
   ```

## Debugging Tests

### Verbose Output

```bash
go test -v ./pkg/omg
```

### Keep Container Running

Comment out the cleanup in your test:
```go
container, client := testomg.SetupOpenFGAContainer(t, ctx, modelDSL)
// defer container.Terminate(ctx)  // Comment this out

// Run your tests
// Container stays running for inspection
```

Then inspect:
```bash
# Find the container
docker ps

# Check logs
docker logs <container-id>

# Access the container
docker exec -it <container-id> sh
```

### Race Detector

Check for race conditions:
```bash
go test -race ./...
```

## Test Coverage

Generate coverage report:
```bash
# Using just
just test-coverage

# Manual
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

Current coverage targets:
- Overall: >80%
- Critical paths: >90%
- Helper functions: >85%

## Best Practices

1. **Always cleanup resources**: Use `defer container.Terminate(ctx)`
2. **Reset state in unit tests**: Call `Reset()` to clear global state
3. **Use descriptive test names**: Follow `Test<Function>_<Scenario>` pattern
4. **Test edge cases**: Empty inputs, nil values, error conditions
5. **Keep tests isolated**: Each test should be independent
6. **Use table-driven tests**: For testing multiple scenarios

## Getting Help

- Check Docker is running: `docker ps`
- View test logs: `go test -v`
- Check testcontainer logs: `docker logs <container-id>`
- Open an issue if tests consistently fail
