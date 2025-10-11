#!/bin/bash

# Build script for UDONGGEUM backend

set -e

echo "========================================="
echo "   UDONGGEUM Backend Build"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓${NC} Go version: $GO_VERSION"
echo ""

# Create bin directory
mkdir -p bin

# Install dependencies
echo "Installing dependencies..."
go mod download
go mod tidy
echo -e "${GREEN}✓${NC} Dependencies installed"
echo ""

# Format code
echo "Formatting code..."
go fmt ./...
echo -e "${GREEN}✓${NC} Code formatted"
echo ""

# Run go vet
echo "Running go vet..."
go vet ./...
echo -e "${GREEN}✓${NC} go vet passed"
echo ""

# Build
echo "Building application..."
go build -o bin/server cmd/server/main.go
BUILD_EXIT_CODE=$?

if [ $BUILD_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Build successful!"
    echo ""
    echo "Binary created: bin/server"
    echo ""
    echo "To run the server:"
    echo "  ./bin/server"
else
    echo -e "${RED}✗${NC} Build failed"
    exit 1
fi

echo ""
echo "========================================="
echo "   Build Complete"
echo "========================================="
