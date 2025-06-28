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

func TestPersistentVolumeHandler(t *testing.T) {
	volumeMode := corev1.PersistentVolumeFilesystem
	pv1 := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pv1",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              "fast",
			VolumeMode:                    &volumeMode,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/data",
				},
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeAvailable,
		},
	}

	pv2 := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pv2",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("20Gi"),
			},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			VolumeMode:                    &volumeMode,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: "nfs.example.com",
					Path:   "/exports",
				},
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}

	pvWithOwner := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-pv",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "StorageClass", Name: "my-sc"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
			AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRecycle,
			VolumeMode:                    &volumeMode,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
					VolumeID: "vol-123",
				},
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeFailed,
		},
	}

	tests := []struct {
		name           string
		pvs            []*corev1.PersistentVolume
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all pvs",
			pvs:           []*corev1.PersistentVolume{pv1, pv2},
			expectedCount: 2,
			expectedNames: []string{"pv1", "pv2"},
		},
		{
			name:          "collect pv with owner reference",
			pvs:           []*corev1.PersistentVolume{pvWithOwner},
			expectedCount: 1,
			expectedNames: []string{"owned-pv"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "StorageClass",
				"created_by_name": "my-sc",
			},
		},
		{
			name:          "collect pv with hostpath volume",
			pvs:           []*corev1.PersistentVolume{pv1},
			expectedCount: 1,
			expectedNames: []string{"pv1"},
			expectedFields: map[string]interface{}{
				"access_modes":       "ReadWriteOnce",
				"storage_class":      "fast",
				"status":             "Available",
				"volume_plugin_name": "hostPath",
				"capacity_bytes":     int64(10737418240), // 10Gi in bytes
			},
		},
		{
			name:          "collect pv with nfs volume",
			pvs:           []*corev1.PersistentVolume{pv2},
			expectedCount: 1,
			expectedNames: []string{"pv2"},
			expectedFields: map[string]interface{}{
				"access_modes":       "ReadOnlyMany",
				"status":             "Bound",
				"volume_plugin_name": "nfs",
				"capacity_bytes":     int64(21474836480), // 20Gi in bytes
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.pvs))
			for i, pv := range tt.pvs {
				objects[i] = pv
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewPersistentVolumeHandler(client)
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
					t.Errorf("Expected to find pv with name %s", expectedName)
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
						if entry.Data["accessModes"] != expectedValue.(string) {
							t.Errorf("Expected access_modes %s, got %v", expectedValue, entry.Data["accessModes"])
						}
					case "storage_class":
						if entry.Data["storageClassName"] != expectedValue.(string) {
							t.Errorf("Expected storage_class %s, got %v", expectedValue, entry.Data["storageClassName"])
						}
					case "status":
						if entry.Data["status"] != expectedValue.(string) {
							t.Errorf("Expected status %s, got %v", expectedValue, entry.Data["status"])
						}
					case "volume_plugin_name":
						if entry.Data["volumePluginName"] != expectedValue.(string) {
							t.Errorf("Expected volume_plugin_name %s, got %v", expectedValue, entry.Data["volumePluginName"])
						}
					case "capacity_bytes":
						if entry.Data["capacityBytes"] != expectedValue.(int64) {
							t.Errorf("Expected capacity_bytes %d, got %v", expectedValue, entry.Data["capacityBytes"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "persistentvolume" {
					t.Errorf("Expected resource type 'persistentvolume', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["accessModes"] == nil {
					t.Error("accessModes should not be nil")
				}
				if entry.Data["status"] == nil {
					t.Error("status should not be nil")
				}
				if entry.Data["volumePluginName"] == nil {
					t.Error("volumePluginName should not be nil")
				}
			}
		})
	}
}

func TestPersistentVolumeHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPersistentVolumeHandler(client)
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

func TestPersistentVolumeHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewPersistentVolumeHandler(client)
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
