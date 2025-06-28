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

// createTestStatefulSet creates a test StatefulSet with various configurations
func createTestStatefulSet(name, namespace string, replicas int32) *appsv1.StatefulSet {
	now := metav1.Now()
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test statefulset",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         name + "-service",
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			UpdateStrategy:      appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:           replicas,
			ReadyReplicas:      replicas,
			UpdatedReplicas:    replicas,
			ObservedGeneration: 1,
			CurrentRevision:    name + "-rev-1",
			UpdateRevision:     name + "-rev-1",
		},
	}
	return sts
}

func TestNewStatefulSetHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStatefulSetHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestStatefulSetHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStatefulSetHandler(client)
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

func TestStatefulSetHandler_Collect(t *testing.T) {
	sts1 := createTestStatefulSet("test-sts-1", "default", 3)
	sts2 := createTestStatefulSet("test-sts-2", "kube-system", 2)
	client := fake.NewSimpleClientset(sts1, sts2)
	handler := NewStatefulSetHandler(client)
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

func TestStatefulSetHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStatefulSetHandler(client)
	sts := createTestStatefulSet("test-sts", "default", 3)
	entry := handler.createLogEntry(sts)
	if entry.ResourceType != "statefulset" {
		t.Errorf("Expected resource type 'statefulset', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-sts" {
		t.Errorf("Expected name 'test-sts', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["desiredReplicas"]
	if !ok || val == nil {
		t.Fatalf("desiredReplicas missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected desired replicas 3, got %d", val.(int32))
	}
	val, ok = data["currentReplicas"]
	if !ok || val == nil {
		t.Fatalf("currentReplicas missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected current replicas 3, got %d", val.(int32))
	}
	val, ok = data["readyReplicas"]
	if !ok || val == nil {
		t.Fatalf("readyReplicas missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected ready replicas 3, got %d", val.(int32))
	}
	val, ok = data["serviceName"]
	if !ok || val == nil {
		t.Fatalf("serviceName missing or nil")
	}
	if val.(string) != "test-sts-service" {
		t.Errorf("Expected service name 'test-sts-service', got '%s'", val.(string))
	}
	val, ok = data["podManagementPolicy"]
	if !ok || val == nil {
		t.Fatalf("podManagementPolicy missing or nil")
	}
	if val.(string) != "OrderedReady" {
		t.Errorf("Expected pod management policy 'OrderedReady', got '%s'", val.(string))
	}
	val, ok = data["updateStrategy"]
	if !ok || val == nil {
		t.Fatalf("updateStrategy missing or nil")
	}
	if val.(string) != "RollingUpdate" {
		t.Errorf("Expected update strategy 'RollingUpdate', got '%s'", val.(string))
	}
}

func TestStatefulSetHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStatefulSetHandler(client)
	sts := createTestStatefulSet("test-sts", "default", 3)
	sts.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deploy",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(sts)
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

func TestStatefulSetHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStatefulSetHandler(client)
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

func TestStatefulSetHandler_Collect_NamespaceFiltering(t *testing.T) {
	sts1 := createTestStatefulSet("test-sts-1", "default", 3)
	sts2 := createTestStatefulSet("test-sts-2", "kube-system", 2)
	sts3 := createTestStatefulSet("test-sts-3", "monitoring", 1)
	client := fake.NewSimpleClientset(sts1, sts2, sts3)
	handler := NewStatefulSetHandler(client)
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
