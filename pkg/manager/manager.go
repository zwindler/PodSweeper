// Package manager orchestrates game lifecycle operations.
// It ties together grid generation, state persistence, and pod spawning.
package manager

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/zwindler/podsweeper/pkg/game"
	"github.com/zwindler/podsweeper/pkg/grid"
	"github.com/zwindler/podsweeper/pkg/spawner"
)

// GameManager orchestrates game creation and lifecycle.
type GameManager struct {
	client    client.Client
	store     game.Store
	namespace string
	config    Config
}

// Config holds configuration for the GameManager.
type Config struct {
	// Namespace where games are created.
	Namespace string

	// CellImage is the container image for cell pods.
	CellImage string

	// HintAgentImage is the container image for hint pods.
	HintAgentImage string

	// SpawnBatchSize is how many pods to create in parallel.
	SpawnBatchSize int

	// WaitTimeout is how long to wait for pods to be ready.
	WaitTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Namespace:      game.DefaultNamespace,
		CellImage:      spawner.CellImage,
		HintAgentImage: "ghcr.io/zwindler/podsweeper-hint-agent:latest",
		SpawnBatchSize: spawner.DefaultBatchSize,
		WaitTimeout:    5 * time.Minute,
	}
}

// NewGameManager creates a new GameManager.
func NewGameManager(c client.Client, store game.Store, cfg Config) *GameManager {
	if cfg.Namespace == "" {
		cfg.Namespace = game.DefaultNamespace
	}
	if cfg.CellImage == "" {
		cfg.CellImage = spawner.CellImage
	}
	if cfg.SpawnBatchSize <= 0 {
		cfg.SpawnBatchSize = spawner.DefaultBatchSize
	}
	if cfg.WaitTimeout <= 0 {
		cfg.WaitTimeout = 5 * time.Minute
	}

	return &GameManager{
		client:    c,
		store:     store,
		namespace: cfg.Namespace,
		config:    cfg,
	}
}

// StartGameOptions contains options for starting a new game.
type StartGameOptions struct {
	// Level is the game level (0-9).
	Level int

	// Seed is the random seed for mine placement.
	// If 0, a random seed will be used.
	Seed int64

	// WaitForReady waits for all pods to be running before returning.
	WaitForReady bool

	// CleanupExisting removes any existing game pods first.
	CleanupExisting bool
}

// StartGameResult contains the result of starting a new game.
type StartGameResult struct {
	// GameState is the created game state.
	GameState *game.GameState

	// SpawnResult contains pod creation statistics.
	SpawnResult *spawner.SpawnResult

	// Duration is the total time to start the game.
	Duration time.Duration

	// LevelConfig describes the level settings.
	LevelConfig grid.LevelConfig
}

// StartGame creates and starts a new game at the specified level.
func (m *GameManager) StartGame(ctx context.Context, opts StartGameOptions) (*StartGameResult, error) {
	logger := log.FromContext(ctx)
	start := time.Now()

	// Validate level
	if err := grid.ValidateLevel(opts.Level); err != nil {
		return nil, fmt.Errorf("invalid level: %w", err)
	}

	levelConfig := grid.GetLevelConfig(opts.Level)
	logger.Info("starting new game",
		"level", opts.Level,
		"tier", levelConfig.Tier,
		"size", levelConfig.Size,
		"mines", levelConfig.MineCount,
	)

	// Create spawner for this operation
	gridSpawner := spawner.NewGridSpawner(m.client, spawner.GridSpawnerConfig{
		Namespace: m.namespace,
		CellImage: m.config.CellImage,
		BatchSize: m.config.SpawnBatchSize,
	})

	// Cleanup existing game if requested
	if opts.CleanupExisting {
		logger.Info("cleaning up existing game")
		if err := gridSpawner.CleanupGrid(ctx); err != nil {
			logger.Error(err, "failed to cleanup existing game, continuing anyway")
		}
		// Also delete existing state
		if err := m.store.Delete(ctx); err != nil {
			logger.V(1).Info("no existing state to delete", "error", err)
		}
	}

	// Generate game state
	seed := opts.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	gameState, err := grid.GenerateForLevel(opts.Level, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to generate game: %w", err)
	}

	logger.Info("game generated",
		"seed", gameState.Seed,
		"mineCount", gameState.MineCount,
	)

	// Save game state
	if err := m.store.Save(ctx, gameState); err != nil {
		return nil, fmt.Errorf("failed to save game state: %w", err)
	}
	logger.Info("game state saved")

	// Spawn the grid pods
	spawnResult, err := gridSpawner.SpawnGrid(ctx, gameState)
	if err != nil {
		return nil, fmt.Errorf("failed to spawn grid: %w", err)
	}

	// Wait for pods to be ready if requested
	if opts.WaitForReady {
		expectedPods := levelConfig.Size * levelConfig.Size
		logger.Info("waiting for pods to be ready", "expected", expectedPods)
		if err := gridSpawner.WaitForPodsReady(ctx, expectedPods, m.config.WaitTimeout); err != nil {
			logger.Error(err, "timed out waiting for pods, game may not be ready")
		}
	}

	result := &StartGameResult{
		GameState:   gameState,
		SpawnResult: spawnResult,
		Duration:    time.Since(start),
		LevelConfig: levelConfig,
	}

	logger.Info("game started successfully",
		"duration", result.Duration,
		"podsCreated", spawnResult.CreatedPods,
	)

	return result, nil
}

