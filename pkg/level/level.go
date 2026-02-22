// Package level handles level-specific resources and configurations for PodSweeper.
// Each level (0-9) has different "cheat" paths and security hardening.
package level

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/zwindler/podsweeper/pkg/game"
)

const (
	// MapConfigMapName is the name of the ConfigMap containing the mine map.
	MapConfigMapName = "map"

	// MapSecretName is the name of the Secret containing the mine map (Level 1+).
	MapSecretName = "map"

	// MapDataKey is the key used to store the visual grid in ConfigMap/Secret.
	MapDataKey = "grid"
)

// Manager handles level-specific resource creation and cleanup.
type Manager struct {
	client.Client
	Namespace string
}

// NewManager creates a new level Manager.
func NewManager(c client.Client, namespace string) *Manager {
	return &Manager{
		Client:    c,
		Namespace: namespace,
	}
}

// ApplyLevel applies level-specific resources for the given game state.
// This should be called after the game state is saved but before the grid is spawned.
func (m *Manager) ApplyLevel(ctx context.Context, state *game.GameState) error {
	// First, clean up resources from any previous level
	if err := m.Cleanup(ctx); err != nil {
		// Log but don't fail - cleanup errors are non-fatal
		_ = err
	}

	switch state.Level {
	case 0:
		// Level 0: The Intern - mine map exposed in a ConfigMap
		return m.applyLevel0(ctx, state)
	case 1:
		// Level 1: The Junior - mine map in a Secret (Base64)
		return m.applyLevel1(ctx, state)
	// Levels 2-9 will be implemented later
	default:
		// No special resources for this level
		return nil
	}
}

// Cleanup removes all level-specific resources.
func (m *Manager) Cleanup(ctx context.Context) error {
	// Delete map ConfigMap if it exists
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapConfigMapName,
			Namespace: m.Namespace,
		},
	}
	if err := m.Delete(ctx, cm); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete map ConfigMap: %w", err)
	}

	// Delete map Secret if it exists
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapSecretName,
			Namespace: m.Namespace,
		},
	}
	if err := m.Delete(ctx, secret); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete map Secret: %w", err)
	}

	return nil
}

// applyLevel0 creates resources for Level 0: The Intern.
// The mine map is exposed as a plaintext ConfigMap that anyone can read.
func (m *Manager) applyLevel0(ctx context.Context, state *game.GameState) error {
	visualGrid := state.ToVisualGrid()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapConfigMapName,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "podsweeper",
				"app.kubernetes.io/component": "map",
				"podsweeper.io/level":         "0",
			},
			Annotations: map[string]string{
				"podsweeper.io/hint": "The intern left the mine map in plain sight. Try: kubectl get cm map -o yaml",
			},
		},
		Data: map[string]string{
			MapDataKey: visualGrid,
		},
	}

	if err := m.Create(ctx, cm); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing ConfigMap
			existing := &corev1.ConfigMap{}
			if err := m.Get(ctx, client.ObjectKey{Namespace: m.Namespace, Name: MapConfigMapName}, existing); err != nil {
				return fmt.Errorf("failed to get existing map ConfigMap: %w", err)
			}
			existing.Data = cm.Data
			existing.Labels = cm.Labels
			existing.Annotations = cm.Annotations
			if err := m.Update(ctx, existing); err != nil {
				return fmt.Errorf("failed to update map ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create map ConfigMap: %w", err)
	}

	return nil
}

// applyLevel1 creates resources for Level 1: The Junior.
// The mine map is stored in a Secret (Base64 encoded).
// Player needs to decode Base64 to read it.
func (m *Manager) applyLevel1(ctx context.Context, state *game.GameState) error {
	visualGrid := state.ToVisualGrid()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapSecretName,
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "podsweeper",
				"app.kubernetes.io/component": "map",
				"podsweeper.io/level":         "1",
			},
			Annotations: map[string]string{
				"podsweeper.io/hint": "The junior hid the map in a Secret. Can you decode it?",
			},
		},
		StringData: map[string]string{
			MapDataKey: visualGrid,
		},
	}

	if err := m.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing Secret
			existing := &corev1.Secret{}
			if err := m.Get(ctx, client.ObjectKey{Namespace: m.Namespace, Name: MapSecretName}, existing); err != nil {
				return fmt.Errorf("failed to get existing map Secret: %w", err)
			}
			existing.StringData = secret.StringData
			existing.Labels = secret.Labels
			existing.Annotations = secret.Annotations
			if err := m.Update(ctx, existing); err != nil {
				return fmt.Errorf("failed to update map Secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create map Secret: %w", err)
	}

	return nil
}
