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

	// Verify BaseHandler is embedded
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
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

	if entries[0].Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entries[0].Namespace)
	}
}

func TestPodHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)

	// Test pod with running status
	pod := createTestPod("test-pod", "default", corev1.PodRunning)
	entry := handler.createPodLogEntry(pod)

	// Verify basic fields
	if entry.ResourceType != "pod" {
		t.Errorf("Expected resource type 'pod', got '%s'", entry.ResourceType)
	}

	if entry.Name != "test-pod" {
		t.Errorf("Expected name 'test-pod', got '%s'", entry.Name)
	}

	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}

	// Verify data structure
	data := entry.Data

	// Verify pod-specific fields
	val, ok := data["nodeName"]
	if !ok || val == nil {
		t.Fatalf("nodeName missing or nil")
	}
	if val.(string) != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", val.(string))
	}

	val, ok = data["hostIP"]
	if !ok || val == nil {
		t.Fatalf("hostIP missing or nil")
	}
	if val.(string) != "192.168.1.1" {
		t.Errorf("Expected host IP '192.168.1.1', got '%s'", val.(string))
	}

	val, ok = data["podIP"]
	if !ok || val == nil {
		t.Fatalf("podIP missing or nil")
	}
	if val.(string) != "10.0.0.1" {
		t.Errorf("Expected pod IP '10.0.0.1', got '%s'", val.(string))
	}

	val, ok = data["phase"]
	if !ok || val == nil {
		t.Fatalf("phase missing or nil")
	}
	if val.(string) != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", val.(string))
	}

	val, ok = data["qosClass"]
	if !ok || val == nil {
		t.Fatalf("qosClass missing or nil")
	}
	if val.(string) != "Burstable" {
		t.Errorf("Expected QoS class 'Burstable', got '%s'", val.(string))
	}

	// Verify conditions
	val, ok = data["ready"]
	if !ok || val == nil {
		t.Fatalf("ready missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected ready condition to be true")
	}

	val, ok = data["initialized"]
	if !ok || val == nil {
		t.Fatalf("initialized missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected initialized condition to be true")
	}

	val, ok = data["scheduled"]
	if !ok || val == nil {
		t.Fatalf("scheduled missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected scheduled condition to be true")
	}

	// Verify labels and annotations
	val, ok = data["labels"]
	if !ok || val == nil {
		t.Fatalf("labels missing or nil")
	}
	if val.(map[string]string)["app"] != "test-pod" {
		t.Errorf("Expected label 'app' to be 'test-pod', got '%s'", val.(map[string]string)["app"])
	}

	val, ok = data["annotations"]
	if !ok || val == nil {
		t.Fatalf("annotations missing or nil")
	}
	if val.(map[string]string)["description"] != "test pod" {
		t.Errorf("Expected annotation 'description' to be 'test pod', got '%s'", val.(map[string]string)["description"])
	}
}

func TestPodHandler_createLogEntry_PendingPod(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)

	// Test pod with pending status and unschedulable condition
	pod := createTestPod("test-pod", "default", corev1.PodPending)
	pod.Status.Conditions = []corev1.PodCondition{
		{
			Type:               corev1.PodScheduled,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "Unschedulable",
			Message:            "Pod is unschedulable",
		},
	}

	entry := handler.createPodLogEntry(pod)
	data := entry.Data

	val, ok := data["phase"]
	if !ok || val == nil {
		t.Fatalf("phase missing or nil")
	}
	if val.(string) != "Pending" {
		t.Errorf("Expected phase 'Pending', got '%s'", val.(string))
	}

	val, ok = data["scheduled"]
	if !ok || val == nil {
		t.Fatalf("scheduled missing or nil")
	}
	if val.(bool) {
		t.Error("Expected scheduled condition to be false")
	}

	val, ok = data["unschedulable"]
	if !ok || val == nil {
		t.Fatalf("unschedulable missing or nil")
	}
	if !val.(bool) {
		t.Error("Expected unschedulable to be true")
	}
}

func TestPodHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)

	pod := createTestPod("test-pod", "default", corev1.PodRunning)

	// Add owner reference
	pod.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "test-replicaset",
			UID:        "test-uid",
		},
	}

	entry := handler.createPodLogEntry(pod)
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

func TestPodHandler_createLogEntry_WithTolerations(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)

	pod := createTestPod("test-pod", "default", corev1.PodRunning)

	// Add tolerations
	pod.Spec.Tolerations = []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/master",
			Value:    "",
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpExists,
		},
		{
			Key:               "node.kubernetes.io/not-ready",
			Value:             "",
			Effect:            corev1.TaintEffectNoExecute,
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &[]int64{300}[0],
		},
	}

	entry := handler.createPodLogEntry(pod)
	data := entry.Data

	val, ok := data["tolerations"]
	if !ok || val == nil {
		t.Fatalf("tolerations missing or nil")
	}

	tolerations := val.([]types.TolerationData)
	if len(tolerations) != 2 {
		t.Fatalf("Expected 2 tolerations, got %d", len(tolerations))
	}

	// Check first toleration
	if tolerations[0].Key != "node-role.kubernetes.io/master" {
		t.Errorf("Expected first toleration key 'node-role.kubernetes.io/master', got '%s'", tolerations[0].Key)
	}
	if tolerations[0].Effect != "NoSchedule" {
		t.Errorf("Expected first toleration effect 'NoSchedule', got '%s'", tolerations[0].Effect)
	}

	// Check second toleration
	if tolerations[1].Key != "node.kubernetes.io/not-ready" {
		t.Errorf("Expected second toleration key 'node.kubernetes.io/not-ready', got '%s'", tolerations[1].Key)
	}
	if tolerations[1].TolerationSeconds != "300" {
		t.Errorf("Expected second toleration seconds '300', got '%s'", tolerations[1].TolerationSeconds)
	}
}

