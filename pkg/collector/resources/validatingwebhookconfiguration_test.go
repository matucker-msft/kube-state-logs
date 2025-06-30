package resources

import (
	"context"
	"testing"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

func TestValidatingWebhookConfigurationHandler(t *testing.T) {
	failurePolicy := admissionregistrationv1.Fail
	matchPolicy := admissionregistrationv1.Equivalent
	sideEffects := admissionregistrationv1.SideEffectClassNone
	timeout := int32(10)
	path := "/validate"
	port := int32(443)

	vwc1 := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "vwc-1",
			Labels:            map[string]string{"env": "prod"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "webhook-1",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL:      strPtr("https://webhook1.example.com"),
					CABundle: []byte("fake-ca-bundle"),
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: "default",
						Name:      "svc-1",
						Path:      &path,
						Port:      &port,
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"apps"},
							APIVersions: []string{"v1"},
							Resources:   []string{"deployments"},
							Scope:       &[]admissionregistrationv1.ScopeType{admissionregistrationv1.NamespacedScope}[0],
						},
					},
				},
				FailurePolicy:           &failurePolicy,
				MatchPolicy:             &matchPolicy,
				NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"team": "devops"}},
				ObjectSelector:          &metav1.LabelSelector{MatchLabels: map[string]string{"tier": "backend"}},
				SideEffects:             &sideEffects,
				TimeoutSeconds:          &timeout,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
			},
		},
	}

	vwc2 := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "vwc-2",
			CreationTimestamp: metav1.Now(),
		},
	}

	vwcWithOwner := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-vwc",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
	}

	tests := []struct {
		name           string
		vwcs           []*admissionregistrationv1.ValidatingWebhookConfiguration
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all validating webhook configurations",
			vwcs:          []*admissionregistrationv1.ValidatingWebhookConfiguration{vwc1, vwc2},
			expectedCount: 2,
			expectedNames: []string{"vwc-1", "vwc-2"},
		},
		{
			name:          "collect validating webhook configuration with owner reference",
			vwcs:          []*admissionregistrationv1.ValidatingWebhookConfiguration{vwcWithOwner},
			expectedCount: 1,
			expectedNames: []string{"owned-vwc"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.vwcs))
			for i, vwc := range tt.vwcs {
				objects[i] = vwc
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewValidatingWebhookConfigurationHandler(client)
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
				webhookConfigData, ok := entry.(types.ValidatingWebhookConfigurationData)
				if !ok {
					t.Fatalf("Expected ValidatingWebhookConfigurationData type, got %T", entry)
				}
				entryNames[i] = webhookConfigData.Name
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
					t.Errorf("Expected to find validating webhook configuration with name %s", expectedName)
				}
			}
			if tt.expectedFields != nil && len(entries) > 0 {
				webhookConfigData, ok := entries[0].(types.ValidatingWebhookConfigurationData)
				if !ok {
					t.Fatalf("Expected ValidatingWebhookConfigurationData type, got %T", entries[0])
				}
				for field, expectedValue := range tt.expectedFields {
					switch field {
					case "created_by_kind":
						if webhookConfigData.CreatedByKind != expectedValue {
							t.Errorf("Expected created_by_kind to be %v, got %v", expectedValue, webhookConfigData.CreatedByKind)
						}
					case "created_by_name":
						if webhookConfigData.CreatedByName != expectedValue {
							t.Errorf("Expected created_by_name to be %v, got %v", expectedValue, webhookConfigData.CreatedByName)
						}
					}
				}
			}
			for _, entry := range entries {
				webhookConfigData, ok := entry.(types.ValidatingWebhookConfigurationData)
				if !ok {
					t.Fatalf("Expected ValidatingWebhookConfigurationData type, got %T", entry)
				}
				if webhookConfigData.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if webhookConfigData.CreatedTimestamp == 0 {
					t.Error("Created timestamp should not be zero")
				}
				if webhookConfigData.Webhooks == nil {
					t.Error("webhooks should not be nil")
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestValidatingWebhookConfigurationHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingWebhookConfigurationHandler(client)
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

func TestValidatingWebhookConfigurationHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingWebhookConfigurationHandler(client)
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

func createTestValidatingWebhookConfiguration(name string) *admissionregistrationv1.ValidatingWebhookConfiguration {
	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Now(),
		},
	}
}

func TestValidatingWebhookConfigurationHandler_Collect(t *testing.T) {
	// Create test validating webhook configurations
	vwc1 := createTestValidatingWebhookConfiguration("test-vwc-1")
	vwc2 := createTestValidatingWebhookConfiguration("test-vwc-2")

	// Create fake client with test validating webhook configurations
	client := fake.NewSimpleClientset(vwc1, vwc2)
	handler := NewValidatingWebhookConfigurationHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}

	// Setup informer
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}

	// Start the factory to populate the cache
	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	// Test collecting all validating webhook configurations
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Verify entries are of correct type
	for _, entry := range entries {
		_, ok := entry.(types.ValidatingWebhookConfigurationData)
		if !ok {
			t.Fatalf("Expected ValidatingWebhookConfigurationData type, got %T", entry)
		}
	}
}
