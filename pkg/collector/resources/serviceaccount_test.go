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

func TestServiceAccountHandler(t *testing.T) {
	automountTrue := true
	automountFalse := false

	sa1 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default",
			Namespace:         "default",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Secrets: []corev1.ObjectReference{
			{Name: "default-token-abc123"},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "docker-registry-secret"},
		},
		AutomountServiceAccountToken: &automountTrue,
	}

	sa2 := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app-sa",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Secrets: []corev1.ObjectReference{
			{Name: "app-sa-token-xyz789"},
			{Name: "app-sa-token-def456"},
		},
		AutomountServiceAccountToken: &automountFalse,
	}

	saWithOwner := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-sa",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Deployment", Name: "my-deployment"}},
			CreationTimestamp: metav1.Now(),
		},
		Secrets: []corev1.ObjectReference{
			{Name: "owned-sa-token-123"},
		},
	}

	saDefault := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default-automount",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		// AutomountServiceAccountToken is nil, should default to true
	}

	tests := []struct {
		name            string
		serviceAccounts []*corev1.ServiceAccount
		namespaces      []string
		expectedCount   int
		expectedNames   []string
		expectedFields  map[string]interface{}
	}{
		{
			name:            "collect all service accounts",
			serviceAccounts: []*corev1.ServiceAccount{sa1, sa2},
			namespaces:      []string{},
			expectedCount:   2,
			expectedNames:   []string{"default", "app-sa"},
		},
		{
			name:            "collect service accounts from specific namespace",
			serviceAccounts: []*corev1.ServiceAccount{sa1, sa2},
			namespaces:      []string{"default"},
			expectedCount:   1,
			expectedNames:   []string{"default"},
		},
		{
			name:            "collect service account with owner reference",
			serviceAccounts: []*corev1.ServiceAccount{saWithOwner},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"owned-sa"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Deployment",
				"created_by_name": "my-deployment",
			},
		},
		{
			name:            "collect service account with secrets and image pull secrets",
			serviceAccounts: []*corev1.ServiceAccount{sa1},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"default"},
			expectedFields: map[string]interface{}{
				"secrets":                         []string{"default-token-abc123"},
				"image_pull_secrets":              []string{"docker-registry-secret"},
				"automount_service_account_token": true,
			},
		},
		{
			name:            "collect service account with multiple secrets and disabled automount",
			serviceAccounts: []*corev1.ServiceAccount{sa2},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"app-sa"},
			expectedFields: map[string]interface{}{
				"secrets":                         []string{"app-sa-token-xyz789", "app-sa-token-def456"},
				"automount_service_account_token": false,
			},
		},
		{
			name:            "collect service account with default automount (nil)",
			serviceAccounts: []*corev1.ServiceAccount{saDefault},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"default-automount"},
			expectedFields: map[string]interface{}{
				"automount_service_account_token": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.serviceAccounts))
			for i, sa := range tt.serviceAccounts {
				objects[i] = sa
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewServiceAccountHandler(client)
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
					t.Errorf("Expected to find service account with name %s", expectedName)
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
					case "secrets":
						expectedSecrets := expectedValue.([]string)
						secrets, ok := entry.Data["secrets"].([]string)
						if !ok {
							t.Errorf("Expected secrets to be []string, got %T", entry.Data["secrets"])
						} else if len(secrets) != len(expectedSecrets) {
							t.Errorf("Expected %d secrets, got %d", len(expectedSecrets), len(secrets))
						}
					case "image_pull_secrets":
						expectedSecrets := expectedValue.([]string)
						secrets, ok := entry.Data["imagePullSecrets"].([]string)
						if !ok {
							t.Errorf("Expected imagePullSecrets to be []string, got %T", entry.Data["imagePullSecrets"])
						} else if len(secrets) != len(expectedSecrets) {
							t.Errorf("Expected %d image pull secrets, got %d", len(expectedSecrets), len(secrets))
						}
					case "automount_service_account_token":
						if entry.Data["automountServiceAccountToken"] != expectedValue.(bool) {
							t.Errorf("Expected automount_service_account_token %v, got %v", expectedValue, entry.Data["automountServiceAccountToken"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "serviceaccount" {
					t.Errorf("Expected resource type 'serviceaccount', got %s", entry.ResourceType)
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
				if entry.Data["secrets"] == nil {
					t.Error("secrets should not be nil")
				}
				if entry.Data["imagePullSecrets"] == nil {
					t.Error("imagePullSecrets should not be nil")
				}
				if entry.Data["automountServiceAccountToken"] == nil {
					t.Error("automountServiceAccountToken should not be nil")
				}
			}
		})
	}
}

func TestServiceAccountHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceAccountHandler(client)
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

func TestServiceAccountHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewServiceAccountHandler(client)
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
