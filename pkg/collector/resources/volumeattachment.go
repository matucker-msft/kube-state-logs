package resources

import (
	"context"
	"time"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// VolumeAttachmentHandler handles collection of volumeattachment metrics
type VolumeAttachmentHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewVolumeAttachmentHandler creates a new VolumeAttachmentHandler
func NewVolumeAttachmentHandler(client *kubernetes.Clientset) *VolumeAttachmentHandler {
	return &VolumeAttachmentHandler{
		client: client,
	}
}

// SetupInformer sets up the volumeattachment informer
func (h *VolumeAttachmentHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create volumeattachment informer
	h.informer = factory.Storage().V1().VolumeAttachments().Informer()

	return nil
}

// Collect gathers volumeattachment metrics from the cluster (uses cache)
func (h *VolumeAttachmentHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all volumeattachments from the cache
	vaList := utils.SafeGetStoreList(h.informer)

	for _, obj := range vaList {
		va, ok := obj.(*storagev1.VolumeAttachment)
		if !ok {
			continue
		}

		entry := h.createLogEntry(va)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a volumeattachment
func (h *VolumeAttachmentHandler) createLogEntry(va *storagev1.VolumeAttachment) types.LogEntry {
	// Get attachment metadata
	// See: https://kubernetes.io/docs/concepts/storage/volume-attachments/
	attachmentMetadata := make(map[string]string)
	if va.Status.AttachmentMetadata != nil {
		for key, value := range va.Status.AttachmentMetadata {
			attachmentMetadata[key] = value
		}
	}

	// Get volume name
	volumeName := ""
	if va.Spec.Source.PersistentVolumeName != nil {
		volumeName = *va.Spec.Source.PersistentVolumeName
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(va)

	// Create data structure
	data := types.VolumeAttachmentData{
		CreatedTimestamp:   va.CreationTimestamp.Unix(),
		Labels:             va.Labels,
		Annotations:        va.Annotations,
		Attacher:           va.Spec.Attacher,
		VolumeName:         volumeName,
		NodeName:           va.Spec.NodeName,
		Attached:           va.Status.Attached,
		AttachmentMetadata: attachmentMetadata,
		CreatedByKind:      createdByKind,
		CreatedByName:      createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "volumeattachment",
		Name:         va.Name,
		Namespace:    "", // VolumeAttachments are cluster-scoped
		Data:         utils.ConvertStructToMap(data),
	}
}