func TestPodHandler_createLogEntry_WithPVCs(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)
	pod := createTestPod("test-pod", "default", corev1.PodRunning)

	// Add PVC volumes
	pod.Spec.Volumes = []corev1.Volume{
		{
			Name: "data-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "data-pvc",
				},
			},
		},
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "config-pvc",
				},
			},
		},
	}

	// Add volume mounts with read-only flag
	pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "data-volume",
			MountPath: "/data",
			ReadOnly:  false,
		},
		{
			Name:      "config-volume",
			MountPath: "/config",
			ReadOnly:  true,
		},
	}

	entry := handler.createPodLogEntry(pod)

	// Verify PVCs are included
	data := entry.Data
	pvcsVal, ok := data["persistentVolumeClaims"]
	if !ok || pvcsVal == nil {
		t.Fatalf("persistentVolumeClaims missing or nil")
	}
	pvcs := pvcsVal.([]types.PVCData)
	if len(pvcs) != 2 {
		t.Errorf("Expected 2 PVCs, got %d", len(pvcs))
	}

	// Check first PVC
	if pvcs[0].ClaimName != "data-pvc" {
		t.Errorf("Expected claim name 'data-pvc', got %s", pvcs[0].ClaimName)
	}
	if pvcs[0].ReadOnly != false {
		t.Errorf("Expected readOnly false, got %t", pvcs[0].ReadOnly)
	}

	// Check second PVC
	if pvcs[1].ClaimName != "config-pvc" {
		t.Errorf("Expected claim name 'config-pvc', got %s", pvcs[1].ClaimName)
	}
	if pvcs[1].ReadOnly != true {
		t.Errorf("Expected readOnly true, got %t", pvcs[1].ReadOnly)
	}
}

func TestPodHandler_createLogEntry_WithResourceArithmetic(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodHandler(client)
	pod := createTestPod("test-pod", "default", corev1.PodRunning)

	// Add multiple containers with overlapping resource requests and limits
	pod.Spec.Containers = []corev1.Container{
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
		{
			Name:  "sidecar",
			Image: "busybox:latest",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
		{
			Name:  "init",
			Image: "alpine:latest",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("25m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		},
	}

	entry := handler.createPodLogEntry(pod)

	// Verify resource arithmetic is working correctly
	data := entry.Data

	// Check resource requests (should be summed: 100m + 50m + 25m = 175m)
	requestsVal, ok := data["resourceRequests"]
	if !ok || requestsVal == nil {
		t.Fatalf("resourceRequests missing or nil")
	}
	requests := requestsVal.(map[string]string)

	cpuRequest, exists := requests["cpu"]
	if !exists {
		t.Fatal("CPU request missing")
	}
	if cpuRequest != "175m" {
		t.Errorf("Expected CPU request '175m', got %s", cpuRequest)
	}

	memoryRequest, exists := requests["memory"]
	if !exists {
		t.Fatal("Memory request missing")
	}
	if memoryRequest != "224Mi" {
		t.Errorf("Expected Memory request '224Mi', got %s", memoryRequest)
	}

	// Check resource limits (should be summed: 200m + 100m + 50m = 350m)
	limitsVal, ok := data["resourceLimits"]
	if !ok || limitsVal == nil {
		t.Fatalf("resourceLimits missing or nil")
	}
	limits := limitsVal.(map[string]string)

	cpuLimit, exists := limits["cpu"]
	if !exists {
		t.Fatal("CPU limit missing")
	}
	if cpuLimit != "350m" {
		t.Errorf("Expected CPU limit '350m', got %s", cpuLimit)
	}

	memoryLimit, exists := limits["memory"]
	if !exists {
		t.Fatal("Memory limit missing")
	}
	if memoryLimit != "448Mi" {
		t.Errorf("Expected Memory limit '448Mi', got %s", memoryLimit)
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
		t.Fatalf("Expected 0 entries for empty cache, got %d", len(entries))
	}
}

func TestPodHandler_Collect_NamespaceFiltering(t *testing.T) {
	// Create test pods in different namespaces
	pod1 := createTestPod("test-pod-1", "default", corev1.PodRunning)
	pod2 := createTestPod("test-pod-2", "kube-system", corev1.PodPending)
	pod3 := createTestPod("test-pod-3", "monitoring", corev1.PodSucceeded)

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
