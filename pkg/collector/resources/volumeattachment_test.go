package resources

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func createTestVolumeAttachment(name string) *storagev1.VolumeAttachment {
	pvName := "test-pv"
	return &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Labels:            map[string]string{"app": "test-app"},
			Annotations:       map[string]string{"test-annotation": "test-value"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Attacher: "test-attacher",
			Source: storagev1.VolumeAttachmentSource{
				PersistentVolumeName: &pvName,
			},
			NodeName: "test-node",
		},
		Status: storagev1.VolumeAttachmentStatus{
			Attached:           true,
			AttachmentMetadata: map[string]string{"key": "value"},
		},
	}
}

func TestNewVolumeAttachmentHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestVolumeAttachmentHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
	logger := &testutils.MockLogger{}
	factory := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&storagev1.VolumeAttachment{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(factory, logger)
	if handler.GetInformer() == nil {
		t.Fatal("Expected informer to be set up")
	}
}

func TestVolumeAttachmentHandler_SetupInformer_Proper(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
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

func TestVolumeAttachmentHandler_Collect(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
	logger := &testutils.MockLogger{}
	va := createTestVolumeAttachment("test-va")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&storagev1.VolumeAttachment{},
		0,
		cache.Indexers{},
	)
	handler.SetupBaseInformer(informer, logger)
	store := informer.GetStore()
	store.Add(va)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.ResourceType != "volumeattachment" {
		t.Errorf("Expected resource type 'volumeattachment', got %s", entry.ResourceType)
	}
	if entry.Name != "test-va" {
		t.Errorf("Expected name 'test-va', got %s", entry.Name)
	}
	data := entry.Data
	if data["attacher"] != "test-attacher" {
		t.Errorf("Expected attacher 'test-attacher', got %s", data["attacher"])
	}
	if data["volumeName"] != "test-pv" {
		t.Errorf("Expected volumeName 'test-pv', got %s", data["volumeName"])
	}
	if data["nodeName"] != "test-node" {
		t.Errorf("Expected nodeName 'test-node', got %s", data["nodeName"])
	}
	if data["attached"] != true {
		t.Errorf("Expected attached true, got %v", data["attached"])
	}
	meta, ok := data["attachmentMetadata"].(map[string]string)
	if !ok || meta["key"] != "value" {
		t.Errorf("Expected attachmentMetadata to have key 'key' with value 'value', got %v", data["attachmentMetadata"])
	}
}

func TestVolumeAttachmentHandler_Collect_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&storagev1.VolumeAttachment{},
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

func TestVolumeAttachmentHandler_Collect_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewVolumeAttachmentHandler(client)
	logger := &testutils.MockLogger{}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{},
		&storagev1.VolumeAttachment{},
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
