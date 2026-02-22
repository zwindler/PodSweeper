# PodSweeper

> **The most impractical, over-engineered, and chaotic way to play Minesweeper.**

**PodSweeper** is a cloud-native "deminer" game where the cells aren't boxes on a screen, but **live Pods** inside a Kubernetes cluster. To "click" on a cell, you don't use a mouse; you use `kubectl delete`.

---

## Project Status: MVP Complete

The core game is **playable**! You can:
- Start a game and see a grid of pods
- Delete pods to reveal cells
- See hint pods with adjacent mine counts
- Trigger chain reactions on empty cells (BFS propagation)
- Win by clearing all safe cells
- Lose by hitting a mine (with ASCII explosion art)

**Coming soon:** 10 hardening levels that turn the game into a Kubernetes CTF.

---

## Quick Start

### Prerequisites
- A Kubernetes cluster (kind, minikube, or any cluster you have access to)
- `kubectl` configured to access the cluster

### Installation

```bash
# Clone the repository
git clone https://github.com/zwindler/podsweeper.git
cd podsweeper

# Deploy to your cluster
kubectl apply -k deploy/base/

# Wait for the gamemaster to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=podsweeper -n podsweeper-game --timeout=60s
```

### Playing the Game

```bash
# Start a new game
kubectl patch configmap podsweeper-config -n podsweeper-game \
  --type merge -p '{"data":{"action":"start"}}'

# See the grid
kubectl get pods -n podsweeper-game

# "Click" a cell by deleting it
kubectl delete pod pod-2-2 -n podsweeper-game

# Check for hints (hint pods expose adjacent mine count)
kubectl get pods -n podsweeper-game

# If you hit a mine... BOOM! Check the explosion pod logs
kubectl logs explosion -n podsweeper-game

# Win by clearing all safe cells, then check the victory message
kubectl logs victory -n podsweeper-game
```

### Grid Layout

The grid scales with difficulty level:
- **Levels 0-4:** 5×5 grid with 4 mines (learning mode)
- **Levels 5-7:** 10×10 grid with 15 mines (intermediate)
- **Levels 8-9:** 20×20 grid with 60 mines (expert)

---

## The Concept

PodSweeper turns your Kubernetes namespace into a minefield.

1. **The Grid:** The Gamemaster (a Go-based controller) spawns a matrix of pods named `pod-x-y`.
2. **The Action:** Deleting a pod is your way of "sweeping" the tile.
   - **Safe Pod:** Replaced by a `hint-x-y` pod exposing the number of adjacent mines via HTTP.
   - **Empty Area:** If no mines are nearby, a chain reaction (BFS) clears the area automatically.
   - **Mined Pod:** The namespace "explodes" (all game pods wiped) and it's Game Over.

---

## Learning through Hardening (CTF Mode)

PodSweeper isn't just a game; it's a **Kubernetes CTF**. The game features **10 levels of increasing difficulty**. 

As you progress, the Gamemaster hardens the cluster to prevent "cheating" and force you to master deeper K8s concepts:
* **RBAC:** Access is stripped away, forcing you to find info in logs or events.
* **NetworkPolicies:** Your "cheat" scripts are blocked by network isolation.
* **Finalizers:** Deletions become sticky, requiring manual patches.
* **Admission Webhooks:** A timing-based challenge where deletions are only accepted within a 100ms window.

---

## Technical Stack

- **Language:** Go
- **Framework:** `controller-runtime` / `client-go`
- **Architecture:** Kubernetes Controller watching Pod deletions (Webhook added for Levels 5-9)
- **Deployment:** Kustomize manifests
- **UI:** 100% terminal-based (`kubectl` + ASCII art)

---

## Building from Source

```bash
# Build binaries
make build

# Run tests
make test

# Build container images (requires podman or docker)
make docker-build

# Deploy to kind cluster
make docker-push  # loads images to kind
kubectl apply -k deploy/base/
```

---

## Why PodSweeper?

Because running `kubectl delete pod` should be scary, and we wanted to make it fun. This project is perfect for:
- **K8s Newbies:** Learn basic CLI and resources
- **SREs/DevOps:** Test your scripting and troubleshooting skills under pressure
- **Security Folks:** Understand how RBAC and Admission Controllers enforce policies

---

## Disclaimer

PodSweeper is designed to be destructive within its own namespace. **Do not run this in a production cluster** unless you really want to explain to your boss why you were playing Minesweeper with the company's infrastructure.

---

*Created with ❤️ during a few vibe coding sessions.*
