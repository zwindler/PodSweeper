package controller

import (
	"context"
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
	if result.Requeue {
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
	if result.Requeue {
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
	if result.Requeue {
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
	if result.Requeue {
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
	if result.Requeue {
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
	if result.Requeue {
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

	handlers := NewGameHandlers(fakeClient, store, testNamespace)
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

	handlers := NewGameHandlers(fakeClient, store, testNamespace)
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

	handlers := NewGameHandlers(fakeClient, store, testNamespace)
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

	handlers := NewGameHandlers(nil, store, testNamespace)
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

	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	if container.Image != HintAgentImage {
		t.Errorf("expected image %q, got %q", HintAgentImage, container.Image)
	}
}

func TestGameHandlers_SpawnExplosionPod(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	store := game.NewMemoryStore()
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

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
	handlers := NewGameHandlers(fakeClient, store, testNamespace)

	coords := game.Coordinate{X: 99, Y: 99}

	// Should not return an error for non-existent pod
	err := handlers.deletePod(ctx, coords)
	if err != nil {
		t.Fatalf("deletePod should not error for non-existent pod: %v", err)
	}
}
