package level

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zwindler/podsweeper/pkg/game"
)

func newFakeClient(objs ...runtime.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		Build()
}

func TestApplyLevel0(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	// Create a namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}

	fakeClient := newFakeClient(ns)
	mgr := NewManager(fakeClient, namespace)

	// Create a game state with known mines
	state := game.NewGameState(5, 12345)
	state.SetMine(2, 0)
	state.SetMine(3, 0)
	state.SetMine(4, 1)
	state.SetMine(1, 4)
	state.Level = 0

	// Apply level 0
	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	// Check that the map ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapConfigMapName}, cm)
	if err != nil {
		t.Fatalf("Failed to get map ConfigMap: %v", err)
	}

	// Verify the visual grid content
	grid, ok := cm.Data[MapDataKey]
	if !ok {
		t.Fatal("ConfigMap missing 'grid' key")
	}

	expected := `. . X X .
. . . . X
. . . . .
. . . . .
. X . . .`

	if grid != expected {
		t.Errorf("Grid mismatch:\nExpected:\n%s\n\nGot:\n%s", expected, grid)
	}

	// Verify labels
	if cm.Labels["podsweeper.io/level"] != "0" {
		t.Errorf("Expected level label '0', got %q", cm.Labels["podsweeper.io/level"])
	}

	// Verify hint annotation
	if cm.Annotations["podsweeper.io/hint"] == "" {
		t.Error("Expected hint annotation to be set")
	}
}

func TestApplyLevel1(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}

	fakeClient := newFakeClient(ns)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(3, 12345)
	state.SetMine(1, 1)
	state.Level = 1

	// Apply level 1
	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	// Check that the map Secret was created (not ConfigMap)
	secret := &corev1.Secret{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapSecretName}, secret)
	if err != nil {
		t.Fatalf("Failed to get map Secret: %v", err)
	}

	// The secret should have the grid data
	// Note: fake client doesn't auto-convert StringData to Data, so check both
	expected := `. . .
. X .
. . .`

	var grid string
	if data, ok := secret.Data[MapDataKey]; ok {
		grid = string(data)
	} else if stringData, ok := secret.StringData[MapDataKey]; ok {
		grid = stringData
	} else {
		t.Fatal("Secret missing 'grid' key in both Data and StringData")
	}

	if grid != expected {
		t.Errorf("Grid mismatch:\nExpected:\n%s\n\nGot:\n%s", expected, grid)
	}

	// Verify no ConfigMap was created for level 1
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapConfigMapName}, cm)
	if err == nil {
		t.Error("Expected no ConfigMap for level 1, but found one")
	}
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	// Create existing map ConfigMap and Secret
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{MapDataKey: "old data"},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{MapDataKey: []byte("old data")},
	}

	fakeClient := newFakeClient(cm, secret)
	mgr := NewManager(fakeClient, namespace)

	// Cleanup
	err := mgr.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify ConfigMap was deleted
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapConfigMapName}, &corev1.ConfigMap{})
	if err == nil {
		t.Error("Expected ConfigMap to be deleted")
	}

	// Verify Secret was deleted
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapSecretName}, &corev1.Secret{})
	if err == nil {
		t.Error("Expected Secret to be deleted")
	}
}

func TestCleanupNoResources(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	fakeClient := newFakeClient()
	mgr := NewManager(fakeClient, namespace)

	// Cleanup should not error when resources don't exist
	err := mgr.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed when no resources exist: %v", err)
	}
}

func TestApplyLevelUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	// Create existing ConfigMap with old data
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MapConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{MapDataKey: "old grid"},
	}

	fakeClient := newFakeClient(existingCM)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(2, 12345)
	state.SetMine(0, 0)
	state.Level = 0

	// Apply level 0 - should update existing ConfigMap
	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	// Check that the ConfigMap was updated
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MapConfigMapName}, cm)
	if err != nil {
		t.Fatalf("Failed to get map ConfigMap: %v", err)
	}

	expected := `X .
. .`

	if cm.Data[MapDataKey] != expected {
		t.Errorf("Grid not updated:\nExpected:\n%s\n\nGot:\n%s", expected, cm.Data[MapDataKey])
	}
}

// --- RBAC Tests ---

func createPlayerRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PlayerRoleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			// Initial placeholder rules
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		},
	}
}

