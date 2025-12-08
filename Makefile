# Docker configuration
DOCKER_REGISTRY := ghcr.io/udonggeum
DOCKER_IMAGE := udonggeum-backend
DOCKER_TAG := $(shell cat VERSION)
DOCKER_FULL_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

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
	@echo "Building and pushing Docker image to GHCR..."
	@echo "Image: $(DOCKER_FULL_IMAGE)"
	@docker build -t $(DOCKER_FULL_IMAGE) .
	@docker push $(DOCKER_FULL_IMAGE)
	@echo "Push complete!"

build_alone:
	@go build -tags migrate -o bin/$(shell basename $(PWD)) ./cmd/server

run:
	@echo "Running..."
	@./bin/$(shell basename $(PWD))

clean:
	rm -f bin/$(shell basename $(PWD))

# Show current Docker tag
docker-tag:
	@echo "Current Docker tag: $(DOCKER_TAG)"
	@echo "Full image name: $(DOCKER_FULL_IMAGE)"

.PHONY: all init build pushall build_alone run clean docker-tag