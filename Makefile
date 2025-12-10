# Makefile for agentmcp

.PHONY: help build test clean install run docker fmt lint

# Variables
BINARY_NAME=agentmcp
VERSION?=1.0.0
BUILD_DIR=dist
LDFLAGS=-ldflags="-s -w -X main.VERSION=$(VERSION)"

# Default target
help:
	@echo "AgentMCP - Makefile targets:"
	@echo "  make build      - Build binary for current platform"
	@echo "  make build-all  - Build binaries for all platforms"
	@echo "  make test       - Run tests"
	@echo "  make test-cov   - Run tests with coverage"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make install    - Install to /usr/local/bin"
	@echo "  make run        - Run server locally"
	@echo "  make docker     - Build Docker image"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Run linters"
	@echo "  make release    - Create release builds"

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@./build.sh

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-cov:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install to system
install: build
	@echo "Installing to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

# Run server locally
run: build
	@echo "Starting server..."
	@./$(BINARY_NAME) -agents ./agents -transport stdio

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .
	@echo "Docker image built: $(BINARY_NAME):latest"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# Run linters
lint:
	@echo "Running linters..."
	@go vet ./...
	@golangci-lint run || echo "Install golangci-lint for full linting"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

# Create release
release: clean build-all
	@echo "Release builds created in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/
