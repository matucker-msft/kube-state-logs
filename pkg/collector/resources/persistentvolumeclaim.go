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

// PersistentVolumeClaimHandler handles collection of persistentvolumeclaim metrics
type PersistentVolumeClaimHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewPersistentVolumeClaimHandler creates a new PersistentVolumeClaimHandler
func NewPersistentVolumeClaimHandler(client *kubernetes.Clientset) *PersistentVolumeClaimHandler {
	return &PersistentVolumeClaimHandler{
		client: client,
	}
}

// SetupInformer sets up the persistentvolumeclaim informer
func (h *PersistentVolumeClaimHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create persistentvolumeclaim informer
	h.informer = factory.Core().V1().PersistentVolumeClaims().Informer()

	return nil
}

// Collect gathers persistentvolumeclaim metrics from the cluster (uses cache)
func (h *PersistentVolumeClaimHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all persistentvolumeclaims from the cache
	pvcs := utils.SafeGetStoreList(h.informer)

	for _, obj := range pvcs {
		pvc, ok := obj.(*corev1.PersistentVolumeClaim)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pvc.Namespace) {
			continue
		}

		entry := h.createLogEntry(pvc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a persistentvolumeclaim
func (h *PersistentVolumeClaimHandler) createLogEntry(pvc *corev1.PersistentVolumeClaim) types.LogEntry {
	var accessModes []string
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	var storageClassName *string
	if pvc.Spec.StorageClassName != nil {
		storageClassName = pvc.Spec.StorageClassName
	}

	capacity := make(map[string]string)
	if pvc.Status.Capacity != nil {
		for resource, quantity := range pvc.Status.Capacity {
			capacity[string(resource)] = quantity.String()
		}
	}

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

	conditionPending := false
	conditionBound := false
	conditionLost := false
	for _, condition := range pvc.Status.Conditions {
		switch condition.Type {
		case "Pending":
			conditionPending = condition.Status == corev1.ConditionTrue
		case "Bound":
			conditionBound = condition.Status == corev1.ConditionTrue
		case "Lost":
			conditionLost = condition.Status == corev1.ConditionTrue
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pvc)

	data := types.PersistentVolumeClaimData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(pvc),
		Labels:           utils.ExtractLabels(pvc),
		Annotations:      utils.ExtractAnnotations(pvc),
		AccessModes:      accessModes,
		StorageClassName: storageClassName,
		VolumeName:       pvc.Spec.VolumeName,
		Phase:            string(pvc.Status.Phase),
		Capacity:         capacity,
		ConditionPending: conditionPending,
		ConditionBound:   conditionBound,
		ConditionLost:    conditionLost,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
		RequestStorage:   requestStorage,
		UsedStorage:      usedStorage,
	}

	return utils.CreateLogEntry("persistentvolumeclaim", utils.ExtractName(pvc), utils.ExtractNamespace(pvc), data)
}
