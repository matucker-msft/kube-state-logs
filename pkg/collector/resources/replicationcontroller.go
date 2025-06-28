package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ReplicationControllerHandler handles collection of replicationcontroller metrics
type ReplicationControllerHandler struct {
	utils.BaseHandler
}

// NewReplicationControllerHandler creates a new ReplicationControllerHandler
func NewReplicationControllerHandler(client kubernetes.Interface) *ReplicationControllerHandler {
	return &ReplicationControllerHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the replicationcontroller informer
func (h *ReplicationControllerHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create replicationcontroller informer
	informer := factory.Core().V1().ReplicationControllers().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers replicationcontroller metrics from the cluster (uses cache)
func (h *ReplicationControllerHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all replicationcontrollers from the cache
	rcList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range rcList {
		rc, ok := obj.(*corev1.ReplicationController)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, rc.Namespace) {
			continue
		}

		entry := h.createLogEntry(rc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a replicationcontroller
func (h *ReplicationControllerHandler) createLogEntry(rc *corev1.ReplicationController) types.LogEntry {
	// Get desired replicas
	// Default to 1 when spec.replicas is nil (Kubernetes API default)
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller/#replicationcontroller
	desiredReplicas := int32(1) // Default value
	if rc.Spec.Replicas != nil {
		desiredReplicas = *rc.Spec.Replicas
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rc)

	// Create data structure
	data := types.ReplicationControllerData{
		CreatedTimestamp:     utils.ExtractCreationTimestamp(rc),
		Labels:               utils.ExtractLabels(rc),
		Annotations:          utils.ExtractAnnotations(rc),
		DesiredReplicas:      desiredReplicas,
		CurrentReplicas:      rc.Status.Replicas,
		ReadyReplicas:        rc.Status.ReadyReplicas,
		AvailableReplicas:    rc.Status.AvailableReplicas,
		FullyLabeledReplicas: rc.Status.FullyLabeledReplicas,
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
		ObservedGeneration:   rc.Status.ObservedGeneration,
	}

	return utils.CreateLogEntry("replicationcontroller", utils.ExtractName(rc), utils.ExtractNamespace(rc), data)
}
