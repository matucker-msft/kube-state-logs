package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// NamespaceHandler handles collection of namespace metrics
type NamespaceHandler struct {
	utils.BaseHandler
}

// NewNamespaceHandler creates a new NamespaceHandler
func NewNamespaceHandler(client *kubernetes.Clientset) *NamespaceHandler {
	return &NamespaceHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the namespace informer
func (h *NamespaceHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create namespace informer
	informer := factory.Core().V1().Namespaces().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers namespace metrics from the cluster (uses cache)
func (h *NamespaceHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all namespaces from the cache
	namespaceList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range namespaceList {
		ns, ok := obj.(*corev1.Namespace)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, ns.Name) {
			continue
		}

		entry := h.createLogEntry(ns)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a namespace
func (h *NamespaceHandler) createLogEntry(ns *corev1.Namespace) types.LogEntry {
	// Determine phase
	phase := string(ns.Status.Phase)

	// Determine conditions
	conditionActive := false
	conditionTerminating := false

	for _, condition := range ns.Status.Conditions {
		switch condition.Type {
		case corev1.NamespaceConditionType("Active"):
			conditionActive = condition.Status == corev1.ConditionTrue
		case corev1.NamespaceConditionType("Terminating"):
			conditionTerminating = condition.Status == corev1.ConditionTrue
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ns)

	var deletionTimestamp *v1.Time
	if t := utils.ExtractDeletionTimestamp(ns); t != nil {
		ts := v1.NewTime(*t)
		deletionTimestamp = &ts
	}

	data := types.NamespaceData{
		CreatedTimestamp:     utils.ExtractCreationTimestamp(ns),
		Labels:               utils.ExtractLabels(ns),
		Annotations:          utils.ExtractAnnotations(ns),
		Phase:                phase,
		ConditionActive:      conditionActive,
		ConditionTerminating: conditionTerminating,
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
		DeletionTimestamp:    deletionTimestamp,
	}

	return utils.CreateLogEntry("namespace", utils.ExtractName(ns), utils.ExtractNamespace(ns), data)
}
