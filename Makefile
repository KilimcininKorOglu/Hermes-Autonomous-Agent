.PHONY: build test lint clean install build-all

BINARY_NAME=hermes
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
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

# Development helpers
fmt:
	go fmt ./...

tidy:
	go mod tidy

deps:
	go mod download
