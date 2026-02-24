package controller

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zwindler/podsweeper/pkg/game"
)

const testNamespace = "podsweeper-game"

// --- Pod name parsing tests ---

func TestParsePodName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOK    bool
		wantCoord game.Coordinate
	}{
		{"valid pod-0-0", "pod-0-0", true, game.Coordinate{X: 0, Y: 0}},
		{"valid pod-3-5", "pod-3-5", true, game.Coordinate{X: 3, Y: 5}},
		{"valid pod-99-99", "pod-99-99", true, game.Coordinate{X: 99, Y: 99}},
		{"hint pod", "hint-3-5", false, game.Coordinate{}},
		{"random name", "nginx", false, game.Coordinate{}},
		{"partial match", "pod-3", false, game.Coordinate{}},
		{"invalid format", "pod-a-b", false, game.Coordinate{}},
		{"empty string", "", false, game.Coordinate{}},
		{"explosion pod", "explosion", false, game.Coordinate{}},
		{"victory pod", "victory", false, game.Coordinate{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord, ok := ParsePodName(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParsePodName(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && (coord.X != tt.wantCoord.X || coord.Y != tt.wantCoord.Y) {
				t.Errorf("ParsePodName(%q) coord = %v, want %v", tt.input, coord, tt.wantCoord)
			}
		})
	}
}

func TestParseHintPodName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOK    bool
		wantCoord game.Coordinate
	}{
		{"valid hint-0-0", "hint-0-0", true, game.Coordinate{X: 0, Y: 0}},
		{"valid hint-3-5", "hint-3-5", true, game.Coordinate{X: 3, Y: 5}},
		{"valid hint-99-99", "hint-99-99", true, game.Coordinate{X: 99, Y: 99}},
		{"game pod", "pod-3-5", false, game.Coordinate{}},
		{"random name", "nginx", false, game.Coordinate{}},
		{"partial match", "hint-3", false, game.Coordinate{}},
		{"invalid format", "hint-a-b", false, game.Coordinate{}},
		{"empty string", "", false, game.Coordinate{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord, ok := ParseHintPodName(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParseHintPodName(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok && (coord.X != tt.wantCoord.X || coord.Y != tt.wantCoord.Y) {
				t.Errorf("ParseHintPodName(%q) coord = %v, want %v", tt.input, coord, tt.wantCoord)
			}
		})
	}
}

func TestIsPodName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"pod-0-0", true},
		{"pod-3-5", true},
		{"hint-3-5", false},
		{"nginx", false},
		{"explosion", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsPodName(tt.input); got != tt.want {
				t.Errorf("IsPodName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHintPodName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hint-0-0", true},
		{"hint-3-5", true},
		{"pod-3-5", false},
		{"nginx", false},
		{"explosion", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsHintPodName(tt.input); got != tt.want {
				t.Errorf("IsHintPodName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGeneratePodName(t *testing.T) {
	tests := []struct {
		x, y int
		want string
	}{
		{0, 0, "pod-0-0"},
		{3, 5, "pod-3-5"},
		{99, 99, "pod-99-99"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := GeneratePodName(tt.x, tt.y); got != tt.want {
				t.Errorf("GeneratePodName(%d, %d) = %q, want %q", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestGenerateHintPodName(t *testing.T) {
	tests := []struct {
		x, y int
		want string
	}{
		{0, 0, "hint-0-0"},
		{3, 5, "hint-3-5"},
		{99, 99, "hint-99-99"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := GenerateHintPodName(tt.x, tt.y); got != tt.want {
				t.Errorf("GenerateHintPodName(%d, %d) = %q, want %q", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

// --- Helper functions for tests ---

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func createTestPod(name, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				LabelApp:       "podsweeper",
				LabelComponent: "cell",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "cell",
					Image: "busybox:latest",
				},
			},
		},
	}
}

func createTestGameState(size int) *game.GameState {
	state := game.NewGameState(size, 12345)
	// Set up a simple mine at (1,1) for testing
	state.SetMine(1, 1)
	return state
}

// --- Controller tests ---

func TestGameController_ReconcileIgnoresOtherNamespaces(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	// Create a pod in a different namespace
	pod := createTestPod("pod-3-5", "other-namespace")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	_ = store.Save(ctx, state)

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "pod-3-5",
			Namespace: "other-namespace",
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue for pod in different namespace")
	}
}

func TestGameController_ReconcileIgnoresNonGamePods(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	// Create a non-game pod (doesn't match pod-X-Y pattern)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment-abc123",
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:latest"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	_ = store.Save(ctx, state)

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nginx-deployment-abc123",
			Namespace: testNamespace,
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue for non-game pod")
	}
}

func TestGameController_ReconcileIgnoresPodWithDeletionTimestamp(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()
	now := metav1.Now()

	// Create a pod that is being deleted (has DeletionTimestamp)
	pod := createTestPod("pod-3-5", testNamespace)
	pod.DeletionTimestamp = &now
	pod.Finalizers = []string{"test-finalizer"} // Required for DeletionTimestamp to be set

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	_ = store.Save(ctx, state)

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "pod-3-5",
			Namespace: testNamespace,
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue for terminating pod")
	}
}

func TestGameController_ReconcileNoGameState(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Empty store - no game in progress
	store := game.NewMemoryStore()

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	// Pod was deleted (not found)
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "pod-3-5",
			Namespace: testNamespace,
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue when no game state exists")
	}
}

func TestGameController_ReconcileIgnoresAlreadyRevealed(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	// Mark cell as already revealed
	state.Reveal(3, 5)
	_ = store.Save(ctx, state)

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	// Pod was deleted (not found) but already revealed
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "pod-3-5",
			Namespace: testNamespace,
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue for already revealed cell")
	}
}

func TestGameController_ReconcileIgnoresGameOver(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	state.SetLost() // Game is already over
	_ = store.Save(ctx, state)

	controller := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "pod-3-5",
			Namespace: testNamespace,
		},
	}

	result, err := controller.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("expected no requeue when game is already over")
	}
}

// --- Handler tests ---

func TestGameHandlers_HandleMineHit(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	state.SetMine(3, 3) // Add a mine
	_ = store.Save(ctx, state)

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})
	coords := game.Coordinate{X: 3, Y: 3}

	_, err := handlers.HandleMineHit(ctx, state, coords)
	if err != nil {
		t.Fatalf("HandleMineHit returned error: %v", err)
	}

	// Check state was updated
	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if loadedState.Status != game.StatusLost {
		t.Errorf("expected status %s, got %s", game.StatusLost, loadedState.Status)
	}

	if !loadedState.IsRevealed(3, 3) {
		t.Error("expected mine cell to be revealed")
	}

	// Check explosion pod was created
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "explosion", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Explosion pod was not created: %v", err)
	}
}

