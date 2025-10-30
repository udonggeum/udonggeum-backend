all: build

init:
	@echo "Initializing..."
	@$(MAKE) build

build:
	@echo "Building..."
	@go mod tidy
	@go mod download
	@$(MAKE) sqlc_gen
	@${MAKE} build_alone

pushall:
	@docker build -t ghcr.io/udonggeum/udonggeum-backend:latest .
	@docker push ghcr.io/udonggeum/udonggeum-backend:latest

build_alone:
	@go build -tags migrate -o bin/$(shell basename $(PWD)) ./cmd

sqlc_gen:
	@echo "Generating sqlc..."
	@cd internal/infra/sqlc && \
	sqlc generate

run:
	@echo "Running..."
	@./bin/$(shell basename $(PWD))

clean:
	rm -f bin/$(shell basename $(PWD))