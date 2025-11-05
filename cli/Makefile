.PHONY: build test clean install deps

BINARY_NAME=fybrk
BUILD_DIR=bin
MAIN_PATH=./cmd/fybrk

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	go clean
	rm -rf $(BUILD_DIR)

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
