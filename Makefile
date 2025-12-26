# Talaria - Comprehensive Backup System
# Makefile for development and build automation

.PHONY: all build build-server build-client \
        build-linux build-linux-arm64 build-windows build-darwin build-all-platforms \
        test test-unit test-integration lint fmt vet clean \
        run-server run-client docker-build docker-push \
        proto deps deps-update generate help

# Variables
BINARY_DIR := bin
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

# Binary names with OS/Arch suffix
SERVER_BINARY := talaria-server-$(GOOS)-$(GOARCH)$(BINARY_SUFFIX)
CLIENT_BINARY := talaria-client-$(GOOS)-$(GOARCH)$(BINARY_SUFFIX)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Proto parameters
PROTOC := protoc
BUF := buf

# Default target
all: build

# ==================== BUILD TARGETS ====================

# Build all binaries
build: build-server build-client
	@echo "All binaries built successfully"

# Build individual binaries
build-server:
	@echo "Building Talaria server ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(SERVER_BINARY) ./cmd/talaria-server

build-client:
	@echo "Building Talaria client ($(GOOS)/$(GOARCH))..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(CLIENT_BINARY) ./cmd/talaria-client

# ==================== CROSS-COMPILATION ====================

build-linux:
	@echo "Building all binaries for Linux (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-linux-amd64 ./cmd/talaria-server
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-linux-amd64 ./cmd/talaria-client

build-linux-arm64:
	@echo "Building all binaries for Linux (arm64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-linux-arm64 ./cmd/talaria-server
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-linux-arm64 ./cmd/talaria-client

build-windows:
	@echo "Building all binaries for Windows (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-windows-amd64.exe ./cmd/talaria-server
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-windows-amd64.exe ./cmd/talaria-client

build-windows-arm64:
	@echo "Building all binaries for Windows (arm64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-windows-arm64.exe ./cmd/talaria-server
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-windows-arm64.exe ./cmd/talaria-client

build-darwin:
	@echo "Building all binaries for macOS (amd64)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-darwin-amd64 ./cmd/talaria-server
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-darwin-amd64 ./cmd/talaria-client

build-darwin-arm64:
	@echo "Building all binaries for macOS (arm64/Apple Silicon)..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-server-darwin-arm64 ./cmd/talaria-server
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/talaria-client-darwin-arm64 ./cmd/talaria-client

build-all-platforms: build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64
	@echo "All platform binaries built successfully"

# ==================== TESTING ====================

test: test-unit
	@echo "All tests completed"

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./...

test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -func=coverage.out

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

# ==================== RUN TARGETS ====================

run-server: build-server
	@echo "Starting Talaria server..."
	./$(BINARY_DIR)/$(SERVER_BINARY) -config configs/talaria.yaml

run-client: build-client
	@echo "Starting Talaria client..."
	./$(BINARY_DIR)/$(CLIENT_BINARY)

# ==================== PROTO ====================

proto:
	@echo "Generating protobuf code..."
	$(BUF) generate

proto-lint:
	@echo "Linting protobuf files..."
	$(BUF) lint

# ==================== DOCKER ====================

docker-build:
	@echo "Building Docker images..."
	docker build -t talaria-server:$(VERSION) -f deploy/docker/Dockerfile.server .
	docker build -t talaria-client:$(VERSION) -f deploy/docker/Dockerfile.client .

docker-push:
	@echo "Pushing Docker images..."
	docker push talaria-server:$(VERSION)
	docker push talaria-client:$(VERSION)

# ==================== DEPENDENCIES ====================

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# ==================== GENERATE ====================

generate:
	@echo "Generating code..."
	$(GOCMD) generate ./...

# ==================== WEB DASHBOARD ====================

web-install:
	@echo "Installing web dashboard dependencies..."
	cd web && npm install

web-build:
	@echo "Building web dashboard..."
	cd web && npm run build

web-dev:
	@echo "Starting web dashboard dev server..."
	cd web && npm run dev

# ==================== HELP ====================

help:
	@echo "Talaria - Comprehensive Backup System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets (native - auto-detects OS/Arch):"
	@echo "  build              Build all binaries for current platform"
	@echo "  build-server       Build server binary"
	@echo "  build-client       Build client binary"
	@echo ""
	@echo "Cross-compilation targets:"
	@echo "  build-linux        Build all binaries for Linux (amd64)"
	@echo "  build-linux-arm64  Build all binaries for Linux (arm64)"
	@echo "  build-windows      Build all binaries for Windows (amd64)"
	@echo "  build-windows-arm64 Build all binaries for Windows (arm64)"
	@echo "  build-darwin       Build all binaries for macOS (amd64)"
	@echo "  build-darwin-arm64 Build all binaries for macOS (arm64/Apple Silicon)"
	@echo "  build-all-platforms Build for all supported platforms"
	@echo ""
	@echo "Test targets:"
	@echo "  test               Run all tests"
	@echo "  test-unit          Run unit tests with coverage"
	@echo "  test-integration   Run integration tests"
	@echo "  test-cover         Run tests with coverage report"
	@echo ""
	@echo "Code quality:"
	@echo "  lint               Run golangci-lint"
	@echo "  fmt                Format code and tidy modules"
	@echo "  vet                Run go vet"
	@echo "  check              Run fmt, vet, lint, and test"
	@echo ""
	@echo "Run targets:"
	@echo "  run-server         Build and run server"
	@echo "  run-client         Build and run client"
	@echo ""
	@echo "Protobuf:"
	@echo "  proto              Generate protobuf code"
	@echo "  proto-lint         Lint protobuf files"
	@echo ""
	@echo "Web Dashboard:"
	@echo "  web-install        Install web dependencies"
	@echo "  web-build          Build web dashboard"
	@echo "  web-dev            Start web dev server"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       Build Docker images"
	@echo "  docker-push        Push Docker images"
	@echo ""
	@echo "Other:"
	@echo "  deps               Download dependencies"
	@echo "  deps-update        Update dependencies"
	@echo "  generate           Run go generate"
	@echo "  clean              Remove build artifacts"
	@echo "  help               Show this help message"
	@echo ""
	@echo "Binary naming convention: talaria-{component}-{os}-{arch}[.exe]"
	@echo "Example: talaria-server-linux-amd64, talaria-client-windows-amd64.exe"
