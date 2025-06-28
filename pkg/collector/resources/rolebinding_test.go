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

func createTestRoleBinding(name, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			Labels:            map[string]string{"app": "test-app"},
			Annotations:       map[string]string{"test-annotation": "test-value"},
			CreationTimestamp: metav1.Now(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "User",
				Name:      "test-user",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "test-role",
		},
	}
}

func TestNewRoleBindingHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestRoleBindingHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.RoleBinding{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestRoleBindingHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
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

func TestRoleBindingHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
	logger := &testutils.MockLogger{}
	rb := createTestRoleBinding("test-rb", "test-ns")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.RoleBinding{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(rb)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.ResourceType != "rolebinding" {
		t.Errorf("Expected resource type 'rolebinding', got %s", entry.ResourceType)
	}
	if entry.Name != "test-rb" {
		t.Errorf("Expected name 'test-rb', got %s", entry.Name)
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
	subjects, ok := data["subjects"].([]types.Subject)
	if !ok || len(subjects) != 1 {
		t.Errorf("Expected subjects to be []types.Subject of length 1, got %v", data["subjects"])
	}
	roleRef, ok := data["roleRef"].(types.RoleRef)
	if !ok || roleRef.Name != "test-role" {
		t.Errorf("Expected roleRef name 'test-role', got %v", roleRef.Name)
	}
}

func TestRoleBindingHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.RoleBinding{},
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

func TestRoleBindingHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRoleBindingHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&rbacv1.RoleBinding{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(&rbacv1.ClusterRoleBinding{})
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}