// GetCurrentGame retrieves the current game state.
func (m *GameManager) GetCurrentGame(ctx context.Context) (*game.GameState, error) {
	state, err := m.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("no active game")
	}
	return state, nil
}

// EndGame cleans up the current game.
func (m *GameManager) EndGame(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// Create spawner for cleanup
	gridSpawner := spawner.NewGridSpawner(m.client, spawner.GridSpawnerConfig{
		Namespace: m.namespace,
	})

	// Cleanup pods
	if err := gridSpawner.CleanupGrid(ctx); err != nil {
		return fmt.Errorf("failed to cleanup grid: %w", err)
	}

	// Delete state
	if err := m.store.Delete(ctx); err != nil {
		logger.V(1).Info("no state to delete", "error", err)
	}

	logger.Info("game ended and cleaned up")
	return nil
}

// ResetGame ends the current game and starts a new one at the same level.
func (m *GameManager) ResetGame(ctx context.Context) (*StartGameResult, error) {
	logger := log.FromContext(ctx)

	// Load current game to get level
	currentGame, err := m.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("no current game to reset: %w", err)
	}
	if currentGame == nil {
		return nil, fmt.Errorf("no current game to reset")
	}

	level := currentGame.Level
	logger.Info("resetting game", "level", level)

	// Start new game at same level with cleanup
	return m.StartGame(ctx, StartGameOptions{
		Level:           level,
		CleanupExisting: true,
		WaitForReady:    false,
	})
}

// GameStatus returns information about the current game.
type GameStatus struct {
	Active      bool
	Level       int
	Tier        int
	TierDesc    string
	Size        int
	MineCount   int
	Revealed    int
	Remaining   int
	GameOver    bool
	Victory     bool
	StartedAt   time.Time
	ElapsedTime time.Duration
}

// GetStatus returns the status of the current game.
func (m *GameManager) GetStatus(ctx context.Context) (*GameStatus, error) {
	state, err := m.store.Load(ctx)
	if err != nil || state == nil {
		return &GameStatus{Active: false}, nil
	}

	config := grid.GetLevelConfig(state.Level)
	totalCells := state.Size * state.Size

	// Count revealed cells
	revealedCount := 0
	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			if state.Revealed[x][y] {
				revealedCount++
			}
		}
	}
	safeCells := totalCells - state.MineCount

	return &GameStatus{
		Active:      true,
		Level:       state.Level,
		Tier:        config.Tier,
		TierDesc:    grid.GetTierDescription(config.Tier),
		Size:        state.Size,
		MineCount:   state.MineCount,
		Revealed:    revealedCount,
		Remaining:   safeCells - revealedCount,
		GameOver:    state.Status == game.StatusLost || state.Status == game.StatusWon,
		Victory:     state.Status == game.StatusWon,
		StartedAt:   state.StartedAt,
		ElapsedTime: time.Since(state.StartedAt),
	}, nil
}

// Namespace returns the namespace where games are managed.
func (m *GameManager) Namespace() string {
	return m.namespace
}
