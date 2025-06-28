package resources

import (
	"context"
	"time"

	nodev1 "k8s.io/api/node/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// RuntimeClassHandler handles collection of runtimeclass metrics
type RuntimeClassHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewRuntimeClassHandler creates a new RuntimeClassHandler
func NewRuntimeClassHandler(client *kubernetes.Clientset) *RuntimeClassHandler {
	return &RuntimeClassHandler{
		client: client,
	}
}

// SetupInformer sets up the runtimeclass informer
func (h *RuntimeClassHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create runtimeclass informer
	h.informer = factory.Node().V1().RuntimeClasses().Informer()

	return nil
}

// Collect gathers runtimeclass metrics from the cluster (uses cache)
func (h *RuntimeClassHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all runtimeclasses from the cache
	rcList := safeGetStoreList(h.informer)

	for _, obj := range rcList {
		rc, ok := obj.(*nodev1.RuntimeClass)
		if !ok {
			continue
		}

		entry := h.createLogEntry(rc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a RuntimeClass
func (h *RuntimeClassHandler) createLogEntry(rc *nodev1.RuntimeClass) types.LogEntry {
	// Extract basic metadata
	createdTimestamp := int64(0)
	if creationTime := rc.GetCreationTimestamp(); !creationTime.IsZero() {
		createdTimestamp = creationTime.Unix()
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rc)

	// Create data structure
	// See: https://kubernetes.io/docs/concepts/containers/runtime-class/
	data := types.RuntimeClassData{
		CreatedTimestamp: createdTimestamp,
		Labels:           rc.GetLabels(),
		Annotations:      rc.GetAnnotations(),
		Handler:          rc.Handler,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "runtimeclass",
		Name:         rc.GetName(),
		Namespace:    "", // RuntimeClass is cluster-scoped
		Data:         convertStructToMap(data),
	}
}
