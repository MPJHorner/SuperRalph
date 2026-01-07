.PHONY: build install clean test run

# Binary name
BINARY=superralph

# Build directory
BUILD_DIR=build

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
	$(GOCLEAN)

# Run tests
test:
	$(GOTEST) -v ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application (for development)
run: build
	./$(BUILD_DIR)/$(BINARY)

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .
