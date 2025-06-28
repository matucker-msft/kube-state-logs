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
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// NamespaceHandler handles collection of namespace metrics
type NamespaceHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewNamespaceHandler creates a new NamespaceHandler
func NewNamespaceHandler(client *kubernetes.Clientset) *NamespaceHandler {
	return &NamespaceHandler{
		client: client,
	}
}

// SetupInformer sets up the namespace informer
func (h *NamespaceHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create namespace informer
	h.informer = factory.Core().V1().Namespaces().Informer()

	return nil
}

// Collect gathers namespace metrics from the cluster (uses cache)
func (h *NamespaceHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all namespaces from the cache using safe utility
	namespaceList := safeGetStoreList(h.informer)

	for _, obj := range namespaceList {
		ns, ok := obj.(*corev1.Namespace)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, ns.Name) {
			continue
		}

		entry := h.createLogEntry(ns)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a namespace
func (h *NamespaceHandler) createLogEntry(ns *corev1.Namespace) types.LogEntry {

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ns)

	data := types.NamespaceData{
		CreatedTimestamp:     ns.CreationTimestamp.Unix(),
		Labels:               ns.Labels,
		Annotations:          ns.Annotations,
		Phase:                string(ns.Status.Phase),
		ConditionActive:      h.getConditionStatus(ns.Status.Conditions, "NamespaceActive"),
		ConditionTerminating: h.getConditionStatus(ns.Status.Conditions, "NamespaceTerminating"),
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
		DeletionTimestamp:    ns.DeletionTimestamp,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "namespace",
		Name:         ns.Name,
		Namespace:    ns.Name, // Namespace name is the same as the resource name
		Data:         h.convertToMap(data),
	}
}

// getConditionStatus checks if a condition is true
func (h *NamespaceHandler) getConditionStatus(conditions []corev1.NamespaceCondition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.NamespaceConditionType(conditionType) {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *NamespaceHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
