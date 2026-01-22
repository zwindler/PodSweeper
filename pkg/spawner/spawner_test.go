package spawner

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zwindler/podsweeper/pkg/game"
)

const testNamespace = "podsweeper-game"

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func TestNewGridSpawner(t *testing.T) {
	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name           string
		config         GridSpawnerConfig
		wantNamespace  string
		wantCellImage  string
		wantBatchSize  int
		wantRetryCount int
	}{
		{
			name:           "defaults",
			config:         GridSpawnerConfig{},
			wantNamespace:  game.DefaultNamespace,
			wantCellImage:  CellImage,
			wantBatchSize:  DefaultBatchSize,
			wantRetryCount: DefaultRetryAttempts,
		},
		{
			name: "custom config",
			config: GridSpawnerConfig{
				Namespace:     "custom-ns",
				CellImage:     "custom-image:v1",
				BatchSize:     5,
				RetryAttempts: 5,
			},
			wantNamespace:  "custom-ns",
			wantCellImage:  "custom-image:v1",
			wantBatchSize:  5,
			wantRetryCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spawner := NewGridSpawner(fakeClient, tt.config)

			if spawner.namespace != tt.wantNamespace {
				t.Errorf("namespace = %q, want %q", spawner.namespace, tt.wantNamespace)
			}
			if spawner.cellImage != tt.wantCellImage {
				t.Errorf("cellImage = %q, want %q", spawner.cellImage, tt.wantCellImage)
			}
			if spawner.batchSize != tt.wantBatchSize {
				t.Errorf("batchSize = %d, want %d", spawner.batchSize, tt.wantBatchSize)
			}
			if spawner.retryAttempts != tt.wantRetryCount {
				t.Errorf("retryAttempts = %d, want %d", spawner.retryAttempts, tt.wantRetryCount)
			}
		})
	}
}

func TestGridSpawner_SpawnGrid(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
		BatchSize: 5, // Small batch for testing
	})

	// Create a 3x3 game state
	state := game.NewGameState(3, 12345)
	state.SetMine(1, 1)

	result, err := spawner.SpawnGrid(ctx, state)
	if err != nil {
		t.Fatalf("SpawnGrid returned error: %v", err)
	}

	// Check result
	if result.TotalPods != 9 {
		t.Errorf("TotalPods = %d, want 9", result.TotalPods)
	}
	if result.CreatedPods != 9 {
		t.Errorf("CreatedPods = %d, want 9", result.CreatedPods)
	}
	if result.FailedPods != 0 {
		t.Errorf("FailedPods = %d, want 0", result.FailedPods)
	}

	// Verify pods were created
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			podName := game.Coordinate{X: x, Y: y}.PodName()
			var pod corev1.Pod
			err := fakeClient.Get(ctx, types.NamespacedName{
				Name:      podName,
				Namespace: testNamespace,
			}, &pod)
			if err != nil {
				t.Errorf("Pod %s was not created: %v", podName, err)
				continue
			}

			// Check labels
			if pod.Labels[LabelApp] != "podsweeper" {
				t.Errorf("Pod %s app label = %q, want 'podsweeper'", podName, pod.Labels[LabelApp])
			}
			if pod.Labels[LabelComponent] != "cell" {
				t.Errorf("Pod %s component label = %q, want 'cell'", podName, pod.Labels[LabelComponent])
			}
		}
	}
}