func TestGameHandlers_HandleHintCell(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	state := createTestGameState(8)
	state.SetMine(1, 1) // Mine at 1,1
	_ = store.Save(ctx, state)

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})
	// Cell at 0,0 is adjacent to the mine at 1,1
	coords := game.Coordinate{X: 0, Y: 0}
	hintValue := state.AdjacentMines(0, 0)

	_, err := handlers.HandleHintCell(ctx, state, coords, hintValue)
	if err != nil {
		t.Fatalf("HandleHintCell returned error: %v", err)
	}

	// Check state was updated
	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if !loadedState.IsRevealed(0, 0) {
		t.Error("expected cell to be revealed")
	}

	// Check hint cells were recorded
	if len(loadedState.HintCells) != 1 {
		t.Errorf("expected 1 hint cell, got %d", len(loadedState.HintCells))
	}

	// Check hint pod was created
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "hint-0-0", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Hint pod was not created: %v", err)
	}

	// Verify hint pod has correct labels
	if pod.Labels[LabelComponent] != "hint" {
		t.Errorf("expected component label 'hint', got %q", pod.Labels[LabelComponent])
	}

	// Verify hint value annotation
	if pod.Annotations[AnnotationHint] != "1" {
		t.Errorf("expected hint annotation '1', got %q", pod.Annotations[AnnotationHint])
	}
}

