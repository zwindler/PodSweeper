package manager

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zwindler/podsweeper/pkg/game"
	"github.com/zwindler/podsweeper/pkg/grid"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Namespace == "" {
		t.Error("expected namespace to have default value")
	}
	if cfg.CellImage == "" {
		t.Error("expected CellImage to have default value")
	}
	if cfg.SpawnBatchSize <= 0 {
		t.Error("expected SpawnBatchSize to be positive")
	}
	if cfg.WaitTimeout <= 0 {
		t.Error("expected WaitTimeout to be positive")
	}
}

func TestNewGameManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "with defaults",
			config: Config{},
		},
		{
			name: "with custom config",
			config: Config{
				Namespace:      "test-ns",
				CellImage:      "custom:image",
				SpawnBatchSize: 5,
				WaitTimeout:    time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewGameManager(fakeClient, store, tt.config)
			if mgr == nil {
				t.Fatal("expected non-nil manager")
			}
			if mgr.Namespace() == "" {
				t.Error("expected namespace to be set")
			}
		})
	}
}

func TestGameManager_StartGame(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, Config{
		Namespace:      "test-game",
		SpawnBatchSize: 25, // Spawn entire 5x5 grid in one batch
	})

	ctx := context.Background()

	// Start a level 0 game
	result, err := mgr.StartGame(ctx, StartGameOptions{
		Level:           0,
		Seed:            12345,
		WaitForReady:    false,
		CleanupExisting: false,
	})

	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify level config
	if result.LevelConfig.Level != 0 {
		t.Errorf("expected level 0, got %d", result.LevelConfig.Level)
	}
	if result.LevelConfig.Size != grid.Tier1Size {
		t.Errorf("expected size %d, got %d", grid.Tier1Size, result.LevelConfig.Size)
	}

	// Verify game state was created
	if result.GameState == nil {
		t.Fatal("expected non-nil game state")
	}
	if result.GameState.Seed != 12345 {
		t.Errorf("expected seed 12345, got %d", result.GameState.Seed)
	}
	if result.GameState.MineCount != grid.Tier1Mines {
		t.Errorf("expected %d mines, got %d", grid.Tier1Mines, result.GameState.MineCount)
	}

	// Verify spawn result
	if result.SpawnResult == nil {
		t.Fatal("expected non-nil spawn result")
	}
	expectedPods := grid.Tier1Size * grid.Tier1Size
	if result.SpawnResult.TotalPods != expectedPods {
		t.Errorf("expected %d total pods, got %d", expectedPods, result.SpawnResult.TotalPods)
	}
	if result.SpawnResult.CreatedPods != expectedPods {
		t.Errorf("expected %d created pods, got %d", expectedPods, result.SpawnResult.CreatedPods)
	}

	// Verify state was saved to store
	savedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("failed to load saved state: %v", err)
	}
	if savedState.Seed != 12345 {
		t.Errorf("saved state has wrong seed: %d", savedState.Seed)
	}
}

func TestGameManager_StartGame_InvalidLevel(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, DefaultConfig())
	ctx := context.Background()

	_, err := mgr.StartGame(ctx, StartGameOptions{
		Level: 99, // Invalid level
	})

	if err == nil {
		t.Error("expected error for invalid level")
	}
}

func TestGameManager_GetCurrentGame(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, DefaultConfig())
	ctx := context.Background()

	// No game yet
	_, err := mgr.GetCurrentGame(ctx)
	if err == nil {
		t.Error("expected error when no game exists")
	}

	// Start a game
	_, err = mgr.StartGame(ctx, StartGameOptions{
		Level: 0,
		Seed:  42,
	})
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	// Now we should be able to get it
	state, err := mgr.GetCurrentGame(ctx)
	if err != nil {
		t.Fatalf("GetCurrentGame failed: %v", err)
	}
	if state.Seed != 42 {
		t.Errorf("expected seed 42, got %d", state.Seed)
	}
}

