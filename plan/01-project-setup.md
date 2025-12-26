# Phase 01: Project Setup and Dependencies

## Goal

Initialize Go module, set up project structure, and configure dependencies.

## Tasks

### 1.1 Initialize Go Module

```bash
cd go
go mod init github.com/user/hermes
```

### 1.2 Create Directory Structure

```
go/
├── cmd/
│   └── hermes/          # Single binary with subcommands
├── internal/
│   ├── ai/              # AI provider (claude-code-sdk-go)
│   ├── cmd/             # Cobra subcommands
│   ├── config/          # Configuration
│   ├── task/            # Task parsing
│   ├── git/             # Git operations
│   ├── circuit/         # Circuit breaker
│   ├── prompt/          # Prompt injection
│   ├── analyzer/        # Response analysis
│   ├── ui/              # Console output
│   └── tui/             # Terminal UI (bubbletea)
├── pkg/
│   └── hermes/
└── templates/
```

### 1.3 Add Dependencies

```bash
# AI Provider - Claude Code CLI SDK
go get github.com/severity1/claude-code-sdk-go@v0.4.0

# CLI Framework
go get github.com/spf13/cobra

# Configuration
go get github.com/spf13/viper

# TUI Framework
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles

# Console colors (non-TUI mode)
go get github.com/fatih/color
```

### 1.4 Create Makefile

```makefile
.PHONY: build test lint clean install

BINARY_NAME=hermes
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Build single binary with all subcommands
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/hermes

# Cross-platform builds
build-all:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/hermes
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/hermes
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/hermes
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/hermes

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/
```

### 1.5 Create Basic Main File

Single entry point with subcommands:

```go
// cmd/hermes/main.go
package main

import (
    "os"

    "github.com/spf13/cobra"
    "hermes/internal/cmd"
)

var version = "dev"

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes",
        Short:   "Hermes Autonomous Agent",
        Long:    "AI-powered autonomous application development using Claude Code SDK",
        Version: version,
    }

    // Add subcommands
    rootCmd.AddCommand(cmd.NewRunCmd())
    rootCmd.AddCommand(cmd.NewPrdCmd())
    rootCmd.AddCommand(cmd.NewAddCmd())
    rootCmd.AddCommand(cmd.NewInitCmd())
    rootCmd.AddCommand(cmd.NewStatusCmd())
    rootCmd.AddCommand(cmd.NewTuiCmd())
    rootCmd.AddCommand(cmd.NewResetCmd())

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Files to Create

| File | Description |
|------|-------------|
| `go.mod` | Go module definition |
| `go.sum` | Dependency checksums (auto-generated) |
| `Makefile` | Build automation |
| `cmd/hermes/main.go` | Main CLI entry with subcommands |

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` succeeds (no tests yet, but should not error)
- [ ] All dependencies resolve correctly
- [ ] Makefile targets work
