```
██████╗  ██████╗ ██████╗ ███████╗██╗    ██╗███████╗███████╗██████╗ ███████╗██████╗ 
██╔══██╗██╔═══██╗██╔══██╗██╔════╝██║    ██║██╔════╝██╔════╝██╔══██╗██╔════╝██╔══██╗
██████╔╝██║   ██║██║  ██║███████╗██║ █╗ ██║█████╗  █████╗  ██████╔╝█████╗  ██████╔╝
██╔═══╝ ██║   ██║██║  ██║╚════██║██║███╗██║██╔══╝  ██╔══╝  ██╔═══╝ ██╔══╝  ██╔══██╗
██║     ╚██████╔╝██████╔╝███████║╚███╔███╔╝███████╗███████╗██║     ███████╗██║  ██║
╚═╝      ╚═════╝ ╚═════╝ ╚══════╝ ╚══╝╚══╝ ╚══════╝╚══════╝╚═╝     ╚══════╝╚═╝  ╚═╝
                                                                                                 
      +-+-+-+-+-+      __  __ _                                      
      |.|1|X|1|.|     |  \/  (_)_ __   ___  _____      _____  ___ _ __ 
      +-+-+-+-+-+     | |\/| | | '_ \ / _ \/ __\ \ /\ / / _ \/ _ \ '__|
      |.|1|2|2|1|     | |  | | | | | |  __/\__ \\ V  V /  __/  __/ |   
      +-+-+-+-+-+     |_|  |_|_|_| |_|\___||___/ \_/\_/ \___|\___|_|   
      |.|.|1|X|1|                                                     
      +-+-+-+-+-+     ...but in Kubernetes. With kubectl delete.     
      |.|.|1|1|1|                                                     
      +-+-+-+-+-+     💥 BOOM! You hit a mine! 💥                    
```

<p align="center">
  <strong>The most impractical, over-engineered, and chaotic way to play Minesweeper.</strong>
</p>

<p align="center">
  <a href="https://github.com/zwindler/podsweeper/actions"><img src="https://img.shields.io/github/actions/workflow/status/zwindler/podsweeper/ci.yaml?branch=main&style=flat-square&logo=github&label=CI" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/zwindler/podsweeper"><img src="https://goreportcard.com/badge/github.com/zwindler/podsweeper?style=flat-square" alt="Go Report Card"></a>
  <a href="https://github.com/zwindler/podsweeper/releases"><img src="https://img.shields.io/github/v/release/zwindler/podsweeper?style=flat-square&logo=github" alt="Release"></a>
  <a href="https://github.com/zwindler/podsweeper/blob/main/LICENSE"><img src="https://img.shields.io/github/license/zwindler/podsweeper?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#the-concept">Concept</a> •
  <a href="#learning-through-hardening-ctf-mode">CTF Mode</a> •
  <a href="#building-from-source">Build</a>
</p>

---

## What is PodSweeper?

**PodSweeper** is a cloud-native Minesweeper game where cells are **live Kubernetes Pods**. To "click" a cell, you don't use a mouse — you use `kubectl delete`.

```
$ kubectl get pods -n podsweeper-game
NAME       READY   STATUS    AGE
pod-0-0    1/1     Running   5s
pod-0-1    1/1     Running   5s
pod-1-0    1/1     Running   5s
hint-1-1   1/1     Running   3s    # <- You revealed this one!
pod-2-0    1/1     Running   5s
...

$ kubectl delete pod pod-0-0 -n podsweeper-game
pod "pod-0-0" deleted

$ kubectl get pods -n podsweeper-game
NAME       READY   STATUS    AGE
explosion  1/1     Running   1s    # <- 💥 BOOM!
```

---

## Project Status

| Component | Status |
|-----------|--------|
| Core Game Engine | ✅ Complete |
| Grid Generation | ✅ Complete |
| Hint System (BFS) | ✅ Complete |
| Victory/Defeat Detection | ✅ Complete |
| Player Terminal | ✅ Complete |
| Level 0-4 (Cheat Paths) | ✅ Complete |
| Levels 5-9 (Webhook) | 📋 Planned |
| CI/CD Pipeline | ✅ Complete |

---

## Disclaimer

> ⚠️ **PodSweeper is designed to be destructive within its own namespace.**
> 
> Do not run this in production unless you want to explain to your boss why you were playing Minesweeper with company infrastructure.

