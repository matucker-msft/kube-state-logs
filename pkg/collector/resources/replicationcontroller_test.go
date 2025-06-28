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

func TestReplicationControllerHandler(t *testing.T) {
	replicas := int32(3)

	rc1 := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "rc-1",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
		},
		Status: corev1.ReplicationControllerStatus{
			Replicas:             3,
			ReadyReplicas:        2,
			AvailableReplicas:    2,
			FullyLabeledReplicas: 3,
			ObservedGeneration:   1,
		},
	}

	rc2 := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "rc-2",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: nil, // Should default to 1
		},
		Status: corev1.ReplicationControllerStatus{
			Replicas:             1,
			ReadyReplicas:        1,
			AvailableReplicas:    1,
			FullyLabeledReplicas: 1,
			ObservedGeneration:   2,
		},
	}

	rcWithOwner := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-rc",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
		},
		Status: corev1.ReplicationControllerStatus{
			Replicas:             3,
			ReadyReplicas:        3,
			AvailableReplicas:    3,
			FullyLabeledReplicas: 3,
			ObservedGeneration:   3,
		},
	}

	rcEmpty := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "empty-rc",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
	}

	tests := []struct {
		name           string
		rcs            []*corev1.ReplicationController
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all replication controllers",
			rcs:           []*corev1.ReplicationController{rc1, rc2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"rc-1", "rc-2"},
		},
		{
			name:          "collect replication controllers from specific namespace",
			rcs:           []*corev1.ReplicationController{rc1, rc2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"rc-1"},
		},
		{
			name:          "collect replication controller with owner reference",
			rcs:           []*corev1.ReplicationController{rcWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-rc"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:          "collect empty replication controller",
			rcs:           []*corev1.ReplicationController{rcEmpty},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"empty-rc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.rcs))
			for i, rc := range tt.rcs {
				objects[i] = rc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewReplicationControllerHandler(client)
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
					t.Errorf("Expected to find replication controller with name %s", expectedName)
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
				if entry.ResourceType != "replicationcontroller" {
					t.Errorf("Expected resource type 'replicationcontroller', got %s", entry.ResourceType)
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
				if entry.Data["desiredReplicas"] == nil {
					t.Error("desiredReplicas should not be nil")
				}
				if entry.Data["currentReplicas"] == nil {
					t.Error("currentReplicas should not be nil")
				}
				if entry.Data["readyReplicas"] == nil {
					t.Error("readyReplicas should not be nil")
				}
				if entry.Data["availableReplicas"] == nil {
					t.Error("availableReplicas should not be nil")
				}
				if entry.Data["fullyLabeledReplicas"] == nil {
					t.Error("fullyLabeledReplicas should not be nil")
				}
				if entry.Data["observedGeneration"] == nil {
					t.Error("observedGeneration should not be nil")
				}
			}
		})
	}
}

func TestReplicationControllerHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicationControllerHandler(client)
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

func TestReplicationControllerHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewReplicationControllerHandler(client)
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