func TestGameHandlers_HandleEmptyCell_BFSPropagation(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	// Create a 5x5 grid with mines only in the bottom-right corner
	state := game.NewGameState(5, 12345)
	state.SetMine(4, 4) // Mine in corner
	state.SetMine(4, 3)
	state.SetMine(3, 4)
	_ = store.Save(ctx, state)

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})
	// Click on empty cell in top-left corner - should propagate
	coords := game.Coordinate{X: 0, Y: 0}

	_, err := handlers.HandleEmptyCell(ctx, state, coords)
	if err != nil {
		t.Fatalf("HandleEmptyCell returned error: %v", err)
	}

	// Check state was updated - multiple cells should be revealed
	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// The top-left corner should be revealed
	if !loadedState.IsRevealed(0, 0) {
		t.Error("expected (0,0) to be revealed")
	}

	// Cell (0,1) should also be revealed (adjacent empty)
	if !loadedState.IsRevealed(0, 1) {
		t.Error("expected (0,1) to be revealed")
	}
}

func TestGameHandlers_BFSPropagation(t *testing.T) {
	store := game.NewMemoryStore()

	// Create a 4x4 grid:
	// . . . M
	// . . . .
	// . . . .
	// . . . .
	// Empty cell at (0,0), mine at (3,0)
	state := game.NewGameState(4, 12345)
	state.SetMine(3, 0)

	handlers := NewGameHandlers(nil, store, GameHandlersConfig{Namespace: testNamespace})
	start := game.Coordinate{X: 0, Y: 0}

	empty, boundary := handlers.bfsPropagation(state, start)

	// Should find many empty cells and some boundary cells
	if len(empty) == 0 {
		t.Error("expected some empty cells from BFS")
	}

	// The cells adjacent to the mine should be boundaries
	// (3,1), (2,0), (2,1) are adjacent to the mine
	hasBoundary := false
	for _, b := range boundary {
		if (b.X == 2 && b.Y == 0) || (b.X == 2 && b.Y == 1) || (b.X == 3 && b.Y == 1) {
			hasBoundary = true
			break
		}
	}
	if !hasBoundary {
		t.Error("expected boundary cells adjacent to mine")
	}
}

func TestGameHandlers_BFSDoesNotRevealMines(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()

	// Create a 5x5 grid with mines scattered around
	// M . . . M
	// . . . . .
	// . . M . .
	// . . . . .
	// M . . . M
	state := game.NewGameState(5, 12345)
	state.SetMine(0, 0)
	state.SetMine(4, 0)
	state.SetMine(2, 2)
	state.SetMine(0, 4)
	state.SetMine(4, 4)
	_ = store.Save(ctx, state)

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	// Click on empty cell at (1,1) - BFS should propagate but NOT reveal mines
	coords := game.Coordinate{X: 1, Y: 1}
	_, err := handlers.HandleEmptyCell(ctx, state, coords)
	if err != nil {
		t.Fatalf("HandleEmptyCell returned error: %v", err)
	}

	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify mines are NOT revealed
	mineCoords := []game.Coordinate{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 2, Y: 2},
		{X: 0, Y: 4},
		{X: 4, Y: 4},
	}
	for _, mc := range mineCoords {
		if loadedState.IsRevealed(mc.X, mc.Y) {
			t.Errorf("Mine at (%d,%d) should NOT be revealed after BFS", mc.X, mc.Y)
		}
	}

	// Verify clicked cell IS revealed
	if !loadedState.IsRevealed(1, 1) {
		t.Error("Clicked cell (1,1) should be revealed")
	}
}

func TestGameHandlers_BFSDoesNotTransformMinesIntoHints(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()

	// Create a 4x4 grid with a mine in the middle
	// . . . .
	// . M . .
	// . . . .
	// . . . .
	state := game.NewGameState(4, 12345)
	state.SetMine(1, 1)
	_ = store.Save(ctx, state)

	// Record mine positions before BFS
	minesBefore := make(map[string]bool)
	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			if state.IsMine(x, y) {
				minesBefore[fmt.Sprintf("%d,%d", x, y)] = true
			}
		}
	}

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	// Click on empty cell at (3,3) - far from mine, should trigger BFS
	coords := game.Coordinate{X: 3, Y: 3}
	_, err := handlers.HandleEmptyCell(ctx, state, coords)
	if err != nil {
		t.Fatalf("HandleEmptyCell returned error: %v", err)
	}

	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify mines are still mines after BFS (not transformed into hints)
	minesAfter := make(map[string]bool)
	for x := 0; x < loadedState.Size; x++ {
		for y := 0; y < loadedState.Size; y++ {
			if loadedState.IsMine(x, y) {
				minesAfter[fmt.Sprintf("%d,%d", x, y)] = true
			}
		}
	}

	// Compare mine maps
	if len(minesBefore) != len(minesAfter) {
		t.Errorf("Mine count changed: before=%d, after=%d", len(minesBefore), len(minesAfter))
	}

	for coord := range minesBefore {
		if !minesAfter[coord] {
			t.Errorf("Mine at %s was removed/transformed after BFS", coord)
		}
	}

	// Explicitly check the mine is still a mine
	if !loadedState.IsMine(1, 1) {
		t.Error("Mine at (1,1) should still be a mine after BFS")
	}
}

