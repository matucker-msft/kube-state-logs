package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"go.goms.io/aks/kube-state-logs/pkg/collector/testutils"
	"go.goms.io/aks/kube-state-logs/pkg/types"
)

func TestRuntimeClassHandler(t *testing.T) {
	rc1 := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "runc",
			Labels:            map[string]string{"type": "default"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Handler: "runc",
	}

	rc2 := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "kata-containers",
			CreationTimestamp: metav1.Now(),
		},
		Handler: "kata-containers",
	}

	rcWithOwner := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-runtime",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Handler: "gvisor",
	}

	rcEmpty := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "empty-runtime",
			CreationTimestamp: metav1.Now(),
		},
		Handler: "default",
	}

	tests := []struct {
		name           string
		runtimeClasses []*nodev1.RuntimeClass
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:           "collect all runtime classes",
			runtimeClasses: []*nodev1.RuntimeClass{rc1, rc2},
			expectedCount:  2,
			expectedNames:  []string{"runc", "kata-containers"},
		},
		{
			name:           "collect runtime class with owner reference",
			runtimeClasses: []*nodev1.RuntimeClass{rcWithOwner},
			expectedCount:  1,
			expectedNames:  []string{"owned-runtime"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:           "collect empty runtime class",
			runtimeClasses: []*nodev1.RuntimeClass{rcEmpty},
			expectedCount:  1,
			expectedNames:  []string{"empty-runtime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.runtimeClasses))
			for i, rc := range tt.runtimeClasses {
				objects[i] = rc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewRuntimeClassHandler(client)
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
			if len(entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries))
			}
			entryNames := make([]string, len(entries))
			for i, entry := range entries {
				runtimeClassData, ok := entry.(types.RuntimeClassData)
				if !ok {
					t.Fatalf("Expected RuntimeClassData type, got %T", entry)
				}
				entryNames[i] = runtimeClassData.Name
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
					t.Errorf("Expected to find runtime class with name %s", expectedName)
				}
			}
			if tt.expectedFields != nil && len(entries) > 0 {
				runtimeClassData, ok := entries[0].(types.RuntimeClassData)
				if !ok {
					t.Fatalf("Expected RuntimeClassData type, got %T", entries[0])
				}
				for field, expectedValue := range tt.expectedFields {
					switch field {
					case "created_by_kind":
						if runtimeClassData.CreatedByKind != expectedValue.(string) {
							t.Errorf("Expected created_by_kind %s, got %v", expectedValue, runtimeClassData.CreatedByKind)
						}
					case "created_by_name":
						if runtimeClassData.CreatedByName != expectedValue.(string) {
							t.Errorf("Expected created_by_name %s, got %v", expectedValue, runtimeClassData.CreatedByName)
						}
					}
				}
			}
			for _, entry := range entries {
				runtimeClassData, ok := entry.(types.RuntimeClassData)
				if !ok {
					t.Fatalf("Expected RuntimeClassData type, got %T", entry)
				}
				if runtimeClassData.ResourceType != "runtimeclass" {
					t.Errorf("Expected resource type 'runtimeclass', got %s", runtimeClassData.ResourceType)
				}
				if runtimeClassData.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if runtimeClassData.CreatedTimestamp == 0 {
					t.Error("Created timestamp should not be zero")
				}
				if runtimeClassData.Handler == "" {
					t.Error("handler should not be empty")
				}
			}
		})
	}
}

func TestRuntimeClassHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRuntimeClassHandler(client)
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

func TestRuntimeClassHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewRuntimeClassHandler(client)
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
