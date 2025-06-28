package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PersistentVolumeHandler handles collection of persistentvolume metrics
type PersistentVolumeHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewPersistentVolumeHandler creates a new PersistentVolumeHandler
func NewPersistentVolumeHandler(client *kubernetes.Clientset) *PersistentVolumeHandler {
	return &PersistentVolumeHandler{
		client: client,
	}
}

// SetupInformer sets up the persistentvolume informer
func (h *PersistentVolumeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create persistentvolume informer
	h.informer = factory.Core().V1().PersistentVolumes().Informer()

	return nil
}

// Collect gathers persistentvolume metrics from the cluster (uses cache)
func (h *PersistentVolumeHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all persistentvolumes from the cache
	pvList := utils.SafeGetStoreList(h.informer)

	for _, obj := range pvList {
		pv, ok := obj.(*corev1.PersistentVolume)
		if !ok {
			continue
		}

		entry := h.createLogEntry(pv)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a persistentvolume
func (h *PersistentVolumeHandler) createLogEntry(pv *corev1.PersistentVolume) types.LogEntry {
	// Calculate capacity in bytes
	capacityBytes := int64(0)
	if pv.Spec.Capacity != nil {
		if storage, exists := pv.Spec.Capacity[corev1.ResourceStorage]; exists {
			capacityBytes = storage.Value()
		}
	}

	// Format access modes
	accessModes := ""
	for i, mode := range pv.Spec.AccessModes {
		if i > 0 {
			accessModes += ","
		}
		accessModes += string(mode)
	}

	// Get volume plugin name
	volumePluginName := h.getVolumePluginName(pv)

	// Get persistent volume source
	persistentVolumeSource := h.getPersistentVolumeSource(pv)

	// Create data structure
	data := types.PersistentVolumeData{
		CreatedTimestamp:       pv.CreationTimestamp.Unix(),
		Labels:                 pv.Labels,
		Annotations:            pv.Annotations,
		CapacityBytes:          capacityBytes,
		AccessModes:            accessModes,
		ReclaimPolicy:          string(pv.Spec.PersistentVolumeReclaimPolicy),
		Status:                 string(pv.Status.Phase),
		StorageClassName:       pv.Spec.StorageClassName,
		VolumeMode:             string(*pv.Spec.VolumeMode),
		VolumePluginName:       volumePluginName,
		PersistentVolumeSource: persistentVolumeSource,
		CreatedByKind:          "",
		CreatedByName:          "",
		IsDefaultClass:         false, // This is typically determined by StorageClass, not PV
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "persistentvolume",
		Name:         pv.Name,
		Namespace:    "", // PVs are cluster-scoped
		Data:         utils.ConvertStructToMap(data),
	}
}

// getVolumePluginName determines the volume plugin name from the PV spec
func (h *PersistentVolumeHandler) getVolumePluginName(pv *corev1.PersistentVolume) string {
	if pv.Spec.HostPath != nil {
		return "hostPath"
	}
	if pv.Spec.GCEPersistentDisk != nil {
		return "gcePersistentDisk"
	}
	if pv.Spec.AWSElasticBlockStore != nil {
		return "awsElasticBlockStore"
	}
	if pv.Spec.NFS != nil {
		return "nfs"
	}
	if pv.Spec.ISCSI != nil {
		return "iscsi"
	}
	if pv.Spec.FC != nil {
		return "fc"
	}
	if pv.Spec.Flocker != nil {
		return "flocker"
	}
	if pv.Spec.FlexVolume != nil {
		return "flexVolume"
	}
	if pv.Spec.Cinder != nil {
		return "cinder"
	}
	if pv.Spec.CephFS != nil {
		return "cephfs"
	}
	if pv.Spec.Flocker != nil {
		return "flocker"
	}
	if pv.Spec.AzureFile != nil {
		return "azureFile"
	}
	if pv.Spec.VsphereVolume != nil {
		return "vsphereVolume"
	}
	if pv.Spec.Quobyte != nil {
		return "quobyte"
	}
	if pv.Spec.AzureDisk != nil {
		return "azureDisk"
	}
	if pv.Spec.PhotonPersistentDisk != nil {
		return "photonPersistentDisk"
	}
	if pv.Spec.PortworxVolume != nil {
		return "portworxVolume"
	}
	if pv.Spec.ScaleIO != nil {
		return "scaleIO"
	}
	if pv.Spec.Local != nil {
		return "local"
	}
	if pv.Spec.StorageOS != nil {
		return "storageOS"
	}
	if pv.Spec.CSI != nil {
		return "csi"
	}
	return "unknown"
}

// getPersistentVolumeSource returns a string representation of the volume source
func (h *PersistentVolumeHandler) getPersistentVolumeSource(pv *corev1.PersistentVolume) string {
	if pv.Spec.HostPath != nil {
		return "hostPath"
	}
	if pv.Spec.GCEPersistentDisk != nil {
		return "gcePersistentDisk"
	}
	if pv.Spec.AWSElasticBlockStore != nil {
		return "awsElasticBlockStore"
	}
	if pv.Spec.NFS != nil {
		return "nfs"
	}
	if pv.Spec.ISCSI != nil {
		return "iscsi"
	}
	if pv.Spec.FC != nil {
		return "fc"
	}
	if pv.Spec.Flocker != nil {
		return "flocker"
	}
	if pv.Spec.FlexVolume != nil {
		return "flexVolume"
	}
	if pv.Spec.Cinder != nil {
		return "cinder"
	}
	if pv.Spec.CephFS != nil {
		return "cephfs"
	}
	if pv.Spec.AzureFile != nil {
		return "azureFile"
	}
	if pv.Spec.VsphereVolume != nil {
		return "vsphereVolume"
	}
	if pv.Spec.Quobyte != nil {
		return "quobyte"
	}
	if pv.Spec.AzureDisk != nil {
		return "azureDisk"
	}
	if pv.Spec.PhotonPersistentDisk != nil {
		return "photonPersistentDisk"
	}
	if pv.Spec.PortworxVolume != nil {
		return "portworxVolume"
	}
	if pv.Spec.ScaleIO != nil {
		return "scaleIO"
	}
	if pv.Spec.Local != nil {
		return "local"
	}
	if pv.Spec.StorageOS != nil {
		return "storageOS"
	}
	if pv.Spec.CSI != nil {
		return "csi"
	}
	return "unknown"
}
