.PHONY: build install clean test run lint lint-fix fmt coverage check setup vet

# Binary name
BINARY=superralph

# Build directory
BUILD_DIR=build

# Coverage output
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt
GOIMPORTS=goimports

# Linker flags for version info
LDFLAGS=-ldflags "-s -w \
	-X github.com/mpjhorner/superralph/internal/version.Version=$(VERSION) \
	-X github.com/mpjhorner/superralph/internal/version.BuildTime=$(BUILD_TIME) \
	-X github.com/mpjhorner/superralph/internal/version.GitCommit=$(GIT_COMMIT)"

# Build the binary
build:
	@echo "Building $(BINARY) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

# Install to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Done! You can now run '$(BINARY)' from anywhere."

# Install to user's go bin (doesn't require sudo)
install-user: build
	@echo "Installing $(BINARY) to ~/go/bin..."
	@mkdir -p ~/go/bin
	@cp $(BUILD_DIR)/$(BINARY) ~/go/bin/$(BINARY)
	@echo "Done! Make sure ~/go/bin is in your PATH."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	$(GOCLEAN)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "Coverage report: $(COVERAGE_HTML)"

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application (for development)
run: build
	./$(BUILD_DIR)/$(BINARY)

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Run linter with auto-fix
lint-fix:
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w -s .
	@if command -v $(GOIMPORTS) > /dev/null 2>&1; then \
		echo "Organizing imports..."; \
		$(GOIMPORTS) -w -local github.com/mpjhorner/superralph .; \
	else \
		echo "goimports not found, skipping import organization"; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@echo "Installing golangci-lint..."
	@if ! command -v golangci-lint > /dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	else \
		echo "golangci-lint already installed"; \
	fi
	@echo "Installing goimports..."
	@if ! command -v goimports > /dev/null 2>&1; then \
		go install golang.org/x/tools/cmd/goimports@latest; \
	else \
		echo "goimports already installed"; \
	fi
	@echo "Installing govulncheck..."
	@if ! command -v govulncheck > /dev/null 2>&1; then \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	else \
		echo "govulncheck already installed"; \
	fi
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Setup complete!"

# Security vulnerability check
vuln:
	@echo "Checking for vulnerabilities..."
	govulncheck ./...

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  install-user- Install to ~/go/bin"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  coverage    - Run tests with coverage report"
	@echo "  lint        - Run golangci-lint"
	@echo "  lint-fix    - Run golangci-lint with auto-fix"
	@echo "  fmt         - Format code with gofmt and goimports"
	@echo "  vet         - Run go vet"
	@echo "  check       - Run all checks (fmt, vet, lint, test)"
	@echo "  setup       - Install development tools"
	@echo "  vuln        - Check for security vulnerabilities"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  run         - Build and run the application"
	@echo "  build-all   - Build for all platforms"
	@echo "  help        - Show this help"
