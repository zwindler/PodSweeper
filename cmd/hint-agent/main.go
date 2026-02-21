// Package main is the entry point for the PodSweeper Hint Agent.
// The Hint Agent is a minimal HTTP server that runs inside hint pods.
// It exposes the hint value (number of adjacent mines) via HTTP.
//
// Configuration via environment variables:
//   - HINT_VALUE: The number to display (1-8)
//   - POD_X: The X coordinate of this pod
//   - POD_Y: The Y coordinate of this pod
//   - PORT: The port to listen on (default: 8080)
//
// Usage:
//
//	kubectl port-forward pod/hint-3-5 8080:8080
//	curl localhost:8080
//	# Returns: 3
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Version is set at build time
var Version = "dev"

func main() {
	// Read configuration from environment
	hintValue := os.Getenv("HINT_VALUE")
	if hintValue == "" {
		hintValue = "?"
	}

	podX := os.Getenv("POD_X")
	podY := os.Getenv("POD_Y")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Validate port is a number
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("Invalid PORT value: %s", port)
	}

	// Main hint endpoint - just returns the number
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s\n", hintValue)
	})

	// Health check endpoint for Kubernetes probes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Readiness check
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Info endpoint with coordinates (JSON)
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := map[string]string{
			"x":       podX,
			"y":       podY,
			"hint":    hintValue,
			"version": Version,
		}
		json.NewEncoder(w).Encode(info)
	})

	// ASCII art hint (for fun)
	http.HandleFunc("/ascii", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		ascii := getASCIIHint(hintValue)
		fmt.Fprint(w, ascii)
	})

	addr := ":" + port
	log.Printf("PodSweeper Hint Agent %s starting on %s", Version, addr)
	log.Printf("  Coordinates: (%s, %s)", podX, podY)
	log.Printf("  Hint value: %s", hintValue)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// getASCIIHint returns an ASCII art representation of the hint number
func getASCIIHint(hint string) string {
	hints := map[string]string{
		"1": `
  ██╗
 ███║
 ╚██║
  ██║
 ███║
 ╚══╝`,
		"2": `
██████╗
╚════██╗
 █████╔╝
██╔═══╝
███████╗
╚══════╝`,
		"3": `
██████╗
╚════██╗
 █████╔╝
 ╚═══██╗
██████╔╝
╚═════╝`,
		"4": `
██╗  ██╗
██║  ██║
███████║
╚════██║
     ██║
     ╚═╝`,
		"5": `
███████╗
██╔════╝
███████╗
╚════██║
███████║
╚══════╝`,
		"6": `
 ██████╗
██╔════╝
███████╗
██╔══██║
╚█████╔╝
 ╚════╝`,
		"7": `
███████╗
╚════██║
    ██╔╝
   ██╔╝
   ██║
   ╚═╝`,
		"8": `
 █████╗
██╔══██╗
╚█████╔╝
██╔══██╗
╚█████╔╝
 ╚════╝`,
	}

	if ascii, ok := hints[hint]; ok {
		return ascii + "\n"
	}
	return fmt.Sprintf("\n  %s\n\n", hint)
}
