.PHONY: test test-unit test-e2e test-go test-file test-run build clean tidy help \
	check-toolchain fmt fmt-check vet lint tidy-check race-test vuln secrets \
	replace-check security check release-check release tools

BINARY := $(CURDIR)/bin/fizzy
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# Test configuration (set these or export as environment variables)
# export FIZZY_TEST_TOKEN=your-token
# export FIZZY_TEST_ACCOUNT=your-account

help:
	@echo "Fizzy CLI"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build the CLI"
	@echo "  make test-unit      Run unit tests (no API required)"
	@echo "  make test-e2e       Run e2e tests (requires API credentials)"
	@echo "  make test           Alias for test-e2e"
	@echo "  make test-file      Run a specific e2e test file"
	@echo "  make test-run       Run a specific e2e test by name"
	@echo "  make clean          Remove build artifacts"
	@echo "  make tidy           Tidy dependencies"
	@echo ""
	@echo "  make fmt            Format Go source files"
	@echo "  make fmt-check      Check formatting (CI gate)"
	@echo "  make vet            Run go vet"
	@echo "  make lint           Run golangci-lint"
	@echo "  make tidy-check     Verify go.mod/go.sum tidiness"
	@echo "  make race-test      Run unit tests with race detector"
	@echo "  make vuln           Run govulncheck"
	@echo "  make secrets        Run gitleaks secret scan"
	@echo "  make replace-check  Guard against replace directives in go.mod"
	@echo ""
	@echo "  make security       lint + vuln + secrets"
	@echo "  make check          fmt-check + vet + lint + test-unit + tidy-check"
	@echo "  make release-check  check + replace-check + vuln + race-test"
	@echo "  make release        Run release preflight and tag"
	@echo "  make tools          Install dev tools"
	@echo ""
	@echo "Environment variables (required for e2e tests):"
	@echo "  FIZZY_TEST_TOKEN   API token"
	@echo "  FIZZY_TEST_ACCOUNT Account slug"
	@echo "  FIZZY_TEST_API_URL API base URL (default: https://app.fizzy.do)"
	@echo "  FIZZY_TEST_USER_ID User ID for user update/deactivate tests (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test-unit"
	@echo "  export FIZZY_TEST_TOKEN=your-token"
	@echo "  export FIZZY_TEST_ACCOUNT=your-account"
	@echo "  make test-e2e"

# Toolchain guard — fails fast when PATH go and GOROOT go disagree
check-toolchain:
	@GOV=$$(go version | awk '{print $$3}'); \
	ROOT=$$(go env GOROOT); \
	ROOTV=$$($$ROOT/bin/go version | awk '{print $$3}'); \
	if [ "$$GOV" != "$$ROOTV" ]; then \
		echo "ERROR: Go toolchain mismatch"; \
		echo "  PATH go:   $$GOV ($$(which go))"; \
		echo "  GOROOT go: $$ROOTV ($$ROOT/bin/go)"; \
		echo "Fix: eval \"\$$(mise hook-env)\" && make <target>"; \
		exit 1; \
	fi

# Build CLI
build: check-toolchain
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/fizzy

# Run unit tests (no API required)
test-unit: check-toolchain
	go test -v ./internal/...

# Run e2e tests (requires API credentials)
test-e2e: build
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	FIZZY_TEST_BINARY=$(BINARY) go test -v ./e2e/tests/...

# Alias for test-e2e
test: test-e2e
test-go: test-e2e

# Run a single test file (e.g., make test-file FILE=board)
test-file: build
	@if [ -z "$(FILE)" ]; then echo "Usage: make test-file FILE=board"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	FIZZY_TEST_BINARY=$(BINARY) go test -v ./e2e/tests/$(FILE)_test.go

# Run a single test by name (e.g., make test-run NAME=TestBoardCRUD)
test-run: build
	@if [ -z "$(NAME)" ]; then echo "Usage: make test-run NAME=TestBoardCRUD"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	FIZZY_TEST_BINARY=$(BINARY) go test -v -run $(NAME) ./e2e/tests/...

# Format Go source
fmt:
	gofmt -s -w .

# Check formatting (CI gate)
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:"; gofmt -l .; exit 1)

# Run go vet
vet: check-toolchain
	go vet ./...

# Run golangci-lint
lint:
	golangci-lint run ./...

# Verify go.mod/go.sum tidiness (non-mutating)
tidy-check: check-toolchain
	@cp go.mod go.mod.bak && cp go.sum go.sum.bak
	@go mod tidy
	@if ! diff -q go.mod go.mod.bak >/dev/null 2>&1 || ! diff -q go.sum go.sum.bak >/dev/null 2>&1; then \
		mv go.mod.bak go.mod; mv go.sum.bak go.sum; \
		echo "go.mod or go.sum is not tidy — run 'go mod tidy'"; \
		exit 1; \
	fi
	@mv go.mod.bak go.mod && mv go.sum.bak go.sum

# Run unit tests with race detector
race-test: check-toolchain
	go test -race -count=1 ./internal/...

# Run govulncheck
vuln:
	govulncheck ./...

# Run gitleaks secret scan
secrets:
	@if ! command -v gitleaks >/dev/null 2>&1 || [ ! -f .gitleaks.toml ]; then \
		echo "Skipping gitleaks (binary not found or .gitleaks.toml absent)"; \
	else \
		gitleaks detect --source . --verbose; \
	fi

# Guard against replace directives in go.mod
replace-check:
	@if grep -q '^replace' go.mod; then \
		echo "ERROR: go.mod contains replace directives"; \
		grep '^replace' go.mod; \
		exit 1; \
	fi

# Security suite
security: lint vuln secrets

# Local CI gate
check: fmt-check vet lint test-unit tidy-check

# Release preflight
release-check: check replace-check vuln race-test

# Release (delegates to script)
release:
	@scripts/release.sh

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "For gitleaks, install via: brew install gitleaks (or see https://github.com/gitleaks/gitleaks)"

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Tidy dependencies
tidy:
	go mod tidy
