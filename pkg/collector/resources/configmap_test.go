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

func TestConfigMapHandler(t *testing.T) {
	// Test data
	configmap1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test-app",
			},
			Annotations: map[string]string{
				"description": "test configmap",
			},
			CreationTimestamp: metav1.Now(),
		},
		Data: map[string]string{
			"config.json": `{"key": "value"}`,
			"env.txt":     "DEBUG=true",
		},
	}

	configmap2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap-2",
			Namespace: "kube-system",
			Labels: map[string]string{
				"component": "system",
			},
			CreationTimestamp: metav1.Now(),
		},
		Data: map[string]string{
			"system.conf": "system_config_value",
		},
	}

	configmapWithOwner := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "owned-configmap",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "test-deployment",
				},
			},
			CreationTimestamp: metav1.Now(),
		},
		Data: map[string]string{
			"owned.conf": "owned_value",
		},
	}

	tests := []struct {
		name           string
		configmaps     []*corev1.ConfigMap
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all configmaps",
			configmaps:    []*corev1.ConfigMap{configmap1, configmap2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"test-configmap-1", "test-configmap-2"},
		},
		{
			name:          "collect configmaps from specific namespace",
			configmaps:    []*corev1.ConfigMap{configmap1, configmap2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"test-configmap-1"},
		},
		{
			name:          "collect configmap with owner reference",
			configmaps:    []*corev1.ConfigMap{configmapWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-configmap"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Deployment",
				"created_by_name": "test-deployment",
			},
		},
		{
			name:          "collect configmap with data keys",
			configmaps:    []*corev1.ConfigMap{configmap1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"test-configmap-1"},
			expectedFields: map[string]interface{}{
				"data_keys": []string{"config.json", "env.txt"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			objects := make([]runtime.Object, len(tt.configmaps))
			for i, cm := range tt.configmaps {
				objects[i] = cm
			}
			client := fake.NewSimpleClientset(objects...)

			// Create handler
			handler := NewConfigMapHandler(client)

			// Create informer factory
			factory := informers.NewSharedInformerFactory(client, time.Hour)

			// Setup informer
			err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
			if err != nil {
				t.Fatalf("Failed to setup informer: %v", err)
			}

			// Start informer
			factory.Start(context.Background().Done())
			if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
				t.Fatal("Failed to sync cache")
			}

			// Collect metrics
			entries, err := handler.Collect(context.Background(), tt.namespaces)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify results
			if len(entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries))
			}

			// Verify entry names
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
					t.Errorf("Expected to find configmap with name %s", expectedName)
				}
			}

			// Verify specific fields if provided
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
					case "data_keys":
						expectedKeys := expectedValue.([]string)
						dataKeys, ok := entry.Data["dataKeys"].([]string)
						if !ok {
							t.Errorf("Expected dataKeys to be []string, got %T", entry.Data["dataKeys"])
						} else if len(dataKeys) != len(expectedKeys) {
							t.Errorf("Expected %d data keys, got %d", len(expectedKeys), len(dataKeys))
						}
						// Note: DataKeys order might vary, so we just check count for now
					}
				}
			}

			// Verify entry structure
			for _, entry := range entries {
				if entry.ResourceType != "configmap" {
					t.Errorf("Expected resource type 'configmap', got %s", entry.ResourceType)
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
			}
		})
	}
}

func TestConfigMapHandler_EmptyCache(t *testing.T) {
	// Create handler with empty client
	client := fake.NewSimpleClientset()
	handler := NewConfigMapHandler(client)

	// Create informer factory
	factory := informers.NewSharedInformerFactory(client, time.Hour)

	// Setup informer
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	// Start informer
	factory.Start(context.Background().Done())
	if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
		t.Fatal("Failed to sync cache")
	}

	// Collect metrics
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Should return empty slice
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestConfigMapHandler_InvalidObject(t *testing.T) {
	// Create handler
	client := fake.NewSimpleClientset()
	handler := NewConfigMapHandler(client)

	// Create informer factory
	factory := informers.NewSharedInformerFactory(client, time.Hour)

	// Setup informer
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	// Manually add invalid object to store
	invalidObj := &corev1.Pod{} // Wrong type
	handler.GetInformer().GetStore().Add(invalidObj)

	// Collect metrics
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Should handle invalid object gracefully
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}