func TestGameHandlers_BFSLargeGrid(t *testing.T) {
	store := game.NewMemoryStore()

	// Create a 10x10 grid with mines only on the edges
	// This tests BFS on a larger grid with a big empty center
	state := game.NewGameState(10, 12345)

	// Place mines around the perimeter
	for i := 0; i < 10; i++ {
		state.SetMine(i, 0) // top row
		state.SetMine(i, 9) // bottom row
		state.SetMine(0, i) // left column
		state.SetMine(9, i) // right column
	}

	handlers := NewGameHandlers(nil, store, GameHandlersConfig{Namespace: testNamespace})

	// Click in the center at (5,5) - should propagate through the interior
	start := game.Coordinate{X: 5, Y: 5}
	empty, boundary := handlers.bfsPropagation(state, start)

	// The interior should have empty cells (8x8 inner area minus boundary)
	// Interior is cells from (1,1) to (8,8)
	if len(empty) == 0 {
		t.Error("Expected empty cells in the interior of large grid")
	}

	// Boundary cells should be adjacent to the perimeter mines
	if len(boundary) == 0 {
		t.Error("Expected boundary cells adjacent to perimeter mines")
	}

	// Verify no mine coordinates appear in empty or boundary lists
	for _, e := range empty {
		if state.IsMine(e.X, e.Y) {
			t.Errorf("Mine at (%d,%d) incorrectly included in empty cells", e.X, e.Y)
		}
	}
	for _, b := range boundary {
		if state.IsMine(b.X, b.Y) {
			t.Errorf("Mine at (%d,%d) incorrectly included in boundary cells", b.X, b.Y)
		}
	}

	// The total revealed area should cover the 8x8 interior (64 cells)
	totalRevealed := len(empty) + len(boundary)
	expectedInterior := 64 // 8x8 interior
	if totalRevealed != expectedInterior {
		t.Errorf("Expected %d interior cells, got %d (empty=%d, boundary=%d)",
			expectedInterior, totalRevealed, len(empty), len(boundary))
	}
}

func TestGameHandlers_HandleVictory(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	// Create a minimal grid where only one safe cell exists
	state := game.NewGameState(2, 12345)
	state.SetMine(0, 0)
	state.SetMine(0, 1)
	state.SetMine(1, 0)
	// Only (1,1) is safe
	_ = store.Save(ctx, state)

	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	// Reveal the only safe cell - should trigger victory
	coords := game.Coordinate{X: 1, Y: 1}
	hintValue := state.AdjacentMines(1, 1)
	// This will reveal the cell and check victory

	_, err := handlers.HandleHintCell(ctx, state, coords, hintValue)
	if err != nil {
		t.Fatalf("HandleHintCell returned error: %v", err)
	}

	// Check state was updated to won
	loadedState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if loadedState.Status != game.StatusWon {
		t.Errorf("expected status %s, got %s", game.StatusWon, loadedState.Status)
	}

	// Check victory pod was created
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "victory", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Victory pod was not created: %v", err)
	}
}

