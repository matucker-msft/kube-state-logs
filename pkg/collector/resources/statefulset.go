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
)

// StatefulSetHandler handles collection of statefulset metrics
type StatefulSetHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewStatefulSetHandler creates a new StatefulSetHandler
func NewStatefulSetHandler(client *kubernetes.Clientset) *StatefulSetHandler {
	return &StatefulSetHandler{
		client: client,
	}
}

// SetupInformer sets up the statefulset informer
func (h *StatefulSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create statefulset informer
	h.informer = factory.Apps().V1().StatefulSets().Informer()

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

// Collect gathers statefulset metrics from the cluster (uses cache)
func (h *StatefulSetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all statefulsets from the cache
	statefulsets := h.informer.GetStore().List()

	for _, obj := range statefulsets {
		sts, ok := obj.(*appsv1.StatefulSet)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, sts.Namespace) {
			continue
		}

		entry := h.createLogEntry(sts)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a statefulset
func (h *StatefulSetHandler) createLogEntry(sts *appsv1.StatefulSet) types.LogEntry {
	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(sts.OwnerReferences) > 0 {
		createdByKind = sts.OwnerReferences[0].Kind
		createdByName = sts.OwnerReferences[0].Name
	}

	// Get service name
	serviceName := ""
	if sts.Spec.ServiceName != "" {
		serviceName = sts.Spec.ServiceName
	}

	// Get pod management policy
	podManagementPolicy := string(sts.Spec.PodManagementPolicy)

	// Get update strategy
	updateStrategy := string(sts.Spec.UpdateStrategy.Type)

	data := types.StatefulSetData{
		CreatedTimestamp:        sts.CreationTimestamp.Unix(),
		Labels:                  sts.Labels,
		Annotations:             sts.Annotations,
		DesiredReplicas:         *sts.Spec.Replicas,
		CurrentReplicas:         sts.Status.Replicas,
		ReadyReplicas:           sts.Status.ReadyReplicas,
		UpdatedReplicas:         sts.Status.UpdatedReplicas,
		ObservedGeneration:      sts.Status.ObservedGeneration,
		CurrentRevision:         sts.Status.CurrentRevision,
		UpdateRevision:          sts.Status.UpdateRevision,
		ConditionAvailable:      h.getConditionStatus(sts.Status.Conditions, "StatefulSetAvailable"),
		ConditionProgressing:    h.getConditionStatus(sts.Status.Conditions, "StatefulSetProgressing"),
		ConditionReplicaFailure: h.getConditionStatus(sts.Status.Conditions, "StatefulSetReplicaFailure"),
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		ServiceName:             serviceName,
		PodManagementPolicy:     podManagementPolicy,
		UpdateStrategy:          updateStrategy,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "statefulset",
		Name:         sts.Name,
		Namespace:    sts.Namespace,
		Data:         h.convertToMap(data),
	}
}

// getConditionStatus checks if a condition is true
func (h *StatefulSetHandler) getConditionStatus(conditions []appsv1.StatefulSetCondition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == appsv1.StatefulSetConditionType(conditionType) {
			return condition.Status == "True"
		}
	}
	return false
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *StatefulSetHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
