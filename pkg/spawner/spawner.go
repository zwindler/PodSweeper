// Package spawner creates the initial game pods when a new game starts.
package spawner

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/zwindler/podsweeper/pkg/game"
)

const (
	// CellImage is the default container image for game cell pods.
	// These pods just sit there waiting to be deleted by the player.
	CellImage = "busybox:latest"

	// LabelApp is the app label for game pods.
	LabelApp = "app.kubernetes.io/name"

	// LabelComponent is the component label.
	LabelComponent = "app.kubernetes.io/component"

	// LabelCoordX is the X coordinate label.
	LabelCoordX = "podsweeper.io/x"

	// LabelCoordY is the Y coordinate label.
	LabelCoordY = "podsweeper.io/y"

	// LabelGameID is the game session identifier.
	LabelGameID = "podsweeper.io/game-id"

	// DefaultBatchSize is the default number of pods to create in parallel.
	DefaultBatchSize = 10

	// DefaultRetryAttempts is the default number of retry attempts for pod creation.
	DefaultRetryAttempts = 3

	// DefaultRetryDelay is the default delay between retries.
	DefaultRetryDelay = 500 * time.Millisecond
)

// GridSpawner creates game pods for a new game.
type GridSpawner struct {
	client        client.Client
	namespace     string
	cellImage     string
	batchSize     int
	retryAttempts int
	retryDelay    time.Duration
}

// GridSpawnerConfig holds configuration for the GridSpawner.
type GridSpawnerConfig struct {
	Namespace     string
	CellImage     string
	BatchSize     int
	RetryAttempts int
	RetryDelay    time.Duration
}

// SpawnResult contains the result of a spawn operation.
type SpawnResult struct {
	TotalPods    int
	CreatedPods  int
	FailedPods   int
	FailedCoords []game.Coordinate
	Duration     time.Duration
}

// NewGridSpawner creates a new GridSpawner.
func NewGridSpawner(c client.Client, config GridSpawnerConfig) *GridSpawner {
	if config.CellImage == "" {
		config.CellImage = CellImage
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = DefaultRetryAttempts
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.Namespace == "" {
		config.Namespace = game.DefaultNamespace
	}

	return &GridSpawner{
		client:        c,
		namespace:     config.Namespace,
		cellImage:     config.CellImage,
		batchSize:     config.BatchSize,
		retryAttempts: config.RetryAttempts,
		retryDelay:    config.RetryDelay,
	}
}

// SpawnGrid creates all game pods for the given game state.
// It creates pods in batches to avoid overwhelming the API server.
func (s *GridSpawner) SpawnGrid(ctx context.Context, state *game.GameState) (*SpawnResult, error) {
	logger := log.FromContext(ctx)
	start := time.Now()

	result := &SpawnResult{
		TotalPods: state.Size * state.Size,
	}

	// Generate all coordinates
	coords := make([]game.Coordinate, 0, result.TotalPods)
	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			coords = append(coords, game.Coordinate{X: x, Y: y})
		}
	}

	// Create pods in batches
	gameID := fmt.Sprintf("%d-%d", state.Seed, state.StartedAt.Unix())

	for i := 0; i < len(coords); i += s.batchSize {
		end := i + s.batchSize
		if end > len(coords) {
			end = len(coords)
		}
		batch := coords[i:end]

		logger.Info("spawning batch", "start", i, "end", end, "total", len(coords))

		for _, coord := range batch {
			if err := s.createPodWithRetry(ctx, coord, gameID); err != nil {
				logger.Error(err, "failed to create pod", "coord", coord)
				result.FailedPods++
				result.FailedCoords = append(result.FailedCoords, coord)
			} else {
				result.CreatedPods++
			}
		}
	}

	result.Duration = time.Since(start)

	logger.Info("grid spawn complete",
		"created", result.CreatedPods,
		"failed", result.FailedPods,
		"duration", result.Duration)

	if result.FailedPods > 0 {
		return result, fmt.Errorf("failed to create %d pods", result.FailedPods)
	}

	return result, nil
}

// createPodWithRetry creates a single pod with retry logic.
func (s *GridSpawner) createPodWithRetry(ctx context.Context, coord game.Coordinate, gameID string) error {
	var lastErr error

	for attempt := 0; attempt < s.retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.retryDelay):
			}
		}

		pod := s.buildCellPod(coord, gameID)
		if err := s.client.Create(ctx, pod); err != nil {
			if errors.IsAlreadyExists(err) {
				// Pod already exists, that's fine
				return nil
			}
			lastErr = err
			continue
		}
		return nil
	}

	return fmt.Errorf("after %d attempts: %w", s.retryAttempts, lastErr)
}

// buildCellPod creates the pod spec for a game cell.
func (s *GridSpawner) buildCellPod(coord game.Coordinate, gameID string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coord.PodName(),
			Namespace: s.namespace,
			Labels: map[string]string{
				LabelApp:       "podsweeper",
				LabelComponent: "cell",
				LabelCoordX:    fmt.Sprintf("%d", coord.X),
				LabelCoordY:    fmt.Sprintf("%d", coord.Y),
				LabelGameID:    gameID,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "cell",
					Image: s.cellImage,
					// The pod just sleeps - it's waiting to be deleted
					Command: []string{"sh", "-c", "echo 'PodSweeper cell ready' && sleep infinity"},
				},
			},
		},
	}
}

// CleanupGrid removes all game pods from the namespace.
func (s *GridSpawner) CleanupGrid(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// List all pods with the podsweeper app label
	podList := &corev1.PodList{}
	if err := s.client.List(ctx, podList,
		client.InNamespace(s.namespace),
		client.MatchingLabels{LabelApp: "podsweeper"},
	); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	logger.Info("cleaning up game pods", "count", len(podList.Items))

	var lastErr error
	deleted := 0

	for i := range podList.Items {
		pod := &podList.Items[i]
		if err := s.client.Delete(ctx, pod); err != nil {
			if !errors.IsNotFound(err) {
				logger.Error(err, "failed to delete pod", "name", pod.Name)
				lastErr = err
			}
		} else {
			deleted++
		}
	}

	logger.Info("cleanup complete", "deleted", deleted)

	return lastErr
}

// WaitForPodsReady waits for all game pods to be in Running phase.
func (s *GridSpawner) WaitForPodsReady(ctx context.Context, expectedCount int, timeout time.Duration) error {
	logger := log.FromContext(ctx)

	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		podList := &corev1.PodList{}
		if err := s.client.List(ctx, podList,
			client.InNamespace(s.namespace),
			client.MatchingLabels{
				LabelApp:       "podsweeper",
				LabelComponent: "cell",
			},
		); err != nil {
			return false, err
		}

		runningCount := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodRunning {
				runningCount++
			}
		}

		logger.V(1).Info("waiting for pods", "running", runningCount, "expected", expectedCount)

		return runningCount >= expectedCount, nil
	})
}

// Namespace returns the namespace where pods are spawned.
func (s *GridSpawner) Namespace() string {
	return s.namespace
}
