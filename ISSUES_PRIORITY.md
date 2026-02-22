# PodSweeper Development Roadmap

This document outlines the recommended order for building PodSweeper incrementally.

---

## Summary

| Phase | Name | Status |
|-------|------|--------|
| 1 | Foundation (MVP) | **Complete** |
| 2 | Level Progression (0-4) | **Complete** |
| 3 | Security Hardening + Webhook (5-9) | Next Up |
| 4 | Pre-Release Polish | Not Started |
| 5 | Documentation & Release | Not Started |

---

## Phase 1: Foundation (MVP) - COMPLETE

**Goal:** Basic working game. Player can delete pods, see hints, win, or lose.

### Completed
- [x] Project scaffolding (Go module, directory structure, Makefile)
- [x] Structured logging with `logr`
- [x] Game state data model (`GameState` struct with JSON serialization)
- [x] State persistence to Kubernetes Secret
- [x] Grid generator with `pod-x-y` naming
- [x] Seeded random mine placement
- [x] Grid spawner with batch creation and retries
- [x] Controller with Pod watcher and reconciliation loop
- [x] Deletion event handler routing (mine/safe/empty)
- [x] Adjacent mine counter (8-cell neighborhood)
- [x] BFS propagation for empty cells
- [x] Hint micro-agent HTTP server
- [x] Hint pod spawner with environment injection
- [x] Victory condition checker
- [x] Defeat handler with explosion pod
- [x] Player ServiceAccount and RBAC
- [x] Namespace initialization
- [x] ConfigMap-based game control
- [x] Dockerfiles for gamemaster and hint-agent
- [x] Kustomize deployment manifests
- [x] Unit tests for controllers and game logic
- [x] Fast pod termination (1s grace period)
- [x] Race condition fixes (WaitForCleanup, save-before-cleanup)

---

## Phase 2: Level Progression (Levels 0-4) - COMPLETE

**Goal:** Implement CTF-style "cheat" paths for early levels. No webhook required.

### Level Infrastructure
- [x] Level state management
  - Track current level in game state (already have `Level` field)
  - Persist level progress across restarts
- [x] Level transition on victory
  - Increment level after winning
  - Apply level-specific resources on transition
- [x] Level selection via ConfigMap
  - Allow starting at specific level for testing

### Level 0: The Intern (Current Default)
- [x] Create `map` ConfigMap with mine positions as visual grid:
  ```
  . . X X .
  . . . . X
  . . . . .
  . . . . .
  . X . . .
  ```
- [x] Player can `kubectl get cm map -o yaml` to see mines
- [x] No RBAC restrictions

### Level 1: The Junior
- [x] Store map in `map` Secret (Base64 encoded, same visual format)
- [x] Restrict player RBAC: remove `get configmaps`
- [x] Player must decode Base64 to read map

### Level 2: The Infiltrator
- [x] Inject map data into game pod environment variables
- [x] Restrict player RBAC: remove `get secrets`
- [x] Player must `kubectl exec` into a pod to read env

### Level 3: The Heart of the Machine
- [x] Write map to file inside Gamemaster pod (`/tmp/map.txt`)
- [x] Remove env vars from game pods
- [x] Player must exec into Gamemaster to read file

### Level 4: Amnesia
- [x] Remove all static map leaks (no CM, no Secret readable, no env, no files)
- [x] Map only exists in encrypted game state Secret
- [x] Forces "legitimate" gameplay - no cheating possible

---

## Phase 3: Security Hardening + Webhook (Levels 5-9)

**Goal:** Advanced levels requiring admission webhook mechanics.

### Admission Webhook Setup
- [ ] Webhook server (HTTP server for admission reviews)
- [ ] TLS certificate management (self-signed or cert-manager)
- [ ] ValidatingWebhookConfiguration for DELETE operations
- [ ] Level-aware validation routing

### Level 5: The Firewall (NetworkPolicies)
- [ ] Gamemaster exposes debug endpoint `:9999/debug/map`
- [ ] NetworkPolicy blocks traffic to debug port
- [ ] Player must create proxy pod with whitelisted labels

