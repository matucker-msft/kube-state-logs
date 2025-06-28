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

// Collect gathers ingressclass metrics from the cluster (uses cache)
func (h *IngressClassHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all ingressclasses from the cache
	icList := safeGetStoreList(h.informer)

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
	// Check if this is the default ingress class
	// See: https://kubernetes.io/docs/concepts/services-networking/ingress/#default-ingress-class
	isDefault := false
	if ic.Annotations != nil {
		if _, exists := ic.Annotations["ingressclass.kubernetes.io/is-default-class"]; exists {
			isDefault = true
		}
	}

	// Create data structure
	data := types.IngressClassData{
		CreatedTimestamp: ic.CreationTimestamp.Unix(),
		Labels:           ic.Labels,
		Annotations:      ic.Annotations,
		Controller:       ic.Spec.Controller,
		IsDefault:        isDefault,
		CreatedByKind:    "",
		CreatedByName:    "",
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "ingressclass",
		Name:         ic.Name,
		Namespace:    "", // Cluster-scoped resource
		Data:         convertStructToMap(data),
	}
}
