# PodSweeper Gamemaster - Development Guide

This document provides instructions for setting up and developing the PodSweeper Gamemaster controller.

## Prerequisites

- **Go**: Version 1.21 or higher (project uses Go 1.21)
- **Docker**: For building container images
- **Kubernetes Cluster**: For testing (minikube, kind, or k3s recommended for local development)
- **kubectl**: Kubernetes CLI tool

## Project Structure

```
.
├── cmd/
│   └── gamemaster/          # Main application entry point
├── pkg/
│   ├── controller/          # Pod watcher and game orchestration logic
│   ├── webhook/             # Validating admission webhook handlers
│   └── game/                # Core game mechanics (grid, mines, BFS, hints)
├── internal/
│   └── config/              # Internal configuration structures
├── config/
│   └── samples/             # Sample Kubernetes manifests
├── Makefile                 # Build and development tasks
└── go.mod                   # Go module dependencies
```

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/zwindler/PodSweeper.git
cd PodSweeper
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Gamemaster

```bash
make build
```

This will create a binary at `bin/gamemaster`.

### 4. Run Tests

```bash
make test
```

### 5. Run Locally (Out-of-Cluster)

To run the gamemaster locally (requires a kubeconfig):

```bash
make run
```

The gamemaster will connect to your current Kubernetes context.

## Development Workflow

### Code Formatting

```bash
make fmt
```

### Linting

Install and run the linter:

```bash
make lint
```

To automatically fix linting issues:

```bash
make lint-fix
```

### Building Docker Image

```bash
make docker-build IMG=your-registry/podsweeper-gamemaster:tag
```

### Pushing Docker Image

```bash
make docker-push IMG=your-registry/podsweeper-gamemaster:tag
```

## Key Components (To Be Implemented)

### Controller (`pkg/controller/`)
- Watches for Pod deletion events in the `podsweeper-game` namespace
- Triggers game logic based on which pod was deleted
- Manages grid state and progression

### Webhook (`pkg/webhook/`)
- Validates deletion requests based on current level constraints
- Implements timing checks, finalizer validation, etc.
- Returns detailed error messages to the player

### Game Engine (`pkg/game/`)
- Generates the minefield grid
- Calculates hints (adjacent mine count)
- Implements BFS for empty cell propagation
- Detects victory and defeat conditions

### Configuration (`internal/config/`)
- Level definitions and progression logic
- Game parameters (grid size, mine density)
- RBAC and NetworkPolicy templates per level

## Debugging

### Enable Verbose Logging

```bash
go run ./cmd/gamemaster/main.go --zap-log-level=debug
```

### Check Controller Metrics

The controller exposes metrics on `:8080/metrics` by default.

### Health Checks

- Liveness: `:8081/healthz`
- Readiness: `:8081/readyz`

## Next Steps

1. Implement the core game logic in `pkg/game/`
2. Create the Pod controller in `pkg/controller/`
3. Implement the admission webhook in `pkg/webhook/`
4. Add Kubernetes manifests for deployment
5. Create end-to-end tests

## Contributing

Please ensure:
- Code is formatted (`make fmt`)
- Tests pass (`make test`)
- Linter passes (`make lint`)
- Documentation is updated

## References

- [controller-runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubernetes API Reference](https://kubernetes.io/docs/reference/kubernetes-api/)
- [Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
