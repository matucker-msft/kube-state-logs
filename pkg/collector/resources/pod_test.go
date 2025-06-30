package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// createTestPod creates a test pod with various configurations
func createTestPod(name, namespace string, phase corev1.PodPhase) *corev1.Pod {
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test pod",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
			RestartPolicy:      corev1.RestartPolicyAlways,
			ServiceAccountName: "default",
			SchedulerName:      "default-scheduler",
		},
		Status: corev1.PodStatus{
			Phase:     phase,
			HostIP:    "192.168.1.1",
			PodIP:     "10.0.0.1",
			StartTime: &now,
			QOSClass:  corev1.PodQOSBurstable,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: now,
						},
					},
				},
			},
			Conditions: []corev1.PodCondition{
				{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "PodReady",
					Message:            "Pod is ready",
				},
				{
					Type:               corev1.PodInitialized,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "PodInitialized",
					Message:            "Pod is initialized",
				},
				{
					Type:               corev1.PodScheduled,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "PodScheduled",
					Message:            "Pod is scheduled",
				},
				{
					Type:               corev1.ContainersReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "ContainersReady",
					Message:            "All containers are ready",
				},
			},
		},
	}

	return pod
}

func TestNewPodHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
}

func TestPodHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)
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

func TestPodHandler_Collect(t *testing.T) {
	// Create test pods
	pod1 := createTestPod("test-pod-1", "default", corev1.PodRunning)
	pod2 := createTestPod("test-pod-2", "kube-system", corev1.PodPending)

	// Create fake client with test pods
	client := fake.NewSimpleClientset(pod1, pod2)
	handler := NewPodHandler(client)
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

	// Test collecting all pods
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

	// Type assert to PodData for assertions
	entry, ok := entries[0].(types.PodData)
	if !ok {
		t.Fatalf("Expected PodData type, got %T", entries[0])
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
}

func TestPodHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)
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
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}

func TestPodHandler_Collect_NamespaceFiltering(t *testing.T) {
	// Create test pods in different namespaces
	pod1 := createTestPod("test-pod-1", "default", corev1.PodRunning)
	pod2 := createTestPod("test-pod-2", "kube-system", corev1.PodRunning)
	pod3 := createTestPod("test-pod-3", "monitoring", corev1.PodRunning)

	client := fake.NewSimpleClientset(pod1, pod2, pod3)
	handler := NewPodHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}

	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	ctx := context.Background()

	// Test filtering by specific namespace
	entries, err := handler.Collect(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for default namespace, got %d", len(entries))
	}

	// Type assert to PodData for assertions
	entry, ok := entries[0].(types.PodData)
	if !ok {
		t.Fatalf("Expected PodData type, got %T", entries[0])
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}

	// Test filtering by multiple namespaces
	entries, err = handler.Collect(ctx, []string{"default", "kube-system"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for default and kube-system namespaces, got %d", len(entries))
	}

	// Verify both namespaces are present
	namespaces := make(map[string]bool)
	for _, entry := range entries {
		podData, ok := entry.(types.PodData)
		if !ok {
			t.Fatalf("Expected PodData type, got %T", entry)
		}
		namespaces[podData.Namespace] = true
	}

	if !namespaces["default"] {
		t.Error("Expected to find pod in default namespace")
	}
	if !namespaces["kube-system"] {
		t.Error("Expected to find pod in kube-system namespace")
	}
}
