package resources

import (
	"context"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// LeaseHandler handles collection of lease metrics
type LeaseHandler struct {
	utils.BaseHandler
}

// NewLeaseHandler creates a new LeaseHandler
func NewLeaseHandler(client *kubernetes.Clientset) *LeaseHandler {
	return &LeaseHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the lease informer
func (h *LeaseHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create lease informer
	informer := factory.Coordination().V1().Leases().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers lease metrics from the cluster (uses cache)
func (h *LeaseHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all leases from the cache
	leaseList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range leaseList {
		lease, ok := obj.(*coordinationv1.Lease)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, lease.Namespace) {
			continue
		}

		entry := h.createLogEntry(lease)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a lease
func (h *LeaseHandler) createLogEntry(lease *coordinationv1.Lease) types.LogEntry {
	// Extract timestamps
	var renewTime *time.Time
	var acquireTime *time.Time

	if lease.Spec.RenewTime != nil {
		renewTime = &lease.Spec.RenewTime.Time
	}
	if lease.Spec.AcquireTime != nil {
		acquireTime = &lease.Spec.AcquireTime.Time
	}

	// Extract holder identity
	holderIdentity := ""
	if lease.Spec.HolderIdentity != nil {
		holderIdentity = *lease.Spec.HolderIdentity
	}

	// Extract lease duration seconds with nil check
	leaseDurationSeconds := int32(0)
	if lease.Spec.LeaseDurationSeconds != nil {
		leaseDurationSeconds = *lease.Spec.LeaseDurationSeconds
	}

	// Extract lease transitions with nil check
	leaseTransitions := int32(0)
	if lease.Spec.LeaseTransitions != nil {
		leaseTransitions = *lease.Spec.LeaseTransitions
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(lease)

	data := types.LeaseData{
		CreatedTimestamp:     utils.ExtractCreationTimestamp(lease),
		Labels:               utils.ExtractLabels(lease),
		Annotations:          utils.ExtractAnnotations(lease),
		HolderIdentity:       holderIdentity,
		LeaseDurationSeconds: leaseDurationSeconds,
		RenewTime:            renewTime,
		AcquireTime:          acquireTime,
		LeaseTransitions:     leaseTransitions,
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
	}

	return utils.CreateLogEntry("lease", utils.ExtractName(lease), utils.ExtractNamespace(lease), data)
}