func TestGameHandlers_WipeGamePods(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	// Create some game pods and hint pods
	gamePod1 := createTestPod("pod-0-0", testNamespace)
	gamePod2 := createTestPod("pod-1-1", testNamespace)
	hintPod := createTestPod("hint-2-2", testNamespace)
	hintPod.Name = "hint-2-2"
	// Non-game pod that should not be deleted
	otherPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: testNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:latest"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gamePod1, gamePod2, hintPod, otherPod).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	err := handlers.wipeGamePods(ctx)
	if err != nil {
		t.Fatalf("wipeGamePods returned error: %v", err)
	}

	// Game pods should be deleted
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "pod-0-0", Namespace: testNamespace}, &pod)
	if err == nil {
		t.Error("expected pod-0-0 to be deleted")
	}

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "pod-1-1", Namespace: testNamespace}, &pod)
	if err == nil {
		t.Error("expected pod-1-1 to be deleted")
	}

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "hint-2-2", Namespace: testNamespace}, &pod)
	if err == nil {
		t.Error("expected hint-2-2 to be deleted")
	}

	// Non-game pod should still exist
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "nginx", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Error("expected nginx pod to still exist")
	}
}

func TestNewGameController(t *testing.T) {
	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := game.NewMemoryStore()

	config := GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	}

	controller := NewGameController(fakeClient, config)

	if controller == nil {
		t.Fatal("expected controller to be created")
	}
	if controller.Namespace != testNamespace {
		t.Errorf("expected namespace %q, got %q", testNamespace, controller.Namespace)
	}
	if controller.Store != store {
		t.Error("expected store to be set")
	}
	if controller.Handlers == nil {
		t.Error("expected handlers to be set")
	}
}

func TestGameHandlers_SpawnHintPod(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	coords := game.Coordinate{X: 5, Y: 7}
	hintValue := 3

	err := handlers.spawnHintPod(ctx, coords, hintValue)
	if err != nil {
		t.Fatalf("spawnHintPod returned error: %v", err)
	}

	// Verify pod was created with correct properties
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "hint-5-7", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Failed to get hint pod: %v", err)
	}

	// Check labels
	if pod.Labels[LabelApp] != "podsweeper" {
		t.Errorf("expected app label 'podsweeper', got %q", pod.Labels[LabelApp])
	}
	if pod.Labels[LabelComponent] != "hint" {
		t.Errorf("expected component label 'hint', got %q", pod.Labels[LabelComponent])
	}
	if pod.Labels[LabelCoordX] != "5" {
		t.Errorf("expected x label '5', got %q", pod.Labels[LabelCoordX])
	}
	if pod.Labels[LabelCoordY] != "7" {
		t.Errorf("expected y label '7', got %q", pod.Labels[LabelCoordY])
	}

	// Check annotations
	if pod.Annotations[AnnotationHint] != "3" {
		t.Errorf("expected hint annotation '3', got %q", pod.Annotations[AnnotationHint])
	}
	if pod.Annotations[AnnotationPort] != "8080" {
		t.Errorf("expected port annotation '8080', got %q", pod.Annotations[AnnotationPort])
	}

	// Check container
	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(pod.Spec.Containers))
	}
	container := pod.Spec.Containers[0]
	if container.Name != "hint" {
		t.Errorf("expected container name 'hint', got %q", container.Name)
	}
	if container.Image != DefaultHintAgentImage {
		t.Errorf("expected image %q, got %q", DefaultHintAgentImage, container.Image)
	}
}

func TestGameHandlers_SpawnExplosionPod(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	coords := game.Coordinate{X: 3, Y: 5}

	err := handlers.spawnExplosionPod(ctx, coords)
	if err != nil {
		t.Fatalf("spawnExplosionPod returned error: %v", err)
	}

	// Verify pod was created
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "explosion", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Failed to get explosion pod: %v", err)
	}

	// Check labels
	if pod.Labels[LabelApp] != "podsweeper" {
		t.Errorf("expected app label 'podsweeper', got %q", pod.Labels[LabelApp])
	}
	if pod.Labels[LabelComponent] != "explosion" {
		t.Errorf("expected component label 'explosion', got %q", pod.Labels[LabelComponent])
	}
}

func TestGameHandlers_SpawnVictoryPod(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	state := createTestGameState(8)
	state.Level = 5
	state.Clicks = 42

	err := handlers.spawnVictoryPod(ctx, state)
	if err != nil {
		t.Fatalf("spawnVictoryPod returned error: %v", err)
	}

	// Verify pod was created
	var pod corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "victory", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Fatalf("Failed to get victory pod: %v", err)
	}

	// Check labels
	if pod.Labels[LabelApp] != "podsweeper" {
		t.Errorf("expected app label 'podsweeper', got %q", pod.Labels[LabelApp])
	}
	if pod.Labels[LabelComponent] != "victory" {
		t.Errorf("expected component label 'victory', got %q", pod.Labels[LabelComponent])
	}
}

