package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestPriorityClassHandler(t *testing.T) {
	preemptionPolicy := corev1.PreemptionPolicy("PreemptLowerOrEqualPriority")

	pc1 := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "high-priority",
			Labels:            map[string]string{"tier": "critical"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Value:            1000000,
		GlobalDefault:    false,
		Description:      "High priority class for critical workloads",
		PreemptionPolicy: &preemptionPolicy,
	}

	pc2 := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "system-cluster-critical",
			CreationTimestamp: metav1.Now(),
		},
		Value:         2000000000,
		GlobalDefault: true,
		Description:   "System cluster critical priority class",
	}

	pcWithOwner := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-priority",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Value:         500000,
		GlobalDefault: false,
		Description:   "Priority class owned by project",
	}

	pcNilPreemption := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "nil-preemption",
			CreationTimestamp: metav1.Now(),
		},
		Value:         100000,
		GlobalDefault: false,
		Description:   "Priority class with nil preemption policy",
	}

	tests := []struct {
		name            string
		priorityClasses []*schedulingv1.PriorityClass
		expectedCount   int
		expectedNames   []string
		expectedFields  map[string]interface{}
	}{
		{
			name:            "collect all priority classes",
			priorityClasses: []*schedulingv1.PriorityClass{pc1, pc2},
			expectedCount:   2,
			expectedNames:   []string{"high-priority", "system-cluster-critical"},
		},
		{
			name:            "collect priority class with owner reference",
			priorityClasses: []*schedulingv1.PriorityClass{pcWithOwner},
			expectedCount:   1,
			expectedNames:   []string{"owned-priority"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:            "collect priority class with preemption policy",
			priorityClasses: []*schedulingv1.PriorityClass{pc1},
			expectedCount:   1,
			expectedNames:   []string{"high-priority"},
			expectedFields: map[string]interface{}{
				"preemption_policy": "PreemptLowerOrEqualPriority",
			},
		},
		{
			name:            "collect priority class with nil preemption policy",
			priorityClasses: []*schedulingv1.PriorityClass{pcNilPreemption},
			expectedCount:   1,
			expectedNames:   []string{"nil-preemption"},
			expectedFields: map[string]interface{}{
				"preemption_policy": "",
			},
		},
		{
			name:            "collect global default priority class",
			priorityClasses: []*schedulingv1.PriorityClass{pc2},
			expectedCount:   1,
			expectedNames:   []string{"system-cluster-critical"},
			expectedFields: map[string]interface{}{
				"global_default": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.priorityClasses))
			for i, pc := range tt.priorityClasses {
				objects[i] = pc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewPriorityClassHandler(client)
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
					t.Errorf("Expected to find priority class with name %s", expectedName)
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
					case "preemption_policy":
						if entry.Data["preemptionPolicy"] != expectedValue.(string) {
							t.Errorf("Expected preemption_policy %s, got %v", expectedValue, entry.Data["preemptionPolicy"])
						}
					case "global_default":
						if entry.Data["globalDefault"] != expectedValue.(bool) {
							t.Errorf("Expected global_default %v, got %v", expectedValue, entry.Data["globalDefault"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "priorityclass" {
					t.Errorf("Expected resource type 'priorityclass', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["value"] == nil {
					t.Error("value should not be nil")
				}
				if entry.Data["globalDefault"] == nil {
					t.Error("globalDefault should not be nil")
				}
				if entry.Data["description"] == nil {
					t.Error("description should not be nil")
				}
				if entry.Data["preemptionPolicy"] == nil {
					t.Error("preemptionPolicy should not be nil")
				}
			}
		})
	}
}

func TestPriorityClassHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPriorityClassHandler(client)
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

func TestPriorityClassHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPriorityClassHandler(client)
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
