# LogFiend Makefile

.PHONY: build clean test run help install deps

# Variables
BINARY_NAME=logfiend
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build the application
build:
	@echo "Building LogFiend..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run with default config
run:
	@echo "Running LogFiend with default config..."
	go run . -config=config.yml

# Run with Elasticsearch example
run-elasticsearch:
	@echo "Running LogFiend with Elasticsearch config..."
	go run . -config=examples/elasticsearch.yml

# Run with Splunk example
run-splunk:
	@echo "Running LogFiend with Splunk config..."
	go run . -config=examples/splunk.yml

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
