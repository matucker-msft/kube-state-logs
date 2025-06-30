package resources

import (
	"context"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PodDisruptionBudgetHandler handles collection of poddisruptionbudget metrics
type PodDisruptionBudgetHandler struct {
	utils.BaseHandler
}

// NewPodDisruptionBudgetHandler creates a new PodDisruptionBudgetHandler
func NewPodDisruptionBudgetHandler(client kubernetes.Interface) *PodDisruptionBudgetHandler {
	return &PodDisruptionBudgetHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the poddisruptionbudget informer
func (h *PodDisruptionBudgetHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create poddisruptionbudget informer
	informer := factory.Policy().V1().PodDisruptionBudgets().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers poddisruptionbudget metrics from the cluster (uses cache)
func (h *PodDisruptionBudgetHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all poddisruptionbudgets from the cache
	pdbs := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range pdbs {
		pdb, ok := obj.(*policyv1.PodDisruptionBudget)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pdb.Namespace) {
			continue
		}

		entry := h.createLogEntry(pdb)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a PodDisruptionBudgetData from a PDB
func (h *PodDisruptionBudgetHandler) createLogEntry(pdb *policyv1.PodDisruptionBudget) types.PodDisruptionBudgetData {
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
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "poddisruptionbudget",
			Name:             utils.ExtractName(pdb),
			Namespace:        utils.ExtractNamespace(pdb),
			CreatedTimestamp: utils.ExtractCreationTimestamp(pdb),
			Labels:           utils.ExtractLabels(pdb),
			Annotations:      utils.ExtractAnnotations(pdb),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		MinAvailable:             minAvailable,
		MaxUnavailable:           maxUnavailable,
		CurrentHealthy:           currentHealthy,
		DesiredHealthy:           desiredHealthy,
		ExpectedPods:             expectedPods,
		DisruptionsAllowed:       disruptionsAllowed,
		TotalReplicas:            0,
		DisruptionAllowed:        disruptionAllowed,
		StatusCurrentHealthy:     currentHealthy,
		StatusDesiredHealthy:     desiredHealthy,
		StatusExpectedPods:       expectedPods,
		StatusDisruptionsAllowed: disruptionsAllowed,
		StatusTotalReplicas:      0,
		StatusDisruptionAllowed:  disruptionAllowed,
	}

	return data
}
