package resources

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// DaemonSetHandler handles collection of daemonset metrics
type DaemonSetHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewDaemonSetHandler creates a new DaemonSetHandler
func NewDaemonSetHandler(client *kubernetes.Clientset) *DaemonSetHandler {
	return &DaemonSetHandler{
		client: client,
	}
}

// SetupInformer sets up the daemonset informer
func (h *DaemonSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create daemonset informer
	h.informer = factory.Apps().V1().DaemonSets().Informer()

	return nil
}

// Collect gathers daemonset metrics from the cluster (uses cache)
func (h *DaemonSetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all daemonsets from the cache
	daemonsets := utils.SafeGetStoreList(h.informer)

	for _, obj := range daemonsets {
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

// createLogEntry creates a LogEntry from a daemonset
func (h *DaemonSetHandler) createLogEntry(ds *appsv1.DaemonSet) types.LogEntry {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ds)

	// Get update strategy
	updateStrategy := string(ds.Spec.UpdateStrategy.Type)

	data := types.DaemonSetData{
		CreatedTimestamp:        utils.ExtractCreationTimestamp(ds),
		Labels:                  utils.ExtractLabels(ds),
		Annotations:             utils.ExtractAnnotations(ds),
		DesiredNumberScheduled:  ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled:  ds.Status.CurrentNumberScheduled,
		NumberReady:             ds.Status.NumberReady,
		NumberAvailable:         ds.Status.NumberAvailable,
		NumberUnavailable:       ds.Status.NumberUnavailable,
		NumberMisscheduled:      ds.Status.NumberMisscheduled,
		UpdatedNumberScheduled:  ds.Status.UpdatedNumberScheduled,
		ObservedGeneration:      ds.Status.ObservedGeneration,
		ConditionAvailable:      utils.GetConditionStatusGeneric(ds.Status.Conditions, "DaemonSetAvailable"),
		ConditionProgressing:    utils.GetConditionStatusGeneric(ds.Status.Conditions, "DaemonSetProgressing"),
		ConditionReplicaFailure: utils.GetConditionStatusGeneric(ds.Status.Conditions, "DaemonSetReplicaFailure"),
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		UpdateStrategy:          updateStrategy,
	}

	return utils.CreateLogEntry("daemonset", utils.ExtractName(ds), utils.ExtractNamespace(ds), data)
}
