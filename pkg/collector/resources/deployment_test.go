package resources

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestDeployment creates a test deployment with various configurations
func createTestDeployment(name, namespace string, replicas int32, strategyType appsv1.DeploymentStrategyType) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test deployment",
			},
			CreationTimestamp: metav1.Now(),
			Generation:        1,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: strategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            3,
			ReadyReplicas:       2,
			AvailableReplicas:   2,
			UnavailableReplicas: 1,
			UpdatedReplicas:     3,
			ObservedGeneration:  1,
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:               appsv1.DeploymentAvailable,
					Status:             corev1.ConditionTrue,
					LastUpdateTime:     metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Reason:             "MinimumReplicasAvailable",
					Message:            "Deployment has minimum availability.",
				},
				{
					Type:               appsv1.DeploymentProgressing,
					Status:             corev1.ConditionTrue,
					LastUpdateTime:     metav1.Now(),
					LastTransitionTime: metav1.Now(),
					Reason:             "NewReplicaSetAvailable",
					Message:            "ReplicaSet is available.",
				},
			},
		},
	}

	// Add rolling update strategy if specified
	if strategyType == appsv1.RollingUpdateDeploymentStrategyType {
		deployment.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
			MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
		}
	}

	return deployment
}

func TestNewDeploymentHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}

	// Verify BaseHandler is embedded
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestDeploymentHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}

	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify informer is set up
	if handler.GetInformer() == nil {
		t.Error("Expected informer to be set up")
	}
}

func TestDeploymentHandler_Collect(t *testing.T) {
	// Create test deployments
	deployment1 := createTestDeployment("test-deployment-1", "default", 3, appsv1.RollingUpdateDeploymentStrategyType)
	deployment2 := createTestDeployment("test-deployment-2", "kube-system", 2, appsv1.RecreateDeploymentStrategyType)

	// Create fake client with test deployments
	client := fake.NewSimpleClientset(deployment1, deployment2)
	handler := NewDeploymentHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}

	// Setup informer
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	// Start the factory to populate the cache
	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	// Test collecting all deployments
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Test collecting from specific namespace
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

func TestDeploymentHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	// Test deployment with rolling update strategy
	deployment := createTestDeployment("test-deployment", "default", 3, appsv1.RollingUpdateDeploymentStrategyType)
	entry := handler.createLogEntry(deployment)

	// Verify basic fields
	if entry.ResourceType != "deployment" {
		t.Errorf("Expected resource type 'deployment', got '%s'", entry.ResourceType)
	}

	if entry.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got '%s'", entry.Name)
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}

	// Verify data structure
	data := entry.Data

	// Verify deployment-specific fields
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

	val, ok = data["strategyType"]
	if !ok || val == nil {
		t.Fatalf("strategyType missing or nil")
	}
	if val.(string) != "RollingUpdate" {
		t.Errorf("Expected strategy type 'RollingUpdate', got '%s'", val.(string))
	}

	val, ok = data["strategyRollingUpdateMaxSurge"]
	if !ok || val == nil {
		t.Fatalf("strategyRollingUpdateMaxSurge missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected max surge 1, got %d", val.(int32))
	}

	val, ok = data["strategyRollingUpdateMaxUnavailable"]
	if !ok || val == nil {
		t.Fatalf("strategyRollingUpdateMaxUnavailable missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected max unavailable 0, got %d", val.(int32))
	}

	// Verify labels and annotations
	val, ok = data["labels"]
	if !ok || val == nil {
		t.Fatalf("labels missing or nil")
	}
	if val.(map[string]string)["app"] != "test-deployment" {
		t.Errorf("Expected label 'app' to be 'test-deployment', got '%s'", val.(map[string]string)["app"])
	}

	val, ok = data["annotations"]
	if !ok || val == nil {
		t.Fatalf("annotations missing or nil")
	}
	if val.(map[string]string)["description"] != "test deployment" {
		t.Errorf("Expected annotation 'description' to be 'test deployment', got '%s'", val.(map[string]string)["description"])
	}

	// Verify conditions
	val, ok = data["conditionAvailable"]
	if !ok || val == nil {
		t.Fatalf("conditionAvailable missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected Available condition to be true")
	}

	val, ok = data["conditionProgressing"]
	if !ok || val == nil {
		t.Fatalf("conditionProgressing missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected Progressing condition to be true")
	}
}

