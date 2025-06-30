package resources

import (
	"context"
	"time"

	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PriorityClassHandler handles collection of priorityclass metrics
type PriorityClassHandler struct {
	utils.BaseHandler
}

// NewPriorityClassHandler creates a new PriorityClassHandler
func NewPriorityClassHandler(client kubernetes.Interface) *PriorityClassHandler {
	return &PriorityClassHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the priorityclass informer
func (h *PriorityClassHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create priorityclass informer
	informer := factory.Scheduling().V1().PriorityClasses().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers priorityclass metrics from the cluster (uses cache)
func (h *PriorityClassHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all priorityclasses from the cache
	priorityClasses := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range priorityClasses {
		pc, ok := obj.(*schedulingv1.PriorityClass)
		if !ok {
			continue
		}

		entry := h.createLogEntry(pc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a PriorityClassData from a priorityclass
func (h *PriorityClassHandler) createLogEntry(pc *schedulingv1.PriorityClass) types.PriorityClassData {
	createdTimestamp := utils.ExtractCreationTimestamp(pc)
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pc)

	preemptionPolicy := ""
	if pc.PreemptionPolicy != nil {
		preemptionPolicy = string(*pc.PreemptionPolicy)
	}

	data := types.PriorityClassData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "priorityclass",
			Name:             utils.ExtractName(pc),
			Namespace:        utils.ExtractNamespace(pc),
			CreatedTimestamp: createdTimestamp,
			Labels:           utils.ExtractLabels(pc),
			Annotations:      utils.ExtractAnnotations(pc),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		Value:            pc.Value,
		GlobalDefault:    pc.GlobalDefault,
		Description:      pc.Description,
		PreemptionPolicy: preemptionPolicy,
	}

	return data
}
