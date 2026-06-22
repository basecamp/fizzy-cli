.PHONY: test test-unit test-e2e e2e test-go test-file e2e-file test-run e2e-run build clean tidy help \
	check-toolchain fmt fmt-check vet lint tidy-check race-test vuln secrets \
	replace-check security check release-check release tools \
	surface-snapshot surface-check lint-actions

BINARY := $(CURDIR)/bin/fizzy
FIZZY_TEST_BINARY ?= $(BINARY)

# Load local test credentials if present, but refuse tracked local secret files.
ifneq ($(shell git ls-files --error-unmatch .env.test >/dev/null 2>&1 && echo tracked),)
$(error .env.test is tracked by Git. Remove it from version control and keep local secret files untracked)
endif
-include .env.test
export FIZZY_TEST_TOKEN FIZZY_TEST_ACCOUNT FIZZY_TEST_API_URL FIZZY_TEST_BINARY
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# Test configuration (set these or export as environment variables)
# export FIZZY_TEST_TOKEN=your-token
# export FIZZY_TEST_ACCOUNT=your-account

# Default target — local CI gate
.DEFAULT_GOAL := check

help:
	@echo "Fizzy CLI"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build the CLI"
	@echo "  make test-unit      Run unit tests (no API required)"
	@echo "  make e2e            Run owner-only CLI contract e2e tests"
	@echo "  make test-e2e       Alias for e2e"
	@echo "  make test           Alias for e2e"
	@echo "  make e2e-file       Run a specific CLI contract e2e test file"
	@echo "  make test-file      Alias for e2e-file"
	@echo "  make e2e-run        Run a specific CLI contract e2e test by name"
	@echo "  make test-run       Alias for e2e-run"
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
	@echo "  make lint-actions   Lint GitHub Actions workflows"
	@echo "  make security       lint + vuln + secrets"
	@echo "  make check          fmt-check + vet + lint + test-unit + tidy-check"
	@echo "  make release-check  check + replace-check + vuln + race-test"
	@echo "  make release        Run release preflight and tag"
	@echo "  make tools          Install dev tools"
	@echo ""
	@echo "Environment variables (required for e2e tests):"
	@echo "  FIZZY_TEST_TOKEN        API token"
	@echo "  FIZZY_TEST_ACCOUNT      Account slug"
	@echo "  FIZZY_TEST_API_URL      API base URL (default: https://app.fizzy.do)"
	@echo "  FIZZY_TEST_BINARY       Prebuilt binary path (optional)"
	@echo "  FIZZY_E2E_KEEP_FIXTURE  Set to 1 to skip final fixture teardown"
	@echo "  FIZZY_E2E_TEARDOWN_DELAY  Delay teardown by N seconds"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test-unit"
	@echo "  export FIZZY_TEST_TOKEN=your-token"
	@echo "  export FIZZY_TEST_ACCOUNT=your-account"
	@echo "  make e2e"

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
e2e: build
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	go test -v -count=1 -timeout 10m ./e2e/cli_tests/...

test-e2e: e2e

test: e2e
test-go: e2e

# Run a single test file (e.g., make e2e-file FILE=crud_board)
e2e-file: build
	@if [ -z "$(FILE)" ]; then echo "Usage: make e2e-file FILE=crud_board"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	go test -v -count=1 ./e2e/cli_tests/$(FILE)_test.go

test-file: e2e-file

# Run a single test by name (e.g., make e2e-run NAME=TestBoardList)
e2e-run: build
	@if [ -z "$(NAME)" ]; then echo "Usage: make e2e-run NAME=TestBoardList"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_TOKEN" ]; then echo "Error: FIZZY_TEST_TOKEN not set"; exit 1; fi
	@if [ -z "$$FIZZY_TEST_ACCOUNT" ]; then echo "Error: FIZZY_TEST_ACCOUNT not set"; exit 1; fi
	go test -v -count=1 -run $(NAME) ./e2e/cli_tests/...

test-run: e2e-run

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

# Local CI gate (fmt, vet, lint, tidy, race-test)
check: fmt-check vet lint tidy-check race-test

# Release preflight
release-check: check replace-check vuln

# Release (delegates to script)
release:
	@scripts/release.sh

# Lint GitHub Actions workflows
lint-actions:
	actionlint
	zizmor .

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "For gitleaks, install via: brew install gitleaks (or see https://github.com/gitleaks/gitleaks)"
	@for tool in actionlint shellcheck zizmor; do \
		if ! command -v "$$tool" > /dev/null 2>&1; then \
			if command -v brew > /dev/null 2>&1; then \
				brew install "$$tool"; \
			elif command -v pacman > /dev/null 2>&1; then \
				sudo pacman -S --noconfirm "$$tool"; \
			else \
				echo "Error: install $$tool manually" >&2; \
				exit 1; \
			fi; \
		fi; \
	done

# Regenerate SURFACE.txt
surface-snapshot:
	GENERATE_SURFACE=1 go test ./internal/commands/ -run TestGenerateSurfaceSnapshot -v

# CI check: SURFACE.txt is up to date
surface-check:
	go test ./internal/commands/ -run TestSurfaceSnapshot -v

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Tidy dependencies
tidy:
	go mod tidy
