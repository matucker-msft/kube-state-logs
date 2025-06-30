package resources

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// DaemonSetHandler handles collection of daemonset metrics
type DaemonSetHandler struct {
	utils.BaseHandler
}

// NewDaemonSetHandler creates a new DaemonSetHandler
func NewDaemonSetHandler(client kubernetes.Interface) *DaemonSetHandler {
	return &DaemonSetHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the daemonset informer
func (h *DaemonSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create daemonset informer
	informer := factory.Apps().V1().DaemonSets().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers daemonset metrics from the cluster (uses cache)
func (h *DaemonSetHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all daemonsets from the cache
	daemonSets := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range daemonSets {
		ds, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, ds.Namespace) {
			continue
		}

		entry := h.createLogEntry(ds)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a DaemonSetData from a daemonset
func (h *DaemonSetHandler) createLogEntry(ds *appsv1.DaemonSet) types.DaemonSetData {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ds)

	// Get update strategy
	updateStrategy := string(ds.Spec.UpdateStrategy.Type)

	data := types.DaemonSetData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "daemonset",
			Name:             utils.ExtractName(ds),
			Namespace:        utils.ExtractNamespace(ds),
			CreatedTimestamp: utils.ExtractCreationTimestamp(ds),
			Labels:           utils.ExtractLabels(ds),
			Annotations:      utils.ExtractAnnotations(ds),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		DesiredNumberScheduled:  ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled:  ds.Status.CurrentNumberScheduled,
		NumberReady:             ds.Status.NumberReady,
		NumberAvailable:         ds.Status.NumberAvailable,
		NumberUnavailable:       ds.Status.NumberUnavailable,
		NumberMisscheduled:      ds.Status.NumberMisscheduled,
		UpdatedNumberScheduled:  ds.Status.UpdatedNumberScheduled,
		ObservedGeneration:      ds.Status.ObservedGeneration,
		ConditionAvailable:      utils.GetConditionStatusGeneric(ds.Status.Conditions, "Available"),
		ConditionProgressing:    utils.GetConditionStatusGeneric(ds.Status.Conditions, "Progressing"),
		ConditionReplicaFailure: utils.GetConditionStatusGeneric(ds.Status.Conditions, "ReplicaFailure"),
		UpdateStrategy:          updateStrategy,
	}

	return data
}
