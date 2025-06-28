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

// LimitRangeHandler handles collection of limitrange metrics
type LimitRangeHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewLimitRangeHandler creates a new LimitRangeHandler
func NewLimitRangeHandler(client *kubernetes.Clientset) *LimitRangeHandler {
	return &LimitRangeHandler{
		client: client,
	}
}

// SetupInformer sets up the limitrange informer
func (h *LimitRangeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create limitrange informer
	h.informer = factory.Core().V1().LimitRanges().Informer()

	return nil
}

// Collect gathers limitrange metrics from the cluster (uses cache)
func (h *LimitRangeHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all limitranges from the cache
	lrList := safeGetStoreList(h.informer)

	for _, obj := range lrList {
		lr, ok := obj.(*corev1.LimitRange)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if !utils.ShouldIncludeNamespace(namespaces, lr.Namespace) {
			continue
		}

		entry := h.createLogEntry(lr)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a limitrange
func (h *LimitRangeHandler) createLogEntry(lr *corev1.LimitRange) types.LogEntry {
	// Convert limits
	// See: https://kubernetes.io/docs/concepts/policy/limit-range/#limit-range
	var limits []types.LimitRangeItem
	for _, limit := range lr.Spec.Limits {
		limitItem := types.LimitRangeItem{
			Type:                 string(limit.Type),
			ResourceType:         "",
			ResourceName:         "",
			Min:                  h.convertResourceList(limit.Min),
			Max:                  h.convertResourceList(limit.Max),
			Default:              h.convertResourceList(limit.Default),
			DefaultRequest:       h.convertResourceList(limit.DefaultRequest),
			MaxLimitRequestRatio: h.convertResourceList(limit.MaxLimitRequestRatio),
		}
		limits = append(limits, limitItem)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(lr)

	// Create data structure
	data := types.LimitRangeData{
		CreatedTimestamp: lr.CreationTimestamp.Unix(),
		Labels:           lr.Labels,
		Annotations:      lr.Annotations,
		Limits:           limits,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "limitrange",
		Name:         lr.Name,
		Namespace:    lr.Namespace,
		Data:         convertStructToMap(data),
	}
}

// convertResourceList converts corev1.ResourceList to map[string]string
func (h *LimitRangeHandler) convertResourceList(resourceList corev1.ResourceList) map[string]string {
	result := make(map[string]string)
	for resource, quantity := range resourceList {
		result[string(resource)] = quantity.String()
	}
	return result
}
