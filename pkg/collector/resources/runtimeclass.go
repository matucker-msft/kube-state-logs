package resources

import (
	"context"
	"time"

	nodev1 "k8s.io/api/node/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// RuntimeClassHandler handles collection of runtimeclass metrics
type RuntimeClassHandler struct {
	utils.BaseHandler
}

// NewRuntimeClassHandler creates a new RuntimeClassHandler
func NewRuntimeClassHandler(client kubernetes.Interface) *RuntimeClassHandler {
	return &RuntimeClassHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the runtimeclass informer
func (h *RuntimeClassHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create runtimeclass informer
	informer := factory.Node().V1().RuntimeClasses().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers runtimeclass metrics from the cluster (uses cache)
func (h *RuntimeClassHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all runtimeclasses from the cache
	rcList := utils.SafeGetStoreList(h.GetInformer())

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
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rc)

	// Create data structure
	// See: https://kubernetes.io/docs/concepts/containers/runtime-class/
	data := types.RuntimeClassData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(rc),
		Labels:           utils.ExtractLabels(rc),
		Annotations:      utils.ExtractAnnotations(rc),
		Handler:          rc.Handler,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("runtimeclass", utils.ExtractName(rc), utils.ExtractNamespace(rc), data)
}
