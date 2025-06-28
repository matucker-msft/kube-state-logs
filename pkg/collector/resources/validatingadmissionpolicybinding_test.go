package resources

import (
	"context"
	"testing"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"k8s.io/client-go/informers"
)

func createTestValidatingAdmissionPolicyBinding(name string) *admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding{
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
		Spec: admissionregistrationv1beta1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: "test-policy",
			ParamRef: &admissionregistrationv1beta1.ParamRef{
				Name: "test-param",
			},
			MatchResources: &admissionregistrationv1beta1.MatchResources{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"namespace-label": "namespace-value",
					},
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"object-label": "object-value",
					},
				},
			},
			ValidationActions: []admissionregistrationv1beta1.ValidationAction{
				admissionregistrationv1beta1.Deny,
				admissionregistrationv1beta1.Warn,
			},
		},
	}
}

func TestNewValidatingAdmissionPolicyBindingHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestValidatingAdmissionPolicyBindingHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestValidatingAdmissionPolicyBindingHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)
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

func TestValidatingAdmissionPolicyBindingHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)
	logger := &testutils.MockLogger{}

	// Create test validatingadmissionpolicybinding
	vapb := createTestValidatingAdmissionPolicyBinding("test-vapb")

	// Create informer and add test object
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)

	// Add test object to store
	store := informer.GetStore()
	store.Add(vapb)

	// Collect entries
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ResourceType != "validatingadmissionpolicybinding" {
		t.Errorf("Expected resource type 'validatingadmissionpolicybinding', got %s", entry.ResourceType)
	}

	if entry.Name != "test-vapb" {
		t.Errorf("Expected name 'test-vapb', got %s", entry.Name)
	}

	// Verify data
	data := entry.Data
	if data["policyName"] != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got %s", data["policyName"])
	}

	if data["paramRef"] != "test-param" {
		t.Errorf("Expected param ref 'test-param', got %s", data["paramRef"])
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

func TestValidatingAdmissionPolicyBindingHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)
	logger := &testutils.MockLogger{}

	// Create empty informer
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding{},
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

func TestValidatingAdmissionPolicyBindingHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)
	logger := &testutils.MockLogger{}

	// Create informer
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding{},
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

func TestValidatingAdmissionPolicyBindingHandler_CreateLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)

	// Create test validatingadmissionpolicybinding with owner reference
	ownerRef := metav1.OwnerReference{
		Kind: "Deployment",
		Name: "test-deployment",
	}
	vapb := createTestValidatingAdmissionPolicyBinding("test-vapb")
	vapb.OwnerReferences = []metav1.OwnerReference{ownerRef}

	// Create log entry
	entry := handler.createLogEntry(vapb)

	if entry.ResourceType != "validatingadmissionpolicybinding" {
		t.Errorf("Expected resource type 'validatingadmissionpolicybinding', got %s", entry.ResourceType)
	}

	if entry.Name != "test-vapb" {
		t.Errorf("Expected name 'test-vapb', got %s", entry.Name)
	}

	// Verify data
	data := entry.Data
	if data["createdByKind"] != "Deployment" {
		t.Errorf("Expected created by kind 'Deployment', got %s", data["createdByKind"])
	}

	if data["createdByName"] != "test-deployment" {
		t.Errorf("Expected created by name 'test-deployment', got %s", data["createdByName"])
	}

	if data["policyName"] != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got %s", data["policyName"])
	}

	if data["paramRef"] != "test-param" {
		t.Errorf("Expected param ref 'test-param', got %s", data["paramRef"])
	}
}

func TestValidatingAdmissionPolicyBindingHandler_CreateLogEntry_NoParamRef(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)

	// Create test validatingadmissionpolicybinding without param ref
	vapb := createTestValidatingAdmissionPolicyBinding("test-vapb")
	vapb.Spec.ParamRef = nil

	// Create log entry
	entry := handler.createLogEntry(vapb)

	// Verify data
	data := entry.Data
	if data["paramRef"] != "" {
		t.Errorf("Expected empty param ref, got %s", data["paramRef"])
	}
}

func TestValidatingAdmissionPolicyBindingHandler_CreateLogEntry_NoPolicyName(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewValidatingAdmissionPolicyBindingHandler(client)

	// Create test validatingadmissionpolicybinding without policy name
	vapb := createTestValidatingAdmissionPolicyBinding("test-vapb")
	vapb.Spec.PolicyName = ""

	// Create log entry
	entry := handler.createLogEntry(vapb)

	// Verify data
	data := entry.Data
	if data["policyName"] != "" {
		t.Errorf("Expected empty policy name, got %s", data["policyName"])
	}
}
