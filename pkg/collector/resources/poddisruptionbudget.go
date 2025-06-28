package resources

import (
	"context"
	"slices"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PodDisruptionBudgetHandler handles collection of poddisruptionbudget metrics
type PodDisruptionBudgetHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewPodDisruptionBudgetHandler creates a new PodDisruptionBudgetHandler
func NewPodDisruptionBudgetHandler(client *kubernetes.Clientset) *PodDisruptionBudgetHandler {
	return &PodDisruptionBudgetHandler{
		client: client,
	}
}

// SetupInformer sets up the poddisruptionbudget informer
func (h *PodDisruptionBudgetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create poddisruptionbudget informer
	h.informer = factory.Policy().V1().PodDisruptionBudgets().Informer()

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

// Collect gathers poddisruptionbudget metrics from the cluster (uses cache)
func (h *PodDisruptionBudgetHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all poddisruptionbudgets from the cache
	pdbList := safeGetStoreList(h.informer)

	for _, obj := range pdbList {
		pdb, ok := obj.(*policyv1.PodDisruptionBudget)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, pdb.Namespace) {
			continue
		}

		entry := h.createLogEntry(pdb)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a poddisruptionbudget
func (h *PodDisruptionBudgetHandler) createLogEntry(pdb *policyv1.PodDisruptionBudget) types.LogEntry {
	// Get min available and max unavailable
	// See: https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets
	minAvailable := int32(0)
	maxUnavailable := int32(0)

	if pdb.Spec.MinAvailable != nil {
		minAvailable = pdb.Spec.MinAvailable.IntVal
	}
	if pdb.Spec.MaxUnavailable != nil {
		maxUnavailable = pdb.Spec.MaxUnavailable.IntVal
	}

	// Get status values
	currentHealthy := pdb.Status.CurrentHealthy
	desiredHealthy := pdb.Status.DesiredHealthy
	expectedPods := pdb.Status.ExpectedPods
	disruptionsAllowed := pdb.Status.DisruptionsAllowed
	disruptionAllowed := disruptionsAllowed > 0

	
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pdb)

	// Create data structure
	data := types.PodDisruptionBudgetData{
		CreatedTimestamp:         pdb.CreationTimestamp.Unix(),
		Labels:                   pdb.Labels,
		Annotations:              pdb.Annotations,
		MinAvailable:             minAvailable,
		MaxUnavailable:           maxUnavailable,
		CurrentHealthy:           currentHealthy,
		DesiredHealthy:           desiredHealthy,
		ExpectedPods:             expectedPods,
		DisruptionsAllowed:       disruptionsAllowed,
		TotalReplicas:            0, // Not available in v1 API
		DisruptionAllowed:        disruptionAllowed,
		StatusCurrentHealthy:     currentHealthy,
		StatusDesiredHealthy:     desiredHealthy,
		StatusExpectedPods:       expectedPods,
		StatusDisruptionsAllowed: disruptionsAllowed,
		StatusTotalReplicas:      0, // Not available in v1 API
		StatusDisruptionAllowed:  disruptionAllowed,
		CreatedByKind:            createdByKind,
		CreatedByName:            createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "poddisruptionbudget",
		Name:         pdb.Name,
		Namespace:    pdb.Namespace,
		Data:         convertStructToMap(data),
	}
}
