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

// ConfigMapHandler handles collection of configmap metrics
type ConfigMapHandler struct {
	utils.BaseHandler
}

// NewConfigMapHandler creates a new ConfigMapHandler
func NewConfigMapHandler(client kubernetes.Interface) *ConfigMapHandler {
	return &ConfigMapHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the configmap informer
func (h *ConfigMapHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create configmap informer
	informer := factory.Core().V1().ConfigMaps().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers configmap metrics from the cluster (uses cache)
func (h *ConfigMapHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all configmaps from the cache
	configmaps := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range configmaps {
		configmap, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, configmap.Namespace) {
			continue
		}

		entry := h.createLogEntry(configmap)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a ConfigMapData from a configmap
func (h *ConfigMapHandler) createLogEntry(configmap *corev1.ConfigMap) types.ConfigMapData {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(configmap)

	var dataKeys []string
	for key := range configmap.Data {
		dataKeys = append(dataKeys, key)
	}
	for key := range configmap.BinaryData {
		dataKeys = append(dataKeys, key)
	}

	data := types.ConfigMapData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "configmap",
			Name:             utils.ExtractName(configmap),
			Namespace:        utils.ExtractNamespace(configmap),
			CreatedTimestamp: utils.ExtractCreationTimestamp(configmap),
			Labels:           utils.ExtractLabels(configmap),
			Annotations:      utils.ExtractAnnotations(configmap),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		DataKeys: dataKeys,
	}

	return data
}
