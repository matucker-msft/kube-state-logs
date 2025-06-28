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

// LimitRangeHandler handles collection of limitrange metrics
type LimitRangeHandler struct {
	utils.BaseHandler
}

// NewLimitRangeHandler creates a new LimitRangeHandler
func NewLimitRangeHandler(client *kubernetes.Clientset) *LimitRangeHandler {
	return &LimitRangeHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the limitrange informer
func (h *LimitRangeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create limitrange informer
	informer := factory.Core().V1().LimitRanges().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers limitrange metrics from the cluster (uses cache)
func (h *LimitRangeHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all limitranges from the cache
	lrList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range lrList {
		lr, ok := obj.(*corev1.LimitRange)
		if !ok {
			continue
		}

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
	var limits []types.LimitRangeItem
	for _, limit := range lr.Spec.Limits {
		limitItem := types.LimitRangeItem{
			Type:                 string(limit.Type),
			ResourceType:         "",
			ResourceName:         "",
			Min:                  make(map[string]string),
			Max:                  make(map[string]string),
			Default:              make(map[string]string),
			DefaultRequest:       make(map[string]string),
			MaxLimitRequestRatio: make(map[string]string),
		}

		// Extract resource type and name
		for resourceName := range limit.Min {
			limitItem.ResourceType = string(resourceName)
			limitItem.ResourceName = string(resourceName)
			break
		}

		// Convert resource maps
		for key, value := range limit.Min {
			limitItem.Min[string(key)] = value.String()
		}
		for key, value := range limit.Max {
			limitItem.Max[string(key)] = value.String()
		}
		for key, value := range limit.Default {
			limitItem.Default[string(key)] = value.String()
		}
		for key, value := range limit.DefaultRequest {
			limitItem.DefaultRequest[string(key)] = value.String()
		}
		for key, value := range limit.MaxLimitRequestRatio {
			limitItem.MaxLimitRequestRatio[string(key)] = value.String()
		}

		limits = append(limits, limitItem)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(lr)

	data := types.LimitRangeData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(lr),
		Labels:           utils.ExtractLabels(lr),
		Annotations:      utils.ExtractAnnotations(lr),
		Limits:           limits,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("limitrange", utils.ExtractName(lr), utils.ExtractNamespace(lr), data)
}
