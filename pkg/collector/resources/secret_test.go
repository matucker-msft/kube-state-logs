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

func TestSecretHandler(t *testing.T) {
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "default",
			Labels:    map[string]string{"env": "prod"},
			Annotations: map[string]string{
				"purpose": "test",
			},
			CreationTimestamp: metav1.Now(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"password": []byte("supersecret"),
			"token":    []byte("tokendata"),
		},
	}

	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "docker-secret",
			Namespace:         "kube-system",
			Labels:            map[string]string{"env": "system"},
			CreationTimestamp: metav1.Now(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte("{\"auths\":{}}"),
		},
	}

	secretWithOwner := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-secret",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "ServiceAccount", Name: "my-sa"}},
			CreationTimestamp: metav1.Now(),
		},
		Type: corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{
			"token": []byte("sometoken"),
		},
	}

	tests := []struct {
		name           string
		secrets        []*corev1.Secret
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all secrets",
			secrets:       []*corev1.Secret{secret1, secret2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"my-secret", "docker-secret"},
		},
		{
			name:          "collect secrets from specific namespace",
			secrets:       []*corev1.Secret{secret1, secret2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"my-secret"},
		},
		{
			name:          "collect secret with owner reference",
			secrets:       []*corev1.Secret{secretWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-secret"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "ServiceAccount",
				"created_by_name": "my-sa",
			},
		},
		{
			name:          "collect secret with data keys",
			secrets:       []*corev1.Secret{secret1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"my-secret"},
			expectedFields: map[string]interface{}{
				"data_keys": []string{"password", "token"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.secrets))
			for i, s := range tt.secrets {
				objects[i] = s
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewSecretHandler(client)
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
					t.Errorf("Expected to find secret with name %s", expectedName)
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
					case "data_keys":
						expectedKeys := expectedValue.([]string)
						dataKeys, ok := entry.Data["dataKeys"].([]string)
						if !ok {
							t.Errorf("Expected dataKeys to be []string, got %T", entry.Data["dataKeys"])
						} else if len(dataKeys) != len(expectedKeys) {
							t.Errorf("Expected %d data keys, got %d", len(expectedKeys), len(dataKeys))
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "secret" {
					t.Errorf("Expected resource type 'secret', got %s", entry.ResourceType)
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
				if entry.Data["type"] == nil {
					t.Error("Secret type should not be nil")
				}
				if entry.Data["dataKeys"] == nil {
					t.Error("dataKeys should not be nil")
				}
				// Ensure secret values are never present
				if _, hasPassword := entry.Data["password"]; hasPassword {
					t.Error("Secret value for 'password' should not be present in log entry")
				}
				if _, hasToken := entry.Data["token"]; hasToken {
					t.Error("Secret value for 'token' should not be present in log entry")
				}
				if _, hasDocker := entry.Data[".dockerconfigjson"]; hasDocker {
					t.Error("Secret value for '.dockerconfigjson' should not be present in log entry")
				}
			}
		})
	}
}

func TestSecretHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewSecretHandler(client)
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

func TestSecretHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewSecretHandler(client)
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
