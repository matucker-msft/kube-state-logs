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

func TestResourceQuotaHandler(t *testing.T) {
	rq1 := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "compute-quota",
			Namespace:         "default",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("10"),
			},
			Scopes: []corev1.ResourceQuotaScope{
				corev1.ResourceQuotaScopeBestEffort,
				corev1.ResourceQuotaScopeNotBestEffort,
			},
		},
		Status: corev1.ResourceQuotaStatus{
			Used: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
				corev1.ResourcePods:   resource.MustParse("5"),
			},
		},
	}

	rq2 := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "storage-quota",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePersistentVolumeClaims: resource.MustParse("5"),
				corev1.ResourceRequestsStorage:        resource.MustParse("100Gi"),
			},
			Scopes: []corev1.ResourceQuotaScope{
				corev1.ResourceQuotaScopeTerminating,
			},
		},
		Status: corev1.ResourceQuotaStatus{
			Used: corev1.ResourceList{
				corev1.ResourcePersistentVolumeClaims: resource.MustParse("2"),
				corev1.ResourceRequestsStorage:        resource.MustParse("50Gi"),
			},
		},
	}

	rqWithOwner := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-quota",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceServices: resource.MustParse("10"),
			},
			Scopes: []corev1.ResourceQuotaScope{
				corev1.ResourceQuotaScopeNotTerminating,
			},
		},
		Status: corev1.ResourceQuotaStatus{
			Used: corev1.ResourceList{
				corev1.ResourceServices: resource.MustParse("3"),
			},
		},
	}

	rqEmpty := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "empty-quota",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{},
		},
		Status: corev1.ResourceQuotaStatus{
			Used: corev1.ResourceList{},
		},
	}

	tests := []struct {
		name           string
		resourceQuotas []*corev1.ResourceQuota
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:           "collect all resource quotas",
			resourceQuotas: []*corev1.ResourceQuota{rq1, rq2},
			namespaces:     []string{},
			expectedCount:  2,
			expectedNames:  []string{"compute-quota", "storage-quota"},
		},
		{
			name:           "collect resource quotas from specific namespace",
			resourceQuotas: []*corev1.ResourceQuota{rq1, rq2},
			namespaces:     []string{"default"},
			expectedCount:  1,
			expectedNames:  []string{"compute-quota"},
		},
		{
			name:           "collect resource quota with owner reference",
			resourceQuotas: []*corev1.ResourceQuota{rqWithOwner},
			namespaces:     []string{},
			expectedCount:  1,
			expectedNames:  []string{"owned-quota"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:           "collect resource quota with compute resources",
			resourceQuotas: []*corev1.ResourceQuota{rq1},
			namespaces:     []string{},
			expectedCount:  1,
			expectedNames:  []string{"compute-quota"},
			expectedFields: map[string]interface{}{
				"scopes": []string{"BestEffort", "NotBestEffort"},
			},
		},
		{
			name:           "collect empty resource quota",
			resourceQuotas: []*corev1.ResourceQuota{rqEmpty},
			namespaces:     []string{},
			expectedCount:  1,
			expectedNames:  []string{"empty-quota"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.resourceQuotas))
			for i, rq := range tt.resourceQuotas {
				objects[i] = rq
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewResourceQuotaHandler(client)
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
					t.Errorf("Expected to find resource quota with name %s", expectedName)
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
					case "scopes":
						expectedScopes := expectedValue.([]string)
						scopes, ok := entry.Data["scopes"].([]string)
						if !ok {
							t.Errorf("Expected scopes to be []string, got %T", entry.Data["scopes"])
						} else if len(scopes) != len(expectedScopes) {
							t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(scopes))
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "resourcequota" {
					t.Errorf("Expected resource type 'resourcequota', got %s", entry.ResourceType)
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
				if entry.Data["hard"] == nil {
					t.Error("hard should not be nil")
				}
				if entry.Data["used"] == nil {
					t.Error("used should not be nil")
				}
				if entry.Data["scopes"] == nil {
					t.Error("scopes should not be nil")
				}
			}
		})
	}
}

func TestResourceQuotaHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewResourceQuotaHandler(client)
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

func TestResourceQuotaHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewResourceQuotaHandler(client)
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
