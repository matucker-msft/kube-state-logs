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
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestContainer creates a test container with various configurations
func createTestContainer(name, image string, ready bool) *corev1.Container {
	container := &corev1.Container{
		Name:  name,
		Image: image,
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
	}

	return container
}

// createTestPodWithContainers creates a test pod with containers
func createTestPodWithContainers(name, namespace string, containers []corev1.Container) *corev1.Pod {
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
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
		Status: corev1.PodStatus{
			ContainerStatuses: make([]corev1.ContainerStatus, len(containers)),
		},
	}

	// Populate container statuses
	for i, container := range containers {
		pod.Status.ContainerStatuses[i] = corev1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			ImageID:      "docker://sha256:test",
			Ready:        true,
			RestartCount: 0,
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Now(),
				},
			},
		}
	}

	return pod
}

func TestNewContainerHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}

	// Verify BaseHandler is embedded
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestContainerHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
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

func TestContainerHandler_Collect(t *testing.T) {
	// Create test containers
	container1 := createTestContainer("app", "nginx:latest", true)
	container2 := createTestContainer("sidecar", "busybox:latest", true)

	// Create test pods with containers
	pod1 := createTestPodWithContainers("test-pod-1", "default", []corev1.Container{*container1})
	pod2 := createTestPodWithContainers("test-pod-2", "kube-system", []corev1.Container{*container2})

	// Create fake client with test pods
	client := fake.NewSimpleClientset(pod1, pod2)
	handler := NewContainerHandler(client)
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

	// Test collecting all containers
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

	// Type assert to ContainerData for assertions
	entry, ok := entries[0].(types.ContainerData)
	if !ok {
		t.Fatalf("Expected ContainerData type, got %T", entries[0])
	}

	if entry.PodName != "test-pod-1" {
		t.Errorf("Expected pod name 'test-pod-1', got '%s'", entry.PodName)
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
}

func TestContainerHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
	container := createTestContainer("app", "nginx:latest", true)
	pod := createTestPodWithContainers("test-pod", "default", []corev1.Container{*container})
	entry := handler.createLogEntry(pod, &pod.Status.ContainerStatuses[0], false)

	if entry.Name != "app" {
		t.Errorf("Expected name 'app', got '%s'", entry.Name)
	}

	if entry.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", entry.Image)
	}

	if entry.PodName != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", entry.PodName)
	}

	// Verify container-specific fields
	if entry.Ready == nil || !*entry.Ready {
		t.Error("Expected container to be ready")
	}

	if entry.RestartCount != 0 {
		t.Errorf("Expected restart count 0, got %d", entry.RestartCount)
	}

	if entry.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", entry.State)
	}

	if entry.StateRunning == nil || !*entry.StateRunning {
		t.Error("Expected StateRunning to be true")
	}

	if entry.StateWaiting != nil && *entry.StateWaiting {
		t.Error("Expected StateWaiting to be false")
	}

	if entry.StateTerminated != nil && *entry.StateTerminated {
		t.Error("Expected StateTerminated to be false")
	}

	// Verify resource requests
	if entry.ResourceRequests["cpu"] != "100m" {
		t.Errorf("Expected CPU request '100m', got '%s'", entry.ResourceRequests["cpu"])
	}

	if entry.ResourceRequests["memory"] != "128Mi" {
		t.Errorf("Expected memory request '128Mi', got '%s'", entry.ResourceRequests["memory"])
	}

	// Verify resource limits
	if entry.ResourceLimits["cpu"] != "200m" {
		t.Errorf("Expected CPU limit '200m', got '%s'", entry.ResourceLimits["cpu"])
	}

	if entry.ResourceLimits["memory"] != "256Mi" {
		t.Errorf("Expected memory limit '256Mi', got '%s'", entry.ResourceLimits["memory"])
	}
}

