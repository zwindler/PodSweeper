# 🚨 SPOILER ALERT: PodSweeper Leveling & Mechanics 🚨
This document contains the full progression logic for PodSweeper. **Reading this will reveal all the "cheat codes" and technical secrets of the game**. If you want to play the game as intended, close this file now!

---

# 🛸 PodSweeper: Gameplay

**PodSweeper** is a cloud-native Minesweeper clone where cells are represented by Kubernetes Pods. The player must clear the namespace by deleting non-mined pods. As the player progresses through levels, the cluster becomes increasingly "hostile," forcing the player to master deeper Kubernetes concepts.

## 🕹️ Core Gameplay

* **The Grid:** A 10x10 (or larger) matrix of static Pods named `pod-x-y`.
* **The "Click":** Performed by running `kubectl delete pod pod-x-y`.
* **The Result:**
    * **Mined Pod:** Triggering a mine results in a "Nuclear Meltdown" (Namespace wipe / Game Over).
    * **Empty Pod:** Triggers a chain reaction (The Gamemaster automatically deletes adjacent empty pods).
    * **Hint Pod:** The Pod remains or is immediately recreated, exposing a "Hint" (number of adjacent mines) via a local HTTP endpoint.

---

## 📈 Leveling Progression: The Hardening Path

| Level | Name | Information Source (The Cheat) | Major Obstacle | Technical Concept Learned |
| :--- | :--- | :--- | :--- | :--- |
| **0** | **The Intern** | `ConfigMap` named `map` | None | `kubectl get cm` |
| **1** | **The Junior** | `Secret` (Base64 encoded) | RBAC restrictions (Secrets) | `kubectl get secrets` & Base64 |
| **2** | **The Infiltrator** | Environment Variables | No access to Secrets/CMs | `kubectl exec` & Env vars |
| **3** | **The Heart of the Machine** | Gamemaster Filesystem | No info in Game Pods | `kubectl exec` into Controller |
| **4** | **Amnesia** | None (In-Memory only) | Static files are gone | Transition to intended gameplay |
| **5** | **The Firewall** | HTTP Admin Endpoint | **NetworkPolicies** | Network isolation & Pod selectors |
| **6** | **The Sand Grain** | Hint Port | **Finalizers** (on 10% to 100% of pods) | `kubectl patch` & Lifecycle |
| **7** | **Port-Hacking** | Annotations | **Dynamic Ports** (Randomized per pod) | `jsonpath` & Metadata |
| **8** | **The Firing Window** | API Timing | **Time-Window Deletion** (0.0s to 0.1s) | Latency & Precise Scripting |
| **9** | **RBAC Blackout** | **Kubernetes Events** | Minimalist RBAC (No exec/get) | `kubectl get events` |

---

## 🛠️ Detailed Level Mechanics

### Level 5: The Firewall (NetworkPolicies)

The Gamemaster exposes a "cheat" endpoint at `:9999/debug/map`. However, a `NetworkPolicy` is applied to the namespace, blocking all traffic to this port. The player must analyze the policy to discover which labels are "whitelisted" and create a "Proxy Pod" with those labels to bypass the restriction.

### Level 6: The Sand Grain (Finalizers)
Some pods are marked with the `podsweeper.io/wait` finalizer. When a player tries to delete them, the pod stays in a `Terminating` state forever. The Gamemaster only validates the "click" once the pod is fully removed. The player must patch the pod to remove the finalizer before it can be truly deleted.

### Level 8: The Firing Window (Time-sync)
To simulate high-pressure production environments, the Gamemaster only accepts a `Delete` event if it hits the API Server within the first 100ms of any given second (e.g., `12:00:01.050` is valid, but `12:00:01.400` is rejected). This forces the player to write a script that synchronizes with the cluster clock.

### Level 9: RBAC Blackout
The player's `ServiceAccount` is stripped of almost all permissions. `kubectl exec`, `describe`, and even `get pod -o yaml` (annotations) are forbidden. The player must rely on `kubectl get events`, as the Gamemaster "leaks" hints into the event stream of the involved objects.

---

## 🏁 Win & Loss Conditions

* **Victory:** All non-mined pods are successfully removed. The Gamemaster spawns a `victory` Pod with a congratulatory message in the logs.
* **Defeat:** A mined pod is deleted. The Gamemaster triggers a `DeleteAll` on the namespace and replaces the grid with an ASCII "Explosion" Pod.