func TestGameHandlers_DeletePod(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	pod := createTestPod("pod-2-3", testNamespace)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	coords := game.Coordinate{X: 2, Y: 3}

	err := handlers.deletePod(ctx, coords)
	if err != nil {
		t.Fatalf("deletePod returned error: %v", err)
	}

	// Verify pod was deleted
	var result corev1.Pod
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "pod-2-3", Namespace: testNamespace}, &result)
	if err == nil {
		t.Error("expected pod to be deleted")
	}
}

func TestGameHandlers_DeletePodNotFound(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, GameHandlersConfig{Namespace: testNamespace})

	coords := game.Coordinate{X: 99, Y: 99}

	// Should not return an error for non-existent pod
	err := handlers.deletePod(ctx, coords)
	if err != nil {
		t.Fatalf("deletePod should not error for non-existent pod: %v", err)
	}
}

// --- Level Transition Tests ---

func TestGameController_HandleGameRestart_LevelUp(t *testing.T) {
	// This test verifies that when the player wins, the level is incremented
	ctx := context.Background()
	scheme := newTestScheme()

	// Create a fake client with the necessary namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	store := game.NewMemoryStore()

	// Create a game state at level 0 that was won
	state := game.NewGameState(5, 12345)
	state.Level = 0
	state.Status = game.StatusWon
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save initial state: %v", err)
	}

	// Create the controller
	gc := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	// Simulate victory pod deletion (triggers handleGameRestart)
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "victory",
			Namespace: testNamespace,
		},
	}

	// The victory pod doesn't exist, so Reconcile will call handleGameRestart
	_, err := gc.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	// Load the new game state
	newState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("failed to load new state: %v", err)
	}

	// Verify level was incremented
	if newState.Level != 1 {
		t.Errorf("expected level 1 after winning level 0, got %d", newState.Level)
	}

	// Verify game is in playing status
	if newState.Status != game.StatusPlaying {
		t.Errorf("expected status 'playing', got %q", newState.Status)
	}
}

func TestGameController_HandleGameRestart_NoLevelUpOnLoss(t *testing.T) {
	// This test verifies that when the player loses, the level stays the same
	ctx := context.Background()
	scheme := newTestScheme()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	store := game.NewMemoryStore()

	// Create a game state at level 2 that was lost
	state := game.NewGameState(5, 12345)
	state.Level = 2
	state.Status = game.StatusLost
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save initial state: %v", err)
	}

	gc := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "explosion",
			Namespace: testNamespace,
		},
	}

	_, err := gc.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	newState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("failed to load new state: %v", err)
	}

	// Verify level stayed the same
	if newState.Level != 2 {
		t.Errorf("expected level 2 after losing, got %d", newState.Level)
	}

	if newState.Status != game.StatusPlaying {
		t.Errorf("expected status 'playing', got %q", newState.Status)
	}
}

func TestGameController_HandleGameRestart_MaxLevel(t *testing.T) {
	// This test verifies that winning at max level doesn't overflow
	ctx := context.Background()
	scheme := newTestScheme()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	store := game.NewMemoryStore()

	// Create a game state at max level (9) that was won
	state := game.NewGameState(20, 12345) // Tier 3 size for level 9
	state.Level = 9
	state.Status = game.StatusWon
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("failed to save initial state: %v", err)
	}

	gc := NewGameController(fakeClient, GameControllerConfig{
		Namespace: testNamespace,
		Store:     store,
	})

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "victory",
			Namespace: testNamespace,
		},
	}

	_, err := gc.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	newState, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("failed to load new state: %v", err)
	}

	// Verify level stayed at max (9) - no overflow
	if newState.Level != 9 {
		t.Errorf("expected level 9 (max) after winning at max level, got %d", newState.Level)
	}

	if newState.Status != game.StatusPlaying {
		t.Errorf("expected status 'playing', got %q", newState.Status)
	}
}
