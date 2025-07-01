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

// StatefulSetHandler handles collection of statefulset metrics
type StatefulSetHandler struct {
	utils.BaseHandler
}

// NewStatefulSetHandler creates a new StatefulSetHandler
func NewStatefulSetHandler(client kubernetes.Interface) *StatefulSetHandler {
	return &StatefulSetHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the statefulset informer
func (h *StatefulSetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create statefulset informer
	informer := factory.Apps().V1().StatefulSets().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers statefulset metrics from the cluster (uses cache)
func (h *StatefulSetHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all statefulsets from the cache
	statefulsets := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range statefulsets {
		statefulset, ok := obj.(*appsv1.StatefulSet)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, statefulset.Namespace) {
			continue
		}

		entry := h.createLogEntry(statefulset)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a StatefulSetData from a statefulset
func (h *StatefulSetHandler) createLogEntry(sts *appsv1.StatefulSet) types.StatefulSetData {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(sts)

	serviceName := ""
	if sts.Spec.ServiceName != "" {
		serviceName = sts.Spec.ServiceName
	}

	podManagementPolicy := string(sts.Spec.PodManagementPolicy)
	updateStrategy := string(sts.Spec.UpdateStrategy.Type)

	desiredReplicas := int32(1)
	if sts.Spec.Replicas != nil {
		desiredReplicas = *sts.Spec.Replicas
	}

	data := types.StatefulSetData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "statefulset",
			Name:             utils.ExtractName(sts),
			Namespace:        utils.ExtractNamespace(sts),
			CreatedTimestamp: utils.ExtractCreationTimestamp(sts),
			Labels:           utils.ExtractLabels(sts),
			Annotations:      utils.ExtractAnnotations(sts),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		DesiredReplicas:         desiredReplicas,
		CurrentReplicas:         sts.Status.Replicas,
		ReadyReplicas:           sts.Status.ReadyReplicas,
		UpdatedReplicas:         sts.Status.UpdatedReplicas,
		ObservedGeneration:      sts.Status.ObservedGeneration,
		CurrentRevision:         sts.Status.CurrentRevision,
		UpdateRevision:          sts.Status.UpdateRevision,
		ConditionAvailable:      utils.GetConditionStatusGeneric(sts.Status.Conditions, "Available"),
		ConditionProgressing:    utils.GetConditionStatusGeneric(sts.Status.Conditions, "Progressing"),
		ConditionReplicaFailure: utils.GetConditionStatusGeneric(sts.Status.Conditions, "ReplicaFailure"),
		ServiceName:             serviceName,
		PodManagementPolicy:     podManagementPolicy,
		UpdateStrategy:          updateStrategy,
	}

	return data
}
