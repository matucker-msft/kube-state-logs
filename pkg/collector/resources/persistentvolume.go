package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PersistentVolumeHandler handles collection of persistentvolume metrics
type PersistentVolumeHandler struct {
	utils.BaseHandler
}

// NewPersistentVolumeHandler creates a new PersistentVolumeHandler
func NewPersistentVolumeHandler(client *kubernetes.Clientset) *PersistentVolumeHandler {
	return &PersistentVolumeHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the persistentvolume informer
func (h *PersistentVolumeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create persistentvolume informer
	informer := factory.Core().V1().PersistentVolumes().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers persistentvolume metrics from the cluster (uses cache)
func (h *PersistentVolumeHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all persistentvolumes from the cache
	pvList := utils.SafeGetStoreList(h.GetInformer())

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
	// Extract access modes
	var accessModes []string
	for _, mode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// Extract capacity
	capacityBytes := int64(0)
	if pv.Spec.Capacity != nil {
		if storage, exists := pv.Spec.Capacity[corev1.ResourceStorage]; exists {
			capacityBytes = storage.Value()
		}
	}

	// Extract reclaim policy
	reclaimPolicy := string(pv.Spec.PersistentVolumeReclaimPolicy)

	// Extract status
	status := string(pv.Status.Phase)

	// Extract storage class name
	storageClassName := ""
	if pv.Spec.StorageClassName != "" {
		storageClassName = pv.Spec.StorageClassName
	}

	// Extract volume mode
	volumeMode := string(*pv.Spec.VolumeMode)

	// Extract volume plugin name
	volumePluginName := ""
	if pv.Spec.PersistentVolumeSource.AWSElasticBlockStore != nil {
		volumePluginName = "awsElasticBlockStore"
	} else if pv.Spec.PersistentVolumeSource.AzureDisk != nil {
		volumePluginName = "azureDisk"
	} else if pv.Spec.PersistentVolumeSource.AzureFile != nil {
		volumePluginName = "azureFile"
	} else if pv.Spec.PersistentVolumeSource.CephFS != nil {
		volumePluginName = "cephFS"
	} else if pv.Spec.PersistentVolumeSource.Cinder != nil {
		volumePluginName = "cinder"
	} else if pv.Spec.PersistentVolumeSource.FC != nil {
		volumePluginName = "fc"
	} else if pv.Spec.PersistentVolumeSource.FlexVolume != nil {
		volumePluginName = "flexVolume"
	} else if pv.Spec.PersistentVolumeSource.Flocker != nil {
		volumePluginName = "flocker"
	} else if pv.Spec.PersistentVolumeSource.GCEPersistentDisk != nil {
		volumePluginName = "gcePersistentDisk"
	} else if pv.Spec.PersistentVolumeSource.Glusterfs != nil {
		volumePluginName = "glusterfs"
	} else if pv.Spec.PersistentVolumeSource.HostPath != nil {
		volumePluginName = "hostPath"
	} else if pv.Spec.PersistentVolumeSource.ISCSI != nil {
		volumePluginName = "iscsi"
	} else if pv.Spec.PersistentVolumeSource.Local != nil {
		volumePluginName = "local"
	} else if pv.Spec.PersistentVolumeSource.NFS != nil {
		volumePluginName = "nfs"
	} else if pv.Spec.PersistentVolumeSource.PhotonPersistentDisk != nil {
		volumePluginName = "photonPersistentDisk"
	} else if pv.Spec.PersistentVolumeSource.PortworxVolume != nil {
		volumePluginName = "portworxVolume"
	} else if pv.Spec.PersistentVolumeSource.Quobyte != nil {
		volumePluginName = "quobyte"
	} else if pv.Spec.PersistentVolumeSource.RBD != nil {
		volumePluginName = "rbd"
	} else if pv.Spec.PersistentVolumeSource.ScaleIO != nil {
		volumePluginName = "scaleIO"
	} else if pv.Spec.PersistentVolumeSource.StorageOS != nil {
		volumePluginName = "storageOS"
	} else if pv.Spec.PersistentVolumeSource.VsphereVolume != nil {
		volumePluginName = "vsphereVolume"
	} else {
		volumePluginName = "unknown"
	}

	// Extract persistent volume source
	persistentVolumeSource := volumePluginName

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pv)

	// Create data structure
	data := types.PersistentVolumeData{
		CreatedTimestamp:       utils.ExtractCreationTimestamp(pv),
		Labels:                 utils.ExtractLabels(pv),
		Annotations:            utils.ExtractAnnotations(pv),
		CapacityBytes:          capacityBytes,
		AccessModes:            accessModes[0], // Take first access mode
		ReclaimPolicy:          reclaimPolicy,
		Status:                 status,
		StorageClassName:       storageClassName,
		VolumeMode:             volumeMode,
		VolumePluginName:       volumePluginName,
		PersistentVolumeSource: persistentVolumeSource,
		CreatedByKind:          createdByKind,
		CreatedByName:          createdByName,
		IsDefaultClass:         false, // This would need to be determined from StorageClass
	}

	return utils.CreateLogEntry("persistentvolume", utils.ExtractName(pv), utils.ExtractNamespace(pv), data)
}