### Level 6: The Sand Grain (Finalizers)
- [ ] Add `podsweeper.io/wait` finalizer to some pods
- [ ] Pods stuck in Terminating until finalizer removed
- [ ] Player must patch pods to remove finalizer before delete

### Level 7: Port-Hacking (Dynamic Ports)
- [ ] Hint pods serve on randomized port (1024-65535)
- [ ] Port stored in `podsweeper.io/hint-port` annotation
- [ ] Player must read annotation, then curl correct port

### Level 8: The Firing Window (Timing)
- [ ] Webhook only accepts deletions in first 100ms of each second
- [ ] Return detailed timing error message
- [ ] Player must write synchronized deletion script

### Level 9: RBAC Blackout
- [ ] Remove: `exec`, `describe`, `get -o yaml`
- [ ] Keep: `delete pods`, `get events`
- [ ] Hints leaked via Kubernetes Events only

---

## Phase 4: Pre-Release Polish

**Goal:** Production-ready quality for public release.

### CI/CD
- [x] GitHub Actions: Build and push gamemaster image
- [x] GitHub Actions: Build and push hint-agent image
- [x] GitHub Actions: Build and push player-terminal image
- [x] Semantic versioning tags
- [x] Multi-arch builds (amd64, arm64)

### Health & Observability
- [ ] Health endpoints (`/healthz`, `/readyz`)
- [ ] Kubernetes Event emission for game actions

### Testing
- [ ] Automated E2E test framework (kind + test harness)
- [ ] E2E tests for all levels
- [ ] Webhook unit tests

### Game Quality
- [ ] Unique ASCII art per level (victory/defeat)
- [ ] Flag generation system for CTF scoring
- [ ] Seed sharing for challenges
- [ ] Installation verification script

### Quality of Life / Polish
- [ ] Make critical errors FATAL instead of ERROR (RBAC failures, CM creation failures)
- [ ] Add more descriptive error messages for common issues

---

## Phase 5: Documentation & Release

**Goal:** Ready for v1.0 public release.

### Player Documentation
- [ ] Quickstart guide (installation, first game)
- [ ] kubectl cheat sheet for gameplay
- [ ] Level hints document (concepts to research, no solutions)

### Developer Documentation
- [ ] Contributing guide
- [ ] Architecture documentation

### Release Assets
- [ ] Demo GIF/video
- [ ] Helm chart (optional, Kustomize already works)

---

## What to Work On Next

**Immediate priority: Phase 3 - Security Hardening + Webhook**

Start with:
1. **Admission Webhook Setup** - HTTP server for admission reviews, TLS certificates
2. **ValidatingWebhookConfiguration** - Hook DELETE operations on game pods
3. **Level 5-9 implementations** - Advanced security challenges

---

## Recent Fixes

### v0.1.3 (2026-02-22)
- Updated golangci-lint-action from v6 to v7 (required for golangci-lint v2.x)
- Fixed all golangci-lint issues (errcheck, staticcheck, unused)
- Added pre-commit hook for lint enforcement (`make install-hooks`)

### v0.1.2 (2026-02-22)
- Fixed RBAC escalation: gamemaster now has all permissions it grants to players
- Fixed cache timing: `EnsureConfigMap` now runs after manager starts
- Added kubectl autocompletion to player terminal
- Added `bash-completion` package to player image

### v0.1.1 (2026-02-22)
- Updated GitHub Actions to latest versions (checkout v6, setup-go v6, codecov v5)
- Fixed golangci-lint compatibility with Go 1.26 (using v2.10.1)

### v0.1.0 (2026-02-22)
- First public release
- CI/CD pipeline with GitHub Actions
- Multi-arch container images (amd64, arm64)
- Levels 0-4 complete

### Earlier
- Fixed race condition: Save game state BEFORE cleanup
- Added `WaitForCleanup()` to wait for all pods to be deleted before spawning
- Set `terminationGracePeriodSeconds=1` on all game pods (~2s vs ~30s termination)

---

*PodSweeper - The most impractical way to play Minesweeper*
