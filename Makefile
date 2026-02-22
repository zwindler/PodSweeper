# PodSweeper Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary names
GAMEMASTER_BINARY=gamemaster
HINT_AGENT_BINARY=hint-agent

# Build directories
BUILD_DIR=bin
CMD_DIR=cmd

# Container runtime: auto-detect podman or docker
CONTAINER_RUNTIME?=$(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
REGISTRY?=ghcr.io/zwindler
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Kubernetes parameters
NAMESPACE=podsweeper-game

.PHONY: all build build-gamemaster build-hint-agent test test-coverage clean run run-gamemaster fmt vet lint deps tidy docker-build docker-push deploy undeploy help play

## Default target
all: fmt vet test build

## Build all binaries
build: build-gamemaster build-hint-agent

## Build the gamemaster binary
build-gamemaster:
	@echo "Building gamemaster..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(GAMEMASTER_BINARY) -v ./$(CMD_DIR)/gamemaster

## Build the hint-agent binary
build-hint-agent:
	@echo "Building hint-agent..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(HINT_AGENT_BINARY) -v ./$(CMD_DIR)/hint-agent

## Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## Run the gamemaster locally (requires kubeconfig)
run: run-gamemaster

## Run the gamemaster
run-gamemaster: build-gamemaster
	@echo "Running gamemaster..."
	./$(BUILD_DIR)/$(GAMEMASTER_BINARY)

## Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## Run golangci-lint (must be installed separately)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -v -t -d ./...

## Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

## Build Docker images
docker-build: docker-build-gamemaster docker-build-hint-agent docker-build-player-terminal

## Build gamemaster Docker image
docker-build-gamemaster:
	@echo "Building gamemaster Docker image..."
	$(CONTAINER_RUNTIME) build -t $(REGISTRY)/podsweeper-gamemaster:$(VERSION) -f build/gamemaster/Dockerfile --build-arg VERSION=$(VERSION) .

## Build hint-agent Docker image
docker-build-hint-agent:
	@echo "Building hint-agent Docker image..."
	$(CONTAINER_RUNTIME) build -t $(REGISTRY)/podsweeper-hint-agent:$(VERSION) -f build/hint-agent/Dockerfile --build-arg VERSION=$(VERSION) .

## Build player-terminal Docker image
docker-build-player-terminal:
	@echo "Building player-terminal Docker image..."
	$(CONTAINER_RUNTIME) build -t $(REGISTRY)/podsweeper-player-terminal:$(VERSION) -f build/player-terminal/Dockerfile build/player-terminal

## Push Docker images
docker-push: docker-push-gamemaster docker-push-hint-agent docker-push-player-terminal

## Push gamemaster Docker image
docker-push-gamemaster:
	@echo "Pushing gamemaster Docker image..."
	$(CONTAINER_RUNTIME) push $(REGISTRY)/podsweeper-gamemaster:$(VERSION)

## Push hint-agent Docker image
docker-push-hint-agent:
	@echo "Pushing hint-agent Docker image..."
	$(CONTAINER_RUNTIME) push $(REGISTRY)/podsweeper-hint-agent:$(VERSION)

## Push player-terminal Docker image
docker-push-player-terminal:
	@echo "Pushing player-terminal Docker image..."
	$(CONTAINER_RUNTIME) push $(REGISTRY)/podsweeper-player-terminal:$(VERSION)

## Generate code (for future CRDs if needed)
generate:
	@echo "Running code generation..."
	$(GOCMD) generate ./...

## Deploy to Kubernetes cluster
deploy:
	@echo "Deploying PodSweeper..."
	kubectl apply -k deploy/base

## Remove PodSweeper from cluster
undeploy:
	@echo "Removing PodSweeper..."
	kubectl delete -k deploy/base --ignore-not-found

## Start a game (creates ConfigMap with action=start)
start-game:
	@echo "Starting game at level 0..."
	kubectl patch configmap podsweeper-config -n $(NAMESPACE) --type merge -p '{"data":{"level":"0","action":"start"}}'

## Start a game at a specific level (use: make start-level LEVEL=3)
start-level:
	@echo "Starting game at level $(LEVEL)..."
	kubectl patch configmap podsweeper-config -n $(NAMESPACE) --type merge -p '{"data":{"level":"$(LEVEL)","action":"start"}}'

## Show game status
game-status:
	@kubectl get configmap podsweeper-config -n $(NAMESPACE) -o yaml | grep -E "^  (level|status|message|progress|gridSize|mines):"

## Join the game as a player (exec into player terminal)
play:
	@echo "Joining PodSweeper as player..."
	@kubectl get pod player -n $(NAMESPACE) > /dev/null 2>&1 || (echo "Player terminal not running. Deploying..." && kubectl apply -f deploy/base/player-terminal.yaml)
	@kubectl wait --for=condition=Ready pod/player -n $(NAMESPACE) --timeout=30s 2>/dev/null || true
	@kubectl exec -it player -n $(NAMESPACE) -- bash

## Show help
help:
	@echo "PodSweeper - The most impractical way to play Minesweeper"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  all                 Format, vet, test, and build (default)"
	@echo "  build               Build all binaries"
	@echo "  build-gamemaster    Build the gamemaster binary"
	@echo "  build-hint-agent    Build the hint-agent binary"
	@echo "  test                Run all tests"
	@echo "  test-coverage       Run tests with coverage report"
	@echo "  clean               Remove build artifacts"
	@echo "  fmt                 Format Go code"
	@echo "  vet                 Run go vet"
	@echo "  lint                Run golangci-lint"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build                 Build all Docker images"
	@echo "  docker-build-gamemaster      Build gamemaster image"
	@echo "  docker-build-hint-agent      Build hint-agent image"
	@echo "  docker-build-player-terminal Build player terminal image"
	@echo "  docker-push                  Push all Docker images"
	@echo ""
	@echo "Kubernetes Targets:"
	@echo "  deploy              Deploy to Kubernetes cluster"
	@echo "  undeploy            Remove from cluster"
	@echo "  start-game          Start a new game at level 0"
	@echo "  start-level         Start at specific level (LEVEL=N)"
	@echo "  game-status         Show current game status"
	@echo "  play                Join the game (exec into player terminal)"
	@echo ""
	@echo "Quick Start:"
	@echo "  make deploy && make start-game && make play"
