// Package controller contains the Kubernetes controller logic for PodSweeper.
package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/zwindler/podsweeper/pkg/game"
	"github.com/zwindler/podsweeper/pkg/grid"
	"github.com/zwindler/podsweeper/pkg/level"
	"github.com/zwindler/podsweeper/pkg/spawner"
)

const (
	// ConfigMapName is the name of the ConfigMap that controls the game.
	ConfigMapName = "podsweeper-config"

	// ConfigKeyLevel is the key for the game level (0-9).
	ConfigKeyLevel = "level"

	// ConfigKeySeed is the key for the random seed (optional).
	ConfigKeySeed = "seed"

	// ConfigKeyAction is the key for triggering actions like "start" or "end".
	ConfigKeyAction = "action"

	// StatusKeyGameState is the key for the current game state in status.
	StatusKeyGameState = "status"

	// StatusKeyMessage is the key for status messages.
	StatusKeyMessage = "message"

	// StatusKeyLastUpdate is the key for the last update timestamp.
	StatusKeyLastUpdate = "lastUpdate"

	// StatusKeyLevel is the key for the current level.
	StatusKeyLevel = "currentLevel"

	// StatusKeyMines is the key for the mine count.
	StatusKeyMines = "mines"

	// StatusKeySize is the key for the grid size.
	StatusKeySize = "gridSize"

	// StatusKeyProgress is the key for game progress.
	StatusKeyProgress = "progress"
)

// ConfigController watches the game configuration ConfigMap and manages game lifecycle.
type ConfigController struct {
	client.Client
	Store     game.Store
	Namespace string

	// lastProcessedGeneration tracks the last processed ConfigMap generation
	// to avoid reprocessing the same config
	lastProcessedGeneration int64
	lastProcessedLevel      int
	lastProcessedSeed       int64
}

// ConfigControllerConfig holds configuration for the ConfigController.
type ConfigControllerConfig struct {
	Namespace string
	Store     game.Store
}

// NewConfigController creates a new ConfigController.
func NewConfigController(c client.Client, config ConfigControllerConfig) *ConfigController {
	return &ConfigController{
		Client:    c,
		Store:     config.Store,
		Namespace: config.Namespace,
	}
}

// Reconcile handles ConfigMap events for the game configuration.
func (r *ConfigController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Only process our specific ConfigMap in our namespace
	if req.Namespace != r.Namespace || req.Name != ConfigMapName {
		return ctrl.Result{}, nil
	}

	// Get the ConfigMap
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, cm); err != nil {
		if errors.IsNotFound(err) {
			// ConfigMap deleted - end the game
			logger.Info("config ConfigMap deleted, ending game")
			return r.handleGameEnd(ctx)
		}
		return ctrl.Result{}, err
	}

	// Parse configuration
	level, seed, action, err := r.parseConfig(cm)
	if err != nil {
		logger.Error(err, "invalid configuration")
		return r.updateStatus(ctx, cm, "error", fmt.Sprintf("Invalid config: %v", err))
	}

	// Handle explicit actions
	if action == "end" {
		logger.Info("ending game by action")
		return r.handleGameEnd(ctx)
	}

	// Check if we need to start/restart a game
	// We start a new game if:
	// 1. Level changed
	// 2. Seed changed (and seed is specified)
	// 3. Action is "start" or "restart"
	// 4. No game is currently active
	needsNewGame := false
	reason := ""

	if action == "start" || action == "restart" {
		needsNewGame = true
		reason = "action requested"
	} else if level != r.lastProcessedLevel {
		needsNewGame = true
		reason = fmt.Sprintf("level changed from %d to %d", r.lastProcessedLevel, level)
	} else if seed != 0 && seed != r.lastProcessedSeed {
		needsNewGame = true
		reason = "seed changed"
	} else {
		// Check if there's an active game
		state, err := r.Store.Load(ctx)
		if err != nil || state == nil {
			needsNewGame = true
			reason = "no active game"
		}
	}

	if needsNewGame {
		logger.Info("starting new game", "level", level, "seed", seed, "reason", reason)
		return r.handleGameStart(ctx, cm, level, seed)
	}

	// Game is already running at the correct level, just update status
	return r.updateStatusFromGame(ctx, cm)
}

