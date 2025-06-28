package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestService creates a test Service with various configurations
func createTestService(name, namespace string, serviceType corev1.ServiceType) *corev1.Service {
	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test service",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					NodePort:   30080,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			ClusterIP: "10.0.0.1",
		},
	}
	return service
}

// createTestEndpoints creates test Endpoints for a service
func createTestEndpoints(name, namespace string, addresses int) *corev1.Endpoints {
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: make([]corev1.EndpointAddress, addresses),
				Ports: []corev1.EndpointPort{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
		},
	}
	return endpoints
}

func TestNewServiceHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestServiceHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if handler.GetInformer() == nil {
		t.Error("Expected informer to be set up")
	}
	if handler.endpointsInformer == nil {
		t.Error("Expected endpoints informer to be set up")
	}
}

func TestServiceHandler_Collect(t *testing.T) {
	service1 := createTestService("test-service-1", "default", corev1.ServiceTypeClusterIP)
	service2 := createTestService("test-service-2", "kube-system", corev1.ServiceTypeNodePort)
	client := fake.NewSimpleClientset(service1, service2)
	handler := NewServiceHandler(client)
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

func TestServiceHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
	service := createTestService("test-service", "default", corev1.ServiceTypeClusterIP)
	entry := handler.createLogEntry(service)
	if entry.ResourceType != "service" {
		t.Errorf("Expected resource type 'service', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["type"]
	if !ok || val == nil {
		t.Fatalf("type missing or nil")
	}
	if val.(string) != "ClusterIP" {
		t.Errorf("Expected type 'ClusterIP', got '%s'", val.(string))
	}
	val, ok = data["clusterIP"]
	if !ok || val == nil {
		t.Fatalf("clusterIP missing or nil")
	}
	if val.(string) != "10.0.0.1" {
		t.Errorf("Expected cluster IP '10.0.0.1', got '%s'", val.(string))
	}
	val, ok = data["endpointsCount"]
	if !ok || val == nil {
		t.Fatalf("endpointsCount missing or nil")
	}
	if val.(int) != 0 {
		t.Errorf("Expected endpoints count 0, got %d", val.(int))
	}
}

func TestServiceHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
	service := createTestService("test-service", "default", corev1.ServiceTypeClusterIP)
	service.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deploy",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(service)
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

func TestServiceHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
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

func TestServiceHandler_Collect_NamespaceFiltering(t *testing.T) {
	service1 := createTestService("test-service-1", "default", corev1.ServiceTypeClusterIP)
	service2 := createTestService("test-service-2", "kube-system", corev1.ServiceTypeNodePort)
	service3 := createTestService("test-service-3", "monitoring", corev1.ServiceTypeLoadBalancer)
	client := fake.NewSimpleClientset(service1, service2, service3)
	handler := NewServiceHandler(client)
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

func TestServiceHandler_countEndpointsForService(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	// Create endpoints before starting informers
	endpoints := createTestEndpoints("test-service", "default", 3)
	_, err = client.CoreV1().Endpoints("default").Create(context.Background(), endpoints, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create endpoints: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	// Test with endpoints
	count := handler.countEndpointsForService("default", "test-service")
	if count != 3 {
		t.Errorf("Expected 3 endpoints, got %d", count)
	}
	// Test with no endpoints
	count = handler.countEndpointsForService("default", "nonexistent-service")
	if count != 0 {
		t.Errorf("Expected 0 endpoints, got %d", count)
	}
}
