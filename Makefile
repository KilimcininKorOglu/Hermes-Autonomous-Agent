# Hermes - AI-Powered Application Development
# Makefile for development and build automation

.PHONY: all build build-linux build-linux-arm64 build-windows build-windows-arm64 \
        build-darwin build-darwin-arm64 build-all-platforms test lint fmt vet \
        clean run run-tui deps deps-update install update-winres help

# Variables
BINARY_DIR := bin
BINARY_NAME := hermes
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)"

# OS/Arch detection
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Binary suffix (.exe for windows, empty for others)
ifeq ($(GOOS),windows)
    BINARY_SUFFIX := .exe
else
    BINARY_SUFFIX :=
endif

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Default target - build for all platforms
all: build-all-platforms

# ==================== BUILD TARGETS ====================

build:
	@echo "Building Hermes ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) ./cmd/hermes

# ==================== CROSS-COMPILATION ====================

build-linux:
	@echo "Building Hermes for Linux (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/hermes

build-linux-arm64:
	@echo "Building Hermes for Linux (arm64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/hermes

build-windows: update-winres
	@echo "Building Hermes for Windows (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/hermes

build-windows-arm64: update-winres
	@echo "Building Hermes for Windows (arm64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/hermes

build-darwin:
	@echo "Building Hermes for macOS (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/hermes

build-darwin-arm64:
	@echo "Building Hermes for macOS (arm64/Apple Silicon)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/hermes

build-all-platforms: build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64
	@echo "All platform binaries built successfully"

# ==================== TESTING ====================

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

# ==================== CODE QUALITY ====================

lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

check: fmt vet lint test
	@echo "All checks passed"

# ==================== CLEAN ====================

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	rm -f hermes.exe hermes

# ==================== RUN ====================

run: build
	@echo "Starting Hermes..."
	./$(BINARY_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX)

run-tui: build
	@echo "Starting Hermes TUI..."
	./$(BINARY_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) tui

# ==================== DEPENDENCIES ====================

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# ==================== INSTALL ====================

install: build
	@echo "Installing Hermes to GOPATH/bin..."
	cp $(BINARY_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) $(GOPATH)/bin/

# ==================== WINDOWS RESOURCES ====================

# Version parsing for Windows resources
WIN_VERSION := $(shell echo $(VERSION) | sed 's/^v//' | sed 's/-.*$$//')
WIN_VERSION_FULL := $(WIN_VERSION).0

update-winres:
	@if command -v go-winres >/dev/null 2>&1; then \
		echo "Updating Windows resources with version $(WIN_VERSION_FULL)..."; \
		sed 's/{{VERSION_FULL}}/$(WIN_VERSION_FULL)/g; s/{{VERSION}}/$(VERSION)/g' winres/winres.template.json > winres/winres.json; \
		go-winres make --in winres/winres.json --out cmd/hermes/rsrc_windows; \
	else \
		echo "go-winres not found, skipping Windows resources"; \
	fi

# ==================== HELP ====================

help:
	@echo "Hermes - AI-Powered Application Development"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  (no target)        Build for all platforms (default)"
	@echo "  build              Build for current platform only"
	@echo "  build-linux        Build for Linux (amd64)"
	@echo "  build-linux-arm64  Build for Linux (arm64)"
	@echo "  build-windows      Build for Windows (amd64)"
	@echo "  build-windows-arm64 Build for Windows (arm64)"
	@echo "  build-darwin       Build for macOS (amd64)"
	@echo "  build-darwin-arm64 Build for macOS (arm64/Apple Silicon)"
	@echo "  build-all-platforms Build for all platforms"
	@echo ""
	@echo "Test targets:"
	@echo "  test               Run all tests with coverage"
	@echo "  test-short         Run short tests only"
	@echo ""
	@echo "Code quality:"
	@echo "  lint               Run golangci-lint"
	@echo "  fmt                Format code and tidy modules"
	@echo "  vet                Run go vet"
	@echo "  check              Run fmt, vet, lint, and test"
	@echo ""
	@echo "Run targets:"
	@echo "  run                Build and run Hermes"
	@echo "  run-tui            Build and run Hermes TUI"
	@echo ""
	@echo "Other:"
	@echo "  deps               Download dependencies"
	@echo "  deps-update        Update dependencies"
	@echo "  install            Install to GOPATH/bin"
	@echo "  clean              Remove build artifacts"
	@echo "  help               Show this help message"
