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

// createTestNode creates a test Node with various configurations
func createTestNode(name string) *corev1.Node {
	now := metav1.Now()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubernetes.io/hostname":         name,
				"node-role.kubernetes.io/worker": "",
			},
			Annotations: map[string]string{
				"description": "test node",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
			Taints: []corev1.Taint{
				{
					Key:    "node.kubernetes.io/not-ready",
					Value:  "true",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
		},
		Status: corev1.NodeStatus{
			Phase: corev1.NodeRunning,
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.100",
				},
				{
					Type:    corev1.NodeExternalIP,
					Address: "203.0.113.1",
				},
				{
					Type:    corev1.NodeHostName,
					Address: name,
				},
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionFalse,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				Architecture:            "amd64",
				OperatingSystem:         "linux",
				KernelVersion:           "5.4.0",
				KubeletVersion:          "v1.24.0",
				KubeProxyVersion:        "v1.24.0",
				ContainerRuntimeVersion: "containerd://1.6.0",
			},
		},
	}
	return node
}

func TestNewNodeHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestNodeHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
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

func TestNodeHandler_Collect(t *testing.T) {
	node1 := createTestNode("test-node-1")
	node2 := createTestNode("test-node-2")
	client := fake.NewSimpleClientset(node1, node2)
	handler := NewNodeHandler(client)
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
}

func TestNodeHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
	node := createTestNode("test-node")
	entry := handler.createLogEntry(node)
	if entry.ResourceType != "node" {
		t.Errorf("Expected resource type 'node', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-node" {
		t.Errorf("Expected name 'test-node', got '%s'", entry.Name)
	}
	if entry.Namespace != "" {
		t.Errorf("Expected empty namespace for node, got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["architecture"]
	if !ok || val == nil {
		t.Fatalf("architecture missing or nil")
	}
	if val.(string) != "amd64" {
		t.Errorf("Expected architecture 'amd64', got '%s'", val.(string))
	}
	val, ok = data["operatingSystem"]
	if !ok || val == nil {
		t.Fatalf("operatingSystem missing or nil")
	}
	if val.(string) != "linux" {
		t.Errorf("Expected operating system 'linux', got '%s'", val.(string))
	}
	val, ok = data["internalIP"]
	if !ok || val == nil {
		t.Fatalf("internalIP missing or nil")
	}
	if val.(string) != "192.168.1.100" {
		t.Errorf("Expected internal IP '192.168.1.100', got '%s'", val.(string))
	}
	val, ok = data["externalIP"]
	if !ok || val == nil {
		t.Fatalf("externalIP missing or nil")
	}
	if val.(string) != "203.0.113.1" {
		t.Errorf("Expected external IP '203.0.113.1', got '%s'", val.(string))
	}
	val, ok = data["hostname"]
	if !ok || val == nil {
		t.Fatalf("hostname missing or nil")
	}
	if val.(string) != "test-node" {
		t.Errorf("Expected hostname 'test-node', got '%s'", val.(string))
	}
	val, ok = data["unschedulable"]
	if !ok || val == nil {
		t.Fatalf("unschedulable missing or nil")
	}
	if val.(bool) != false {
		t.Errorf("Expected unschedulable false, got %t", val.(bool))
	}
	val, ok = data["ready"]
	if !ok || val == nil {
		t.Fatalf("ready missing or nil")
	}
	if val.(bool) != true {
		t.Errorf("Expected ready true, got %t", val.(bool))
	}
	val, ok = data["role"]
	if !ok || val == nil {
		t.Fatalf("role missing or nil")
	}
	if val.(string) != "worker" {
		t.Errorf("Expected role 'worker', got '%s'", val.(string))
	}
	val, ok = data["phase"]
	if !ok || val == nil {
		t.Fatalf("phase missing or nil")
	}
	if val.(string) != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", val.(string))
	}
}

func TestNodeHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
	node := createTestNode("test-node")
	node.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "machine.openshift.io/v1beta1",
			Kind:       "Machine",
			Name:       "test-machine",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(node)
	data := entry.Data
	val, ok := data["createdByKind"]
	if !ok || val == nil {
		t.Fatalf("createdByKind missing or nil")
	}
	if val.(string) != "Machine" {
		t.Errorf("Expected created by kind 'Machine', got '%s'", val.(string))
	}
	val, ok = data["createdByName"]
	if !ok || val == nil {
		t.Fatalf("createdByName missing or nil")
	}
	if val.(string) != "test-machine" {
		t.Errorf("Expected created by name 'test-machine', got '%s'", val.(string))
	}
}

func TestNodeHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
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

func TestNodeHandler_createLogEntry_WithTaints(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNodeHandler(client)
	node := createTestNode("test-node")

	// Add taints
	node.Spec.Taints = []corev1.Taint{
		{
			Key:    "node-role.kubernetes.io/master",
			Value:  "",
			Effect: corev1.TaintEffectNoSchedule,
		},
		{
			Key:    "dedicated",
			Value:  "gpu",
			Effect: corev1.TaintEffectPreferNoSchedule,
		},
	}

	entry := handler.createLogEntry(node)

	// Verify taints are included
	data := entry.Data
	taintsVal, ok := data["taints"]
	if !ok || taintsVal == nil {
		t.Fatalf("taints missing or nil")
	}
	taints := taintsVal.([]types.TaintData)
	if len(taints) != 2 {
		t.Errorf("Expected 2 taints, got %d", len(taints))
	}

	// Check first taint
	if taints[0].Key != "node-role.kubernetes.io/master" {
		t.Errorf("Expected taint key 'node-role.kubernetes.io/master', got %s", taints[0].Key)
	}
	if taints[0].Effect != "NoSchedule" {
		t.Errorf("Expected taint effect 'NoSchedule', got %s", taints[0].Effect)
	}
}
