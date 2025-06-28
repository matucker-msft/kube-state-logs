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

// ConfigMapHandler handles collection of configmap metrics
type ConfigMapHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewConfigMapHandler creates a new ConfigMapHandler
func NewConfigMapHandler(client *kubernetes.Clientset) *ConfigMapHandler {
	return &ConfigMapHandler{
		client: client,
	}
}

// SetupInformer sets up the configmap informer
func (h *ConfigMapHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create configmap informer
	h.informer = factory.Core().V1().ConfigMaps().Informer()

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

// Collect gathers configmap metrics from the cluster (uses cache)
func (h *ConfigMapHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all configmaps from the cache
	configmaps := safeGetStoreList(h.informer)

	for _, obj := range configmaps {
		configmap, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, configmap.Namespace) {
			continue
		}

		entry := h.createLogEntry(configmap)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a configmap
func (h *ConfigMapHandler) createLogEntry(configmap *corev1.ConfigMap) types.LogEntry {
	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(configmap.OwnerReferences) > 0 {
		createdByKind = configmap.OwnerReferences[0].Kind
		createdByName = configmap.OwnerReferences[0].Name
	}

	// Get data keys
	var dataKeys []string
	for key := range configmap.Data {
		dataKeys = append(dataKeys, key)
	}

	data := types.ConfigMapData{
		CreatedTimestamp: configmap.CreationTimestamp.Unix(),
		Labels:           configmap.Labels,
		Annotations:      configmap.Annotations,
		DataKeys:         dataKeys,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "configmap",
		Name:         configmap.Name,
		Namespace:    configmap.Namespace,
		Data:         h.convertToMap(data),
	}
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *ConfigMapHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
