# 🚨 SPOILER ALERT: PodSweeper Full Functional Specification 🚨
This document contains the complete architectural and logic breakdown of PodSweeper. 
If you are a player, reading this will reveal all the internal mechanics and 100% of the solutions for the hardening levels.

---

# 🛸 PodSweeper: Functional Specification

**PodSweeper** is a cloud-native Minesweeper implementation where Kubernetes Pods are the tiles. It is designed as a technical challenge to teach Kubernetes internals, security hardening, and automation through a progressively difficult "game" environment.

## 1. Core Mechanics

### 1.1 The Grid
- **Namespace:** `podsweeper-game`
- **Initial State:** A grid of $N \times N$ pods named `pod-x-y` (e.g., `pod-3-5`).
- **The "Click":** A player "clicks" by executing `kubectl delete pod pod-x-y`.

### 1.2 Deletion Logic (The State Machine)
When a pod `pod-x-y` is targeted for deletion, the **Gamemaster (Admission Webhook + Controller)** handles the event:

1. **Mined Pod:** - The deletion is allowed. 
   - The controller detects the deletion of a mine, wipes the namespace, and spawns an `explosion` pod with heavy ASCII art in its logs.
2. **Safe Pod with Adjacent Mines (>0):** - The deletion is allowed.
   - The controller immediately creates a new pod named **`hint-x-y`**.
   - This pod runs a micro-agent exposing the hint via HTTP.
3. **Safe Pod with No Adjacent Mines (=0):** - The deletion is allowed.
   - The controller triggers a **Breadth-First Search (BFS) propagation**.
   - All adjacent "0-hint" pods are deleted and not recreated (creating a hole in the grid).
   - Any "numbered-hint" pods on the boundary of the empty zone are created as `hint-x-y`.

## 2. Technical Architecture

### 2.1 The Gamemaster (Go Controller & Webhook)
A Go binary running in the cluster.
- **Validating Webhook:** Intercepts `DELETE` requests. Validates level-specific constraints and returns explicit error messages to the CLI (e.g., Timing errors or missing Finalizers).
- **Watcher:** Monitors successful deletions to trigger game logic (hints, propagation, game over).
- **Orchestrator:** Manages levels by dynamically applying NetworkPolicies, RBAC Roles, and Webhook configurations.

### 2.2 State Management (The Secret)
- **Storage:** A Kubernetes Secret `podsweeper-state` stores the game state in JSON.
- **Content:** The mine map (bool matrix), the seed, the current level, and the list of revealed coordinates.

## 3. The Hardening Levels (CTF Path)

| Level | Name | Info Leak | Security Obstacle |
| :--- | :--- | :--- | :--- |
| **0** | **The Intern** | `ConfigMap` named `map`. | None. |
| **1** | **The Junior** | `Secret` (Base64). | RBAC: No access to ConfigMaps. |
| **2** | **The Infiltrator** | Pod Env Vars. | RBAC: No access to Secrets. |
| **3** | **The Heart** | Controller Filesystem. | Pods have no Env Vars. |
| **4** | **Amnesia** | None (In-memory). | Controller filesystem is hardened. |
| **5** | **The Firewall** | HTTP Admin Port. | **NetworkPolicies** block unauthorized pods. |
| **6** | **The Sand Grain**| Hint via HTTP. | **Finalizers**: Manual patch required. |
| **7** | **Port-Hacking** | Hint via HTTP. | **Dynamic Ports**: Port randomized per pod. |
| **8** | **The Window** | Hint via HTTP. | **Admission Webhook**: Precise timing (0ms-100ms). |
| **9** | **Blackout** | **K8s Events**. | **Minimalist RBAC**: No Exec, No Describe. |

## 4. Operational Rules & Victory

### 4.1 Timing (Level 8)
The **Validating Webhook** captures the request arrival time. If it falls outside the 100ms window, it denies the deletion with an explicit message:  
`Forbidden: Timing error. Request arrived at 450ms. Target window is [0ms - 100ms].`

### 4.2 Victory Condition
Victory is declared when the only pods remaining in the namespace with the name pattern `pod-x-y` are exactly the mines. All other safe pods must have been either deleted (hint 0) or replaced by `hint-x-y` pods.

### 4.3 Rewards (Flags & ASCII)
Upon victory, a `victory` pod is spawned:
- **ASCII Art:** The logs will display a unique ASCII art trophy for each level.
- **The Flag:** A static but obfuscated (Base64/XOR) flag is delivered (e.g., in logs, env, or secret depending on the level). 
- **Validation:** Anyone can decode the flag from the source code, but the obfuscation prevents accidental spoilers while browsing the repo.
