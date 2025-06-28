package resources

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// ReplicationControllerHandler handles collection of replicationcontroller metrics
type ReplicationControllerHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewReplicationControllerHandler creates a new ReplicationControllerHandler
func NewReplicationControllerHandler(client *kubernetes.Clientset) *ReplicationControllerHandler {
	return &ReplicationControllerHandler{
		client: client,
	}
}

// SetupInformer sets up the replicationcontroller informer
func (h *ReplicationControllerHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create replicationcontroller informer
	h.informer = factory.Core().V1().ReplicationControllers().Informer()

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

// Collect gathers replicationcontroller metrics from the cluster (uses cache)
func (h *ReplicationControllerHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all replicationcontrollers from the cache
	rcList := safeGetStoreList(h.informer)

	for _, obj := range rcList {
		rc, ok := obj.(*corev1.ReplicationController)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, rc.Namespace) {
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
	desiredReplicas := int32(1) // Default value
	if rc.Spec.Replicas != nil {
		desiredReplicas = *rc.Spec.Replicas
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(rc.OwnerReferences) > 0 {
		createdByKind = rc.OwnerReferences[0].Kind
		createdByName = rc.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.ReplicationControllerData{
		CreatedTimestamp:     rc.CreationTimestamp.Unix(),
		Labels:               rc.Labels,
		Annotations:          rc.Annotations,
		DesiredReplicas:      desiredReplicas,
		CurrentReplicas:      rc.Status.Replicas,
		ReadyReplicas:        rc.Status.ReadyReplicas,
		AvailableReplicas:    rc.Status.AvailableReplicas,
		FullyLabeledReplicas: rc.Status.FullyLabeledReplicas,
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
		ObservedGeneration:   rc.Status.ObservedGeneration,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "replicationcontroller",
		Name:         rc.Name,
		Namespace:    rc.Namespace,
		Data:         convertStructToMap(data),
	}
}
