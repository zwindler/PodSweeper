# PodSweeper Development Roadmap

This document outlines the recommended order for building PodSweeper incrementally.
Each phase builds on the previous one, ensuring dependencies are resolved before moving forward.

---

## Summary

| Phase | Name | Tasks | Milestone |
|-------|------|-------|-----------|
| 1 | Foundation (MVP) | 18 | Level 0 Playable |
| 2 | Deployment & Ops | 8 | Public Alpha |
| 3 | Testing | 5 | Quality Gates |
| 4 | Level Progression (0-4) | 12 | Levels 0-4 Complete |
| 5 | Security Hardening + Webhook (5-9) | 16 | All 10 Levels Complete |
| 6 | Polish & Victory | 7 | Full Game Experience |
| 7 | Documentation & Release | 7 | v1.0 Release |
| **Total** | | **73** | |

---

## Phase 1: Foundation (MVP - Level 0 Playable)

**Goal:** Get a basic working game without levels or hardening. Player can delete pods, see hints, win, or lose.

**Milestone:** `Level 0 Playable`

### Core Infrastructure
- [ ] Project scaffolding
  - Initialize Go module with `controller-runtime` and `client-go`
  - Set up directory structure: `cmd/`, `pkg/`, `internal/`
  - Create Makefile with `build`, `test`, `run` targets
- [ ] Structured logging setup
  - Configure structured logging (e.g., `zap` or `logr`)
  - Define log levels and formats for controller/webhook
- [ ] Error handling and user feedback
  - Define error types for game events (invalid move, game over, etc.)
  - Ensure errors surface meaningful messages to `kubectl` users

### Game State
- [ ] Game state data model
  - Define `GameState` struct: mine map, revealed cells, level, seed
  - Implement JSON marshalling/unmarshalling
  - Write unit tests for serialization
- [ ] State persistence layer
  - Implement read/write to Kubernetes Secret `podsweeper-state`
  - Handle concurrent access and conflicts

### Grid Management
- [ ] Grid generator
  - Create function to generate N×N pod specs
  - Implement `pod-x-y` naming convention
  - Make grid size configurable
- [ ] Mine placement algorithm
  - Implement seeded random mine placement
  - Make mine density configurable
  - Ensure reproducibility with same seed
- [ ] Grid spawner
  - Create pods in `podsweeper-game` namespace
  - Handle partial failures and retries

### Controller
- [ ] Controller bootstrap and reconciliation
  - Set up controller-runtime manager
  - Implement Pod watcher for `podsweeper-game` namespace
  - Basic reconciliation loop
- [ ] Deletion event handler
  - Detect pod deletion events
  - Route to appropriate handler (mine/safe/empty)
- [ ] Mine detection logic
  - Check if deleted pod coordinates match mine map
  - Trigger appropriate game flow
- [ ] Adjacent mine counter
  - Calculate number of mines in 8-cell neighborhood
  - Handle edge/corner cases correctly
- [ ] BFS propagation with chain deletion
  - Implement Breadth-First Search for empty cell propagation
  - Automatically delete connected empty pods
  - Stop at numbered hint boundaries
  - Create hint pods at propagation edges

### Hint Micro-Agent
- [ ] HTTP server implementation
  - Create minimal Go HTTP server (~50 lines)
  - Serve hint value on `/` endpoint
  - Configurable port binding
- [ ] Environment-based configuration
  - Read hint value from `HINT_VALUE` env var
  - Read coordinates from `POD_X`, `POD_Y` env vars
  - Support dynamic port via `PORT` env var (for Level 7)
- [ ] Container image build
  - Create minimal Dockerfile (scratch/distroless base)
  - Multi-stage build for small image size
- [ ] Hint pod template
  - Define pod spec with hint container
  - Configure environment variables injection
  - Set resource limits
- [ ] Hint pod spawner
  - Create `hint-x-y` pods after safe cell deletion
  - Inject correct hint value based on adjacent mine count

### Victory & Defeat
- [ ] Victory condition checker
  - Detect when all non-mine pods are cleared
  - Compare remaining pods against mine map
