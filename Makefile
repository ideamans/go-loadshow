.PHONY: all build build-static test unit-test integration-test e2e-test lint clean deps deps-static package help

# Variables
VERSION ?= dev
OUTPUT ?= loadshow

# Default target
all: build

# ============================================================================
# Dependencies
# ============================================================================

# Install dependencies (dynamic linking, for development/testing)
deps:
	@bash scripts/setup-deps.sh

# Install dependencies (static linking, for release builds)
deps-static:
	@bash scripts/setup-deps.sh --static

# ============================================================================
# Build
# ============================================================================

# Build the CLI (dynamic linking)
build:
	@bash scripts/build.sh --output $(OUTPUT)

# Build the CLI with version info (dynamic linking)
build-version:
	@bash scripts/build.sh --output $(OUTPUT) --version $(VERSION)

# Build the CLI (static linking, for release)
build-static:
	@bash scripts/build.sh --static --output $(OUTPUT) --version $(VERSION)

# ============================================================================
# Test
# ============================================================================

# Run all tests (unit + integration, excluding E2E)
test: unit-test integration-test

# Run unit tests only
unit-test:
	@bash scripts/test.sh --unit

# Run unit tests with coverage
unit-test-coverage:
	@bash scripts/test.sh --unit --coverage

# Run integration tests (requires Chrome)
integration-test:
	@bash scripts/test.sh --integration

# Run E2E tests (requires Chrome and network access)
e2e-test:
	@bash scripts/test.sh --e2e

# Run all tests including E2E
test-all:
	@bash scripts/test.sh --all

# Run quick tests (unit tests only, no race detector)
test-quick:
	@echo "Running quick tests..."
	go test ./pkg/...

# ============================================================================
# Package (for release)
# ============================================================================

# Create release archive
package:
	@bash scripts/package.sh --version $(VERSION) --binary $(OUTPUT)

# ============================================================================
# Development Tools
# ============================================================================

# Lint the code
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format the code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -f loadshow loadshow.exe loadshow-test
	rm -f coverage.out coverage.html
	rm -f *.tar.gz *.zip
	rm -rf ./debug ./tmp ./aom

# Run a quick demo recording
demo:
	@echo "Recording demo video..."
	./loadshow record https://dummy-ec-site.ideamans.com/ -o demo.mp4 -p mobile
	@echo "Demo video saved to demo.mp4"

# ============================================================================
# Help
# ============================================================================

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Dependencies:"
	@echo "    make deps             - Install dependencies (dynamic, for dev/test)"
	@echo "    make deps-static      - Install dependencies (static, for release)"
	@echo ""
	@echo "  Build:"
	@echo "    make build            - Build the CLI (dynamic linking)"
	@echo "    make build-version    - Build with version (dynamic)"
	@echo "    make build-static     - Build for release (static linking)"
	@echo ""
	@echo "  Test:"
	@echo "    make test             - Run unit and integration tests"
	@echo "    make unit-test        - Run unit tests only"
	@echo "    make unit-test-coverage - Run unit tests with coverage"
	@echo "    make integration-test - Run integration tests"
	@echo "    make e2e-test         - Run E2E tests"
	@echo "    make test-all         - Run all tests including E2E"
	@echo "    make test-quick       - Quick unit tests (no race detector)"
	@echo ""
	@echo "  Release:"
	@echo "    make package          - Create release archive"
	@echo ""
	@echo "  Development:"
	@echo "    make lint             - Run linter"
	@echo "    make fmt              - Format code"
	@echo "    make clean            - Clean build artifacts"
	@echo "    make demo             - Record a demo video"
	@echo ""
	@echo "  Variables:"
	@echo "    VERSION=v1.0.0        - Set version for build/package"
	@echo "    OUTPUT=myapp          - Set output binary name"
	@echo ""
	@echo "  Examples:"
	@echo "    make deps && make test"
	@echo "    make deps-static && make build-static VERSION=v1.0.0 && make package VERSION=v1.0.0"