func TestApplyRBACLevel0(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	role := createPlayerRole(namespace)
	fakeClient := newFakeClient(role)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(5, 12345)
	state.Level = 0

	// Apply level 0
	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	// Get the updated role
	updatedRole := &rbacv1.Role{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: PlayerRoleName}, updatedRole)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	// Level 0 should have ConfigMaps and Secrets access
	hasConfigMaps := false
	hasSecrets := false
	hasExec := false

	for _, rule := range updatedRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "configmaps" {
				hasConfigMaps = true
			}
			if resource == "secrets" {
				hasSecrets = true
			}
			if resource == "pods/exec" {
				hasExec = true
			}
		}
	}

	if !hasConfigMaps {
		t.Error("Level 0 should have configmaps access")
	}
	if !hasSecrets {
		t.Error("Level 0 should have secrets access")
	}
	if !hasExec {
		t.Error("Level 0 should have pods/exec access")
	}
}

func TestApplyRBACLevel1(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	role := createPlayerRole(namespace)
	fakeClient := newFakeClient(role)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(5, 12345)
	state.Level = 1

	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	updatedRole := &rbacv1.Role{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: PlayerRoleName}, updatedRole)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	// Level 1 should have Secrets but NOT ConfigMaps
	hasConfigMaps := false
	hasSecrets := false

	for _, rule := range updatedRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "configmaps" {
				hasConfigMaps = true
			}
			if resource == "secrets" {
				hasSecrets = true
			}
		}
	}

	if hasConfigMaps {
		t.Error("Level 1 should NOT have configmaps access")
	}
	if !hasSecrets {
		t.Error("Level 1 should have secrets access")
	}
}

func TestApplyRBACLevel2(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	role := createPlayerRole(namespace)
	fakeClient := newFakeClient(role)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(5, 12345)
	state.Level = 2

	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	updatedRole := &rbacv1.Role{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: PlayerRoleName}, updatedRole)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	// Level 2 should have exec but NOT ConfigMaps or Secrets
	hasConfigMaps := false
	hasSecrets := false
	hasExec := false

	for _, rule := range updatedRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "configmaps" {
				hasConfigMaps = true
			}
			if resource == "secrets" {
				hasSecrets = true
			}
			if resource == "pods/exec" {
				hasExec = true
			}
		}
	}

	if hasConfigMaps {
		t.Error("Level 2 should NOT have configmaps access")
	}
	if hasSecrets {
		t.Error("Level 2 should NOT have secrets access")
	}
	if !hasExec {
		t.Error("Level 2 should have pods/exec access")
	}
}

func TestApplyRBACLevel4Plus(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	role := createPlayerRole(namespace)
	fakeClient := newFakeClient(role)
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(5, 12345)
	state.Level = 4

	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel failed: %v", err)
	}

	updatedRole := &rbacv1.Role{}
	err = fakeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: PlayerRoleName}, updatedRole)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	// Level 4+ should have minimal permissions - no ConfigMaps, Secrets, or exec
	hasConfigMaps := false
	hasSecrets := false
	hasExec := false
	hasPods := false

	for _, rule := range updatedRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "configmaps" {
				hasConfigMaps = true
			}
			if resource == "secrets" {
				hasSecrets = true
			}
			if resource == "pods/exec" {
				hasExec = true
			}
			if resource == "pods" {
				hasPods = true
			}
		}
	}

	if hasConfigMaps {
		t.Error("Level 4+ should NOT have configmaps access")
	}
	if hasSecrets {
		t.Error("Level 4+ should NOT have secrets access")
	}
	if hasExec {
		t.Error("Level 4+ should NOT have pods/exec access")
	}
	if !hasPods {
		t.Error("Level 4+ should still have pods access for gameplay")
	}
}

func TestApplyRBACNoRole(t *testing.T) {
	ctx := context.Background()
	namespace := "test-ns"

	// No role exists - should not error
	fakeClient := newFakeClient()
	mgr := NewManager(fakeClient, namespace)

	state := game.NewGameState(5, 12345)
	state.Level = 0

	// Should not error when role doesn't exist
	err := mgr.ApplyLevel(ctx, state)
	if err != nil {
		t.Fatalf("ApplyLevel should not fail when role doesn't exist: %v", err)
	}
}
