package resources

import (
	"context"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// IngressClassHandler handles collection of ingressclass metrics
type IngressClassHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewIngressClassHandler creates a new IngressClassHandler
func NewIngressClassHandler(client *kubernetes.Clientset) *IngressClassHandler {
	return &IngressClassHandler{
		client: client,
	}
}

// SetupInformer sets up the ingressclass informer
func (h *IngressClassHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create ingressclass informer
	h.informer = factory.Networking().V1().IngressClasses().Informer()

	return nil
}

// Collect gathers ingressclass metrics from the cluster (uses cache)
func (h *IngressClassHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all ingressclasses from the cache
	icList := utils.SafeGetStoreList(h.informer)

	for _, obj := range icList {
		ic, ok := obj.(*networkingv1.IngressClass)
		if !ok {
			continue
		}

		entry := h.createLogEntry(ic)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from an ingressclass
func (h *IngressClassHandler) createLogEntry(ic *networkingv1.IngressClass) types.LogEntry {
	isDefault := false
	if ic.Annotations != nil {
		if _, exists := ic.Annotations["ingressclass.kubernetes.io/is-default-class"]; exists {
			isDefault = true
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ic)

	data := types.IngressClassData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(ic),
		Labels:           utils.ExtractLabels(ic),
		Annotations:      utils.ExtractAnnotations(ic),
		Controller:       ic.Spec.Controller,
		IsDefault:        isDefault,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("ingressclass", utils.ExtractName(ic), utils.ExtractNamespace(ic), data)
}
