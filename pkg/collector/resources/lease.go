package resources

import (
	"context"
	"slices"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// LeaseHandler handles collection of lease metrics
type LeaseHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewLeaseHandler creates a new LeaseHandler
func NewLeaseHandler(client *kubernetes.Clientset) *LeaseHandler {
	return &LeaseHandler{
		client: client,
	}
}

// SetupInformer sets up the lease informer
func (h *LeaseHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create lease informer
	h.informer = factory.Coordination().V1().Leases().Informer()

	return nil
}

// Collect gathers lease metrics from the cluster (uses cache)
func (h *LeaseHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all leases from the cache
	leaseList := safeGetStoreList(h.informer)

	for _, obj := range leaseList {
		lease, ok := obj.(*coordinationv1.Lease)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, lease.Namespace) {
			continue
		}

		entry := h.createLogEntry(lease)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a lease
func (h *LeaseHandler) createLogEntry(lease *coordinationv1.Lease) types.LogEntry {
	// Get holder identity
	holderIdentity := ""
	if lease.Spec.HolderIdentity != nil {
		holderIdentity = *lease.Spec.HolderIdentity
	}

	// Get lease duration seconds
	// Default is 15 seconds when leaseDurationSeconds is nil
	// See: https://kubernetes.io/docs/concepts/architecture/leases/
	leaseDurationSeconds := int32(15) // Default value
	if lease.Spec.LeaseDurationSeconds != nil {
		leaseDurationSeconds = *lease.Spec.LeaseDurationSeconds
	}

	// Get renew time
	var renewTime *time.Time
	if lease.Spec.RenewTime != nil {
		renewTime = &lease.Spec.RenewTime.Time
	}

	// Get acquire time
	var acquireTime *time.Time
	if lease.Spec.AcquireTime != nil {
		acquireTime = &lease.Spec.AcquireTime.Time
	}

	// Get lease transitions
	leaseTransitions := int32(0)
	if lease.Spec.LeaseTransitions != nil {
		leaseTransitions = *lease.Spec.LeaseTransitions
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(lease)

	// Create data structure
	data := types.LeaseData{
		CreatedTimestamp:     lease.CreationTimestamp.Unix(),
		Labels:               lease.Labels,
		Annotations:          lease.Annotations,
		HolderIdentity:       holderIdentity,
		LeaseDurationSeconds: leaseDurationSeconds,
		RenewTime:            renewTime,
		AcquireTime:          acquireTime,
		LeaseTransitions:     leaseTransitions,
		CreatedByKind:        createdByKind,
		CreatedByName:        createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "lease",
		Name:         lease.Name,
		Namespace:    lease.Namespace,
		Data:         convertStructToMap(data),
	}
}