func TestGameManager_GetStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, DefaultConfig())
	ctx := context.Background()

	// No game - should return inactive status
	status, err := mgr.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Active {
		t.Error("expected inactive status when no game")
	}

	// Start a game
	_, err = mgr.StartGame(ctx, StartGameOptions{
		Level: 2,
		Seed:  12345,
	})
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	// Check status
	status, err = mgr.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if !status.Active {
		t.Error("expected active status")
	}
	if status.Level != 2 {
		t.Errorf("expected level 2, got %d", status.Level)
	}
	if status.Tier != 1 {
		t.Errorf("expected tier 1, got %d", status.Tier)
	}
	if status.Size != grid.Tier1Size {
		t.Errorf("expected size %d, got %d", grid.Tier1Size, status.Size)
	}
	if status.MineCount != grid.Tier1Mines {
		t.Errorf("expected %d mines, got %d", grid.Tier1Mines, status.MineCount)
	}
	if status.GameOver {
		t.Error("expected game not to be over")
	}
	if status.Victory {
		t.Error("expected no victory yet")
	}
	if status.Remaining != (grid.Tier1Size*grid.Tier1Size - grid.Tier1Mines) {
		t.Errorf("expected %d remaining safe cells", grid.Tier1Size*grid.Tier1Size-grid.Tier1Mines)
	}
}

func TestGameManager_EndGame(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create some game pods in the fake client
	pods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-0-0",
				Namespace: "test-ns",
				Labels:    map[string]string{"app.kubernetes.io/name": "podsweeper"},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(pods...).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, Config{Namespace: "test-ns"})
	ctx := context.Background()

	// Start a game first
	_, err := mgr.StartGame(ctx, StartGameOptions{
		Level:           0,
		CleanupExisting: true,
	})
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	// End the game
	err = mgr.EndGame(ctx)
	if err != nil {
		t.Fatalf("EndGame failed: %v", err)
	}

	// Verify state was deleted
	_, err = mgr.GetCurrentGame(ctx)
	if err == nil {
		t.Error("expected error after ending game")
	}
}

func TestGameManager_ResetGame(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	mgr := NewGameManager(fakeClient, store, DefaultConfig())
	ctx := context.Background()

	// Can't reset without existing game
	_, err := mgr.ResetGame(ctx)
	if err == nil {
		t.Error("expected error when resetting with no game")
	}

	// Start a game at level 3
	_, err = mgr.StartGame(ctx, StartGameOptions{
		Level: 3,
		Seed:  100,
	})
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	// Reset - should create new game at same level
	result, err := mgr.ResetGame(ctx)
	if err != nil {
		t.Fatalf("ResetGame failed: %v", err)
	}

	if result.LevelConfig.Level != 3 {
		t.Errorf("expected level 3 after reset, got %d", result.LevelConfig.Level)
	}
	// Seed should be different (new random)
	if result.GameState.Seed == 100 {
		t.Error("expected new seed after reset")
	}
}

func TestGameManager_AllLevels(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	for level := 0; level <= 9; level++ {
		t.Run(game.DefaultNamespace, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			store := game.NewMemoryStore()
			mgr := NewGameManager(fakeClient, store, DefaultConfig())
			ctx := context.Background()

			result, err := mgr.StartGame(ctx, StartGameOptions{
				Level: level,
				Seed:  int64(level * 1000),
			})

			if err != nil {
				t.Fatalf("StartGame failed for level %d: %v", level, err)
			}

			config := grid.GetLevelConfig(level)
			if result.LevelConfig.Size != config.Size {
				t.Errorf("level %d: expected size %d, got %d", level, config.Size, result.LevelConfig.Size)
			}
			if result.LevelConfig.MineCount != config.MineCount {
				t.Errorf("level %d: expected %d mines, got %d", level, config.MineCount, result.LevelConfig.MineCount)
			}
		})
	}
}
