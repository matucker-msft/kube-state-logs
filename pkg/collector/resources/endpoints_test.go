package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestEndpointsHandler(t *testing.T) {
	endpoints1 := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "service1",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "10.0.0.1",
						Hostname: "pod1",
						NodeName: stringPtr("node1"),
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "pod1",
						},
					},
					{
						IP:       "10.0.0.2",
						Hostname: "pod2",
						NodeName: stringPtr("node2"),
						TargetRef: &corev1.ObjectReference{
							Kind: "Pod",
							Name: "pod2",
						},
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "http",
						Protocol: corev1.ProtocolTCP,
						Port:     80,
					},
					{
						Name:     "https",
						Protocol: corev1.ProtocolTCP,
						Port:     443,
					},
				},
			},
		},
	}

	endpoints2 := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "service2",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "10.0.0.3",
						Hostname: "pod3",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Protocol: corev1.ProtocolUDP,
						Port:     53,
					},
				},
			},
		},
	}

	endpointsWithOwner := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-service",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Service", Name: "my-service"}},
			CreationTimestamp: metav1.Now(),
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "10.0.0.4",
						Hostname: "pod4",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "api",
						Protocol: corev1.ProtocolTCP,
						Port:     8080,
					},
				},
			},
		},
	}

	endpointsEmpty := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "empty-service",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Subsets: []corev1.EndpointSubset{},
	}

	tests := []struct {
		name           string
		endpoints      []*corev1.Endpoints
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all endpoints",
			endpoints:     []*corev1.Endpoints{endpoints1, endpoints2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"service1", "service2"},
		},
		{
			name:          "collect endpoints from specific namespace",
			endpoints:     []*corev1.Endpoints{endpoints1, endpoints2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"service1"},
		},
		{
			name:          "collect endpoints with owner reference",
			endpoints:     []*corev1.Endpoints{endpointsWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-service"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Service",
				"created_by_name": "my-service",
			},
		},
		{
			name:          "collect endpoints with multiple addresses and ports",
			endpoints:     []*corev1.Endpoints{endpoints1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"service1"},
			expectedFields: map[string]interface{}{
				"ready": true,
			},
		},
		{
			name:          "collect empty endpoints",
			endpoints:     []*corev1.Endpoints{endpointsEmpty},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"empty-service"},
			expectedFields: map[string]interface{}{
				"ready": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.endpoints))
			for i, ep := range tt.endpoints {
				objects[i] = ep
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewEndpointsHandler(client)
			factory := informers.NewSharedInformerFactory(client, time.Hour)
			err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
			if err != nil {
				t.Fatalf("Failed to setup informer: %v", err)
			}
			factory.Start(context.Background().Done())
			if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
				t.Fatal("Failed to sync cache")
			}
			entries, err := handler.Collect(context.Background(), tt.namespaces)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}
			if len(entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries))
			}
			entryNames := make([]string, len(entries))
			for i, entry := range entries {
				entryNames[i] = entry.Name
			}
			for _, expectedName := range tt.expectedNames {
				found := false
				for _, name := range entryNames {
					if name == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find endpoints with name %s", expectedName)
				}
			}
			if tt.expectedFields != nil && len(entries) > 0 {
				entry := entries[0]
				for field, expectedValue := range tt.expectedFields {
					switch field {
					case "created_by_kind":
						if entry.Data["createdByKind"] != expectedValue.(string) {
							t.Errorf("Expected created_by_kind %s, got %v", expectedValue, entry.Data["createdByKind"])
						}
					case "created_by_name":
						if entry.Data["createdByName"] != expectedValue.(string) {
							t.Errorf("Expected created_by_name %s, got %v", expectedValue, entry.Data["createdByName"])
						}
					case "ready":
						if entry.Data["ready"] != expectedValue.(bool) {
							t.Errorf("Expected ready %v, got %v", expectedValue, entry.Data["ready"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "endpoints" {
					t.Errorf("Expected resource type 'endpoints', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Namespace == "" {
					t.Error("Entry namespace should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["addresses"] == nil {
					t.Error("addresses should not be nil")
				}
				if entry.Data["ports"] == nil {
					t.Error("ports should not be nil")
				}
				if entry.Data["ready"] == nil {
					t.Error("ready should not be nil")
				}
			}
		})
	}
}

func TestEndpointsHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewEndpointsHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(context.Background().Done())
	if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
		t.Fatal("Failed to sync cache")
	}
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestEndpointsHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewEndpointsHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &corev1.Pod{}
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}

func stringPtr(s string) *string {
	return &s
}
