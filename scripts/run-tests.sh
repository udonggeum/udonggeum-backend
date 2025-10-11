#!/bin/bash

# Test execution script for UDONGGEUM backend

set -e

echo "========================================="
echo "   UDONGGEUM Backend Test Suite"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go 1.21 or later"
    echo "Visit: https://go.dev/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓${NC} Go version: $GO_VERSION"
echo ""

# Install dependencies
echo "Installing dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓${NC} Dependencies installed"
echo ""

# Run tests with coverage
echo "Running tests with coverage..."
echo "----------------------------------------"
go test -v -coverprofile=coverage.out ./...
TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "----------------------------------------"
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""

    # Generate coverage report
    echo "Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html

    # Show coverage summary
    echo "Coverage Summary:"
    echo "----------------------------------------"
    go tool cover -func=coverage.out | grep total

    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

    echo ""
    if (( $(echo "$COVERAGE >= 90" | bc -l) )); then
        echo -e "${GREEN}✓ Coverage goal achieved: ${COVERAGE}%${NC}"
    elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then
        echo -e "${YELLOW}⚠ Coverage: ${COVERAGE}% (Goal: 90%)${NC}"
    else
        echo -e "${RED}✗ Coverage: ${COVERAGE}% (Goal: 90%)${NC}"
    fi

    echo ""
    echo "Coverage report saved to: coverage.html"
    echo "Open it in your browser to see detailed coverage"
else
    echo ""
    echo "----------------------------------------"
    echo -e "${RED}✗ Tests failed${NC}"
    exit 1
fi

echo ""
echo "========================================="
echo "   Test Suite Complete"
echo "========================================="
