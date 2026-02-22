// Package level handles level-specific resources and configurations for PodSweeper.
// Each level (0-9) has different "cheat" paths and security hardening.
package level

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

	// PlayerRoleName is the name of the player Role that gets updated per level.
	PlayerRoleName = "podsweeper-player"

	// MapFilePath is the path where the map file is written for Level 3.
	MapFilePath = "/tmp/map.txt"
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

	// Apply RBAC rules for this level
	if err := m.applyRBAC(ctx, state.Level); err != nil {
		return fmt.Errorf("failed to apply RBAC for level %d: %w", state.Level, err)
	}

	switch state.Level {
	case 0:
		// Level 0: The Intern - mine map exposed in a ConfigMap
		return m.applyLevel0(ctx, state)
	case 1:
		// Level 1: The Junior - mine map in a Secret (Base64)
		return m.applyLevel1(ctx, state)
	case 2:
		// Level 2: The Infiltrator - mine map in pod env vars
		// Handled by spawner, no additional resources needed here
		return nil
	case 3:
		// Level 3: The Heart of the Machine - mine map in Gamemaster's filesystem
		return m.applyLevel3(ctx, state)
	default:
		// Level 4+: No cheat path - must play legitimately
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

	// Delete map file if it exists (Level 3)
	if err := os.Remove(MapFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete map file: %w", err)
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

// applyLevel3 creates resources for Level 3: The Heart of the Machine.
// The mine map is written to a file inside the Gamemaster pod.
// Player needs to exec into the Gamemaster pod to read it.
func (m *Manager) applyLevel3(_ context.Context, state *game.GameState) error {
	visualGrid := state.ToVisualGrid()

	// Write the map to a file in the Gamemaster's filesystem
	// The Gamemaster controller runs in the pod, so this writes to /tmp/map.txt
	if err := os.WriteFile(MapFilePath, []byte(visualGrid), 0644); err != nil {
		return fmt.Errorf("failed to write map file: %w", err)
	}

	return nil
}

// applyRBAC updates the player Role with permissions appropriate for the given level.
// Permissions are progressively restricted as levels increase.
func (m *Manager) applyRBAC(ctx context.Context, level int) error {
	rules := m.getRBACRulesForLevel(level)

	// Get the existing Role
	role := &rbacv1.Role{}
	if err := m.Get(ctx, client.ObjectKey{Namespace: m.Namespace, Name: PlayerRoleName}, role); err != nil {
		if errors.IsNotFound(err) {
			// Role doesn't exist yet - this is OK during initial setup
			// The Role is created by kustomize, not by the controller
			return nil
		}
		return fmt.Errorf("failed to get player Role: %w", err)
	}

	// Update the rules
	role.Rules = rules

	if err := m.Update(ctx, role); err != nil {
		return fmt.Errorf("failed to update player Role: %w", err)
	}

	return nil
}

// getRBACRulesForLevel returns the RBAC rules for a given level.
// Each level progressively removes permissions to close cheat paths.
func (m *Manager) getRBACRulesForLevel(level int) []rbacv1.PolicyRule {
	// Base rules that all levels have
	baseRules := []rbacv1.PolicyRule{
		// Core gameplay: see and delete pods
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch", "delete"},
		},
		// Port-forward to hint pods
		{
			APIGroups: []string{""},
			Resources: []string{"pods/portforward"},
			Verbs:     []string{"create"},
		},
		// View pod logs
		{
			APIGroups: []string{""},
			Resources: []string{"pods/log"},
			Verbs:     []string{"get"},
		},
		// View events (always available, required for Level 9)
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	switch {
	case level <= 0:
		// Level 0: The Intern - full access
		// Can read ConfigMaps, Secrets, exec into pods
		return append(baseRules,
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		)

	case level == 1:
		// Level 1: The Junior - remove ConfigMap access
		// Map is in a Secret (Base64 encoded)
		return append(baseRules,
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		)

	case level == 2:
		// Level 2: The Infiltrator - remove Secret access
		// Map is in pod environment variables
		return append(baseRules,
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
		)

	case level == 3:
		// Level 3: The Heart of the Machine - exec only into player pod
		// Map is inside Gamemaster pod only
		// Player can exec, but needs to figure out which pod has the data
		return append(baseRules,
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
		)

	default:
		// Level 4+: Amnesia / advanced levels - minimal permissions
		// No cheat path available - must play legitimately
		return baseRules
	}
}
