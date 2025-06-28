package resources

import (
	"context"
	"slices"
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

	// Add event handlers (no logging on events)
	h.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			// No logging on add events
		},
		UpdateFunc: func(oldObj, newObj any) {
			// No logging on update events
		},
		DeleteFunc: func(obj any) {
			// No logging on delete events
		},
	})

	return nil
}

// Collect gathers daemonset metrics from the cluster (uses cache)
func (h *DaemonSetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all daemonsets from the cache
	daemonsets := safeGetStoreList(h.informer)

	for _, obj := range daemonsets {
		ds, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, ds.Namespace) {
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
		CreatedTimestamp:        ds.CreationTimestamp.Unix(),
		Labels:                  ds.Labels,
		Annotations:             ds.Annotations,
		DesiredNumberScheduled:  ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled:  ds.Status.CurrentNumberScheduled,
		NumberReady:             ds.Status.NumberReady,
		NumberAvailable:         ds.Status.NumberAvailable,
		NumberUnavailable:       ds.Status.NumberUnavailable,
		NumberMisscheduled:      ds.Status.NumberMisscheduled,
		UpdatedNumberScheduled:  ds.Status.UpdatedNumberScheduled,
		ObservedGeneration:      ds.Status.ObservedGeneration,
		ConditionAvailable:      h.getConditionStatus(ds.Status.Conditions, "DaemonSetAvailable"),
		ConditionProgressing:    h.getConditionStatus(ds.Status.Conditions, "DaemonSetProgressing"),
		ConditionReplicaFailure: h.getConditionStatus(ds.Status.Conditions, "DaemonSetReplicaFailure"),
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		UpdateStrategy:          updateStrategy,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "daemonset",
		Name:         ds.Name,
		Namespace:    ds.Namespace,
		Data:         h.convertToMap(data),
	}
}

// getConditionStatus checks if a condition is true
func (h *DaemonSetHandler) getConditionStatus(conditions []appsv1.DaemonSetCondition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == appsv1.DaemonSetConditionType(conditionType) {
			return condition.Status == "True"
		}
	}
	return false
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *DaemonSetHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