- [ ] Defeat sequence handler
  - Detect mine pod deletion
  - Wipe namespace (delete all game pods)
  - Spawn explosion pod with ASCII art in logs

### Game Initialization
- [ ] Player ServiceAccount setup
  - Create ServiceAccount for player
  - Define base RBAC (delete pods, get pods)
  - Generate kubeconfig for player
- [ ] Namespace initializer
  - Create `podsweeper-game` namespace if not exists
  - Apply required labels and annotations
- [ ] Game start command
  - CLI or kubectl plugin to start new game
  - Accept seed and difficulty parameters

---

## Phase 2: Deployment & Basic Ops

**Goal:** Make the game installable by others

**Milestone:** `Public Alpha`

### Container Images
- [ ] Gamemaster Dockerfile
  - Multi-stage build for Go binary
  - Non-root user, minimal base image
- [ ] Gamemaster image CI
  - GitHub Actions workflow for building/pushing
  - Semantic versioning tags
- [ ] Hint Micro-Agent image CI
  - Separate workflow for hint agent image
  - Keep image minimal (<10MB)

### Gamemaster Operations
- [ ] Gamemaster health endpoints
  - `/healthz` for liveness probe
  - `/readyz` for readiness probe
  - Proper startup probe configuration

### Helm Chart
- [ ] Helm chart skeleton
  - Chart.yaml, values.yaml structure
  - Namespace creation option
- [ ] Helm: Gamemaster Deployment template
  - Deployment with configurable replicas
  - Resource requests/limits
  - Environment configuration
- [ ] Helm: RBAC resources
  - ServiceAccount for Gamemaster
  - ClusterRole/Role for watching pods, managing secrets

### Game Operations
- [ ] Game reset function
  - Wipe current game and start fresh
  - Preserve or reset level progress (configurable)

---

## Phase 3: Testing

**Goal:** Establish quality gates before building complex level mechanics

**Milestone:** `Quality Gates`

### Unit Testing
- [ ] Unit test framework setup
  - Configure test harness with mocks for Kubernetes client
  - Set up test fixtures for game state
- [ ] Controller unit tests
  - Test mine detection logic
  - Test BFS propagation algorithm
  - Test victory/defeat conditions
  - Test adjacent mine counting (edge cases)

### Integration/E2E Testing
- [ ] E2E test framework
  - Set up kind/k3d for local cluster testing
  - Create test harness for deploying game
  - Implement test utilities (wait for pods, simulate clicks)
- [ ] E2E tests for Level 0
  - Test: Click safe cell → hint pod appears
  - Test: Click empty cell → BFS propagation works
  - Test: Click mine → game over sequence
  - Test: Clear all safe cells → victory

### Webhook Testing (for later phases)
- [ ] Webhook unit tests
  - Test validation logic (prepared for Phase 5)
  - Test timing window validation
  - Test finalizer validation

---

## Phase 4: Level Progression (Levels 0-4)

**Goal:** Implement the CTF path for early levels (no webhook required)

**Milestone:** `Levels 0-4 Complete`

### Level Infrastructure
- [ ] Level state management
  - Track current level in game state
  - Persist level progress
  - Handle level-specific configuration
- [ ] Level transition orchestrator
  - Define interface for level setup/teardown
  - Trigger transitions on victory
  - Apply level-specific resources (RBAC, ConfigMaps, etc.)
- [ ] Kubernetes Event emission system
  - Emit events for game actions (cell revealed, hint shown)
  - Used for Level 9 mechanic and general debugging
  - Attach events to relevant pods

### RBAC System
- [ ] RBAC template system
  - Define templated Roles/RoleBindings per level
  - Dynamic application based on current level
  - Clean removal on level transition

### Level Implementations

#### Level 0: The Intern
- [ ] Level 0 setup - ConfigMap cheat
  - Create `ConfigMap` named `map` with mine positions
  - Player can `kubectl get cm map -o yaml` to cheat
  - No restrictions

#### Level 1: The Junior  
- [ ] Level 1 RBAC - Restrict ConfigMap access
  - Remove `get configmaps` from player Role
- [ ] Level 1 setup - Secret cheat
  - Store map in `Secret` (Base64 encoded)
  - Player must `kubectl get secret` and decode

