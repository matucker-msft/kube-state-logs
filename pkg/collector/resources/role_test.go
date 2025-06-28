package resources

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

func createTestRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			Labels:            map[string]string{"app": "test-app"},
			Annotations:       map[string]string{"test-annotation": "test-value"},
			CreationTimestamp: metav1.Now(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}

func TestNewRoleHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestRoleHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.Role{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestRoleHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	logger := &testutils.MockLogger{}

	// Create a proper informer factory
	factory := informers.NewSharedInformerFactory(client, 0)

	err := handler.SetupInformer(factory, logger, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestRoleHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	logger := &testutils.MockLogger{}
	role := createTestRole("test-role", "test-ns")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.Role{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(role)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.ResourceType != "role" {
		t.Errorf("Expected resource type 'role', got %s", entry.ResourceType)
	}
	if entry.Name != "test-role" {
		t.Errorf("Expected name 'test-role', got %s", entry.Name)
	}
	if entry.Namespace != "test-ns" {
		t.Errorf("Expected namespace 'test-ns', got %s", entry.Namespace)
	}
	data := entry.Data
	if data["labels"].(map[string]string)["app"] != "test-app" {
		t.Errorf("Expected label 'app' to be 'test-app', got %s", data["labels"].(map[string]string)["app"])
	}
	if data["annotations"].(map[string]string)["test-annotation"] != "test-value" {
		t.Errorf("Expected annotation 'test-annotation' to be 'test-value', got %s", data["annotations"].(map[string]string)["test-annotation"])
	}
	rules, ok := data["rules"].([]types.PolicyRule)
	if !ok || len(rules) != 1 {
		t.Errorf("Expected rules to be []PolicyRule of length 1, got %v", data["rules"])
	}
}

func TestRoleHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.Role{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}

func TestRoleHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.Role{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(&rbacv1.ClusterRole{})
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}
