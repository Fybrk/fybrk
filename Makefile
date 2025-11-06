# Fybrk - Simple P2P File Sync
# Makefile for building and testing

.PHONY: all build test clean install coverage lint fmt vet

# Build configuration
BINARY_NAME=fybrk
BUILD_DIR=bin
CMD_DIR=cmd/fybrk
PKG_DIR=pkg/core

# Go configuration
GO=go
GOFLAGS=-ldflags="-s -w"
COVERAGE_FILE=coverage.out

all: fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Run all tests with coverage
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	@echo "Tests completed"

# Run tests with coverage report
coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	$(GO) tool cover -func=$(COVERAGE_FILE)
	@echo "Coverage report generated: coverage.html"

# Run core package tests only
test-core:
	@echo "Running core package tests..."
	$(GO) test -v -race -coverprofile=core_coverage.out ./$(PKG_DIR)
	$(GO) tool cover -func=core_coverage.out

# Run CLI tests only  
test-cli:
	@echo "Running CLI tests..."
	$(GO) test -v -race ./$(CMD_DIR)

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install ./$(CMD_DIR)
	@echo "Installed $(BINARY_NAME) to $(shell go env GOPATH)/bin"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GO) vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) core_coverage.out coverage.html
	$(GO) clean

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Quick development cycle
dev: fmt vet test-core build
	@echo "Development build complete"

# Integration test (builds and runs basic functionality)
integration: build
	@echo "Running integration test..."
	@mkdir -p test_temp
	@cd test_temp && ../$(BUILD_DIR)/$(BINARY_NAME) help
	@rm -rf test_temp
	@echo "Integration test passed"

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Check for race conditions
race:
	@echo "Running race detection tests..."
	$(GO) test -race ./...

# Generate test coverage badge (requires gopherbadger)
badge:
	@if command -v gopherbadger >/dev/null 2>&1; then \
		gopherbadger -png -o coverage_badge.png; \
	else \
		echo "gopherbadger not installed, install with: go install github.com/jpoles1/gopherbadger@latest"; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  all         - Format, vet, test, and build"
	@echo "  build       - Build the binary"
	@echo "  test        - Run all tests with coverage"
	@echo "  test-core   - Run core package tests only"
	@echo "  test-cli    - Run CLI tests only"
	@echo "  coverage    - Generate HTML coverage report"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  fmt         - Format code"
	@echo "  vet         - Vet code"
	@echo "  lint        - Lint code (requires golangci-lint)"
	@echo "  clean       - Clean build artifacts"
	@echo "  dev-deps    - Install development dependencies"
	@echo "  dev         - Quick development cycle (fmt, vet, test-core, build)"
	@echo "  integration - Build and run basic integration test"
	@echo "  bench       - Run benchmark tests"
	@echo "  race        - Run race detection tests"
	@echo "  badge       - Generate coverage badge (requires gopherbadger)"
	@echo "  help        - Show this help"