func TestContainerHandler_createLogEntry_Waiting(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
	container := createTestContainer("app", "nginx:latest", false)
	pod := createTestPodWithContainers("test-pod", "default", []corev1.Container{*container})

	// Set container status to waiting
	pod.Status.ContainerStatuses[0].State = corev1.ContainerState{
		Waiting: &corev1.ContainerStateWaiting{
			Reason:  "ImagePullBackOff",
			Message: "Back-off pulling image",
		},
	}
	pod.Status.ContainerStatuses[0].Ready = false

	entry := handler.createLogEntry(pod, &pod.Status.ContainerStatuses[0], false)

	if entry.State != "waiting" {
		t.Errorf("Expected state 'waiting', got '%s'", entry.State)
	}

	if entry.StateWaiting == nil || !*entry.StateWaiting {
		t.Error("Expected StateWaiting to be true")
	}

	if entry.WaitingReason != "ImagePullBackOff" {
		t.Errorf("Expected waiting reason 'ImagePullBackOff', got '%s'", entry.WaitingReason)
	}

	if entry.WaitingMessage != "Back-off pulling image" {
		t.Errorf("Expected waiting message 'Back-off pulling image', got '%s'", entry.WaitingMessage)
	}

	if entry.Ready != nil && *entry.Ready {
		t.Error("Expected container to not be ready")
	}
}

func TestContainerHandler_Collect_NamespaceFiltering(t *testing.T) {
	// Create test containers
	container1 := createTestContainer("app", "nginx:latest", true)
	container2 := createTestContainer("sidecar", "busybox:latest", true)

	// Create test pods with containers
	pod1 := createTestPodWithContainers("test-pod-1", "default", []corev1.Container{*container1})
	pod2 := createTestPodWithContainers("test-pod-2", "kube-system", []corev1.Container{*container2})

	// Create fake client with test pods
	client := fake.NewSimpleClientset(pod1, pod2)
	handler := NewContainerHandler(client)
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

	// Test collecting from specific namespace
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for default namespace, got %d", len(entries))
	}

	// Type assert to ContainerData for assertions
	entry, ok := entries[0].(types.ContainerData)
	if !ok {
		t.Fatalf("Expected ContainerData type, got %T", entries[0])
	}

	if entry.PodName != "test-pod-1" {
		t.Errorf("Expected pod name 'test-pod-1', got '%s'", entry.PodName)
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
}

func TestContainerHandler_InitContainerResources(t *testing.T) {
	// Create test init container
	initContainer := createTestContainer("init", "busybox:latest", true)
	regularContainer := createTestContainer("app", "nginx:latest", true)

	// Create test pod with both init and regular containers
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{*initContainer},
			Containers:     []corev1.Container{*regularContainer},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "init",
					Image:        "busybox:latest",
					ImageID:      "docker://sha256:init",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					Image:        "nginx:latest",
					ImageID:      "docker://sha256:app",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
				},
			},
		},
	}

	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)

	// Test init container resource extraction
	initEntry := handler.createLogEntry(pod, &pod.Status.InitContainerStatuses[0], true)
	if initEntry.Name != "init" {
		t.Errorf("Expected init container name 'init', got '%s'", initEntry.Name)
	}
	if len(initEntry.ResourceRequests) == 0 {
		t.Error("Expected init container to have resource requests")
	}
	if len(initEntry.ResourceLimits) == 0 {
		t.Error("Expected init container to have resource limits")
	}

	// Test regular container resource extraction
	regularEntry := handler.createLogEntry(pod, &pod.Status.ContainerStatuses[0], false)
	if regularEntry.Name != "app" {
		t.Errorf("Expected regular container name 'app', got '%s'", regularEntry.Name)
	}
	if len(regularEntry.ResourceRequests) == 0 {
		t.Error("Expected regular container to have resource requests")
	}
	if len(regularEntry.ResourceLimits) == 0 {
		t.Error("Expected regular container to have resource limits")
	}
}
