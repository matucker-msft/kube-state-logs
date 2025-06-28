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
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestPodWithContainers creates a test Pod with containers
func createTestPodWithContainers(name, namespace string) *corev1.Pod {
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
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "nginx:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
				{
					Name:  "sidecar",
					Image: "busybox:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init",
					Image: "busybox:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("25m"),
							corev1.ResourceMemory: resource.MustParse("32Mi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "main",
					Ready:        true,
					RestartCount: 0,
					Image:        "nginx:latest",
					ImageID:      "docker-pullable://nginx@sha256:123456",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: now,
						},
					},
				},
				{
					Name:         "sidecar",
					Ready:        true,
					RestartCount: 1,
					Image:        "busybox:latest",
					ImageID:      "docker-pullable://busybox@sha256:654321",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: now,
						},
					},
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "init",
					Ready:        false,
					RestartCount: 0,
					Image:        "busybox:latest",
					ImageID:      "docker-pullable://busybox@sha256:654321",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   0,
							Reason:     "Completed",
							StartedAt:  now,
							FinishedAt: now,
						},
					},
				},
			},
		},
	}
	return pod
}

func TestNewContainerHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
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
	if handler.GetInformer() == nil {
		t.Error("Expected informer to be set up")
	}
}

func TestContainerHandler_Collect(t *testing.T) {
	pod1 := createTestPodWithContainers("test-pod-1", "default")
	pod2 := createTestPodWithContainers("test-pod-2", "kube-system")
	client := fake.NewSimpleClientset(pod1, pod2)
	handler := NewContainerHandler(client)
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
	// Each pod has 2 containers + 1 init container = 3 containers per pod
	if len(entries) != 6 {
		t.Fatalf("Expected 6 entries (3 containers per pod), got %d", len(entries))
	}
	entries, err = handler.Collect(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries for default namespace, got %d", len(entries))
	}
	if entries[0].Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entries[0].Namespace)
	}
}

func TestContainerHandler_createContainerLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
	pod := createTestPodWithContainers("test-pod", "default")
	container := pod.Spec.Containers[0]
	entry := handler.createContainerLogEntry(pod, &container, false)
	if entry.ResourceType != "container" {
		t.Errorf("Expected resource type 'container', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-pod-main" {
		t.Errorf("Expected name 'test-pod-main', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["name"]
	if !ok || val == nil {
		t.Fatalf("name missing or nil")
	}
	if val.(string) != "main" {
		t.Errorf("Expected container name 'main', got '%s'", val.(string))
	}
	val, ok = data["image"]
	if !ok || val == nil {
		t.Fatalf("image missing or nil")
	}
	if val.(string) != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", val.(string))
	}
	val, ok = data["podName"]
	if !ok || val == nil {
		t.Fatalf("podName missing or nil")
	}
	if val.(string) != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", val.(string))
	}
	val, ok = data["ready"]
	if !ok || val == nil {
		t.Fatalf("ready missing or nil")
	}
	if val.(bool) != true {
		t.Errorf("Expected ready true, got %t", val.(bool))
	}
	val, ok = data["state"]
	if !ok || val == nil {
		t.Fatalf("state missing or nil")
	}
	if val.(string) != "running" {
		t.Errorf("Expected state 'running', got '%s'", val.(string))
	}
	val, ok = data["restartCount"]
	if !ok || val == nil {
		t.Fatalf("restartCount missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected restart count 0, got %d", val.(int32))
	}
}

func TestContainerHandler_createContainerLogEntry_InitContainer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
	pod := createTestPodWithContainers("test-pod", "default")
	container := pod.Spec.InitContainers[0]
	entry := handler.createContainerLogEntry(pod, &container, true)
	if entry.ResourceType != "container" {
		t.Errorf("Expected resource type 'container', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-pod-init" {
		t.Errorf("Expected name 'test-pod-init', got '%s'", entry.Name)
	}
	data := entry.Data
	val, ok := data["name"]
	if !ok || val == nil {
		t.Fatalf("name missing or nil")
	}
	if val.(string) != "init" {
		t.Errorf("Expected container name 'init', got '%s'", val.(string))
	}
	val, ok = data["state"]
	if !ok || val == nil {
		t.Fatalf("state missing or nil")
	}
	if val.(string) != "terminated" {
		t.Errorf("Expected state 'terminated', got '%s'", val.(string))
	}
	val, ok = data["exitCode"]
	if !ok || val == nil {
		t.Fatalf("exitCode missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected exit code 0, got %d", val.(int32))
	}
}

func TestContainerHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewContainerHandler(client)
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

func TestContainerHandler_Collect_NamespaceFiltering(t *testing.T) {
	pod1 := createTestPodWithContainers("test-pod-1", "default")
	pod2 := createTestPodWithContainers("test-pod-2", "kube-system")
	pod3 := createTestPodWithContainers("test-pod-3", "monitoring")
	client := fake.NewSimpleClientset(pod1, pod2, pod3)
	handler := NewContainerHandler(client)
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
	// 2 pods * 3 containers each = 6 entries
	if len(entries) != 6 {
		t.Fatalf("Expected 6 entries for default and monitoring namespaces, got %d", len(entries))
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
