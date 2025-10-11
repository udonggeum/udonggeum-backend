.PHONY: help run build test clean install migrate seed

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod tidy

run: ## Run the application
	go run cmd/server/main.go

build: ## Build the application
	go build -o bin/server cmd/server/main.go

build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/server/main.go

build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 go build -o bin/server.exe cmd/server/main.go

build-mac: ## Build for macOS
	GOOS=darwin GOARCH=amd64 go build -o bin/server-mac cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-short: ## Run tests without verbose
	go test ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total

test-unit: ## Run unit tests only
	go test -v -short ./pkg/... ./internal/app/repository/... ./internal/app/service/...

test-integration: ## Run integration tests
	go test -v ./internal/app/integration_test.go

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -t udonggeum-backend:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env udonggeum-backend:latest

dev: ## Run in development mode with hot reload (requires air)
	air

.env: ## Create .env file from example
	cp .env.example .env
	@echo ".env file created. Please update it with your configuration."
