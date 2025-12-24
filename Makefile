.PHONY: all build test unit-test integration-test e2e-test lint clean install-deps help

# Default target
all: build

# Build the CLI
build:
	go build -o loadshow ./cmd/loadshow

# Build with version info
build-release:
	go build -ldflags "-X main.version=$(VERSION)" -o loadshow ./cmd/loadshow

# Install dependencies (for development)
install-deps:
	@echo "Installing dependencies..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		brew install aom pkg-config || true; \
	elif [ "$$(uname)" = "Linux" ]; then \
		sudo apt-get update && sudo apt-get install -y libaom-dev pkg-config || true; \
	fi

# Run all tests (unit + integration, excluding E2E)
test: unit-test integration-test

# Run unit tests only
unit-test:
	@echo "Running unit tests..."
	go test -v -race ./pkg/...

# Run unit tests with coverage
unit-test-coverage:
	@echo "Running unit tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run integration tests (requires Chrome)
integration-test:
	@echo "Running integration tests..."
	go test -v -timeout 5m ./tests/integration/...

# Run E2E tests (requires Chrome and network access)
e2e-test:
	@echo "Running E2E tests..."
	LOADSHOW_E2E=1 go test -v -timeout 10m ./tests/e2e/...

# Run all tests including E2E
test-all: unit-test integration-test e2e-test

# Run quick tests (unit tests only, no race detector)
test-quick:
	@echo "Running quick tests..."
	go test ./pkg/...

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
	rm -f loadshow loadshow-test
	rm -f coverage.out coverage.html
	rm -rf ./debug ./tmp

# Run a quick demo recording
demo:
	@echo "Recording demo video..."
	./loadshow record https://dummy-ec-site.ideamans.com/ -o demo.mp4 -p mobile
	@echo "Demo video saved to demo.mp4"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build              - Build the CLI"
	@echo "  make test               - Run unit and integration tests"
	@echo "  make unit-test          - Run unit tests only"
	@echo "  make unit-test-coverage - Run unit tests with coverage report"
	@echo "  make integration-test   - Run integration tests (requires Chrome)"
	@echo "  make e2e-test           - Run E2E tests (requires Chrome + network)"
	@echo "  make test-all           - Run all tests including E2E"
	@echo "  make test-quick         - Run quick unit tests without race detector"
	@echo "  make lint               - Run linter"
	@echo "  make fmt                - Format code"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make install-deps       - Install system dependencies"
	@echo "  make demo               - Record a demo video"
	@echo "  make help               - Show this help"
