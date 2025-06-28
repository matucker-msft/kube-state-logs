package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
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

	return nil
}

// Collect gathers configmap metrics from the cluster (uses cache)
func (h *ConfigMapHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all configmaps from the cache
	configmaps := utils.SafeGetStoreList(h.informer)

	for _, obj := range configmaps {
		configmap, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, configmap.Namespace) {
			continue
		}

		entry := h.createLogEntry(configmap)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a configmap
func (h *ConfigMapHandler) createLogEntry(configmap *corev1.ConfigMap) types.LogEntry {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(configmap)

	var dataKeys []string
	for key := range configmap.Data {
		dataKeys = append(dataKeys, key)
	}

	data := types.ConfigMapData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(configmap),
		Labels:           utils.ExtractLabels(configmap),
		Annotations:      utils.ExtractAnnotations(configmap),
		DataKeys:         dataKeys,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("configmap", utils.ExtractName(configmap), utils.ExtractNamespace(configmap), data)
}
