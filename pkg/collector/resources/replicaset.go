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

// ReplicaSetHandler handles collection of replicaset metrics
type ReplicaSetHandler struct {
	utils.BaseHandler
}

// NewReplicaSetHandler creates a new ReplicaSetHandler
func NewReplicaSetHandler(client kubernetes.Interface) *ReplicaSetHandler {
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
func (h *ReplicaSetHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all replicasets from the cache
	replicasets := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range replicasets {
		replicaset, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, replicaset.Namespace) {
			continue
		}

		entry := h.createLogEntry(replicaset)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a ReplicaSetData from a replicaset
func (h *ReplicaSetHandler) createLogEntry(rs *appsv1.ReplicaSet) types.ReplicaSetData {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rs)

	// Get desired replicas with nil check
	desiredReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		desiredReplicas = *rs.Spec.Replicas
	}

	data := types.ReplicaSetData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "replicaset",
			Name:             utils.ExtractName(rs),
			Namespace:        utils.ExtractNamespace(rs),
			CreatedTimestamp: utils.ExtractCreationTimestamp(rs),
			Labels:           utils.ExtractLabels(rs),
			Annotations:      utils.ExtractAnnotations(rs),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		DesiredReplicas:         desiredReplicas,
		CurrentReplicas:         rs.Status.Replicas,
		ReadyReplicas:           rs.Status.ReadyReplicas,
		AvailableReplicas:       rs.Status.AvailableReplicas,
		FullyLabeledReplicas:    rs.Status.FullyLabeledReplicas,
		ObservedGeneration:      rs.Status.ObservedGeneration,
		ConditionAvailable:      utils.GetConditionStatusGeneric(rs.Status.Conditions, "Available"),
		ConditionProgressing:    utils.GetConditionStatusGeneric(rs.Status.Conditions, "Progressing"),
		ConditionReplicaFailure: utils.GetConditionStatusGeneric(rs.Status.Conditions, "ReplicaFailure"),
		IsCurrent: func() bool {
			if rs.Labels != nil {
				return rs.Labels["kube-state-logs/current"] == "true"
			}
			return false
		}(),
	}

	return data
}
