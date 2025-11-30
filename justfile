# OpenFGA Migration Tool - Justfile

# Default recipe (show help)
default:
    @just --list

# Build the CLI binary
build:
    @echo "Building omg..."
    @go build -o omg ./cmd/omg
    @echo "Build complete: ./omg"

# Install the CLI globally
install:
    @echo "Installing omg..."
    @go install ./cmd/omg
    @echo "Installed to $(go env GOPATH)/bin/omg"

# Run unit tests only (no Docker required)
test-unit:
    @echo "Running unit tests..."
    @go test -v ./pkg/migration -run "TestRegister|TestGetAll|TestReset|TestGetAllReturnsCopy"

# Run all tests (requires Docker)
test:
    @echo "Running all tests (requires Docker)..."
    @echo "Make sure Docker is running!"
    @go test ./...

# Run tests with verbose output
test-verbose:
    @echo "Running tests (verbose)..."
    @go test -v ./...

# Run tests with coverage (requires Docker)
test-coverage:
    @echo "Running tests with coverage (requires Docker)..."
    @echo "Make sure Docker is running!"
    @go test -cover ./...
    @go test -coverprofile=coverage.out ./...
    @go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Check if Docker is running
check-docker:
    @docker ps > /dev/null 2>&1 || (echo "Error: Docker is not running. Please start Docker first." && exit 1)
    @echo "Docker is running ✓"

# Run integration tests (requires Docker)
test-integration: check-docker
    @echo "Running integration tests..."
    @go test -v ./pkg/openfga ./pkg/migration ./pkg/helpers -run "TestClient|TestTracker|TestRename|TestCopy|TestDelete|TestMigrate|TestCount|TestBackup"

# Clean build artifacts
clean:
    @echo "Cleaning..."
    @rm -f omg
    @rm -f coverage.out coverage.html
    @echo "Clean complete"

# Run migrations up
up: build
    @./omg up

# Run migrations down
down: build
    @./omg down

# Show migration status
status: build
    @./omg status

# Create a new migration
create NAME: build
    @./omg create {{NAME}}

# List all tuples (optionally filtered by type)
list-tuples TYPE="": build
    @./omg list-tuples {{TYPE}}

# Show current authorization model
show-model: build
    @./omg show-model

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    @go mod download
    @go mod tidy
    @echo "Dependencies updated"

# Format code
fmt:
    @echo "Formatting code..."
    @go fmt ./...
    @echo "Format complete"

# Run linter (requires golangci-lint)
lint:
    @echo "Running linter..."
    @golangci-lint run ./...

# Run all checks (fmt, lint, test)
check: fmt lint test
    @echo "All checks passed!"

# Start local OpenFGA server with docker
start-openfga:
    @echo "Starting OpenFGA with docker-compose..."
    @docker run -d --name openfga -p 8080:8080 -p 8081:8081 -p 3000:3000 openfga/openfga run

# Stop local OpenFGA server
stop-openfga:
    @echo "Stopping OpenFGA..."
    @docker stop openfga
    @docker rm openfga

# Initialize a new OpenFGA store (use the omg CLI instead)
init NAME: build
    @./omg init {{NAME}}

# List OpenFGA stores
list-stores: build
    @./omg list-stores
