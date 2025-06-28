package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestPersistentVolumeClaimHandler(t *testing.T) {
	storageClass := "fast"
	pvc1 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pvc1",
			Namespace:         "default",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			VolumeName:       "pv1",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("8Gi"),
			},
			Conditions: []corev1.PersistentVolumeClaimCondition{{Type: "Bound", Status: corev1.ConditionTrue}},
		},
	}

	pvc2 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pvc2",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase:      corev1.ClaimPending,
			Conditions: []corev1.PersistentVolumeClaimCondition{{Type: "Pending", Status: corev1.ConditionTrue}},
		},
	}

	pvcWithOwner := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-pvc",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "StatefulSet", Name: "my-sts"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase:      corev1.ClaimLost,
			Conditions: []corev1.PersistentVolumeClaimCondition{{Type: "Lost", Status: corev1.ConditionTrue}},
		},
	}

	tests := []struct {
		name           string
		pvcs           []*corev1.PersistentVolumeClaim
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all pvcs",
			pvcs:          []*corev1.PersistentVolumeClaim{pvc1, pvc2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"pvc1", "pvc2"},
		},
		{
			name:          "collect pvcs from specific namespace",
			pvcs:          []*corev1.PersistentVolumeClaim{pvc1, pvc2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"pvc1"},
		},
		{
			name:          "collect pvc with owner reference",
			pvcs:          []*corev1.PersistentVolumeClaim{pvcWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-pvc"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "StatefulSet",
				"created_by_name": "my-sts",
			},
		},
		{
			name:          "collect pvc with access modes and storage class",
			pvcs:          []*corev1.PersistentVolumeClaim{pvc1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"pvc1"},
			expectedFields: map[string]interface{}{
				"access_modes":    []string{"ReadWriteOnce"},
				"storage_class":   "fast",
				"phase":           "Bound",
				"request_storage": "10Gi",
				"used_storage":    "8Gi",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.pvcs))
			for i, pvc := range tt.pvcs {
				objects[i] = pvc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewPersistentVolumeClaimHandler(client)
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
					t.Errorf("Expected to find pvc with name %s", expectedName)
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
					case "access_modes":
						expectedModes := expectedValue.([]string)
						modes, ok := entry.Data["accessModes"].([]string)
						if !ok {
							t.Errorf("Expected accessModes to be []string, got %T", entry.Data["accessModes"])
						} else if len(modes) != len(expectedModes) {
							t.Errorf("Expected %d access modes, got %d", len(expectedModes), len(modes))
						}
					case "storage_class":
						if entry.Data["storageClassName"] != expectedValue.(string) {
							t.Errorf("Expected storageClassName %s, got %v", expectedValue, entry.Data["storageClassName"])
						}
					case "phase":
						if entry.Data["phase"] != expectedValue.(string) {
							t.Errorf("Expected phase %s, got %v", expectedValue, entry.Data["phase"])
						}
					case "request_storage":
						if entry.Data["requestStorage"] != expectedValue.(string) {
							t.Errorf("Expected requestStorage %s, got %v", expectedValue, entry.Data["requestStorage"])
						}
					case "used_storage":
						if entry.Data["usedStorage"] != expectedValue.(string) {
							t.Errorf("Expected usedStorage %s, got %v", expectedValue, entry.Data["usedStorage"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "persistentvolumeclaim" {
					t.Errorf("Expected resource type 'persistentvolumeclaim', got %s", entry.ResourceType)
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
				if entry.Data["accessModes"] == nil {
					t.Error("accessModes should not be nil")
				}
				if entry.Data["phase"] == nil {
					t.Error("phase should not be nil")
				}
				if entry.Data["requestStorage"] == nil {
					t.Error("requestStorage should not be nil")
				}
				if entry.Data["usedStorage"] == nil {
					t.Error("usedStorage should not be nil")
				}
			}
		})
	}
}

func TestPersistentVolumeClaimHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPersistentVolumeClaimHandler(client)
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

func TestPersistentVolumeClaimHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPersistentVolumeClaimHandler(client)
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
