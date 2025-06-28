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

// ResourceQuotaHandler handles collection of resourcequota metrics
type ResourceQuotaHandler struct {
	utils.BaseHandler
}

// NewResourceQuotaHandler creates a new ResourceQuotaHandler
func NewResourceQuotaHandler(client *kubernetes.Clientset) *ResourceQuotaHandler {
	return &ResourceQuotaHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the resourcequota informer
func (h *ResourceQuotaHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create resourcequota informer
	informer := factory.Core().V1().ResourceQuotas().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers resourcequota metrics from the cluster (uses cache)
func (h *ResourceQuotaHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all resourcequotas from the cache
	rqList := utils.SafeGetStoreList(h.GetInformer())

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
	hard := resourceListToInt64Map(rq.Spec.Hard)
	used := resourceListToInt64Map(rq.Status.Used)

	// Format scopes
	// See: https://kubernetes.io/docs/concepts/policy/resource-quotas/#quota-scopes
	scopes := make([]string, len(rq.Spec.Scopes))
	for i, scope := range rq.Spec.Scopes {
		scopes[i] = string(scope)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rq)

	// Create data structure
	data := types.ResourceQuotaData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(rq),
		Labels:           utils.ExtractLabels(rq),
		Annotations:      utils.ExtractAnnotations(rq),
		Hard:             hard,
		Used:             used,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
		Scopes:           scopes,
	}

	return utils.CreateLogEntry("resourcequota", utils.ExtractName(rq), utils.ExtractNamespace(rq), data)
}

// resourceListToInt64Map converts corev1.ResourceList to map[string]int64
func resourceListToInt64Map(rl corev1.ResourceList) map[string]int64 {
	result := make(map[string]int64)
	for resourceName, quantity := range rl {
		result[string(resourceName)] = quantity.Value()
	}
	return result
}
