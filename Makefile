.PHONY: build install install-alias clean test test-verbose test-cover test-race lint fmt vet check ci run help

# Build variables
BINARY_NAME := staghorn
ALIAS_NAME := stag
BUILD_DIR := .
GO_FILES := $(shell find . -name '*.go' -type f)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/HartBrook/staghorn/internal/cli.Version=$(VERSION)"
GOBIN := $(shell go env GOPATH)/bin

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/staghorn

# Install to $GOPATH/bin
install: install-alias
	go install $(LDFLAGS) ./cmd/staghorn

# Create stag alias symlink
install-alias:
	@if [ -f "$(GOBIN)/$(BINARY_NAME)" ] || go install $(LDFLAGS) ./cmd/staghorn 2>/dev/null; then \
		ln -sf $(BINARY_NAME) $(GOBIN)/$(ALIAS_NAME) && \
		echo "Created alias: $(ALIAS_NAME) -> $(BINARY_NAME)"; \
	fi

# Remove build artifacts
clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	go clean -cache -testcache

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -cover ./...

# Run tests with coverage report
test-cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with race detection and coverage (same as CI)
test-race:
	go test -race -coverprofile=coverage.out ./...

# Run linter (installs golangci-lint if missing)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOBIN)/golangci-lint run

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

# Run all CI steps locally (mirrors GitHub Actions workflow)
ci: fmt vet
	@echo "==> Running tests with race detection..."
	go test -race -coverprofile=coverage.out ./...
	@echo "==> Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOBIN)/golangci-lint run --timeout=5m
	@echo "==> Building binary..."
	go build -v ./cmd/staghorn
	@echo "==> Verifying binary runs..."
	./$(BINARY_NAME) version
	@echo "==> CI passed!"

# Run the binary (builds first if needed)
run: build
	./$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build          Build the binary"
	@echo "  install        Install to GOPATH/bin (includes stag alias)"
	@echo "  install-alias  Create stag -> staghorn symlink"
	@echo "  clean          Remove build artifacts"
	@echo "  test           Run tests"
	@echo "  test-verbose   Run tests with verbose output"
	@echo "  test-cover     Run tests with coverage"
	@echo "  test-cover-html Generate HTML coverage report"
	@echo "  test-race      Run tests with race detection (same as CI)"
	@echo "  lint           Run golangci-lint"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo "  check          Run all checks (fmt, vet, lint, test)"
	@echo "  ci             Run all CI steps locally"
	@echo "  run            Build and run the binary"
	@echo "  help           Show this help"
