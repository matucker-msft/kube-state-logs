package resources

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestCRDHandler(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "myresources",
	}

	crd1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "MyResource",
			"metadata": map[string]interface{}{
				"name":      "resource-1",
				"namespace": "default",
				"labels": map[string]interface{}{
					"app": "web",
				},
				"annotations": map[string]interface{}{
					"purpose": "test",
				},
				"creationTimestamp": metav1.Now().Format(time.RFC3339),
			},
			"spec": map[string]interface{}{
				"replicas": float64(3),
				"image":    "nginx:latest",
			},
			"status": map[string]interface{}{
				"ready":     true,
				"phase":     "Running",
				"replicas":  float64(3),
				"available": float64(2),
			},
		},
	}

	crd2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "MyResource",
			"metadata": map[string]interface{}{
				"name":              "resource-2",
				"namespace":         "kube-system",
				"creationTimestamp": metav1.Now().Format(time.RFC3339),
			},
			"spec": map[string]interface{}{
				"replicas": float64(1),
				"image":    "busybox:latest",
			},
			"status": map[string]interface{}{
				"ready":     false,
				"phase":     "Pending",
				"replicas":  float64(1),
				"available": float64(0),
			},
		},
	}

	crdEmpty := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "MyResource",
			"metadata": map[string]interface{}{
				"name":              "empty-resource",
				"namespace":         "default",
				"creationTimestamp": metav1.Now().Format(time.RFC3339),
			},
		},
	}

	tests := []struct {
		name           string
		crds           []*unstructured.Unstructured
		namespaces     []string
		customFields   []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all CRD resources",
			crds:          []*unstructured.Unstructured{crd1, crd2},
			namespaces:    []string{},
			customFields:  []string{"spec.replicas", "status.phase"},
			expectedCount: 2,
			expectedNames: []string{"resource-1", "resource-2"},
		},
		{
			name:          "collect CRD resources from specific namespace",
			crds:          []*unstructured.Unstructured{crd1, crd2},
			namespaces:    []string{"default"},
			customFields:  []string{"spec.replicas"},
			expectedCount: 1,
			expectedNames: []string{"resource-1"},
		},
		{
			name:          "collect empty CRD resource",
			crds:          []*unstructured.Unstructured{crdEmpty},
			namespaces:    []string{},
			customFields:  []string{},
			expectedCount: 1,
			expectedNames: []string{"empty-resource"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.crds))
			for i, crd := range tt.crds {
				objects[i] = crd
			}
			client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
				gvr: "MyResourceList",
			}, objects...)
			handler := NewCRDHandler(client, gvr, "myresource", tt.customFields)
			factory := dynamicinformer.NewDynamicSharedInformerFactory(client, time.Hour)
			err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
			if err != nil {
				t.Fatalf("Failed to setup informer: %v", err)
			}
			factory.Start(context.Background().Done())
			if !cache.WaitForCacheSync(context.Background().Done(), handler.informer.HasSynced) {
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
					t.Errorf("Expected to find CRD resource with name %s", expectedName)
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
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "crd" {
					t.Errorf("Expected resource type 'crd', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["apiVersion"] == nil {
					t.Error("apiVersion should not be nil")
				}
				if entry.Data["kind"] == nil {
					t.Error("kind should not be nil")
				}
				if entry.Data["spec"] == nil {
					t.Error("spec should not be nil")
				}
				if entry.Data["status"] == nil {
					t.Error("status should not be nil")
				}
				if entry.Data["customFields"] == nil {
					t.Error("customFields should not be nil")
				}
			}
		})
	}
}

func TestCRDHandler_EmptyCache(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "myresources",
	}
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr: "MyResourceList",
	})
	handler := NewCRDHandler(client, gvr, "myresource", []string{})
	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(context.Background().Done())
	if !cache.WaitForCacheSync(context.Background().Done(), handler.informer.HasSynced) {
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

func TestCRDHandler_InvalidObject(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "myresources",
	}
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr: "MyResourceList",
	})
	handler := NewCRDHandler(client, gvr, "myresource", []string{})
	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &metav1.Status{} // Use a non-Unstructured object
	handler.informer.GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}

func TestCRDHandler_CustomFields(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "myresources",
	}

	crd := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "MyResource",
			"metadata": map[string]interface{}{
				"name":              "test-resource",
				"namespace":         "default",
				"creationTimestamp": metav1.Now().Format(time.RFC3339),
			},
			"spec": map[string]interface{}{
				"replicas": float64(3),
				"image":    "nginx:latest",
				"config": map[string]interface{}{
					"port": float64(8080),
				},
			},
			"status": map[string]interface{}{
				"ready":     true,
				"phase":     "Running",
				"replicas":  float64(3),
				"available": float64(2),
			},
		},
	}

	customFields := []string{
		"spec.replicas",
		"spec.image",
		"spec.config.port",
		"status.phase",
		"status.available",
		"metadata.nonexistent",
	}

	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		gvr: "MyResourceList",
	}, crd)
	handler := NewCRDHandler(client, gvr, "myresource", customFields)
	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(context.Background().Done())
	if !cache.WaitForCacheSync(context.Background().Done(), handler.informer.HasSynced) {
		t.Fatal("Failed to sync cache")
	}
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	customFieldsData := entry.Data["customFields"].(map[string]interface{})

	// Check that custom fields are extracted correctly
	if customFieldsData["spec.replicas"] != float64(3) {
		t.Errorf("Expected spec.replicas to be 3, got %v", customFieldsData["spec.replicas"])
	}
	if customFieldsData["spec.image"] != "nginx:latest" {
		t.Errorf("Expected spec.image to be 'nginx:latest', got %v", customFieldsData["spec.image"])
	}
	if customFieldsData["spec.config.port"] != float64(8080) {
		t.Errorf("Expected spec.config.port to be 8080, got %v", customFieldsData["spec.config.port"])
	}
	if customFieldsData["status.phase"] != "Running" {
		t.Errorf("Expected status.phase to be 'Running', got %v", customFieldsData["status.phase"])
	}
	if customFieldsData["status.available"] != float64(2) {
		t.Errorf("Expected status.available to be 2, got %v", customFieldsData["status.available"])
	}
	if customFieldsData["metadata.nonexistent"] != nil {
		t.Errorf("Expected metadata.nonexistent to be nil, got %v", customFieldsData["metadata.nonexistent"])
	}
}
