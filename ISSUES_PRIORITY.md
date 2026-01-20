# 🎯 PodSweeper Issues Priority & Roadmap

This document outlines the recommended order for tackling issues to build PodSweeper incrementally. 
Each phase builds on the previous one, ensuring dependencies are resolved before moving forward.

---

## 📊 Summary

| Phase | Name | Issues | Milestone |
|-------|------|--------|-----------|
| 1 | Foundation (MVP) | 20 | Level 0 Playable |
| 2 | Deployment & Basic Ops | 9 | Public Alpha |
| 3 | Admission Webhook | 6 | Webhook Ready |
| 4 | Level Progression | 8 | Levels 0-4 Complete |
| 5 | Security Hardening | 14 | All 10 Levels Complete |
| 6 | Polish & Victory | 9 | Full Game Experience |
| 7 | Documentation & Release | 6 | v1.0 Release |
| **Total** | | **72** | |

---

## Phase 1: Foundation (MVP - Playable Game)
**Goal:** Get a basic working game without levels or hardening

**Milestone:** `Level 0 Playable`

- [ ] [Core Game Engine] Project scaffolding [#1](https://github.com/zwindler/PodSweeper/issues/1)
- [ ] [Core Game Engine] Game state data model [#3](https://github.com/zwindler/PodSweeper/issues/3)
- [ ] [Core Game Engine] State persistence layer [#4](https://github.com/zwindler/PodSweeper/issues/4)
- [ ] [Grid Management] Grid generator [#16](https://github.com/zwindler/PodSweeper/issues/16)
- [ ] [Grid Management] Mine placement algorithm [#17](https://github.com/zwindler/PodSweeper/issues/17)
- [ ] [Grid Management] Grid spawner [#18](https://github.com/zwindler/PodSweeper/issues/18)
- [ ] [Core Game Engine] Controller bootstrap and reconciliation [#2](https://github.com/zwindler/PodSweeper/issues/2)
- [ ] [Core Game Engine] Deletion event handler [#8](https://github.com/zwindler/PodSweeper/issues/8)
- [ ] [Core Game Engine] Implement mine detection logic [#5](https://github.com/zwindler/PodSweeper/issues/5)
- [ ] [Core Game Engine] Adjacent mine counter [#6](https://github.com/zwindler/PodSweeper/issues/6)
- [ ] [Core Game Engine] BFS propagation algorithm [#7](https://github.com/zwindler/PodSweeper/issues/7)
- [ ] [Hint Micro-Agent] HTTP server implementation [#40](https://github.com/zwindler/PodSweeper/issues/40)
- [ ] [Hint Micro-Agent] Container image build [#43](https://github.com/zwindler/PodSweeper/issues/43)
- [ ] [Grid Management] Hint pod template [#19](https://github.com/zwindler/PodSweeper/issues/19)
- [ ] [Grid Management] Hint pod spawner [#20](https://github.com/zwindler/PodSweeper/issues/20)
- [ ] [Victory/Defeat] Victory condition checker [#51](https://github.com/zwindler/PodSweeper/issues/51)
- [ ] [Victory/Defeat] Defeat detection [#55](https://github.com/zwindler/PodSweeper/issues/55)
- [ ] [Grid Management] Namespace wipe function [#22](https://github.com/zwindler/PodSweeper/issues/22)
- [ ] [Game Init] Namespace initializer [#45](https://github.com/zwindler/PodSweeper/issues/45)
- [ ] [Game Init] Game start command [#46](https://github.com/zwindler/PodSweeper/issues/46)

---

## Phase 2: Deployment & Basic Ops
**Goal:** Make it installable by others

**Milestone:** `Public Alpha`

- [ ] [Deployment] Gamemaster Dockerfile [#58](https://github.com/zwindler/PodSweeper/issues/58)
- [ ] [Deployment] Gamemaster image CI [#59](https://github.com/zwindler/PodSweeper/issues/59)
- [ ] [Hint Micro-Agent] Image CI pipeline [#44](https://github.com/zwindler/PodSweeper/issues/44)
- [ ] [Deployment] Helm chart skeleton [#60](https://github.com/zwindler/PodSweeper/issues/60)
- [ ] [Deployment] Helm: Gamemaster Deployment template [#61](https://github.com/zwindler/PodSweeper/issues/61)
- [ ] [Deployment] Helm: RBAC resources [#62](https://github.com/zwindler/PodSweeper/issues/62)
- [ ] [Game Init] Player ServiceAccount setup [#50](https://github.com/zwindler/PodSweeper/issues/50)
- [ ] [Game Init] Game reset function [#48](https://github.com/zwindler/PodSweeper/issues/48)
- [ ] [Documentation] Player quickstart guide [#67](https://github.com/zwindler/PodSweeper/issues/67)

---

## Phase 3: Admission Webhook
**Goal:** Enable advanced level mechanics

**Milestone:** `Webhook Ready`

- [ ] [Admission Webhook] Webhook server setup [#9](https://github.com/zwindler/PodSweeper/issues/9)
- [ ] [Admission Webhook] TLS certificate management [#10](https://github.com/zwindler/PodSweeper/issues/10)
- [ ] [Admission Webhook] ValidatingWebhookConfiguration manifest [#11](https://github.com/zwindler/PodSweeper/issues/11)
- [ ] [Admission Webhook] Base validation logic [#12](https://github.com/zwindler/PodSweeper/issues/12)
- [ ] [Deployment] Helm: Webhook resources [#63](https://github.com/zwindler/PodSweeper/issues/63)
- [ ] [Deployment] Helm: Certificate management [#64](https://github.com/zwindler/PodSweeper/issues/64)

---

## Phase 4: Level Progression System
**Goal:** Implement the CTF path (Levels 0-4)

**Milestone:** `Levels 0-4 Complete`

- [ ] [Level Progression] Level state management [#24](https://github.com/zwindler/PodSweeper/issues/24)
- [ ] [Level Progression] Level 0 setup - ConfigMap cheat [#25](https://github.com/zwindler/PodSweeper/issues/25)
- [ ] [Level Progression] Level 1 setup - Secret cheat [#26](https://github.com/zwindler/PodSweeper/issues/26)
- [ ] [Level Progression] Level 2 setup - Environment variable cheat [#27](https://github.com/zwindler/PodSweeper/issues/27)
- [ ] [Level Progression] Level 3 setup - Gamemaster filesystem cheat [#28](https://github.com/zwindler/PodSweeper/issues/28)
- [ ] [Level Progression] Level 4+ cleanup - Remove static leaks [#29](https://github.com/zwindler/PodSweeper/issues/29)
- [ ] [Level Progression] Level transition orchestrator [#30](https://github.com/zwindler/PodSweeper/issues/30)
- [ ] [Game Init] Level skip/select mechanism [#49](https://github.com/zwindler/PodSweeper/issues/49)

---

## Phase 5: Security Hardening (Levels 5-9)
**Goal:** Implement the hard levels

**Milestone:** `All 10 Levels Complete`

### RBAC Foundation
- [ ] [Security Hardening] RBAC template system [#31](https://github.com/zwindler/PodSweeper/issues/31)
- [ ] [Security Hardening] Level 1 RBAC - Restrict ConfigMap access [#32](https://github.com/zwindler/PodSweeper/issues/32)
- [ ] [Security Hardening] Level 2 RBAC - Restrict Secret access [#33](https://github.com/zwindler/PodSweeper/issues/33)

### Level 5: The Firewall
- [ ] [Security Hardening] Level 5 NetworkPolicy - Block debug endpoint [#34](https://github.com/zwindler/PodSweeper/issues/34)

### Level 6: The Sand Grain
- [ ] [Security Hardening] Level 6 Finalizer injection [#35](https://github.com/zwindler/PodSweeper/issues/35)
- [ ] [Admission Webhook] Level-aware validation [#13](https://github.com/zwindler/PodSweeper/issues/13)
- [ ] [Admission Webhook] Finalizer validation (Level 6) [#15](https://github.com/zwindler/PodSweeper/issues/15)

### Level 7: Port-Hacking
- [ ] [Hint Micro-Agent] Environment-based configuration [#42](https://github.com/zwindler/PodSweeper/issues/42)
- [ ] [Hint Micro-Agent] Hint endpoint implementation [#41](https://github.com/zwindler/PodSweeper/issues/41)
- [ ] [Security Hardening] Level 7 Dynamic port system [#36](https://github.com/zwindler/PodSweeper/issues/36)

### Level 8: The Firing Window
- [ ] [Admission Webhook] Timing validation (Level 8) [#14](https://github.com/zwindler/PodSweeper/issues/14)
- [ ] [Security Hardening] Level 8 Timing enforcement [#37](https://github.com/zwindler/PodSweeper/issues/37)

### Level 9: RBAC Blackout
- [ ] [Security Hardening] Level 9 Minimal RBAC [#38](https://github.com/zwindler/PodSweeper/issues/38)

### Orchestration
- [ ] [Security Hardening] Security resource applicator [#39](https://github.com/zwindler/PodSweeper/issues/39)

---

## Phase 6: Polish & Victory
**Goal:** Complete the game experience

**Milestone:** `Full Game Experience`

### Victory Experience
- [ ] [Victory/Defeat] Victory pod spawner [#52](https://github.com/zwindler/PodSweeper/issues/52)
- [ ] [Victory/Defeat] ASCII art assets - Victory [#53](https://github.com/zwindler/PodSweeper/issues/53)
- [ ] [Victory/Defeat] Flag generation system [#54](https://github.com/zwindler/PodSweeper/issues/54)

### Defeat Experience
- [ ] [Victory/Defeat] Explosion sequence [#56](https://github.com/zwindler/PodSweeper/issues/56)
- [ ] [Victory/Defeat] ASCII art assets - Explosion [#57](https://github.com/zwindler/PodSweeper/issues/57)
- [ ] [Grid Management] Explosion pod spawner [#23](https://github.com/zwindler/PodSweeper/issues/23)

### Game Quality
- [ ] [Game Init] Seed-based reproducibility [#47](https://github.com/zwindler/PodSweeper/issues/47)
- [ ] [Grid Management] Chain deletion handler [#21](https://github.com/zwindler/PodSweeper/issues/21)
- [ ] [Deployment] Installation verification [#66](https://github.com/zwindler/PodSweeper/issues/66)

---

## Phase 7: Documentation & Release
**Goal:** Ready for public release

**Milestone:** `v1.0 Release`

- [ ] [Documentation] kubectl cheat sheet [#68](https://github.com/zwindler/PodSweeper/issues/68)
- [ ] [Documentation] Level hints document [#69](https://github.com/zwindler/PodSweeper/issues/69)
- [ ] [Documentation] Contributing guide [#70](https://github.com/zwindler/PodSweeper/issues/70)
- [ ] [Documentation] Architecture documentation [#71](https://github.com/zwindler/PodSweeper/issues/71)
- [ ] [Documentation] Demo GIF/Video [#72](https://github.com/zwindler/PodSweeper/issues/72)
- [ ] [Deployment] Kustomize alternative [#65](https://github.com/zwindler/PodSweeper/issues/65)

---

## 🚀 Quick Start: First 5 Issues

If you want to start coding immediately, tackle these first:

1. **[Core Game Engine] Project scaffolding** [#1](https://github.com/zwindler/PodSweeper/issues/1)
   - Initialize Go module with `controller-runtime` and `client-go`
   - Set up directory structure: `cmd/`, `pkg/`, `internal/`
   - Create Makefile with `build`, `test`, `run` targets

2. **[Core Game Engine] Game state data model** [#3](https://github.com/zwindler/PodSweeper/issues/3)
   - Define `GameState` struct with mine map, revealed cells, level, seed
   - Implement JSON marshalling/unmarshalling
   - Write unit tests for serialization

3. **[Grid Management] Grid generator** [#16](https://github.com/zwindler/PodSweeper/issues/16)
   - Create function to generate N×N pod specs
   - Implement `pod-x-y` naming convention
   - Make grid size configurable

4. **[Grid Management] Mine placement algorithm** [#17](https://github.com/zwindler/PodSweeper/issues/17)
   - Implement seeded random mine placement
   - Make mine density configurable
   - Ensure reproducibility with same seed

5. **[Hint Micro-Agent] HTTP server implementation** [#40](https://github.com/zwindler/PodSweeper/issues/40)
   - Create minimal Go HTTP server (~50 lines)
   - Serve hint value on `/` endpoint
   - Read configuration from environment variables

---

## 📝 Notes

### Dependencies Between Issues
- All **Core Game Engine** issues depend on **Project scaffolding** [#1](https://github.com/zwindler/PodSweeper/issues/1)
- All **Grid Management** issues depend on **Game state data model** [#3](https://github.com/zwindler/PodSweeper/issues/3)
- All **Security Hardening** issues depend on **Level state management** [#24](https://github.com/zwindler/PodSweeper/issues/24)
- **Admission Webhook** issues are needed for Levels 6, 8
- **Helm** issues depend on **Dockerfiles** being complete

### Labels Used
- `enhancement` - Feature implementation
- `documentation` - Docs and guides

### Suggested Milestones
1. `Level 0 Playable` - Basic working game
2. `Public Alpha` - Installable by others
3. `Webhook Ready` - Advanced mechanics enabled
4. `Levels 0-4 Complete` - Half the CTF done
5. `All 10 Levels Complete` - Full CTF experience
6. `Full Game Experience` - Polished gameplay
7. `v1.0 Release` - Production ready

---

## 🔄 Progress Tracking

Update this section as you complete issues:

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Foundation | 🔴 Not Started | 0/20 |
| Phase 2: Deployment | 🔴 Not Started | 0/9 |
| Phase 3: Webhook | 🔴 Not Started | 0/6 |
| Phase 4: Levels | 🔴 Not Started | 0/8 |
| Phase 5: Security | 🔴 Not Started | 0/14 |
| Phase 6: Polish | 🔴 Not Started | 0/9 |
| Phase 7: Docs | 🔴 Not Started | 0/6 |
| **Total** | | **0/72** |

Legend: 🔴 Not Started | 🟡 In Progress | 🟢 Complete

---

*Generated for PodSweeper - The most impractical way to play Minesweeper* 🛸