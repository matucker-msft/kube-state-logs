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

// ResourceQuotaHandler handles collection of resourcequota metrics
type ResourceQuotaHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewResourceQuotaHandler creates a new ResourceQuotaHandler
func NewResourceQuotaHandler(client *kubernetes.Clientset) *ResourceQuotaHandler {
	return &ResourceQuotaHandler{
		client: client,
	}
}

// SetupInformer sets up the resourcequota informer
func (h *ResourceQuotaHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create resourcequota informer
	h.informer = factory.Core().V1().ResourceQuotas().Informer()

	return nil
}

// Collect gathers resourcequota metrics from the cluster (uses cache)
func (h *ResourceQuotaHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all resourcequotas from the cache
	rqList := safeGetStoreList(h.informer)

	for _, obj := range rqList {
		rq, ok := obj.(*corev1.ResourceQuota)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, rq.Namespace) {
			continue
		}

		entry := h.createLogEntry(rq)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a resourcequota
func (h *ResourceQuotaHandler) createLogEntry(rq *corev1.ResourceQuota) types.LogEntry {
	// Convert hard limits to int64 values
	hard := make(map[string]int64)
	for resourceName, quantity := range rq.Spec.Hard {
		hard[resourceName.String()] = quantity.Value()
	}

	// Convert used limits to int64 values
	used := make(map[string]int64)
	for resourceName, quantity := range rq.Status.Used {
		used[resourceName.String()] = quantity.Value()
	}

	// Format scopes
	// See: https://kubernetes.io/docs/concepts/policy/resource-quotas/#quota-scopes
	scopes := make([]string, len(rq.Spec.Scopes))
	for i, scope := range rq.Spec.Scopes {
		scopes[i] = string(scope)
	}

	// Create data structure
	data := types.ResourceQuotaData{
		CreatedTimestamp: rq.CreationTimestamp.Unix(),
		Labels:           rq.Labels,
		Annotations:      rq.Annotations,
		Hard:             hard,
		Used:             used,
		CreatedByKind:    "",
		CreatedByName:    "",
		Scopes:           scopes,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "resourcequota",
		Name:         rq.Name,
		Namespace:    rq.Namespace,
		Data:         convertStructToMap(data),
	}
}
