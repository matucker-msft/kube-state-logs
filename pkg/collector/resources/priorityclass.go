package resources

import (
	"context"
	"time"

	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PriorityClassHandler handles collection of priorityclass metrics
type PriorityClassHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewPriorityClassHandler creates a new PriorityClassHandler
func NewPriorityClassHandler(client *kubernetes.Clientset) *PriorityClassHandler {
	return &PriorityClassHandler{
		client: client,
	}
}

// SetupInformer sets up the priorityclass informer
func (h *PriorityClassHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create priorityclass informer
	h.informer = factory.Scheduling().V1().PriorityClasses().Informer()

	return nil
}

// Collect gathers priorityclass metrics from the cluster (uses cache)
func (h *PriorityClassHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all priorityclasses from the cache
	pcList := safeGetStoreList(h.informer)

	for _, obj := range pcList {
		pc, ok := obj.(*schedulingv1.PriorityClass)
		if !ok {
			continue
		}

		entry := h.createLogEntry(pc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a PriorityClass
func (h *PriorityClassHandler) createLogEntry(pc *schedulingv1.PriorityClass) types.LogEntry {
	// Extract basic metadata
	createdTimestamp := int64(0)
	if creationTime := pc.GetCreationTimestamp(); !creationTime.IsZero() {
		createdTimestamp = creationTime.Unix()
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pc)

	// Create data structure
	// Default preemption policy is "PreemptLowerOrEqual" when not specified
	// See: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#preemption-policies
	data := types.PriorityClassData{
		CreatedTimestamp: createdTimestamp,
		Labels:           pc.GetLabels(),
		Annotations:      pc.GetAnnotations(),
		Value:            pc.Value,
		GlobalDefault:    pc.GlobalDefault,
		Description:      pc.Description,
		PreemptionPolicy: string(*pc.PreemptionPolicy),
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "priorityclass",
		Name:         pc.GetName(),
		Namespace:    "", // PriorityClass is cluster-scoped
		Data:         convertStructToMap(data),
	}
}
