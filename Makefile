all: build

init:
	@echo "Initializing..."
	@$(MAKE) build

build:
	@echo "Building..."
	@go mod tidy
	@go mod download
	@${MAKE} build_alone

pushall:
	@docker build -t ghcr.io/udonggeum/udonggeum-backend:latest .
	@docker push ghcr.io/udonggeum/udonggeum-backend:latest

build_alone:
	@go build -tags migrate -o bin/$(shell basename $(PWD)) ./cmd/server

run:
	@echo "Running..."
	@./bin/$(shell basename $(PWD))

clean:
	rm -f bin/$(shell basename $(PWD))