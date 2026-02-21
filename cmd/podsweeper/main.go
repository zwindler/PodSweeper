// Package main is the entry point for the PodSweeper CLI.
// This tool allows players to start, manage, and monitor PodSweeper games.
//
// Usage:
//
//	podsweeper start [--level N] [--seed N]  Start a new game
//	podsweeper status                        Show current game status
//	podsweeper reset                         Reset current game
//	podsweeper end                           End current game and cleanup
//	podsweeper levels                        List all available levels
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/zwindler/podsweeper/pkg/game"
	"github.com/zwindler/podsweeper/pkg/grid"
	"github.com/zwindler/podsweeper/pkg/manager"
)

var Version = "dev"

func main() {
	// Set up logging
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "start":
		cmdStart(os.Args[2:])
	case "status":
		cmdStatus()
	case "reset":
		cmdReset()
	case "end":
		cmdEnd()
	case "levels":
		cmdLevels()
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("podsweeper %s\n", Version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`PodSweeper - The most impractical way to play Minesweeper

Usage:
  podsweeper <command> [options]

Commands:
  start     Start a new game
  status    Show current game status
  reset     Reset current game (same level, new mines)
  end       End current game and cleanup pods
  levels    List all available levels
  help      Show this help message
  version   Show version

Start Options:
  --level N    Game level 0-9 (default: 0)
  --seed N     Random seed for mine placement (default: random)
  --wait       Wait for all pods to be ready before returning

Examples:
  podsweeper start                    # Start level 0 game
  podsweeper start --level 3          # Start level 3 game
  podsweeper start --level 0 --seed 42  # Reproducible game
  podsweeper status                   # Check game progress
  podsweeper end                      # Cleanup and end game

Playing the game:
  kubectl get pods -n podsweeper-game        # View the grid
  kubectl delete pod pod-2-3 -n podsweeper-game  # Click cell (2,3)
  kubectl port-forward hint-2-3 8080:8080    # View hint value`)
}

func cmdStart(args []string) {
	level := 0
	seed := int64(0)
	wait := false

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--level", "-l":
			if i+1 >= len(args) {
				fatal("--level requires a value")
			}
			i++
			var err error
			level, err = strconv.Atoi(args[i])
			if err != nil {
				fatal("invalid level: %s", args[i])
			}
		case "--seed", "-s":
			if i+1 >= len(args) {
				fatal("--seed requires a value")
			}
			i++
			var err error
			seed, err = strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				fatal("invalid seed: %s", args[i])
			}
		case "--wait", "-w":
			wait = true
		default:
			fatal("unknown option: %s", args[i])
		}
	}

	// Validate level
	if err := grid.ValidateLevel(level); err != nil {
		fatal("invalid level: %v", err)
	}

	mgr := getManager()
	ctx := context.Background()

	fmt.Printf("Starting PodSweeper level %d...\n", level)

	result, err := mgr.StartGame(ctx, manager.StartGameOptions{
		Level:           level,
		Seed:            seed,
		WaitForReady:    wait,
		CleanupExisting: true,
	})
	if err != nil {
		fatal("failed to start game: %v", err)
	}

	fmt.Println()
	fmt.Printf("Game started!\n")
	fmt.Printf("  Level: %d (Tier %d)\n", result.LevelConfig.Level, result.LevelConfig.Tier)
	fmt.Printf("  Grid:  %dx%d (%d cells)\n", result.LevelConfig.Size, result.LevelConfig.Size, result.LevelConfig.Size*result.LevelConfig.Size)
	fmt.Printf("  Mines: %d\n", result.LevelConfig.MineCount)
	fmt.Printf("  Seed:  %d\n", result.GameState.Seed)
	fmt.Printf("  Pods:  %d created in %s\n", result.SpawnResult.CreatedPods, result.Duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("How to play:")
	fmt.Printf("  kubectl get pods -n %s\n", mgr.Namespace())
	fmt.Printf("  kubectl delete pod pod-X-Y -n %s\n", mgr.Namespace())
	fmt.Println()
	fmt.Println("Good luck!")
}

