.PHONY: build test test-verbose clean deps lint fmt vet install

# Go variables
GOCMD=go
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build variables
BINARY_NAME=fybrk
BUILD_DIR=bin
MAIN_PATH=./cli/cmd/fybrk

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOCMD) build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Install binary
install: 
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOCMD) build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race ./...

# Run specific test
test-pkg:
	@echo "Running tests for specific package..."
	@read -p "Enter package path (e.g., ./pkg/types): " pkg; \
	$(GOTEST) -v $$pkg

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run all checks
check: fmt vet lint test

# Development workflow
dev: deps fmt vet test

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build fybrk binary"
	@echo "  install       - Build and install fybrk binary"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-pkg      - Run tests for specific package"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  check         - Run all checks (fmt, vet, lint, test)"
	@echo "  dev           - Development workflow (deps, fmt, vet, test)"
	@echo "  help          - Show this help"

# Default target
default: help
