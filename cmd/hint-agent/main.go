// Package main is the entry point for the PodSweeper Hint Agent.
// The Hint Agent is a minimal HTTP server that runs inside hint pods.
// It exposes the hint value (number of adjacent mines) via HTTP.
//
// Configuration via environment variables:
//   - HINT_VALUE: The number to display (0-8)
//   - POD_X: The X coordinate of this pod
//   - POD_Y: The Y coordinate of this pod
//   - PORT: The port to listen on (default: 8080)
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

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

	// Create HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%s\n", hintValue)
	})

	// Health check endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Info endpoint with coordinates
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"x":%q,"y":%q,"hint":%q}`, podX, podY, hintValue)
	})

	addr := ":" + port
	log.Printf("Hint Agent starting on %s (hint=%s, x=%s, y=%s)", addr, hintValue, podX, podY)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