func cmdStatus() {
	mgr := getManager()
	ctx := context.Background()

	status, err := mgr.GetStatus(ctx)
	if err != nil {
		fatal("failed to get status: %v", err)
	}

	if !status.Active {
		fmt.Println("No active game.")
		fmt.Println("Start a new game with: podsweeper start")
		return
	}

	fmt.Println("PodSweeper Game Status")
	fmt.Println("======================")
	fmt.Printf("Level:     %d (Tier %d)\n", status.Level, status.Tier)
	fmt.Printf("Grid:      %dx%d\n", status.Size, status.Size)
	fmt.Printf("Mines:     %d\n", status.MineCount)
	fmt.Printf("Revealed:  %d cells\n", status.Revealed)
	fmt.Printf("Remaining: %d safe cells\n", status.Remaining)
	fmt.Printf("Time:      %s\n", status.ElapsedTime.Round(time.Second))

	if status.GameOver {
		if status.Victory {
			fmt.Println()
			fmt.Println("STATUS: VICTORY!")
			fmt.Println("Congratulations! You cleared all safe cells!")
		} else {
			fmt.Println()
			fmt.Println("STATUS: GAME OVER")
			fmt.Println("You hit a mine!")
		}
	} else {
		fmt.Println()
		fmt.Println("STATUS: In Progress")
		fmt.Printf("Progress: %.1f%%\n", float64(status.Revealed)/float64(status.Size*status.Size-status.MineCount)*100)
	}
}

func cmdReset() {
	mgr := getManager()
	ctx := context.Background()

	fmt.Println("Resetting game...")

	result, err := mgr.ResetGame(ctx)
	if err != nil {
		fatal("failed to reset game: %v", err)
	}

	fmt.Printf("Game reset! New seed: %d\n", result.GameState.Seed)
	fmt.Printf("Pods: %d created in %s\n", result.SpawnResult.CreatedPods, result.Duration.Round(time.Millisecond))
}

func cmdEnd() {
	mgr := getManager()
	ctx := context.Background()

	fmt.Println("Ending game and cleaning up...")

	if err := mgr.EndGame(ctx); err != nil {
		fatal("failed to end game: %v", err)
	}

	fmt.Println("Game ended. All pods cleaned up.")
}

func cmdLevels() {
	fmt.Println("PodSweeper Levels")
	fmt.Println("=================")
	fmt.Println()

	configs := grid.AllLevelConfigs()
	currentTier := 0

	for _, cfg := range configs {
		if cfg.Tier != currentTier {
			currentTier = cfg.Tier
			fmt.Printf("\n%s\n", grid.GetTierDescription(cfg.Tier))
			fmt.Println("---")
		}
		fmt.Printf("  Level %d: %dx%d grid, %d mines\n", cfg.Level, cfg.Size, cfg.Size, cfg.MineCount)
	}

	fmt.Println()
	fmt.Println("Security features by level:")
	fmt.Println("  Level 0: No security (learn kubectl)")
	fmt.Println("  Level 1: Resource limits")
	fmt.Println("  Level 2: Read-only root filesystem")
	fmt.Println("  Level 3: Non-root user")
	fmt.Println("  Level 4: Network policies")
	fmt.Println("  Level 5: Pod Security Standards")
	fmt.Println("  Level 6: Seccomp profiles")
	fmt.Println("  Level 7: Service mesh (mTLS)")
	fmt.Println("  Level 8: OPA/Gatekeeper policies")
	fmt.Println("  Level 9: Full hardening + admission control")
}

func getManager() *manager.GameManager {
	// Build Kubernetes client
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = home + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fatal("failed to load kubeconfig: %v", err)
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		fatal("failed to create Kubernetes client: %v", err)
	}

	// Create store and manager
	namespace := os.Getenv("PODSWEEPER_NAMESPACE")
	if namespace == "" {
		namespace = game.DefaultNamespace
	}

	store := game.NewSecretStore(k8sClient,
		game.WithNamespace(namespace),
	)

	return manager.NewGameManager(k8sClient, store, manager.Config{
		Namespace: namespace,
	})
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
