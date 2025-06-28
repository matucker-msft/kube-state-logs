package resources

import (
	"context"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

func createTestMutatingWebhookConfiguration(name string) *admissionregistrationv1.MutatingWebhookConfiguration {
	url := "https://webhook.example.com/mutate"
	path := "/mutate"
	port := int32(443)
	failurePolicy := admissionregistrationv1.Fail
	matchPolicy := admissionregistrationv1.Equivalent
	sideEffects := admissionregistrationv1.SideEffectClassNone
	timeoutSeconds := int32(30)
	scope := admissionregistrationv1.NamespacedScope

	return &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "test-app",
			},
			Annotations: map[string]string{
				"test-annotation": "test-value",
			},
			CreationTimestamp: metav1.Now(),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test-webhook",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: &url,
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: "default",
						Name:      "webhook-service",
						Path:      &path,
						Port:      &port,
					},
					CABundle: []byte("test-ca-bundle"),
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"apps"},
							APIVersions: []string{"v1"},
							Resources:   []string{"deployments"},
							Scope:       &scope,
						},
					},
				},
				FailurePolicy:           &failurePolicy,
				MatchPolicy:             &matchPolicy,
				SideEffects:             &sideEffects,
				TimeoutSeconds:          &timeoutSeconds,
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
}

func TestNewMutatingWebhookConfigurationHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestMutatingWebhookConfigurationHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestMutatingWebhookConfigurationHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)
	logger := &testutils.MockLogger{}

	// Create a proper informer factory
	factory := informers.NewSharedInformerFactory(client, 0)

	err := handler.SetupInformer(factory, logger, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestMutatingWebhookConfigurationHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)
	logger := &testutils.MockLogger{}

	// Create test mutatingwebhookconfiguration
	mwc := createTestMutatingWebhookConfiguration("test-mwc")

	// Create informer and add test object
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)

	// Add test object to store
	store := informer.GetStore()
	store.Add(mwc)

	// Collect entries
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ResourceType != "mutatingwebhookconfiguration" {
		t.Errorf("Expected resource type 'mutatingwebhookconfiguration', got %s", entry.ResourceType)
	}

	if entry.Name != "test-mwc" {
		t.Errorf("Expected name 'test-mwc', got %s", entry.Name)
	}

	// Verify data - webhooks are stored as the original struct type
	data := entry.Data
	webhooks, ok := data["webhooks"].([]types.WebhookData)
	if !ok {
		t.Fatal("Expected webhooks to be []types.WebhookData")
	}

	if len(webhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(webhooks))
	}

	webhook := webhooks[0]
	if webhook.Name != "test-webhook" {
		t.Errorf("Expected webhook name 'test-webhook', got %s", webhook.Name)
	}

	if webhook.ClientConfig.URL != "https://webhook.example.com/mutate" {
		t.Errorf("Expected URL 'https://webhook.example.com/mutate', got %s", webhook.ClientConfig.URL)
	}

	if webhook.ClientConfig.Service == nil {
		t.Fatal("Expected service config to be present")
	}

	if webhook.ClientConfig.Service.Name != "webhook-service" {
		t.Errorf("Expected service name 'webhook-service', got %s", webhook.ClientConfig.Service.Name)
	}

	if len(webhook.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(webhook.Rules))
	}

	rule := webhook.Rules[0]
	if rule.Scope != "Namespaced" {
		t.Errorf("Expected scope 'Namespaced', got %s", rule.Scope)
	}
}

func TestMutatingWebhookConfigurationHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)
	logger := &testutils.MockLogger{}

	// Create empty informer
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)

	// Collect entries
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}

func TestMutatingWebhookConfigurationHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)
	logger := &testutils.MockLogger{}

	// Create informer
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)

	// Add invalid object to store
	store := informer.GetStore()
	store.Add(&corev1.Pod{}) // Wrong type

	// Collect entries
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}

func TestMutatingWebhookConfigurationHandler_CreateLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)

	// Create test mutatingwebhookconfiguration with owner reference
	ownerRef := metav1.OwnerReference{
		Kind: "Deployment",
		Name: "test-deployment",
	}
	mwc := createTestMutatingWebhookConfiguration("test-mwc")
	mwc.OwnerReferences = []metav1.OwnerReference{ownerRef}

	// Create log entry
	entry := handler.createLogEntry(mwc)

	if entry.ResourceType != "mutatingwebhookconfiguration" {
		t.Errorf("Expected resource type 'mutatingwebhookconfiguration', got %s", entry.ResourceType)
	}

	if entry.Name != "test-mwc" {
		t.Errorf("Expected name 'test-mwc', got %s", entry.Name)
	}

	// Verify data
	data := entry.Data
	if data["createdByKind"] != "Deployment" {
		t.Errorf("Expected created by kind 'Deployment', got %s", data["createdByKind"])
	}

	if data["createdByName"] != "test-deployment" {
		t.Errorf("Expected created by name 'test-deployment', got %s", data["createdByName"])
	}

	labels, ok := data["labels"].(map[string]string)
	if !ok {
		t.Fatal("Expected labels to be map[string]string")
	}

	if labels["app"] != "test-app" {
		t.Errorf("Expected label 'app' to be 'test-app', got %s", labels["app"])
	}

	annotations, ok := data["annotations"].(map[string]string)
	if !ok {
		t.Fatal("Expected annotations to be map[string]string")
	}

	if annotations["test-annotation"] != "test-value" {
		t.Errorf("Expected annotation 'test-annotation' to be 'test-value', got %s", annotations["test-annotation"])
	}
}

func TestMutatingWebhookConfigurationHandler_CreateLogEntry_WithSelectors(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewMutatingWebhookConfigurationHandler(client)

	// Create test mutatingwebhookconfiguration with selectors
	mwc := createTestMutatingWebhookConfiguration("test-mwc")
	mwc.Webhooks[0].NamespaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"namespace-label": "namespace-value",
		},
	}
	mwc.Webhooks[0].ObjectSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"object-label": "object-value",
		},
	}

	// Create log entry
	entry := handler.createLogEntry(mwc)

	// Verify data - webhooks are stored as the original struct type
	data := entry.Data
	webhooks, ok := data["webhooks"].([]types.WebhookData)
	if !ok {
		t.Fatal("Expected webhooks to be []types.WebhookData")
	}

	webhook := webhooks[0]
	if webhook.NamespaceSelector["namespace-label"] != "namespace-value" {
		t.Errorf("Expected namespace selector 'namespace-label' to be 'namespace-value', got %s", webhook.NamespaceSelector["namespace-label"])
	}

	if webhook.ObjectSelector["object-label"] != "object-value" {
		t.Errorf("Expected object selector 'object-label' to be 'object-value', got %s", webhook.ObjectSelector["object-label"])
	}
}