#### Level 2: The Infiltrator
- [ ] Level 2 RBAC - Restrict Secret access
  - Remove `get secrets` from player Role
- [ ] Level 2 setup - Environment variable cheat
  - Inject map data into pod environment variables
  - Player must `kubectl exec` to read env

#### Level 3: The Heart of the Machine
- [ ] Level 3 setup - Gamemaster filesystem cheat
  - Write map to file in Gamemaster pod
  - Player must exec into Gamemaster to read
  - No env vars in game pods

#### Level 4: Amnesia
- [ ] Level 4+ cleanup - Remove static leaks
  - Map only in memory (GameState Secret, but encrypted/obfuscated)
  - No ConfigMaps, no readable Secrets, no env vars, no files
  - Forces "legitimate" gameplay

### Level Selection
- [ ] Level skip/select mechanism
  - Allow starting at specific level (for testing/speedruns)
  - Require flag from previous level to unlock (optional)

---

## Phase 5: Security Hardening + Webhook (Levels 5-9)

**Goal:** Implement advanced levels requiring admission webhook

**Milestone:** `All 10 Levels Complete`

### Admission Webhook Setup
- [ ] Webhook server setup
  - HTTP server for admission reviews
  - Integration with controller-runtime
- [ ] TLS certificate management
  - Self-signed cert generation
  - Certificate rotation strategy
  - Mount certs into webhook pod
- [ ] ValidatingWebhookConfiguration manifest
  - Target DELETE operations on pods
  - Namespace selector for `podsweeper-game`
  - Failure policy configuration
- [ ] Base validation logic
  - Parse AdmissionReview requests
  - Return Allow/Deny responses
  - Include meaningful denial messages
- [ ] Level-aware validation
  - Check current level from game state
  - Route to appropriate validation rules
- [ ] Helm: Webhook resources
  - ValidatingWebhookConfiguration template
  - Service for webhook endpoint
- [ ] Helm: Certificate management
  - cert-manager integration OR
  - Self-signed cert Job

### Level 5: The Firewall (NetworkPolicies)
- [ ] Level 5 NetworkPolicy - Block debug endpoint
  - Gamemaster exposes `:9999/debug/map`
  - NetworkPolicy blocks all traffic to this port
  - Player must create "proxy pod" with whitelisted labels

### Level 6: The Sand Grain (Finalizers)
- [ ] Level 6 Finalizer injection
  - Add `podsweeper.io/wait` finalizer to 10-100% of pods
  - Pods stuck in `Terminating` state until finalizer removed
- [ ] Finalizer validation
  - Webhook validates finalizer was properly removed
  - Deny deletion if finalizer still present (for game logic)

### Level 7: Port-Hacking (Dynamic Ports)
- [ ] Hint endpoint implementation
  - Hint pods serve on randomized port (1024-65535)
  - Port stored in pod annotation
- [ ] Level 7 Dynamic port system
  - Generate random port per hint pod
  - Store in `podsweeper.io/hint-port` annotation
  - Player must read annotation, then curl correct port

### Level 8: The Firing Window (Timing)
- [ ] Timing validation
  - Webhook captures request timestamp
  - Only accept deletions in first 100ms of each second
- [ ] Level 8 Timing enforcement
  - Return detailed error: "Request at 450ms. Target: [0-100ms]"
  - Force player to write synchronized deletion script

### Level 9: RBAC Blackout
- [ ] Level 9 Minimal RBAC
  - Remove: `exec`, `describe`, `get -o yaml`
  - Keep: `delete pods`, `get events`
  - Hints leaked via Kubernetes Events only

### Orchestration
- [ ] Security resource applicator
  - Apply/remove NetworkPolicies per level
  - Apply/remove webhook configurations per level
  - Coordinate RBAC changes

---

## Phase 6: Polish & Victory

**Goal:** Complete the game experience with rewards and visual feedback

**Milestone:** `Full Game Experience`

### Victory Experience
- [ ] Victory pod spawner
  - Spawn `victory` pod on game completion
  - Pod stays running for player to inspect
- [ ] ASCII art assets - Victory
  - Unique trophy art per level (10 designs)
  - Store as embedded strings or ConfigMap
