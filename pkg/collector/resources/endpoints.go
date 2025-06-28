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

// EndpointsHandler handles collection of endpoints metrics
type EndpointsHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewEndpointsHandler creates a new EndpointsHandler
func NewEndpointsHandler(client *kubernetes.Clientset) *EndpointsHandler {
	return &EndpointsHandler{
		client: client,
	}
}

// SetupInformer sets up the endpoints informer
func (h *EndpointsHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create endpoints informer
	h.informer = factory.Core().V1().Endpoints().Informer()

	return nil
}

// Collect gathers endpoints metrics from the cluster (uses cache)
func (h *EndpointsHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all endpoints from the cache
	endpointsList := safeGetStoreList(h.informer)

	for _, obj := range endpointsList {
		endpoints, ok := obj.(*corev1.Endpoints)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, endpoints.Namespace) {
			continue
		}

		entry := h.createLogEntry(endpoints)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from an endpoints
func (h *EndpointsHandler) createLogEntry(endpoints *corev1.Endpoints) types.LogEntry {
	// Extract addresses and ports from all subsets
	var addresses []types.EndpointAddressData
	var ports []types.EndpointPortData

	for _, subset := range endpoints.Subsets {
		// Extract addresses
		for _, address := range subset.Addresses {
			targetRef := ""
			if address.TargetRef != nil {
				targetRef = address.TargetRef.Name
			}

			nodeName := ""
			if address.NodeName != nil {
				nodeName = *address.NodeName
			}

			addresses = append(addresses, types.EndpointAddressData{
				IP:        address.IP,
				Hostname:  address.Hostname,
				NodeName:  nodeName,
				TargetRef: targetRef,
			})
		}

		// Extract ports
		for _, port := range subset.Ports {
			ports = append(ports, types.EndpointPortData{
				Name:     port.Name,
				Protocol: string(port.Protocol),
				Port:     port.Port,
			})
		}
	}

	// Determine if endpoints are ready (have addresses)
	ready := len(addresses) > 0

	// Create data structure
	data := types.EndpointsData{
		CreatedTimestamp: endpoints.CreationTimestamp.Unix(),
		Labels:           endpoints.Labels,
		Annotations:      endpoints.Annotations,
		Addresses:        addresses,
		Ports:            ports,
		CreatedByKind:    "",
		CreatedByName:    "",
		Ready:            ready,
	}

	// Convert to map[string]any for the LogEntry
	dataMap := map[string]any{
		"createdTimestamp": data.CreatedTimestamp,
		"labels":           data.Labels,
		"annotations":      data.Annotations,
		"addresses":        data.Addresses,
		"ports":            data.Ports,
		"createdByKind":    data.CreatedByKind,
		"createdByName":    data.CreatedByName,
		"ready":            data.Ready,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "endpoints",
		Name:         endpoints.Name,
		Namespace:    endpoints.Namespace,
		Data:         dataMap,
	}
}
