package controller

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/zwindler/podsweeper/pkg/game"
)

const (
	// HintAgentImage is the container image for hint pods.
	// This should be configurable in production.
	HintAgentImage = "ghcr.io/zwindler/podsweeper-hint-agent:latest"

	// ExplosionImage is the container image for the explosion pod.
	ExplosionImage = "busybox:latest"

	// VictoryImage is the container image for the victory pod.
	VictoryImage = "busybox:latest"

	// LabelApp is the app label for game pods.
	LabelApp = "app.kubernetes.io/name"

	// LabelComponent is the component label.
	LabelComponent = "app.kubernetes.io/component"

	// LabelCoordX is the X coordinate label.
	LabelCoordX = "podsweeper.io/x"

	// LabelCoordY is the Y coordinate label.
	LabelCoordY = "podsweeper.io/y"

	// AnnotationHint is the annotation storing the hint value.
	AnnotationHint = "podsweeper.io/hint"

	// AnnotationPort is the annotation storing the hint port (for Level 7).
	AnnotationPort = "podsweeper.io/port"
)

// GameHandlers contains the logic for handling game events.
type GameHandlers struct {
	client    client.Client
	store     game.Store
	namespace string
}

// NewGameHandlers creates a new GameHandlers instance.
func NewGameHandlers(c client.Client, store game.Store, namespace string) *GameHandlers {
	return &GameHandlers{
		client:    c,
		store:     store,
		namespace: namespace,
	}
}

// HandleMineHit processes a mine being clicked - game over!
func (h *GameHandlers) HandleMineHit(ctx context.Context, state *game.GameState, coords game.Coordinate) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Mark game as lost
	state.Reveal(coords.X, coords.Y)
	state.SetLost()

	// Save state
	if err := h.store.Save(ctx, state); err != nil {
		logger.Error(err, "failed to save game state after mine hit")
		return ctrl.Result{}, err
	}

	// Wipe the namespace (delete all game pods)
	if err := h.wipeGamePods(ctx); err != nil {
		logger.Error(err, "failed to wipe game pods")
		return ctrl.Result{}, err
	}

	// Spawn explosion pod
	if err := h.spawnExplosionPod(ctx, coords); err != nil {
		logger.Error(err, "failed to spawn explosion pod")
		return ctrl.Result{}, err
	}

	logger.Info("game over - mine hit", "coords", coords)
	return ctrl.Result{}, nil
}

// HandleHintCell processes a safe cell with adjacent mines.
func (h *GameHandlers) HandleHintCell(ctx context.Context, state *game.GameState, coords game.Coordinate, hintValue int) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Mark cell as revealed
	state.Reveal(coords.X, coords.Y)
	state.AddHintCell(coords.X, coords.Y)

	// Create hint pod
	if err := h.spawnHintPod(ctx, coords, hintValue); err != nil {
		logger.Error(err, "failed to spawn hint pod")
		return ctrl.Result{}, err
	}

	// Check for victory
	if state.CheckVictory() {
		return h.handleVictory(ctx, state)
	}

	// Save state
	if err := h.store.Save(ctx, state); err != nil {
		logger.Error(err, "failed to save game state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// HandleEmptyCell processes an empty cell (no adjacent mines) with BFS propagation.
func (h *GameHandlers) HandleEmptyCell(ctx context.Context, state *game.GameState, coords game.Coordinate) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// BFS to find all connected empty cells and boundary hint cells
	toReveal, boundaryHints := h.bfsPropagation(state, coords)

	logger.Info("BFS propagation complete",
		"emptyCount", len(toReveal),
		"boundaryCount", len(boundaryHints))

	// Reveal all empty cells
	for _, c := range toReveal {
		state.Reveal(c.X, c.Y)
	}

	// Delete pods for empty cells (they don't get hint pods)
	for _, c := range toReveal {
		if err := h.deletePod(ctx, c); err != nil {
			logger.Error(err, "failed to delete pod during propagation", "coords", c)
			// Continue with other deletions
		}
	}

	// Create hint pods for boundary cells
	for _, c := range boundaryHints {
		hintValue := state.AdjacentMines(c.X, c.Y)
		state.Reveal(c.X, c.Y)
		state.AddHintCell(c.X, c.Y)

		// Delete the original pod first
		if err := h.deletePod(ctx, c); err != nil {
			logger.Error(err, "failed to delete pod for hint", "coords", c)
		}

		// Spawn hint pod
		if err := h.spawnHintPod(ctx, c, hintValue); err != nil {
			logger.Error(err, "failed to spawn hint pod", "coords", c)
		}
	}

	// Check for victory
	if state.CheckVictory() {
		return h.handleVictory(ctx, state)
	}

	// Save state
	if err := h.store.Save(ctx, state); err != nil {
		logger.Error(err, "failed to save game state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// bfsPropagation performs BFS from the starting coordinate to find all connected
// empty cells and the boundary cells that have adjacent mines.
func (h *GameHandlers) bfsPropagation(state *game.GameState, start game.Coordinate) (empty []game.Coordinate, boundary []game.Coordinate) {
	visited := make(map[string]bool)
	queue := []game.Coordinate{start}

	key := func(c game.Coordinate) string {
		return fmt.Sprintf("%d,%d", c.X, c.Y)
	}

	visited[key(start)] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		adjacentMines := state.AdjacentMines(current.X, current.Y)

		if adjacentMines == 0 {
			// Empty cell - add to empty list and explore neighbors
			empty = append(empty, current)

			for _, neighbor := range state.GetNeighbors(current.X, current.Y) {
				nKey := key(neighbor)
				if !visited[nKey] && !state.IsRevealed(neighbor.X, neighbor.Y) && !state.IsMine(neighbor.X, neighbor.Y) {
					visited[nKey] = true
					queue = append(queue, neighbor)
				}
			}
		} else {
			// Boundary cell with adjacent mines - add to boundary list
			boundary = append(boundary, current)
		}
	}

	return empty, boundary
}

// handleVictory processes a victory condition.
func (h *GameHandlers) handleVictory(ctx context.Context, state *game.GameState) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	state.SetWon()

	// Save state
	if err := h.store.Save(ctx, state); err != nil {
		logger.Error(err, "failed to save game state after victory")
		return ctrl.Result{}, err
	}

	// Spawn victory pod
	if err := h.spawnVictoryPod(ctx, state); err != nil {
		logger.Error(err, "failed to spawn victory pod")
		return ctrl.Result{}, err
	}

	logger.Info("victory!", "clicks", state.Clicks, "level", state.Level)
	return ctrl.Result{}, nil
}

// spawnHintPod creates a hint pod at the given coordinates.
func (h *GameHandlers) spawnHintPod(ctx context.Context, coords game.Coordinate, hintValue int) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coords.HintPodName(),
			Namespace: h.namespace,
			Labels: map[string]string{
				LabelApp:       "podsweeper",
				LabelComponent: "hint",
				LabelCoordX:    strconv.Itoa(coords.X),
				LabelCoordY:    strconv.Itoa(coords.Y),
			},
			Annotations: map[string]string{
				AnnotationHint: strconv.Itoa(hintValue),
				AnnotationPort: "8080",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "hint",
					Image: HintAgentImage,
					Env: []corev1.EnvVar{
						{Name: "HINT_VALUE", Value: strconv.Itoa(hintValue)},
						{Name: "POD_X", Value: strconv.Itoa(coords.X)},
						{Name: "POD_Y", Value: strconv.Itoa(coords.Y)},
						{Name: "PORT", Value: "8080"},
					},
					Ports: []corev1.ContainerPort{
						{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
					},
				},
			},
		},
	}

	return h.client.Create(ctx, pod)
}

