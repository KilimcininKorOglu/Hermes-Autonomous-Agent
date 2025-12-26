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
│   ├── hermes/
│   ├── hermes-prd/
│   ├── hermes-add/
│   └── hermes-setup/
├── internal/
│   ├── ai/
│   ├── config/
│   ├── task/
│   ├── git/
│   ├── circuit/
│   ├── prompt/
│   ├── analyzer/
│   └── ui/
├── pkg/
│   └── hermes/
└── templates/
```

### 1.3 Add Dependencies

```bash
go get github.com/severity1/claude-code-sdk-go
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/fatih/color
```

### 1.4 Create Makefile

```makefile
.PHONY: build test lint clean install

BINARY_NAME=hermes
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/hermes
	go build $(LDFLAGS) -o bin/hermes-prd ./cmd/hermes-prd
	go build $(LDFLAGS) -o bin/hermes-add ./cmd/hermes-add
	go build $(LDFLAGS) -o bin/hermes-setup ./cmd/hermes-setup

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install: build
	cp bin/* $(GOPATH)/bin/
```

### 1.5 Create Basic Main Files

Each command gets a minimal main.go as placeholder:

```go
// cmd/hermes/main.go
package main

import "fmt"

var version = "dev"

func main() {
    fmt.Println("Hermes Autonomous Agent", version)
}
```

## Files to Create

| File | Description |
|------|-------------|
| `go.mod` | Go module definition |
| `go.sum` | Dependency checksums (auto-generated) |
| `Makefile` | Build automation |
| `cmd/hermes/main.go` | Main CLI entry |
| `cmd/hermes-prd/main.go` | PRD parser entry |
| `cmd/hermes-add/main.go` | Feature add entry |
| `cmd/hermes-setup/main.go` | Setup entry |

## Acceptance Criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` succeeds (no tests yet, but should not error)
- [ ] All dependencies resolve correctly
- [ ] Makefile targets work
