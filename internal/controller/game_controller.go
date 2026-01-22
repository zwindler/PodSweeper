// Package controller contains the Kubernetes controller logic for PodSweeper.
package controller

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/zwindler/podsweeper/pkg/game"
)

// PodNameRegex matches pod names in the format "pod-X-Y" where X and Y are integers.
var PodNameRegex = regexp.MustCompile(`^pod-(\d+)-(\d+)$`)

// HintPodNameRegex matches hint pod names in the format "hint-X-Y".
var HintPodNameRegex = regexp.MustCompile(`^hint-(\d+)-(\d+)$`)

// GameController reconciles Pod objects in the game namespace.
type GameController struct {
	client.Client
	Store     game.Store
	Namespace string
	Handlers  *GameHandlers
}

// GameControllerConfig holds configuration for the GameController.
type GameControllerConfig struct {
	Namespace string
	Store     game.Store
}

// NewGameController creates a new GameController.
func NewGameController(c client.Client, config GameControllerConfig) *GameController {
	gc := &GameController{
		Client:    c,
		Store:     config.Store,
		Namespace: config.Namespace,
	}
	gc.Handlers = NewGameHandlers(c, config.Store, config.Namespace)
	return gc
}

// Reconcile handles pod events in the game namespace.
func (r *GameController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Only process pods in our namespace
	if req.Namespace != r.Namespace {
		return ctrl.Result{}, nil
	}

	// Check if this is a game pod (pod-X-Y format)
	coords, ok := ParsePodName(req.Name)
	if !ok {
		// Not a game pod, ignore
		return ctrl.Result{}, nil
	}

	// Try to get the pod
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)

	if errors.IsNotFound(err) {
		// Pod was deleted - this is the main game action
		logger.Info("pod deleted", "name", req.Name, "x", coords.X, "y", coords.Y)
		return r.handlePodDeletion(ctx, coords)
	}

	if err != nil {
		logger.Error(err, "failed to get pod")
		return ctrl.Result{}, err
	}

	// Pod exists - check if it's being deleted (has deletion timestamp)
	if !pod.DeletionTimestamp.IsZero() {
		logger.Info("pod is being deleted", "name", req.Name)
		// Pod is terminating, we'll handle it when it's fully gone
		return ctrl.Result{}, nil
	}

	// Pod exists and is not being deleted - nothing to do
	return ctrl.Result{}, nil
}

// handlePodDeletion processes a pod deletion event (the "click").
func (r *GameController) handlePodDeletion(ctx context.Context, coords game.Coordinate) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Load current game state
	state, err := r.Store.Load(ctx)
	if err != nil {
		logger.Error(err, "failed to load game state")
		return ctrl.Result{}, err
	}

	if state == nil {
		logger.Info("no active game, ignoring deletion")
		return ctrl.Result{}, nil
	}

	// Check if game is already over
	if state.Status != game.StatusPlaying {
		logger.Info("game already ended", "status", state.Status)
		return ctrl.Result{}, nil
	}

	// Check if cell was already revealed
	if state.IsRevealed(coords.X, coords.Y) {
		logger.Info("cell already revealed", "coords", coords)
		return ctrl.Result{}, nil
	}

	// Determine what type of cell was clicked
	if state.IsMine(coords.X, coords.Y) {
		// BOOM! Game over
		logger.Info("mine hit!", "coords", coords)
		return r.Handlers.HandleMineHit(ctx, state, coords)
	}

	// Safe cell - check adjacent mines
	adjacentMines := state.AdjacentMines(coords.X, coords.Y)

	if adjacentMines > 0 {
		// Cell with adjacent mines - create hint pod
		logger.Info("safe cell with hints", "coords", coords, "adjacent", adjacentMines)
		return r.Handlers.HandleHintCell(ctx, state, coords, adjacentMines)
	}

	// Empty cell (no adjacent mines) - trigger BFS propagation
	logger.Info("empty cell, triggering propagation", "coords", coords)
	return r.Handlers.HandleEmptyCell(ctx, state, coords)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			// Only watch pods in our namespace
			return object.GetNamespace() == r.Namespace
		})).
		Complete(r)
}

// ParsePodName extracts coordinates from a pod name like "pod-3-5".
// Returns the coordinate and true if successful, or zero coordinate and false if not a game pod.
func ParsePodName(name string) (game.Coordinate, bool) {
	matches := PodNameRegex.FindStringSubmatch(name)
	if matches == nil {
		return game.Coordinate{}, false
	}

	x, err1 := strconv.Atoi(matches[1])
	y, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return game.Coordinate{}, false
	}

	return game.Coordinate{X: x, Y: y}, true
}

// ParseHintPodName extracts coordinates from a hint pod name like "hint-3-5".
func ParseHintPodName(name string) (game.Coordinate, bool) {
	matches := HintPodNameRegex.FindStringSubmatch(name)
	if matches == nil {
		return game.Coordinate{}, false
	}

	x, err1 := strconv.Atoi(matches[1])
	y, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return game.Coordinate{}, false
	}

	return game.Coordinate{X: x, Y: y}, true
}

// IsPodName checks if a name matches the game pod pattern.
func IsPodName(name string) bool {
	return PodNameRegex.MatchString(name)
}

// IsHintPodName checks if a name matches the hint pod pattern.
func IsHintPodName(name string) bool {
	return HintPodNameRegex.MatchString(name)
}

// GeneratePodName creates a pod name from coordinates.
func GeneratePodName(x, y int) string {
	return fmt.Sprintf("pod-%d-%d", x, y)
}

// GenerateHintPodName creates a hint pod name from coordinates.
func GenerateHintPodName(x, y int) string {
	return fmt.Sprintf("hint-%d-%d", x, y)
}