// parseConfig extracts game configuration from the ConfigMap.
func (r *ConfigController) parseConfig(cm *corev1.ConfigMap) (level int, seed int64, action string, err error) {
	// Parse level (default 0)
	levelStr := cm.Data[ConfigKeyLevel]
	if levelStr == "" {
		level = 0
	} else {
		level, err = strconv.Atoi(levelStr)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid level %q: %w", levelStr, err)
		}
		if err := grid.ValidateLevel(level); err != nil {
			return 0, 0, "", err
		}
	}

	// Parse seed (default 0 = random)
	seedStr := cm.Data[ConfigKeySeed]
	if seedStr != "" {
		seed, err = strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid seed %q: %w", seedStr, err)
		}
	}

	// Parse action
	action = cm.Data[ConfigKeyAction]

	return level, seed, action, nil
}

// handleGameStart starts a new game with the given configuration.
func (r *ConfigController) handleGameStart(ctx context.Context, cm *corev1.ConfigMap, lvl int, seed int64) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create spawner for cleanup and spawning
	gridSpawner := spawner.NewGridSpawner(r.Client, spawner.GridSpawnerConfig{
		Namespace: r.Namespace,
	})

	// Create level manager for level-specific resources
	levelMgr := level.NewManager(r.Client, r.Namespace)

	// Generate seed if not provided
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	// Generate new game state
	gameState, err := grid.GenerateForLevel(lvl, seed)
	if err != nil {
		logger.Error(err, "failed to generate game", "level", lvl)
		return r.updateStatus(ctx, cm, "error", fmt.Sprintf("Failed to generate game: %v", err))
	}

	// IMPORTANT: Save game state BEFORE cleanup to avoid race condition.
	// When we cleanup the old grid, pod deletions trigger the GameController.
	// If the old state is still saved (e.g., "lost" status), GameController
	// would see it and potentially trigger unwanted behavior.
	// By saving the new "playing" state first, GameController sees the new game
	// and ignores the cleanup deletions.
	if err := r.Store.Save(ctx, gameState); err != nil {
		logger.Error(err, "failed to save game state")
		return r.updateStatus(ctx, cm, "error", fmt.Sprintf("Failed to save state: %v", err))
	}

	// Apply level-specific resources (e.g., map ConfigMap for Level 0)
	if err := levelMgr.ApplyLevel(ctx, gameState); err != nil {
		logger.Error(err, "failed to apply level resources", "level", lvl)
		// Continue anyway - level resources are "cheat" paths, not required for gameplay
	}

	// Cleanup existing game (now safe - new state already saved)
	if err := gridSpawner.CleanupGrid(ctx); err != nil {
		logger.Error(err, "failed to cleanup existing game")
		// Continue anyway
	}

	// Wait for cleanup to complete before spawning new pods
	// This prevents race conditions where deletion events from cleanup
	// get processed as player clicks on the new game
	if err := gridSpawner.WaitForCleanup(ctx, 30*time.Second); err != nil {
		logger.Error(err, "timeout waiting for cleanup to complete")
		// Continue anyway - we'll try to spawn the new grid
	}

	// Spawn grid
	result, err := gridSpawner.SpawnGrid(ctx, gameState)
	if err != nil {
		logger.Error(err, "failed to spawn grid")
		return r.updateStatus(ctx, cm, "error", fmt.Sprintf("Failed to spawn grid: %v", err))
	}

	// Update tracking
	r.lastProcessedLevel = lvl
	r.lastProcessedSeed = seed

	logger.Info("game started successfully",
		"level", lvl,
		"seed", seed,
		"size", gameState.Size,
		"mines", gameState.MineCount,
		"pods", result.CreatedPods,
	)

	// Clear the action field after processing
	return r.updateStatusAndClearAction(ctx, cm, "playing",
		fmt.Sprintf("Game started: Level %d, %dx%d grid, %d mines, seed %d",
			lvl, gameState.Size, gameState.Size, gameState.MineCount, seed))
}

// handleGameEnd ends the current game.
func (r *ConfigController) handleGameEnd(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Create spawner for cleanup
	gridSpawner := spawner.NewGridSpawner(r.Client, spawner.GridSpawnerConfig{
		Namespace: r.Namespace,
	})

	// Create level manager for cleanup
	levelMgr := level.NewManager(r.Client, r.Namespace)

	// Cleanup pods
	if err := gridSpawner.CleanupGrid(ctx); err != nil {
		logger.Error(err, "failed to cleanup grid")
	}

	// Cleanup level-specific resources (map ConfigMap/Secret, etc.)
	if err := levelMgr.Cleanup(ctx); err != nil {
		logger.Error(err, "failed to cleanup level resources")
	}

	// Delete state
	if err := r.Store.Delete(ctx); err != nil {
		logger.V(1).Info("no state to delete", "error", err)
	}

	// Reset tracking
	r.lastProcessedLevel = 0
	r.lastProcessedSeed = 0

	logger.Info("game ended")

	return ctrl.Result{}, nil
}

