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

// createTestReplicaSet creates a test ReplicaSet with owner, labels, annotations, and status
func createTestReplicaSet(name, namespace, ownerKind, ownerName string, replicas, ready, available, fullyLabeled int32, isCurrent bool) *appsv1.ReplicaSet {
	now := metav1.Now()
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test replicaset",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.ReplicaSetStatus{
			Replicas:             replicas,
			ReadyReplicas:        ready,
			AvailableReplicas:    available,
			FullyLabeledReplicas: fullyLabeled,
			ObservedGeneration:   1,
		},
	}
	if ownerKind != "" && ownerName != "" {
		rs.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       ownerKind,
			Name:       ownerName,
			UID:        "test-uid",
		}}
	}
	if isCurrent {
		if rs.Labels == nil {
			rs.Labels = map[string]string{}
		}
		rs.Labels["kube-state-logs/current"] = "true"
	}
	return rs
}

func TestNewReplicaSetHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicaSetHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestReplicaSetHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicaSetHandler(client)
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

func TestReplicaSetHandler_Collect(t *testing.T) {
	// Create test ReplicaSets
	rs1 := createTestReplicaSet("test-rs-1", "default", "Deployment", "test-deploy", 3, 2, 2, 2, true)
	rs2 := createTestReplicaSet("test-rs-2", "kube-system", "Deployment", "test-deploy2", 2, 2, 2, 2, true)
	client := fake.NewSimpleClientset(rs1, rs2)
	handler := NewReplicaSetHandler(client)
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

func TestReplicaSetHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicaSetHandler(client)

	rs := createTestReplicaSet("test-rs", "default", "Deployment", "test-deploy", 3, 2, 2, 2, true)
	entry := handler.createLogEntry(rs)

	if entry.ResourceType != "replicaset" {
		t.Errorf("Expected resource type 'replicaset', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-rs" {
		t.Errorf("Expected name 'test-rs', got '%s'", entry.Name)
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
	if val.(int32) != 2 {
		t.Errorf("Expected ready replicas 2, got %d", val.(int32))
	}
	val, ok = data["availableReplicas"]
	if !ok || val == nil {
		t.Fatalf("availableReplicas missing or nil")
	}
	if val.(int32) != 2 {
		t.Errorf("Expected available replicas 2, got %d", val.(int32))
	}
	val, ok = data["fullyLabeledReplicas"]
	if !ok || val == nil {
		t.Fatalf("fullyLabeledReplicas missing or nil")
	}
	if val.(int32) != 2 {
		t.Errorf("Expected fully labeled replicas 2, got %d", val.(int32))
	}
	val, ok = data["isCurrent"]
	if !ok || val == nil {
		t.Fatalf("isCurrent missing or nil")
	}
	if !val.(bool) {
		t.Errorf("Expected isCurrent to be true")
	}
}

func TestReplicaSetHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicaSetHandler(client)

	rs := createTestReplicaSet("test-rs", "default", "Deployment", "test-deploy", 3, 2, 2, 2, true)
	entry := handler.createLogEntry(rs)
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

func TestReplicaSetHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicaSetHandler(client)
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

func TestReplicaSetHandler_Collect_NamespaceFiltering(t *testing.T) {
	rs1 := createTestReplicaSet("test-rs-1", "default", "Deployment", "test-deploy", 3, 2, 2, 2, true)
	rs2 := createTestReplicaSet("test-rs-2", "kube-system", "Deployment", "test-deploy2", 2, 2, 2, 2, true)
	rs3 := createTestReplicaSet("test-rs-3", "monitoring", "Deployment", "test-deploy3", 1, 1, 1, 1, true)
	client := fake.NewSimpleClientset(rs1, rs2, rs3)
	handler := NewReplicaSetHandler(client)
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
