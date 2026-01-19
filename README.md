# 🛸 PodSweeper

> **The most impractical, over-engineered, and chaotic way to play Minesweeper.**

**PodSweeper** is a cloud-native "deminer" game where the cells aren't boxes on a screen, but **Live Pods** inside a Kubernetes cluster. To "click" on a cell, you don't use a mouse; you use `kubectl delete`.

---

## ⚠️ PROJECT STATUS: UNDER CONSTRUCTION 🏗️

**Note:** This project is currently in the **ideation/design phase**. 
The game is **not yet playable**. If you don't care for spoilers, feel free to check the `SPECIFICATION.md` and `LEVELING.md` files to see where we are headed!

---

## 🕹️ The Concept

PodSweeper turns your Kubernetes namespace into a minefield. 

1.  **The Grid:** The Gamemaster (a Go-based controller) spawns a matrix of pods named `pod-x-y`.
2.  **The Action:** Deleting a pod is your way of "sweeping" the tile. 
    * **Safe Pod:** If the pod is safe, it is replaced by a `hint-x-y` pod exposing the number of adjacent mines via HTTP.
    * **Empty Area:** If no mines are nearby, a chain reaction (BFS) clears the area automatically.
    * **Mined Pod:** If you delete a mine, the Namespace "explodes" (namespace wipe) and it's Game Over.

## 📈 Learning through Hardening (CTF Mode)

PodSweeper isn't just a game; it's a **Kubernetes CTF**. The game features **10 levels of increasing difficulty**. 

As you progress, the Gamemaster hardens the cluster to prevent "cheating" and force you to master deeper K8s concepts:
* **RBAC:** Access is stripped away, forcing you to find info in logs or events.
* **NetworkPolicies:** Your "cheat" scripts are blocked by network isolation.
* **Finalizers:** Deletions become sticky, requiring manual patches.
* **Admission Webhooks:** A timing-based challenge where deletions are only accepted within a 100ms window.

## 🚀 Why PodSweeper?

Because running `kubectl delete pod` should be scary, and we wanted to make it fun. This project is perfect for:
* **K8s Newbies:** Learn basic CLI and resources.
* **SREs/DevOps:** Test your scripting and troubleshooting skills under pressure.
* **Security Folks:** Understand how RBAC and Admission Controllers can be used to enforce strict policies.

---

## 🛠️ Technical Stack (Planned)

* **Language:** Go (Golang)
* **Framework:** `client-go` / `controller-runtime`
* **Architecture:** Native Controller + Validating Admission Webhook (No CRDs for maximum portability).
* **UI:** 100% terminal-based (`kubectl` + ASCII Art).

## 🛑 Disclaimer

PodSweeper is designed to be destructive within its own namespace. **Do not run this in a production cluster** unless you really want to explain to your boss why you were playing Minesweeper with the company's infrastructure.

---
*Created with ❤️ during a few vibe coding sessions.*