- [ ] Flag generation system
  - Generate obfuscated flag per level
  - XOR or Base64 encoding (spoiler prevention)
  - Deliver via logs, env, or secret (level-dependent)

### Defeat Experience
- [ ] Explosion sequence
  - Dramatic namespace wipe animation (staggered deletes)
  - Final explosion pod with logs
- [ ] ASCII art assets - Explosion
  - Nuclear explosion / mushroom cloud art
  - "GAME OVER" messaging

### Game Quality
- [ ] Seed-based reproducibility
  - Same seed = same mine placement
  - Allow sharing seeds for challenges
- [ ] Installation verification
  - `helm test` or verify script
  - Check all components running
  - Validate webhook connectivity

---

## Phase 7: Documentation & Release

**Goal:** Ready for public release

**Milestone:** `v1.0 Release`

### Player Documentation
- [ ] Player quickstart guide
  - Installation steps (Helm)
  - First game walkthrough
  - Basic kubectl commands
- [ ] kubectl cheat sheet
  - Common commands for gameplay
  - How to read hints, check status
- [ ] Level hints document
  - Subtle hints for each level (no solutions)
  - Concepts to research per level

### Developer Documentation
- [ ] Contributing guide
  - Development setup
  - PR process
  - Code style guidelines
- [ ] Architecture documentation
  - System design overview
  - Component interactions
  - State machine diagrams

### Release Assets
- [ ] Demo GIF/Video
  - Recording of Level 0 gameplay
  - Show click → hint → victory flow
- [ ] Kustomize alternative
  - For users who prefer Kustomize over Helm
  - Base + overlays structure

---

## Quick Start: First 5 Tasks

If you want to start coding immediately, tackle these first:

1. **Project scaffolding**
   - `go mod init github.com/zwindler/podsweeper`
   - Set up `cmd/gamemaster/main.go`
   - Create Makefile with basic targets

2. **Game state data model**
   - Define `GameState` struct in `pkg/game/state.go`
   - Include: `MineMap [][]bool`, `Revealed [][]bool`, `Level int`, `Seed int64`
   - Add JSON tags and serialization tests

3. **Grid generator**
   - Create `pkg/grid/generator.go`
   - Function: `GenerateGrid(size int, seed int64, density float64) *GameState`
   - Unit tests for edge cases

4. **Mine placement algorithm**
   - Implement in grid generator
   - Use `math/rand` with seed for reproducibility
   - Ensure mines don't exceed density percentage

5. **HTTP server implementation (Hint Agent)**
   - Create `cmd/hint-agent/main.go`
   - Minimal server: read `HINT_VALUE` env, serve on `/`
   - Target: <100 lines of code

---

## Dependency Graph

```
Phase 1: Foundation
    └── All other phases depend on this

Phase 2: Deployment
    ├── Depends on: Phase 1 (need something to deploy)
    └── Enables: External testing, CI/CD

Phase 3: Testing
    ├── Depends on: Phase 2 (need deployable artifacts)
    └── Enables: Confident iteration on Phases 4-6

Phase 4: Levels 0-4
    ├── Depends on: Phase 1 (core game), Phase 3 (tests)
    └── No webhook required

Phase 5: Levels 5-9 + Webhook
    ├── Depends on: Phase 4 (level infrastructure)
    └── Introduces admission webhook

Phase 6: Polish
    ├── Depends on: Phase 5 (all levels working)
    └── Can be parallelized with Phase 5

Phase 7: Documentation
    ├── Depends on: Phase 6 (complete game)
    └── Can start drafts earlier
```

---

## Progress Tracking

Update this section as you complete tasks:

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Foundation | Not Started | 0/18 |
| Phase 2: Deployment | Not Started | 0/8 |
| Phase 3: Testing | Not Started | 0/5 |
| Phase 4: Levels 0-4 | Not Started | 0/12 |
| Phase 5: Levels 5-9 + Webhook | Not Started | 0/16 |
| Phase 6: Polish | Not Started | 0/7 |
| Phase 7: Documentation | Not Started | 0/7 |
| **Total** | | **0/73** |

---

*PodSweeper - The most impractical way to play Minesweeper*
