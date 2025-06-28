package resources

import (
	"context"
	"testing"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestPodDisruptionBudgetHandler(t *testing.T) {
	minAvailable := intstrFromInt(2)
	maxUnavailable := intstrFromInt(1)

	pdb1 := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pdb-1",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: minAvailable,
		},
		Status: policyv1.PodDisruptionBudgetStatus{
			CurrentHealthy:     3,
			DesiredHealthy:     2,
			ExpectedPods:       4,
			DisruptionsAllowed: 1,
		},
	}

	pdb2 := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pdb-2",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: maxUnavailable,
		},
		Status: policyv1.PodDisruptionBudgetStatus{
			CurrentHealthy:     5,
			DesiredHealthy:     4,
			ExpectedPods:       6,
			DisruptionsAllowed: 0,
		},
	}

	pdbWithOwner := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-pdb",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: minAvailable,
		},
		Status: policyv1.PodDisruptionBudgetStatus{
			CurrentHealthy:     2,
			DesiredHealthy:     2,
			ExpectedPods:       2,
			DisruptionsAllowed: 1,
		},
	}

	pdbEmpty := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "empty-pdb",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
	}

	tests := []struct {
		name           string
		pdbs           []*policyv1.PodDisruptionBudget
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all pod disruption budgets",
			pdbs:          []*policyv1.PodDisruptionBudget{pdb1, pdb2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"pdb-1", "pdb-2"},
		},
		{
			name:          "collect pod disruption budgets from specific namespace",
			pdbs:          []*policyv1.PodDisruptionBudget{pdb1, pdb2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"pdb-1"},
		},
		{
			name:          "collect pod disruption budget with owner reference",
			pdbs:          []*policyv1.PodDisruptionBudget{pdbWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-pdb"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:          "collect empty pod disruption budget",
			pdbs:          []*policyv1.PodDisruptionBudget{pdbEmpty},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"empty-pdb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.pdbs))
			for i, pdb := range tt.pdbs {
				objects[i] = pdb
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewPodDisruptionBudgetHandler(client)
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
					t.Errorf("Expected to find pod disruption budget with name %s", expectedName)
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
				if entry.ResourceType != "poddisruptionbudget" {
					t.Errorf("Expected resource type 'poddisruptionbudget', got %s", entry.ResourceType)
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
				if entry.Data["minAvailable"] == nil {
					t.Error("minAvailable should not be nil")
				}
				if entry.Data["maxUnavailable"] == nil {
					t.Error("maxUnavailable should not be nil")
				}
				if entry.Data["currentHealthy"] == nil {
					t.Error("currentHealthy should not be nil")
				}
				if entry.Data["desiredHealthy"] == nil {
					t.Error("desiredHealthy should not be nil")
				}
				if entry.Data["expectedPods"] == nil {
					t.Error("expectedPods should not be nil")
				}
				if entry.Data["disruptionsAllowed"] == nil {
					t.Error("disruptionsAllowed should not be nil")
				}
				if entry.Data["disruptionAllowed"] == nil {
					t.Error("disruptionAllowed should not be nil")
				}
			}
		})
	}
}

func intstrFromInt(i int32) *intstr.IntOrString {
	v := intstr.IntOrString{Type: intstr.Int, IntVal: i}
	return &v
}

func TestPodDisruptionBudgetHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodDisruptionBudgetHandler(client)
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

func TestPodDisruptionBudgetHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPodDisruptionBudgetHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &metav1.Status{} // Use a non-PDB object
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}