func TestGridSpawner_SpawnGridLarge(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
		BatchSize: 25,
	})

	// Create an 8x8 grid (64 pods)
	state := game.NewGameState(8, 12345)

	result, err := spawner.SpawnGrid(ctx, state)
	if err != nil {
		t.Fatalf("SpawnGrid returned error: %v", err)
	}

	if result.TotalPods != 64 {
		t.Errorf("TotalPods = %d, want 64", result.TotalPods)
	}
	if result.CreatedPods != 64 {
		t.Errorf("CreatedPods = %d, want 64", result.CreatedPods)
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestGridSpawner_BuildCellPod(t *testing.T) {
	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
		CellImage: "custom:latest",
	})

	coord := game.Coordinate{X: 5, Y: 7}
	gameID := "12345-1234567890"

	pod := spawner.buildCellPod(coord, gameID)

	// Check name
	if pod.Name != "pod-5-7" {
		t.Errorf("pod.Name = %q, want 'pod-5-7'", pod.Name)
	}

	// Check namespace
	if pod.Namespace != testNamespace {
		t.Errorf("pod.Namespace = %q, want %q", pod.Namespace, testNamespace)
	}

	// Check labels
	expectedLabels := map[string]string{
		LabelApp:       "podsweeper",
		LabelComponent: "cell",
		LabelCoordX:    "5",
		LabelCoordY:    "7",
		LabelGameID:    gameID,
	}
	for k, v := range expectedLabels {
		if pod.Labels[k] != v {
			t.Errorf("pod.Labels[%q] = %q, want %q", k, pod.Labels[k], v)
		}
	}

	// Check container
	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(pod.Spec.Containers))
	}

	container := pod.Spec.Containers[0]
	if container.Name != "cell" {
		t.Errorf("container.Name = %q, want 'cell'", container.Name)
	}
	if container.Image != "custom:latest" {
		t.Errorf("container.Image = %q, want 'custom:latest'", container.Image)
	}

	// Check restart policy
	if pod.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("RestartPolicy = %q, want Never", pod.Spec.RestartPolicy)
	}
}

func TestGridSpawner_CleanupGrid(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	// Create some existing pods
	existingPods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-0-0",
				Namespace: testNamespace,
				Labels:    map[string]string{LabelApp: "podsweeper"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "c", Image: "i"}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1-1",
				Namespace: testNamespace,
				Labels:    map[string]string{LabelApp: "podsweeper"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "c", Image: "i"}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-app",
				Namespace: testNamespace,
				Labels:    map[string]string{LabelApp: "other"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "c", Image: "i"}},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&existingPods[0], &existingPods[1], &existingPods[2]).
		Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
	})

	err := spawner.CleanupGrid(ctx)
	if err != nil {
		t.Fatalf("CleanupGrid returned error: %v", err)
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

	// Other app pod should still exist
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "other-app", Namespace: testNamespace}, &pod)
	if err != nil {
		t.Error("expected other-app pod to still exist")
	}
}

func TestGridSpawner_CleanupEmptyNamespace(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
	})

	// Should not error on empty namespace
	err := spawner.CleanupGrid(ctx)
	if err != nil {
		t.Fatalf("CleanupGrid should not error on empty namespace: %v", err)
	}
}

func TestGridSpawner_SpawnGridIdempotent(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: testNamespace,
		BatchSize: 5,
	})

	state := game.NewGameState(3, 12345)

	// First spawn
	result1, err := spawner.SpawnGrid(ctx, state)
	if err != nil {
		t.Fatalf("First SpawnGrid returned error: %v", err)
	}
	if result1.CreatedPods != 9 {
		t.Errorf("First spawn CreatedPods = %d, want 9", result1.CreatedPods)
	}

	// Second spawn (should handle existing pods gracefully)
	result2, err := spawner.SpawnGrid(ctx, state)
	if err != nil {
		t.Fatalf("Second SpawnGrid returned error: %v", err)
	}
	// All pods already exist, so creation returns success (idempotent)
	if result2.FailedPods != 0 {
		t.Errorf("Second spawn FailedPods = %d, want 0", result2.FailedPods)
	}
}

func TestGridSpawner_Namespace(t *testing.T) {
	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	spawner := NewGridSpawner(fakeClient, GridSpawnerConfig{
		Namespace: "test-ns",
	})

	if spawner.Namespace() != "test-ns" {
		t.Errorf("Namespace() = %q, want 'test-ns'", spawner.Namespace())
	}
}

func TestSpawnResult(t *testing.T) {
	result := &SpawnResult{
		TotalPods:    100,
		CreatedPods:  98,
		FailedPods:   2,
		FailedCoords: []game.Coordinate{{X: 1, Y: 1}, {X: 2, Y: 2}},
		Duration:     5 * time.Second,
	}

	if result.TotalPods != 100 {
		t.Errorf("TotalPods = %d, want 100", result.TotalPods)
	}
	if result.CreatedPods != 98 {
		t.Errorf("CreatedPods = %d, want 98", result.CreatedPods)
	}
	if result.FailedPods != 2 {
		t.Errorf("FailedPods = %d, want 2", result.FailedPods)
	}
	if len(result.FailedCoords) != 2 {
		t.Errorf("len(FailedCoords) = %d, want 2", len(result.FailedCoords))
	}
}
