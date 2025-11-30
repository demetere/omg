# Contributing to OpenFGA Migration Tool

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.23 or higher
- Docker (for running tests with testcontainers)
- [just](https://github.com/casey/just) command runner (optional, but recommended)

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/demetere/omg.git
   cd omg
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   just build
   # or
   go build -o omg ./cmd/omg
   ```

## Project Structure

```
.
├── cmd/
│   └── omg/     # CLI application entry point
├── pkg/
│   ├── migration/           # Migration registry and tracking
│   ├── openfga/            # OpenFGA client wrapper
│   └── helpers/            # Migration helper functions
├── internal/
│   └── testhelpers/        # Shared test utilities
├── migrations/             # Migration files
├── justfile                # Task runner commands
├── go.mod                  # Go module definition
└── README.md              # Project documentation
```

## Development Workflow

### Running Tests

```bash
# Run all tests
just test

# Run tests with verbose output
just test-verbose

# Run tests with coverage
just test-coverage

# Run specific package tests
go test -v ./pkg/omg
go test -v ./pkg/omg
go test -v ./pkg/omg
```

### Code Quality

```bash
# Format code
just fmt

# Run linter (requires golangci-lint)
just lint

# Run all checks
just check
```

### Building

```bash
# Build the CLI
just build

# Install globally
just install
```

## Writing Tests

All new features should include tests. We use:
- `testcontainers-go` for integration tests with OpenFGA
- `testify` for assertions

Example test:

```go
func TestMyFeature(t *testing.T) {
	ctx := context.Background()

	// Setup OpenFGA container with a model
	container, client := testomg.SetupOpenFGAContainer(t, ctx, `
type user
type document
  relations
    define owner: [user]
`)
	defer container.Terminate(ctx)

	// Your test logic here
	// ...
}
```

## Adding New Features

### Adding New Helper Functions

1. Add the function to `pkg/omg/omg.go`
2. Add tests to `pkg/omg/helpers_test.go`
3. Update `migrations/00000000000000_example.go` with usage examples
4. Update the README with documentation

### Adding New CLI Commands

1. Add the command handler in `cmd/omg/main.go`
2. Update `printUsage()` function
3. Add tests if applicable
4. Update README documentation

## Pull Request Process

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Make** your changes
4. **Add** tests for new functionality
5. **Run** tests and ensure they pass: `just test`
6. **Format** code: `just fmt`
7. **Commit** your changes (`git commit -m 'Add amazing feature'`)
8. **Push** to your branch (`git push origin feature/amazing-feature`)
9. **Open** a Pull Request

### Pull Request Guidelines

- Write clear, descriptive commit messages
- Include tests for new features
- Update documentation as needed
- Ensure all tests pass
- Follow existing code style
- Keep changes focused (one feature per PR)

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

## Testing Guidelines

### Unit Tests

- Test individual functions in isolation
- Use table-driven tests when appropriate
- Mock external dependencies

### Integration Tests

- Use testcontainers for OpenFGA
- Clean up resources in defer statements
- Test realistic scenarios

### Test Coverage

- Aim for >80% test coverage
- Focus on critical paths and edge cases
- Don't test trivial code just for coverage

## Documentation

- Update README.md for user-facing changes
- Add code comments for complex logic
- Include examples in documentation
- Keep documentation up-to-date

## Common Tasks

### Adding a New Migration Helper

```go
// 1. Add function to pkg/omg/omg.go
func MyNewHelper(ctx context.Context, client *openfga.Client, params ...string) error {
	// Implementation
	return nil
}

// 2. Add test to pkg/omg/helpers_test.go
func TestMyNewHelper(t *testing.T) {
	ctx := context.Background()
	container, client := testomg.SetupOpenFGAContainer(t, ctx, `...`)
	defer container.Terminate(ctx)

	err := omg.MyNewHelper(ctx, client, "param1")
	require.NoError(t, err)

	// Verify results
}

// 3. Add usage example to migrations/00000000000000_example.go
```

### Debugging Tests

```bash
# Run a single test with verbose output
go test -v -run TestSpecificTest ./pkg/omg

# Run with race detector
go test -race ./...

# Keep container running for debugging
# (comment out defer container.Terminate(ctx) in test)
```

## Release Process

1. Update version in code/documentation
2. Update CHANGELOG.md
3. Create and push a git tag
4. Create a GitHub release
5. Build and upload binaries

## Getting Help

- Open an issue for bugs or feature requests
- Ask questions in discussions
- Check existing issues and documentation first

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
