package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"go.goms.io/aks/kube-state-logs/pkg/interfaces"
	"go.goms.io/aks/kube-state-logs/pkg/types"
	"go.goms.io/aks/kube-state-logs/pkg/utils"
)

// PersistentVolumeClaimHandler handles collection of persistentvolumeclaim metrics
type PersistentVolumeClaimHandler struct {
	utils.BaseHandler
}

// NewPersistentVolumeClaimHandler creates a new PersistentVolumeClaimHandler
func NewPersistentVolumeClaimHandler(client kubernetes.Interface) *PersistentVolumeClaimHandler {
	return &PersistentVolumeClaimHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the persistentvolumeclaim informer
func (h *PersistentVolumeClaimHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create persistentvolumeclaim informer
	informer := factory.Core().V1().PersistentVolumeClaims().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers persistentvolumeclaim metrics from the cluster (uses cache)
func (h *PersistentVolumeClaimHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all persistentvolumeclaims from the cache
	pvcs := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range pvcs {
		pvc, ok := obj.(*corev1.PersistentVolumeClaim)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pvc.Namespace) {
			continue
		}

		entry := h.createLogEntry(pvc)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a PersistentVolumeClaimData from a persistentvolumeclaim
func (h *PersistentVolumeClaimHandler) createLogEntry(pvc *corev1.PersistentVolumeClaim) types.PersistentVolumeClaimData {
	var accessModes []string
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	var storageClassName *string
	if pvc.Spec.StorageClassName != nil {
		storageClassName = pvc.Spec.StorageClassName
	}

	capacity := utils.ExtractResourceMap(pvc.Status.Capacity)

	requestStorage := ""
	if pvc.Spec.Resources.Requests != nil {
		if storage, exists := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; exists {
			requestStorage = storage.String()
		}
	}

	usedStorage := ""
	if pvc.Status.Capacity != nil {
		if storage, exists := pvc.Status.Capacity[corev1.ResourceStorage]; exists {
			usedStorage = storage.String()
		}
	}

	// Check conditions in a single loop
	var conditionPending, conditionBound, conditionLost *bool
	conditions := make(map[string]*bool)

	for _, condition := range pvc.Status.Conditions {
		val := utils.ConvertCoreConditionStatus(condition.Status)

		switch condition.Type {
		case "Pending":
			conditionPending = val
		case "Bound":
			conditionBound = val
		case "Lost":
			conditionLost = val
		default:
			// Add unknown conditions to the map
			conditions[string(condition.Type)] = val
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pvc)

	data := types.PersistentVolumeClaimData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "persistentvolumeclaim",
			Name:             utils.ExtractName(pvc),
			Namespace:        utils.ExtractNamespace(pvc),
			CreatedTimestamp: utils.ExtractCreationTimestamp(pvc),
			Labels:           utils.ExtractLabels(pvc),
			Annotations:      utils.ExtractAnnotations(pvc),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		AccessModes:      accessModes,
		StorageClassName: storageClassName,
		VolumeName:       pvc.Spec.VolumeName,
		Phase:            string(pvc.Status.Phase),
		Capacity:         capacity,
		ConditionPending: conditionPending,
		ConditionBound:   conditionBound,
		ConditionLost:    conditionLost,
		RequestStorage:   requestStorage,
		UsedStorage:      usedStorage,
		Conditions:       conditions,
	}

	return data
}