func TestDeploymentHandler_createLogEntry_RecreateStrategy(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	// Test deployment with recreate strategy
	deployment := createTestDeployment("test-deployment", "default", 2, appsv1.RecreateDeploymentStrategyType)
	entry := handler.createLogEntry(deployment)

	data := entry.Data

	val, ok := data["strategyType"]
	if !ok || val == nil {
		t.Fatalf("strategyType missing or nil")
	}
	if val.(string) != "Recreate" {
		t.Errorf("Expected strategy type 'Recreate', got '%s'", val.(string))
	}

	// Rolling update fields should be 0 for recreate strategy
	val, ok = data["strategyRollingUpdateMaxSurge"]
	if !ok || val == nil {
		t.Fatalf("strategyRollingUpdateMaxSurge missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected max surge 0 for recreate strategy, got %d", val.(int32))
	}

	val, ok = data["strategyRollingUpdateMaxUnavailable"]
	if !ok || val == nil {
		t.Fatalf("strategyRollingUpdateMaxUnavailable missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected max unavailable 0 for recreate strategy, got %d", val.(int32))
	}
}

func TestDeploymentHandler_createLogEntry_NilReplicas(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	// Create deployment with nil replicas (should default to 1)
	deployment := createTestDeployment("test-deployment", "default", 0, appsv1.RollingUpdateDeploymentStrategyType)
	deployment.Spec.Replicas = nil // Set to nil explicitly

	entry := handler.createLogEntry(deployment)
	data := entry.Data

	// Should default to 1 when replicas is nil
	val, ok := data["desiredReplicas"]
	if !ok || val == nil {
		t.Fatalf("desiredReplicas missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected desired replicas 1 when nil, got %d", val.(int32))
	}
}

func TestDeploymentHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	deployment := createTestDeployment("test-deployment", "default", 3, appsv1.RollingUpdateDeploymentStrategyType)

	// Add owner reference
	deployment.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "test-replicaset",
			UID:        "test-uid",
		},
	}

	entry := handler.createLogEntry(deployment)
	data := entry.Data

	val, ok := data["createdByKind"]
	if !ok || val == nil {
		t.Fatalf("createdByKind missing or nil")
	}
	if val.(string) != "ReplicaSet" {
		t.Errorf("Expected created by kind 'ReplicaSet', got '%s'", val.(string))
	}

	val, ok = data["createdByName"]
	if !ok || val == nil {
		t.Fatalf("createdByName missing or nil")
	}
	if val.(string) != "test-replicaset" {
		t.Errorf("Expected created by name 'test-replicaset', got '%s'", val.(string))
	}
}

func TestDeploymentHandler_getConditionStatus(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)

	conditions := []appsv1.DeploymentCondition{
		{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   appsv1.DeploymentProgressing,
			Status: corev1.ConditionFalse,
		},
	}

	// Test available condition
	if !handler.getConditionStatus(conditions, "Available") {
		t.Error("Expected Available condition to be true")
	}

	// Test progressing condition
	if handler.getConditionStatus(conditions, "Progressing") {
		t.Error("Expected Progressing condition to be false")
	}

	// Test non-existent condition
	if handler.getConditionStatus(conditions, "NonExistent") {
		t.Error("Expected non-existent condition to be false")
	}
}

func TestDeploymentHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(client)
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

func TestDeploymentHandler_Collect_NamespaceFiltering(t *testing.T) {
	// Create test deployments in different namespaces
	deployment1 := createTestDeployment("test-deployment-1", "default", 3, appsv1.RollingUpdateDeploymentStrategyType)
	deployment2 := createTestDeployment("test-deployment-2", "kube-system", 2, appsv1.RecreateDeploymentStrategyType)
	deployment3 := createTestDeployment("test-deployment-3", "monitoring", 1, appsv1.RollingUpdateDeploymentStrategyType)

	client := fake.NewSimpleClientset(deployment1, deployment2, deployment3)
	handler := NewDeploymentHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}

	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	ctx := context.Background()

	// Test multiple namespace filtering
	entries, err := handler.Collect(ctx, []string{"default", "monitoring"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for default and monitoring namespaces, got %d", len(entries))
	}

	// Verify correct namespaces
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
