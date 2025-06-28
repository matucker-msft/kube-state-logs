package resources

import (
	"context"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// CertificateSigningRequestHandler handles collection of certificatesigningrequest metrics
type CertificateSigningRequestHandler struct {
	utils.BaseHandler
}

// NewCertificateSigningRequestHandler creates a new CertificateSigningRequestHandler
func NewCertificateSigningRequestHandler(client kubernetes.Interface) *CertificateSigningRequestHandler {
	return &CertificateSigningRequestHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the certificatesigningrequest informer
func (h *CertificateSigningRequestHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create certificatesigningrequest informer
	informer := factory.Certificates().V1().CertificateSigningRequests().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers certificatesigningrequest metrics from the cluster (uses cache)
func (h *CertificateSigningRequestHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all certificatesigningrequests from the cache
	csrList := utils.SafeGetStoreList(h.GetInformer())

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

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(csr)

	// Create data structure
	data := types.CertificateSigningRequestData{
		CreatedTimestamp:  utils.ExtractCreationTimestamp(csr),
		Labels:            utils.ExtractLabels(csr),
		Annotations:       utils.ExtractAnnotations(csr),
		Status:            status,
		SignerName:        csr.Spec.SignerName,
		ExpirationSeconds: csr.Spec.ExpirationSeconds,
		Usages:            usages,
		CreatedByKind:     createdByKind,
		CreatedByName:     createdByName,
	}

	return utils.CreateLogEntry("certificatesigningrequest", utils.ExtractName(csr), utils.ExtractNamespace(csr), data)
}
