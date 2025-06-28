package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestStorageClassHandler(t *testing.T) {
	reclaimPolicyDelete := corev1.PersistentVolumeReclaimDelete
	reclaimPolicyRetain := corev1.PersistentVolumeReclaimRetain
	bindingModeImmediate := storagev1.VolumeBindingImmediate
	bindingModeWaitForFirstConsumer := storagev1.VolumeBindingWaitForFirstConsumer
	allowExpansion := true

	sc1 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "fast-ssd",
			Labels:            map[string]string{"type": "ssd"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Provisioner:          "kubernetes.io/aws-ebs",
		ReclaimPolicy:        &reclaimPolicyDelete,
		VolumeBindingMode:    &bindingModeImmediate,
		AllowVolumeExpansion: &allowExpansion,
		Parameters: map[string]string{
			"type": "gp3",
			"iops": "3000",
		},
		MountOptions: []string{"debug"},
	}

	sc2 := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "slow-hdd",
			CreationTimestamp: metav1.Now(),
		},
		Provisioner:          "kubernetes.io/azure-disk",
		ReclaimPolicy:        &reclaimPolicyRetain,
		VolumeBindingMode:    &bindingModeWaitForFirstConsumer,
		AllowVolumeExpansion: nil, // Should default to false
		Parameters: map[string]string{
			"storageaccounttype": "Standard_LRS",
		},
	}

	scWithOwner := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-sc",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Cluster", Name: "my-cluster"}},
			CreationTimestamp: metav1.Now(),
		},
		Provisioner:          "kubernetes.io/gce-pd",
		ReclaimPolicy:        &reclaimPolicyDelete,
		VolumeBindingMode:    &bindingModeImmediate,
		AllowVolumeExpansion: &allowExpansion,
	}

	scDefault := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-sc",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
			CreationTimestamp: metav1.Now(),
		},
		Provisioner:          "kubernetes.io/aws-ebs",
		ReclaimPolicy:        &reclaimPolicyDelete,
		VolumeBindingMode:    &bindingModeImmediate,
		AllowVolumeExpansion: &allowExpansion,
	}

	scWithTopologies := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "topology-sc",
			CreationTimestamp: metav1.Now(),
		},
		Provisioner:       "kubernetes.io/aws-ebs",
		ReclaimPolicy:     &reclaimPolicyDelete,
		VolumeBindingMode: &bindingModeImmediate,
		AllowedTopologies: []corev1.TopologySelectorTerm{
			{
				MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
					{
						Key:    "topology.kubernetes.io/zone",
						Values: []string{"us-west-2a", "us-west-2b"},
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		storageClasses []*storagev1.StorageClass
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:           "collect all storage classes",
			storageClasses: []*storagev1.StorageClass{sc1, sc2},
			expectedCount:  2,
			expectedNames:  []string{"fast-ssd", "slow-hdd"},
		},
		{
			name:           "collect storage class with owner reference",
			storageClasses: []*storagev1.StorageClass{scWithOwner},
			expectedCount:  1,
			expectedNames:  []string{"owned-sc"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Cluster",
				"created_by_name": "my-cluster",
			},
		},
		{
			name:           "collect storage class with parameters and mount options",
			storageClasses: []*storagev1.StorageClass{sc1},
			expectedCount:  1,
			expectedNames:  []string{"fast-ssd"},
			expectedFields: map[string]interface{}{
				"provisioner":            "kubernetes.io/aws-ebs",
				"reclaim_policy":         "Delete",
				"volume_binding_mode":    "Immediate",
				"allow_volume_expansion": true,
			},
		},
		{
			name:           "collect default storage class",
			storageClasses: []*storagev1.StorageClass{scDefault},
			expectedCount:  1,
			expectedNames:  []string{"default-sc"},
			expectedFields: map[string]interface{}{
				"is_default_class": true,
			},
		},
		{
			name:           "collect storage class with allowed topologies",
			storageClasses: []*storagev1.StorageClass{scWithTopologies},
			expectedCount:  1,
			expectedNames:  []string{"topology-sc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.storageClasses))
			for i, sc := range tt.storageClasses {
				objects[i] = sc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewStorageClassHandler(client)
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
					t.Errorf("Expected to find storage class with name %s", expectedName)
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
					case "provisioner":
						if entry.Data["provisioner"] != expectedValue.(string) {
							t.Errorf("Expected provisioner %s, got %v", expectedValue, entry.Data["provisioner"])
						}
					case "reclaim_policy":
						if entry.Data["reclaimPolicy"] != expectedValue.(string) {
							t.Errorf("Expected reclaim_policy %s, got %v", expectedValue, entry.Data["reclaimPolicy"])
						}
					case "volume_binding_mode":
						if entry.Data["volumeBindingMode"] != expectedValue.(string) {
							t.Errorf("Expected volume_binding_mode %s, got %v", expectedValue, entry.Data["volumeBindingMode"])
						}
					case "allow_volume_expansion":
						if entry.Data["allowVolumeExpansion"] != expectedValue.(bool) {
							t.Errorf("Expected allow_volume_expansion %v, got %v", expectedValue, entry.Data["allowVolumeExpansion"])
						}
					case "is_default_class":
						if entry.Data["isDefaultClass"] != expectedValue.(bool) {
							t.Errorf("Expected is_default_class %v, got %v", expectedValue, entry.Data["isDefaultClass"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "storageclass" {
					t.Errorf("Expected resource type 'storageclass', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["provisioner"] == nil {
					t.Error("provisioner should not be nil")
				}
				if entry.Data["reclaimPolicy"] == nil {
					t.Error("reclaimPolicy should not be nil")
				}
				if entry.Data["volumeBindingMode"] == nil {
					t.Error("volumeBindingMode should not be nil")
				}
				if entry.Data["allowVolumeExpansion"] == nil {
					t.Error("allowVolumeExpansion should not be nil")
				}
				if entry.Data["parameters"] == nil {
					t.Error("parameters should not be nil")
				}
				if entry.Data["mountOptions"] == nil {
					t.Error("mountOptions should not be nil")
				}
			}
		})
	}
}

func TestStorageClassHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStorageClassHandler(client)
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

func TestStorageClassHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewStorageClassHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &corev1.PersistentVolume{}
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}
