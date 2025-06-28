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

func TestNamespaceHandler(t *testing.T) {
	namespace1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
			Conditions: []corev1.NamespaceCondition{
				{Type: "Active", Status: corev1.ConditionTrue},
			},
		},
	}

	namespace2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
			Conditions: []corev1.NamespaceCondition{
				{Type: "Active", Status: corev1.ConditionTrue},
			},
		},
	}

	namespaceTerminating := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "terminating-ns",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceTerminating,
			Conditions: []corev1.NamespaceCondition{
				{Type: "Terminating", Status: corev1.ConditionTrue},
			},
		},
	}

	namespaceWithOwner := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-ns",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
			Conditions: []corev1.NamespaceCondition{
				{Type: "Active", Status: corev1.ConditionTrue},
			},
		},
	}

	tests := []struct {
		name             string
		namespaces       []*corev1.Namespace
		filterNamespaces []string
		expectedCount    int
		expectedNames    []string
		expectedFields   map[string]interface{}
	}{
		{
			name:             "collect all namespaces",
			namespaces:       []*corev1.Namespace{namespace1, namespace2},
			filterNamespaces: []string{},
			expectedCount:    2,
			expectedNames:    []string{"default", "kube-system"},
		},
		{
			name:             "collect namespaces with filter",
			namespaces:       []*corev1.Namespace{namespace1, namespace2},
			filterNamespaces: []string{"default"},
			expectedCount:    1,
			expectedNames:    []string{"default"},
		},
		{
			name:             "collect namespace with owner reference",
			namespaces:       []*corev1.Namespace{namespaceWithOwner},
			filterNamespaces: []string{},
			expectedCount:    1,
			expectedNames:    []string{"owned-ns"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:             "collect terminating namespace",
			namespaces:       []*corev1.Namespace{namespaceTerminating},
			filterNamespaces: []string{},
			expectedCount:    1,
			expectedNames:    []string{"terminating-ns"},
			expectedFields: map[string]interface{}{
				"phase":                 "Terminating",
				"condition_terminating": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.namespaces))
			for i, ns := range tt.namespaces {
				objects[i] = ns
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewNamespaceHandler(client)
			factory := informers.NewSharedInformerFactory(client, time.Hour)
			err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
			if err != nil {
				t.Fatalf("Failed to setup informer: %v", err)
			}
			factory.Start(context.Background().Done())
			if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
				t.Fatal("Failed to sync cache")
			}
			entries, err := handler.Collect(context.Background(), tt.filterNamespaces)
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
					t.Errorf("Expected to find namespace with name %s", expectedName)
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
					case "phase":
						if entry.Data["phase"] != expectedValue.(string) {
							t.Errorf("Expected phase %s, got %v", expectedValue, entry.Data["phase"])
						}
					case "condition_terminating":
						if entry.Data["conditionTerminating"] != expectedValue.(bool) {
							t.Errorf("Expected condition_terminating %v, got %v", expectedValue, entry.Data["conditionTerminating"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "namespace" {
					t.Errorf("Expected resource type 'namespace', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["phase"] == nil {
					t.Error("phase should not be nil")
				}
			}
		})
	}
}

func TestNamespaceHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNamespaceHandler(client)
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

func TestNamespaceHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNamespaceHandler(client)
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