## Quick Start

### Prerequisites

- A Kubernetes cluster ([kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/), or any cluster)
- `kubectl` configured to access the cluster

### Installation

```bash
# Deploy directly from GitHub (no clone needed!)
kubectl apply -k https://github.com/zwindler/podsweeper//deploy/base?ref=v0.1.3

# Wait for the gamemaster to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=podsweeper \
  -n podsweeper-game --timeout=60s
```

Or if you prefer to clone:
```bash
git clone https://github.com/zwindler/podsweeper.git
cd podsweeper
kubectl apply -k deploy/base/
```

### Play the Game

```bash
# Start a new game
kubectl patch configmap podsweeper-config -n podsweeper-game \
  --type merge -p '{"data":{"level":"0","action":"start"}}'

# Join the player terminal
kubectl exec -it player -n podsweeper-game -- bash
```

Inside the player terminal:
```bash
podsweeper> pods              # See the grid
podsweeper> sweep 2 2         # Click cell at (2,2)
podsweeper> hint 2 2          # Get hint value
podsweeper> map               # Cheat! (Level 0 only)
```

Or play directly with kubectl:
```bash
kubectl get pods -n podsweeper-game                    # See the grid
kubectl delete pod pod-2-2 -n podsweeper-game          # Click a cell
kubectl port-forward hint-2-2 8080:8080 -n podsweeper-game &
curl localhost:8080                                    # Read hint value
kubectl logs explosion -n podsweeper-game              # See defeat message
kubectl logs victory -n podsweeper-game                # See victory message
```

### Grid Sizes

| Level | Grid | Mines | Difficulty |
|-------|------|-------|------------|
| 0-4 | 5×5 | 4 | Beginner |
| 5-7 | 10×10 | 15 | Intermediate |
| 8-9 | 20×20 | 60 | Expert |

---

## The Concept

PodSweeper transforms your Kubernetes namespace into a minefield:

```
   0   1   2   3   4
 +---+---+---+---+---+
0| . | . | X | X | . |  X = Mine (hidden)
 +---+---+---+---+---+  . = Safe cell
1| . | . | . | . | X |  
 +---+---+---+---+---+  Click a cell = kubectl delete pod
2| . | . | . | . | . |  
 +---+---+---+---+---+  Hit a mine = 💥 Game Over
3| . | . | . | . | . |  
 +---+---+---+---+---+  Clear all safe = 🎉 Victory!
4| . | X | . | . | . |  
 +---+---+---+---+---+
```

1. **The Grid:** The Gamemaster spawns a matrix of pods named `pod-x-y`
2. **Safe Cell:** Replaced by `hint-x-y` showing adjacent mine count
3. **Empty Area:** Chain reaction (BFS) auto-clears connected empty cells
4. **Mine:** All pods explode, game over!

---

## Learning through Hardening (CTF Mode)

PodSweeper isn't just a game — it's a **Kubernetes CTF**. Each level hardens the cluster to prevent "cheating":

| Level | Name |
|-------|------|
| 0 | The Intern |
| 1 | The Junior |
| 2 | The Infiltrator |
| 3 | The Heart |
| 4 | Amnesia |
| 5 | The Firewall |
| 6 | The Sand Grain |
| 7 | Port Hacking |
| 8 | Firing Window |
| 9 | RBAC Blackout |

---

## Technical Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.26 |
| Framework | controller-runtime / client-go |
| Container Runtime | Podman / Docker |
| Deployment | Kustomize |
| Base Images | Alpine 3.23, distroless |

---

## Building from Source

```bash
# Build binaries
make build

# Run tests
make test

# Build container images
make docker-build

# Load into kind cluster
make docker-push

# Deploy
make deploy

# Start playing
make start-game && make play
```

---

## Why PodSweeper?

Because `kubectl delete pod` should be **scary**, and we wanted to make it **fun**.

Perfect for:
- **K8s Beginners:** Learn kubectl and pod lifecycle
- **SREs/DevOps:** Practice troubleshooting under pressure  
- **Security Teams:** Understand RBAC and admission control
- **Conference Demos:** Impressive and interactive!

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

<p align="center">
  <i>Created with ❤️ and a few LLM driven coding sessions.</i>
</p>

<p align="center">
  <a href="https://github.com/zwindler/podsweeper/stargazers">⭐ Star us on GitHub!</a>
</p>
