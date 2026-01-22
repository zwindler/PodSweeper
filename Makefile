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

# Docker parameters
DOCKER=docker
REGISTRY?=ghcr.io/zwindler
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Kubernetes parameters
NAMESPACE=podsweeper-game

.PHONY: all build build-gamemaster build-hint-agent test test-coverage clean run run-gamemaster fmt vet lint deps tidy docker-build docker-push help

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
docker-build: docker-build-gamemaster docker-build-hint-agent

## Build gamemaster Docker image
docker-build-gamemaster:
	@echo "Building gamemaster Docker image..."
	$(DOCKER) build -t $(REGISTRY)/podsweeper-gamemaster:$(VERSION) -f build/docker/gamemaster/Dockerfile .

## Build hint-agent Docker image
docker-build-hint-agent:
	@echo "Building hint-agent Docker image..."
	$(DOCKER) build -t $(REGISTRY)/podsweeper-hint-agent:$(VERSION) -f build/docker/hint-agent/Dockerfile .

## Push Docker images
docker-push: docker-push-gamemaster docker-push-hint-agent

## Push gamemaster Docker image
docker-push-gamemaster:
	@echo "Pushing gamemaster Docker image..."
	$(DOCKER) push $(REGISTRY)/podsweeper-gamemaster:$(VERSION)

## Push hint-agent Docker image
docker-push-hint-agent:
	@echo "Pushing hint-agent Docker image..."
	$(DOCKER) push $(REGISTRY)/podsweeper-hint-agent:$(VERSION)

## Generate code (for future CRDs if needed)
generate:
	@echo "Running code generation..."
	$(GOCMD) generate ./...

## Install gamemaster binary to GOPATH/bin
install: build-gamemaster
	@echo "Installing gamemaster..."
	cp $(BUILD_DIR)/$(GAMEMASTER_BINARY) $(GOPATH)/bin/

## Show help
help:
	@echo "PodSweeper - The most impractical way to play Minesweeper"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all                 Format, vet, test, and build (default)"
	@echo "  build               Build all binaries"
	@echo "  build-gamemaster    Build the gamemaster binary"
	@echo "  build-hint-agent    Build the hint-agent binary"
	@echo "  test                Run all tests"
	@echo "  test-coverage       Run tests with coverage report"
	@echo "  clean               Remove build artifacts"
	@echo "  run                 Run gamemaster locally"
	@echo "  fmt                 Format Go code"
	@echo "  vet                 Run go vet"
	@echo "  lint                Run golangci-lint"
	@echo "  deps                Download dependencies"
	@echo "  tidy                Tidy go.mod"
	@echo "  docker-build        Build all Docker images"
	@echo "  docker-push         Push all Docker images"
	@echo "  help                Show this help message"
