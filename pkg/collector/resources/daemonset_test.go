package resources

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestDaemonSet creates a test DaemonSet with various configurations
func createTestDaemonSet(name, namespace string) *appsv1.DaemonSet {
	now := metav1.Now()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test daemonset",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberReady:            3,
			NumberAvailable:        3,
			NumberUnavailable:      0,
			NumberMisscheduled:     0,
			UpdatedNumberScheduled: 3,
			ObservedGeneration:     1,
		},
	}
	return ds
}

func TestNewDaemonSetHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDaemonSetHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestDaemonSetHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDaemonSetHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if handler.GetInformer() == nil {
		t.Error("Expected informer to be set up")
	}
}

func TestDaemonSetHandler_Collect(t *testing.T) {
	ds1 := createTestDaemonSet("test-ds-1", "default")
	ds2 := createTestDaemonSet("test-ds-2", "kube-system")
	client := fake.NewSimpleClientset(ds1, ds2)
	handler := NewDaemonSetHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	entries, err = handler.Collect(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for default namespace, got %d", len(entries))
	}
	if entries[0].Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entries[0].Namespace)
	}
}

func TestDaemonSetHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDaemonSetHandler(client)
	ds := createTestDaemonSet("test-ds", "default")
	entry := handler.createLogEntry(ds)
	if entry.ResourceType != "daemonset" {
		t.Errorf("Expected resource type 'daemonset', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-ds" {
		t.Errorf("Expected name 'test-ds', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["desiredNumberScheduled"]
	if !ok || val == nil {
		t.Fatalf("desiredNumberScheduled missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected desired number scheduled 3, got %d", val.(int32))
	}
	val, ok = data["currentNumberScheduled"]
	if !ok || val == nil {
		t.Fatalf("currentNumberScheduled missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected current number scheduled 3, got %d", val.(int32))
	}
	val, ok = data["numberReady"]
	if !ok || val == nil {
		t.Fatalf("numberReady missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected number ready 3, got %d", val.(int32))
	}
	val, ok = data["numberAvailable"]
	if !ok || val == nil {
		t.Fatalf("numberAvailable missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected number available 3, got %d", val.(int32))
	}
	val, ok = data["updateStrategy"]
	if !ok || val == nil {
		t.Fatalf("updateStrategy missing or nil")
	}
	if val.(string) != "RollingUpdate" {
		t.Errorf("Expected update strategy 'RollingUpdate', got '%s'", val.(string))
	}
}

func TestDaemonSetHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDaemonSetHandler(client)
	ds := createTestDaemonSet("test-ds", "default")
	ds.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deploy",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(ds)
	data := entry.Data
	val, ok := data["createdByKind"]
	if !ok || val == nil {
		t.Fatalf("createdByKind missing or nil")
	}
	if val.(string) != "Deployment" {
		t.Errorf("Expected created by kind 'Deployment', got '%s'", val.(string))
	}
	val, ok = data["createdByName"]
	if !ok || val == nil {
		t.Fatalf("createdByName missing or nil")
	}
	if val.(string) != "test-deploy" {
		t.Errorf("Expected created by name 'test-deploy', got '%s'", val.(string))
	}
}

func TestDaemonSetHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDaemonSetHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for empty cache, got %d", len(entries))
	}
}

func TestDaemonSetHandler_Collect_NamespaceFiltering(t *testing.T) {
	ds1 := createTestDaemonSet("test-ds-1", "default")
	ds2 := createTestDaemonSet("test-ds-2", "kube-system")
	ds3 := createTestDaemonSet("test-ds-3", "monitoring")
	client := fake.NewSimpleClientset(ds1, ds2, ds3)
	handler := NewDaemonSetHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{"default", "monitoring"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for default and monitoring namespaces, got %d", len(entries))
	}
	namespaces := make(map[string]bool)
	for _, entry := range entries {
		namespaces[entry.Namespace] = true
	}
	if !namespaces["default"] {
		t.Error("Expected entry from default namespace")
	}
	if !namespaces["monitoring"] {
		t.Error("Expected entry from monitoring namespace")
	}
	if namespaces["kube-system"] {
		t.Error("Did not expect entry from kube-system namespace")
	}
}