// spawnExplosionPod creates the explosion pod after a mine is hit.
func (h *GameHandlers) spawnExplosionPod(ctx context.Context, coords game.Coordinate) error {
	explosionASCII := `
    _ ._  _ , _ ._
  (_ ' ( \` + "`" + `)_  .__)
( (  (    )   \` + "`" + `) ) _)
(__ (_   (_ . _) _) ,__)
    \` + "`" + `~~\` + "`" + `\ ' . /\` + "`" + `~~\` + "`" + `
         ;   ;
         /   \
_________/_ __ \_________

    💥 BOOM! 💥
    
  You hit a mine at (%d, %d)!
  
     GAME OVER
`
	message := fmt.Sprintf(explosionASCII, coords.X, coords.Y)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "explosion",
			Namespace: h.namespace,
			Labels: map[string]string{
				LabelApp:       "podsweeper",
				LabelComponent: "explosion",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "explosion",
					Image:   ExplosionImage,
					Command: []string{"sh", "-c", fmt.Sprintf("echo '%s' && sleep infinity", message)},
				},
			},
		},
	}

	return h.client.Create(ctx, pod)
}

// spawnVictoryPod creates the victory pod after winning.
func (h *GameHandlers) spawnVictoryPod(ctx context.Context, state *game.GameState) error {
	victoryASCII := `
    ___________
   '._==_==_=_.'
   .-\:      /-.
  | (|:.     |) |
   '-|:.     |-'
     \::.    /
      '::. .'
        ) (
      _.' '._
     \` + "`" + `"""""""\` + "`" + `

  🎉 VICTORY! 🎉
  
  Level: %d
  Clicks: %d
  Mines: %d
  
  Congratulations!
`
	message := fmt.Sprintf(victoryASCII, state.Level, state.Clicks, state.MineCount)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "victory",
			Namespace: h.namespace,
			Labels: map[string]string{
				LabelApp:       "podsweeper",
				LabelComponent: "victory",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "victory",
					Image:   VictoryImage,
					Command: []string{"sh", "-c", fmt.Sprintf("echo '%s' && sleep infinity", message)},
				},
			},
		},
	}

	return h.client.Create(ctx, pod)
}

// deletePod deletes a game pod at the given coordinates.
func (h *GameHandlers) deletePod(ctx context.Context, coords game.Coordinate) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coords.PodName(),
			Namespace: h.namespace,
		},
	}

	return client.IgnoreNotFound(h.client.Delete(ctx, pod))
}

// wipeGamePods deletes all game pods (pod-X-Y pattern) from the namespace.
func (h *GameHandlers) wipeGamePods(ctx context.Context) error {
	podList := &corev1.PodList{}
	if err := h.client.List(ctx, podList, client.InNamespace(h.namespace)); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		// Only delete game pods (pod-X-Y or hint-X-Y)
		if IsPodName(pod.Name) || IsHintPodName(pod.Name) {
			if err := h.client.Delete(ctx, &pod); err != nil {
				// Log but continue with other deletions
				log.FromContext(ctx).Error(err, "failed to delete pod", "name", pod.Name)
			}
		}
	}

	return nil
}
