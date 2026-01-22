// Package game contains the core game logic and state management for PodSweeper.
package game

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultSecretName is the name of the Secret storing game state.
	DefaultSecretName = "podsweeper-state"

	// DefaultNamespace is the default game namespace.
	DefaultNamespace = "podsweeper-game"

	// StateKey is the key in the Secret data map for the game state JSON.
	StateKey = "state"
)

// Store defines the interface for persisting game state.
type Store interface {
	// Load retrieves the current game state.
	// Returns nil, nil if no game state exists.
	Load(ctx context.Context) (*GameState, error)

	// Save persists the game state.
	// Creates or updates the underlying storage.
	Save(ctx context.Context, state *GameState) error

	// Delete removes the game state.
	// Returns nil if the state doesn't exist.
	Delete(ctx context.Context) error

	// Exists checks if a game state exists.
	Exists(ctx context.Context) (bool, error)
}

// SecretStore persists game state in a Kubernetes Secret.
type SecretStore struct {
	client    client.Client
	namespace string
	name      string
}

// SecretStoreOption configures a SecretStore.
type SecretStoreOption func(*SecretStore)

// WithNamespace sets the namespace for the Secret.
func WithNamespace(namespace string) SecretStoreOption {
	return func(s *SecretStore) {
		s.namespace = namespace
	}
}

// WithSecretName sets the name of the Secret.
func WithSecretName(name string) SecretStoreOption {
	return func(s *SecretStore) {
		s.name = name
	}
}

// NewSecretStore creates a new SecretStore.
func NewSecretStore(c client.Client, opts ...SecretStoreOption) *SecretStore {
	store := &SecretStore{
		client:    c,
		namespace: DefaultNamespace,
		name:      DefaultSecretName,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Load retrieves the game state from the Secret.
func (s *SecretStore) Load(ctx context.Context) (*GameState, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: s.namespace,
		Name:      s.name,
	}

	if err := s.client.Get(ctx, key, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // No game state exists
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	data, ok := secret.Data[StateKey]
	if !ok {
		return nil, fmt.Errorf("secret exists but missing '%s' key", StateKey)
	}

	state, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse game state: %w", err)
	}

	return state, nil
}

// Save persists the game state to the Secret.
func (s *SecretStore) Save(ctx context.Context, state *GameState) error {
	data, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize game state: %w", err)
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: s.namespace,
		Name:      s.name,
	}

	err = s.client.Get(ctx, key, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new secret
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      s.name,
					Namespace: s.namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "podsweeper",
						"app.kubernetes.io/component": "game-state",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					StateKey: data,
				},
			}
			if err := s.client.Create(ctx, secret); err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get secret: %w", err)
	}

	// Update existing secret
	secret.Data[StateKey] = data
	if err := s.client.Update(ctx, secret); err != nil {
		if errors.IsConflict(err) {
			return fmt.Errorf("conflict updating secret (concurrent modification): %w", err)
		}
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// Delete removes the game state Secret.
func (s *SecretStore) Delete(ctx context.Context) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.namespace,
		},
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// Exists checks if the game state Secret exists.
func (s *SecretStore) Exists(ctx context.Context) (bool, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: s.namespace,
		Name:      s.name,
	}

	if err := s.client.Get(ctx, key, secret); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check secret: %w", err)
	}

	return true, nil
}

// Namespace returns the namespace where the Secret is stored.
func (s *SecretStore) Namespace() string {
	return s.namespace
}

// SecretName returns the name of the Secret.
func (s *SecretStore) SecretName() string {
	return s.name
}

// MemoryStore is an in-memory Store implementation for testing.
type MemoryStore struct {
	mu    sync.RWMutex
	state *GameState
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Load retrieves the game state from memory.
func (m *MemoryStore) Load(ctx context.Context) (*GameState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return nil, nil
	}

	// Return a clone to prevent external modification
	return m.state.Clone(), nil
}

// Save stores the game state in memory.
func (m *MemoryStore) Save(ctx context.Context, state *GameState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a clone to prevent external modification
	m.state = state.Clone()
	return nil
}

// Delete removes the game state from memory.
func (m *MemoryStore) Delete(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = nil
	return nil
}

// Exists checks if a game state exists in memory.
func (m *MemoryStore) Exists(ctx context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state != nil, nil
}

// Reset clears the store (useful for testing).
func (m *MemoryStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = nil
}

// EncodeForConfigMap encodes the game state for storage in a ConfigMap.
// This is used for Level 0 where the map is exposed as a "cheat".
func EncodeForConfigMap(state *GameState) (string, error) {
	data, err := state.ToJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EncodeForSecret encodes the game state for storage in a Secret.
// The data is base64-encoded (standard Secret behavior).
func EncodeForSecret(state *GameState) (string, error) {
	data, err := state.ToJSON()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeFromSecret decodes a base64-encoded game state from a Secret.
func DecodeFromSecret(encoded string) (*GameState, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return FromJSON(data)
}
