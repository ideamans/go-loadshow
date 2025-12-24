#!/bin/bash
set -e

# Parse arguments
TEST_TYPE="unit"  # unit, integration, e2e, all
COVERAGE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --unit) TEST_TYPE="unit"; shift ;;
        --integration) TEST_TYPE="integration"; shift ;;
        --e2e) TEST_TYPE="e2e"; shift ;;
        --all) TEST_TYPE="all"; shift ;;
        --coverage) COVERAGE=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

OS=$(detect_os)
echo "Running tests on OS: $OS"
echo "Test type: $TEST_TYPE"
echo "Coverage: $COVERAGE"

# Set up environment for Windows
setup_windows_env() {
    export CGO_ENABLED=1
    export CC=gcc
    export PKG_CONFIG_PATH="/ucrt64/lib/pkgconfig"
}

# Run unit tests
run_unit_tests() {
    echo "Running unit tests..."
    local args="-v"

    # Race detector not available on Windows with CGO
    if [ "$OS" != "windows" ]; then
        args="$args -race"
    fi

    if [ "$COVERAGE" = true ]; then
        args="$args -coverprofile=coverage.out"
    fi

    go test $args ./pkg/...

    if [ "$COVERAGE" = true ] && [ -f coverage.out ]; then
        echo "Generating coverage report..."
        go tool cover -html=coverage.out -o coverage.html
        echo "Coverage report: coverage.html"
    fi
}

# Run integration tests
run_integration_tests() {
    echo "Running integration tests..."
    go test -v -timeout 5m ./tests/integration/...
}

# Run E2E tests
run_e2e_tests() {
    echo "Running E2E tests..."
    export LOADSHOW_E2E=1
    go test -v -timeout 10m ./tests/e2e/...
}

# Set up Windows environment if needed
if [ "$OS" = "windows" ]; then
    setup_windows_env
fi

# Run selected tests
case $TEST_TYPE in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    e2e)
        run_e2e_tests
        ;;
    all)
        run_unit_tests
        run_integration_tests
        run_e2e_tests
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        exit 1
        ;;
esac

echo "Tests complete!"
