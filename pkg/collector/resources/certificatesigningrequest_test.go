package resources

import (
	"context"
	"testing"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func createTestCertificateSigningRequest(name string) *certificatesv1.CertificateSigningRequest {
	expirationSeconds := int32(3600)
	return &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Labels:            map[string]string{"app": "test-app"},
			Annotations:       map[string]string{"test-annotation": "test-value"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			SignerName:        "kubernetes.io/kube-apiserver-client",
			ExpirationSeconds: &expirationSeconds,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageClientAuth,
				certificatesv1.UsageServerAuth,
			},
		},
		Status: certificatesv1.CertificateSigningRequestStatus{
			Conditions: []certificatesv1.CertificateSigningRequestCondition{
				{Type: certificatesv1.CertificateApproved},
			},
		},
	}
}

func TestNewCertificateSigningRequestHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestCertificateSigningRequestHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&certificatesv1.CertificateSigningRequest{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestCertificateSigningRequestHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
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

func TestCertificateSigningRequestHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
	logger := &testutils.MockLogger{}
	csr := createTestCertificateSigningRequest("test-csr")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&certificatesv1.CertificateSigningRequest{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(csr)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.ResourceType != "certificatesigningrequest" {
		t.Errorf("Expected resource type 'certificatesigningrequest', got %s", entry.ResourceType)
	}
	if entry.Name != "test-csr" {
		t.Errorf("Expected name 'test-csr', got %s", entry.Name)
	}
	data := entry.Data
	if data["status"] != "Approved" {
		t.Errorf("Expected status 'Approved', got %s", data["status"])
	}
	if data["signerName"] != "kubernetes.io/kube-apiserver-client" {
		t.Errorf("Expected signerName 'kubernetes.io/kube-apiserver-client', got %s", data["signerName"])
	}
	if data["expirationSeconds"].(int32) != 3600 {
		t.Errorf("Expected expirationSeconds 3600, got %v", data["expirationSeconds"])
	}
	usages, ok := data["usages"].([]string)
	if !ok || len(usages) != 2 {
		t.Errorf("Expected usages to be []string of length 2, got %v", data["usages"])
	}
}

func TestCertificateSigningRequestHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&certificatesv1.CertificateSigningRequest{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}

func TestCertificateSigningRequestHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCertificateSigningRequestHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&certificatesv1.CertificateSigningRequest{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(&corev1.Pod{})
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries, got %d", len(entries))
	}
}