// updateStatus updates the ConfigMap with status information.
func (r *ConfigController) updateStatus(ctx context.Context, cm *corev1.ConfigMap, status, message string) (ctrl.Result, error) {
	// Create a copy to avoid modifying the cache
	cmCopy := cm.DeepCopy()

	if cmCopy.Data == nil {
		cmCopy.Data = make(map[string]string)
	}

	cmCopy.Data[StatusKeyGameState] = status
	cmCopy.Data[StatusKeyMessage] = message
	cmCopy.Data[StatusKeyLastUpdate] = time.Now().Format(time.RFC3339)

	if err := r.Update(ctx, cmCopy); err != nil {
		if errors.IsConflict(err) {
			// Requeue to retry
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateStatusAndClearAction updates status and clears the action field.
func (r *ConfigController) updateStatusAndClearAction(ctx context.Context, cm *corev1.ConfigMap, status, message string) (ctrl.Result, error) {
	cmCopy := cm.DeepCopy()

	if cmCopy.Data == nil {
		cmCopy.Data = make(map[string]string)
	}

	cmCopy.Data[StatusKeyGameState] = status
	cmCopy.Data[StatusKeyMessage] = message
	cmCopy.Data[StatusKeyLastUpdate] = time.Now().Format(time.RFC3339)

	// Clear action to prevent re-triggering
	delete(cmCopy.Data, ConfigKeyAction)

	if err := r.Update(ctx, cmCopy); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateStatusFromGame updates the ConfigMap with current game state.
func (r *ConfigController) updateStatusFromGame(ctx context.Context, cm *corev1.ConfigMap) (ctrl.Result, error) {
	state, err := r.Store.Load(ctx)
	if err != nil || state == nil {
		return r.updateStatus(ctx, cm, "no-game", "No active game")
	}

	status := "playing"
	message := fmt.Sprintf("Level %d in progress", state.Level)

	switch state.Status {
	case game.StatusWon:
		status = "won"
		message = "Victory! All safe cells revealed."
	case game.StatusLost:
		status = "lost"
		message = "Game over - hit a mine!"
	}

	// Count revealed cells for progress
	revealed := 0
	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			if state.Revealed[x][y] {
				revealed++
			}
		}
	}
	safeCells := state.Size*state.Size - state.MineCount
	progress := float64(revealed) / float64(safeCells) * 100

	cmCopy := cm.DeepCopy()
	if cmCopy.Data == nil {
		cmCopy.Data = make(map[string]string)
	}

	cmCopy.Data[StatusKeyGameState] = status
	cmCopy.Data[StatusKeyMessage] = message
	cmCopy.Data[StatusKeyLastUpdate] = time.Now().Format(time.RFC3339)
	cmCopy.Data[StatusKeyLevel] = strconv.Itoa(state.Level)
	cmCopy.Data[StatusKeySize] = fmt.Sprintf("%dx%d", state.Size, state.Size)
	cmCopy.Data[StatusKeyMines] = strconv.Itoa(state.MineCount)
	cmCopy.Data[StatusKeyProgress] = fmt.Sprintf("%.1f%%", progress)

	if err := r.Update(ctx, cmCopy); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			// Only watch our specific ConfigMap
			return object.GetNamespace() == r.Namespace && object.GetName() == ConfigMapName
		})).
		Complete(r)
}

// EnsureConfigMap creates the ConfigMap if it doesn't exist.
func (r *ConfigController) EnsureConfigMap(ctx context.Context) error {
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: ConfigMapName}, cm)
	if err == nil {
		// Already exists
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	// Create default ConfigMap
	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "podsweeper",
				"app.kubernetes.io/component": "config",
			},
		},
		Data: map[string]string{
			ConfigKeyLevel:      "0",
			StatusKeyGameState:  "pending",
			StatusKeyMessage:    "Set 'action: start' to begin a game",
			StatusKeyLastUpdate: time.Now().Format(time.RFC3339),
		},
	}

	return r.Create(ctx, cm)
}
