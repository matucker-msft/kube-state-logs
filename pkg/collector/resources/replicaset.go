package resources

import (
	"context"
	"slices"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ReplicaSetHandler handles collection of replicaset metrics
type ReplicaSetHandler struct {
	utils.BaseHandler
}

// NewReplicaSetHandler creates a new ReplicaSetHandler
func NewReplicaSetHandler(client *kubernetes.Clientset) *ReplicaSetHandler {
	return &ReplicaSetHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the replicaset informer
func (h *ReplicaSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create replicaset informer
	informer := factory.Apps().V1().ReplicaSets().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers replicaset metrics from the cluster (uses cache)
func (h *ReplicaSetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all replicasets from the cache
	replicasets := utils.SafeGetStoreList(h.GetInformer())

	// Group replicasets by owner to identify current ones
	ownerReplicaSets := make(map[string][]*appsv1.ReplicaSet)

	for _, obj := range replicasets {
		rs, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			continue
		}

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
			if rsList[0].Labels == nil {
				rsList[0].Labels = make(map[string]string)
			}
			rsList[0].Labels["kube-state-logs/current"] = "true"
		}

		// Only log current replicasets
		for _, rs := range rsList {
			if rs.Labels != nil && rs.Labels["kube-state-logs/current"] == "true" {
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
	desiredReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		desiredReplicas = *rs.Spec.Replicas
	}

	data := types.ReplicaSetData{
		CreatedTimestamp:        utils.ExtractCreationTimestamp(rs),
		Labels:                  utils.ExtractLabels(rs),
		Annotations:             utils.ExtractAnnotations(rs),
		DesiredReplicas:         desiredReplicas,
		CurrentReplicas:         rs.Status.Replicas,
		ReadyReplicas:           rs.Status.ReadyReplicas,
		AvailableReplicas:       rs.Status.AvailableReplicas,
		FullyLabeledReplicas:    rs.Status.FullyLabeledReplicas,
		ObservedGeneration:      rs.Status.ObservedGeneration,
		ConditionAvailable:      utils.GetConditionStatusGeneric(rs.Status.Conditions, "ReplicaSetAvailable"),
		ConditionProgressing:    utils.GetConditionStatusGeneric(rs.Status.Conditions, "ReplicaSetProgressing"),
		ConditionReplicaFailure: utils.GetConditionStatusGeneric(rs.Status.Conditions, "ReplicaSetReplicaFailure"),
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		IsCurrent: func() bool {
			if rs.Labels != nil {
				return rs.Labels["kube-state-logs/current"] == "true"
			}
			return false
		}(),
	}

	return utils.CreateLogEntry("replicaset", utils.ExtractName(rs), utils.ExtractNamespace(rs), data)
}
