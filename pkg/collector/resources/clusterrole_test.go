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

func createTestClusterRole(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
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

func TestNewClusterRoleHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestClusterRoleHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.ClusterRole{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestClusterRoleHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
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

func TestClusterRoleHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
	logger := &testutils.MockLogger{}
	cr := createTestClusterRole("test-cr")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.ClusterRole{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(cr)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.ResourceType != "clusterrole" {
		t.Errorf("Expected resource type 'clusterrole', got %s", entry.ResourceType)
	}
	if entry.Name != "test-cr" {
		t.Errorf("Expected name 'test-cr', got %s", entry.Name)
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
		t.Errorf("Expected rules to be []types.PolicyRule of length 1, got %v", data["rules"])
	}
}

func TestClusterRoleHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.ClusterRole{},
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

func TestClusterRoleHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewClusterRoleHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.ClusterRole{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(&rbacv1.Role{})
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}
