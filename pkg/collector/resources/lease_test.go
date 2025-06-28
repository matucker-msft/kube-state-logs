package resources

import (
	"context"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestLease creates a test Lease with various configurations
func createTestLease(name, namespace string) *coordinationv1.Lease {
	now := metav1.Now()
	nowMicro := metav1.NewMicroTime(now.Time)
	holderIdentity := "test-holder"
	leaseDurationSeconds := int32(15)
	leaseTransitions := int32(1)
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test lease",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &holderIdentity,
			LeaseDurationSeconds: &leaseDurationSeconds,
			RenewTime:            &nowMicro,
			AcquireTime:          &nowMicro,
			LeaseTransitions:     &leaseTransitions,
		},
	}
	return lease
}

func TestNewLeaseHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewLeaseHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestLeaseHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewLeaseHandler(client)
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

func TestLeaseHandler_Collect(t *testing.T) {
	lease1 := createTestLease("test-lease-1", "default")
	lease2 := createTestLease("test-lease-2", "kube-system")
	client := fake.NewSimpleClientset(lease1, lease2)
	handler := NewLeaseHandler(client)
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

func TestLeaseHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewLeaseHandler(client)
	lease := createTestLease("test-lease", "default")
	entry := handler.createLogEntry(lease)
	if entry.ResourceType != "lease" {
		t.Errorf("Expected resource type 'lease', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-lease" {
		t.Errorf("Expected name 'test-lease', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["holderIdentity"]
	if !ok || val == nil {
		t.Fatalf("holderIdentity missing or nil")
	}
	if val.(string) != "test-holder" {
		t.Errorf("Expected holder identity 'test-holder', got '%s'", val.(string))
	}
	val, ok = data["leaseDurationSeconds"]
	if !ok || val == nil {
		t.Fatalf("leaseDurationSeconds missing or nil")
	}
	if val.(int32) != 15 {
		t.Errorf("Expected lease duration seconds 15, got %d", val.(int32))
	}
	val, ok = data["leaseTransitions"]
	if !ok || val == nil {
		t.Fatalf("leaseTransitions missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected lease transitions 1, got %d", val.(int32))
	}
	val, ok = data["renewTime"]
	if !ok || val == nil {
		t.Fatalf("renewTime missing or nil")
	}
	if _, ok := val.(time.Time); !ok {
		t.Error("Expected renew time to be time.Time")
	}
	val, ok = data["acquireTime"]
	if !ok || val == nil {
		t.Fatalf("acquireTime missing or nil")
	}
	if _, ok := val.(time.Time); !ok {
		t.Error("Expected acquire time to be time.Time")
	}
}

func TestLeaseHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewLeaseHandler(client)
	lease := createTestLease("test-lease", "default")
	lease.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deploy",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(lease)
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

func TestLeaseHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewLeaseHandler(client)
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

func TestLeaseHandler_Collect_NamespaceFiltering(t *testing.T) {
	lease1 := createTestLease("test-lease-1", "default")
	lease2 := createTestLease("test-lease-2", "kube-system")
	lease3 := createTestLease("test-lease-3", "monitoring")
	client := fake.NewSimpleClientset(lease1, lease2, lease3)
	handler := NewLeaseHandler(client)
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
