package game

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryStore_LoadEmpty(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	state, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if state != nil {
		t.Error("expected nil state for empty store")
	}
}

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create and save a state
	original := NewGameState(10, 12345)
	original.SetMine(3, 5)
	original.Level = 2

	if err := store.Save(ctx, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load it back
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil state")
	}

	// Verify contents
	if loaded.Size != original.Size {
		t.Errorf("Size mismatch: expected %d, got %d", original.Size, loaded.Size)
	}
	if loaded.Seed != original.Seed {
		t.Errorf("Seed mismatch: expected %d, got %d", original.Seed, loaded.Seed)
	}
	if loaded.Level != original.Level {
		t.Errorf("Level mismatch: expected %d, got %d", original.Level, loaded.Level)
	}
	if !loaded.IsMine(3, 5) {
		t.Error("mine at (3,5) not preserved")
	}
}

func TestMemoryStore_SaveReturnsClone(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	original := NewGameState(10, 12345)
	store.Save(ctx, original)

	// Modify original after saving
	original.Level = 99

	// Load should return the saved value, not the modified one
	loaded, _ := store.Load(ctx)
	if loaded.Level == 99 {
		t.Error("store should keep a clone, not a reference")
	}
}

func TestMemoryStore_LoadReturnsClone(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	original := NewGameState(10, 12345)
	store.Save(ctx, original)

	// Load and modify
	loaded1, _ := store.Load(ctx)
	loaded1.Level = 99

	// Load again - should not see the modification
	loaded2, _ := store.Load(ctx)
	if loaded2.Level == 99 {
		t.Error("Load should return a clone each time")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Save then delete
	store.Save(ctx, NewGameState(10, 12345))
	if err := store.Delete(ctx); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should be gone
	state, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load after delete failed: %v", err)
	}
	if state != nil {
		t.Error("state should be nil after delete")
	}
}

func TestMemoryStore_DeleteNonExistent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Delete on empty store should not error
	if err := store.Delete(ctx); err != nil {
		t.Errorf("Delete on empty store should not error: %v", err)
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Initially doesn't exist
	exists, err := store.Exists(ctx)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("should not exist initially")
	}

	// After save, should exist
	store.Save(ctx, NewGameState(10, 0))
	exists, err = store.Exists(ctx)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("should exist after save")
	}

	// After delete, should not exist
	store.Delete(ctx)
	exists, _ = store.Exists(ctx)
	if exists {
		t.Error("should not exist after delete")
	}
}

func TestMemoryStore_Reset(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Save(ctx, NewGameState(10, 0))
	store.Reset()

	exists, _ := store.Exists(ctx)
	if exists {
		t.Error("should not exist after reset")
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent saves
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(level int) {
			defer wg.Done()
			state := NewGameState(10, int64(level))
			state.Level = level
			store.Save(ctx, state)
		}(i)
	}

	// Concurrent loads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Load(ctx)
		}()
	}

	wg.Wait()

	// Should still be in a consistent state
	exists, err := store.Exists(ctx)
	if err != nil {
		t.Fatalf("Exists failed after concurrent access: %v", err)
	}
	if !exists {
		t.Error("state should exist after concurrent saves")
	}
}

func TestEncodeForConfigMap(t *testing.T) {
	state := NewGameState(5, 12345)
	state.SetMine(1, 2)

	encoded, err := EncodeForConfigMap(state)
	if err != nil {
		t.Fatalf("EncodeForConfigMap failed: %v", err)
	}

	// Should be valid JSON
	decoded, err := FromJSON([]byte(encoded))
	if err != nil {
		t.Fatalf("encoded data is not valid JSON: %v", err)
	}

	if decoded.Size != state.Size {
		t.Errorf("Size mismatch after encode/decode")
	}
	if !decoded.IsMine(1, 2) {
		t.Error("mine not preserved after encode/decode")
	}
}

func TestEncodeDecodeForSecret(t *testing.T) {
	state := NewGameState(5, 12345)
	state.SetMine(1, 2)
	state.Level = 3

	// Encode
	encoded, err := EncodeForSecret(state)
	if err != nil {
		t.Fatalf("EncodeForSecret failed: %v", err)
	}

	// Should be base64
	if len(encoded) == 0 {
		t.Error("encoded string should not be empty")
	}

	// Decode
	decoded, err := DecodeFromSecret(encoded)
	if err != nil {
		t.Fatalf("DecodeFromSecret failed: %v", err)
	}

	// Verify
	if decoded.Size != state.Size {
		t.Errorf("Size mismatch: expected %d, got %d", state.Size, decoded.Size)
	}
	if decoded.Seed != state.Seed {
		t.Errorf("Seed mismatch: expected %d, got %d", state.Seed, decoded.Seed)
	}
	if decoded.Level != state.Level {
		t.Errorf("Level mismatch: expected %d, got %d", state.Level, decoded.Level)
	}
	if !decoded.IsMine(1, 2) {
		t.Error("mine not preserved after encode/decode")
	}
}

func TestDecodeFromSecret_Invalid(t *testing.T) {
	// Invalid base64
	_, err := DecodeFromSecret("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	// Valid base64 but invalid JSON
	_, err = DecodeFromSecret("bm90LWpzb24=") // "not-json" in base64
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestStoreInterface(t *testing.T) {
	// Verify MemoryStore implements Store interface
	var _ Store = (*MemoryStore)(nil)
	var _ Store = (*SecretStore)(nil)
}

func TestSecretStoreOptions(t *testing.T) {
	// We can't test the actual K8s operations without a fake client,
	// but we can test the option functions

	store := NewSecretStore(nil,
		WithNamespace("custom-namespace"),
		WithSecretName("custom-secret"),
	)

	if store.Namespace() != "custom-namespace" {
		t.Errorf("expected namespace 'custom-namespace', got '%s'", store.Namespace())
	}
	if store.SecretName() != "custom-secret" {
		t.Errorf("expected secret name 'custom-secret', got '%s'", store.SecretName())
	}
}

func TestSecretStoreDefaults(t *testing.T) {
	store := NewSecretStore(nil)

	if store.Namespace() != DefaultNamespace {
		t.Errorf("expected default namespace '%s', got '%s'", DefaultNamespace, store.Namespace())
	}
	if store.SecretName() != DefaultSecretName {
		t.Errorf("expected default secret name '%s', got '%s'", DefaultSecretName, store.SecretName())
	}
}

func TestConstants(t *testing.T) {
	if DefaultSecretName != "podsweeper-state" {
		t.Errorf("unexpected default secret name: %s", DefaultSecretName)
	}
	if DefaultNamespace != "podsweeper-game" {
		t.Errorf("unexpected default namespace: %s", DefaultNamespace)
	}
	if StateKey != "state" {
		t.Errorf("unexpected state key: %s", StateKey)
	}
}
