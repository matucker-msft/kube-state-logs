package resources

import (
	"context"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// CertificateSigningRequestHandler handles collection of certificatesigningrequest metrics
type CertificateSigningRequestHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewCertificateSigningRequestHandler creates a new CertificateSigningRequestHandler
func NewCertificateSigningRequestHandler(client *kubernetes.Clientset) *CertificateSigningRequestHandler {
	return &CertificateSigningRequestHandler{
		client: client,
	}
}

// SetupInformer sets up the certificatesigningrequest informer
func (h *CertificateSigningRequestHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create certificatesigningrequest informer
	h.informer = factory.Certificates().V1().CertificateSigningRequests().Informer()

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

// Collect gathers certificatesigningrequest metrics from the cluster (uses cache)
func (h *CertificateSigningRequestHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all certificatesigningrequests from the cache
	csrList := h.informer.GetStore().List()

	for _, obj := range csrList {
		csr, ok := obj.(*certificatesv1.CertificateSigningRequest)
		if !ok {
			continue
		}

		entry := h.createLogEntry(csr)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a certificatesigningrequest
func (h *CertificateSigningRequestHandler) createLogEntry(csr *certificatesv1.CertificateSigningRequest) types.LogEntry {
	// Convert usages to strings
	var usages []string
	for _, usage := range csr.Spec.Usages {
		usages = append(usages, string(usage))
	}

	// Get status
	status := ""
	if len(csr.Status.Conditions) > 0 {
		status = string(csr.Status.Conditions[0].Type)
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(csr.OwnerReferences) > 0 {
		createdByKind = csr.OwnerReferences[0].Kind
		createdByName = csr.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.CertificateSigningRequestData{
		CreatedTimestamp:  csr.CreationTimestamp.Unix(),
		Labels:            csr.Labels,
		Annotations:       csr.Annotations,
		Status:            status,
		SignerName:        csr.Spec.SignerName,
		ExpirationSeconds: csr.Spec.ExpirationSeconds,
		Usages:            usages,
		CreatedByKind:     createdByKind,
		CreatedByName:     createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "certificatesigningrequest",
		Name:         csr.Name,
		Namespace:    "", // CertificateSigningRequests are cluster-scoped
		Data:         convertStructToMap(data),
	}
}
