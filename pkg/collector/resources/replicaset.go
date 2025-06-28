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

// ReplicaSetHandler handles collection of replicaset metrics
type ReplicaSetHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewReplicaSetHandler creates a new ReplicaSetHandler
func NewReplicaSetHandler(client *kubernetes.Clientset) *ReplicaSetHandler {
	return &ReplicaSetHandler{
		client: client,
	}
}

// SetupInformer sets up the replicaset informer
func (h *ReplicaSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create replicaset informer
	h.informer = factory.Apps().V1().ReplicaSets().Informer()

	return nil
}

// Collect gathers replicaset metrics from the cluster (uses cache)
func (h *ReplicaSetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all replicasets from the cache
	replicasets := safeGetStoreList(h.informer)

	// Group replicasets by owner to identify current ones
	ownerReplicaSets := make(map[string][]*appsv1.ReplicaSet)

	for _, obj := range replicasets {
		rs, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if !utils.ShouldIncludeNamespace(namespaces, rs.Namespace) {
			continue
		}

		// Group by owner
		for _, ownerRef := range rs.OwnerReferences {
			key := rs.Namespace + "/" + ownerRef.Kind + "/" + ownerRef.Name
			ownerReplicaSets[key] = append(ownerReplicaSets[key], rs)
		}
	}

	// Process each group and identify current replicasets
	for _, rsList := range ownerReplicaSets {
		if len(rsList) == 0 {
			continue
		}

		// Sort by creation timestamp (newest first) and generation
		slices.SortFunc(rsList, func(a, b *appsv1.ReplicaSet) int {
			if a.CreationTimestamp.Equal(&b.CreationTimestamp) {
				return int(b.Generation - a.Generation)
			}
			return b.CreationTimestamp.Compare(a.CreationTimestamp.Time)
		})

		// Mark the first (newest) replicaset as current
		if len(rsList) > 0 {
			rsList[0].Labels["kube-state-logs/current"] = "true"
		}

		// Only log current replicasets
		for _, rs := range rsList {
			if rs.Labels["kube-state-logs/current"] == "true" {
				entry := h.createLogEntry(rs)
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a replicaset
func (h *ReplicaSetHandler) createLogEntry(rs *appsv1.ReplicaSet) types.LogEntry {

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rs)

	// Get desired replicas with nil check
	// Default to 1 when spec.replicas is nil (Kubernetes API default)
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#replicaset-basics
	desiredReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		desiredReplicas = *rs.Spec.Replicas
	}

	data := types.ReplicaSetData{
		CreatedTimestamp:        rs.CreationTimestamp.Unix(),
		Labels:                  rs.Labels,
		Annotations:             rs.Annotations,
		DesiredReplicas:         desiredReplicas,
		CurrentReplicas:         rs.Status.Replicas,
		ReadyReplicas:           rs.Status.ReadyReplicas,
		AvailableReplicas:       rs.Status.AvailableReplicas,
		FullyLabeledReplicas:    rs.Status.FullyLabeledReplicas,
		ObservedGeneration:      rs.Status.ObservedGeneration,
		ConditionAvailable:      h.getConditionStatus(rs.Status.Conditions, "ReplicaSetAvailable"),
		ConditionProgressing:    h.getConditionStatus(rs.Status.Conditions, "ReplicaSetProgressing"),
		ConditionReplicaFailure: h.getConditionStatus(rs.Status.Conditions, "ReplicaSetReplicaFailure"),
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		IsCurrent:               rs.Labels["kube-state-logs/current"] == "true",
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "replicaset",
		Name:         rs.Name,
		Namespace:    rs.Namespace,
		Data:         h.convertToMap(data),
	}
}

// getConditionStatus checks if a condition is true
func (h *ReplicaSetHandler) getConditionStatus(conditions []appsv1.ReplicaSetCondition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == appsv1.ReplicaSetConditionType(conditionType) {
			return condition.Status == "True"
		}
	}
	return false
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *ReplicaSetHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
