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

// ServiceAccountHandler handles collection of serviceaccount metrics
type ServiceAccountHandler struct {
	utils.BaseHandler
}

// NewServiceAccountHandler creates a new ServiceAccountHandler
func NewServiceAccountHandler(client kubernetes.Interface) *ServiceAccountHandler {
	return &ServiceAccountHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the serviceaccount informer
func (h *ServiceAccountHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create serviceaccount informer
	informer := factory.Core().V1().ServiceAccounts().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers serviceaccount metrics from the cluster (uses cache)
func (h *ServiceAccountHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all serviceaccounts from the cache
	serviceAccounts := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range serviceAccounts {
		sa, ok := obj.(*corev1.ServiceAccount)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, sa.Namespace) {
			continue
		}

		entry := h.createLogEntry(sa)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a serviceaccount
func (h *ServiceAccountHandler) createLogEntry(sa *corev1.ServiceAccount) types.LogEntry {
	// Extract secrets
	var secrets []string
	for _, secret := range sa.Secrets {
		secrets = append(secrets, secret.Name)
	}

	// Extract image pull secrets
	var imagePullSecrets []string
	for _, secret := range sa.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, secret.Name)
	}

	// Get automount service account token setting
	// Default is true when automountServiceAccountToken is nil
	// See: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server
	automountToken := true
	if sa.AutomountServiceAccountToken != nil {
		automountToken = *sa.AutomountServiceAccountToken
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(sa)

	// Create data structure
	data := types.ServiceAccountData{
		CreatedTimestamp:             utils.ExtractCreationTimestamp(sa),
		Labels:                       utils.ExtractLabels(sa),
		Annotations:                  utils.ExtractAnnotations(sa),
		Secrets:                      secrets,
		ImagePullSecrets:             imagePullSecrets,
		CreatedByKind:                createdByKind,
		CreatedByName:                createdByName,
		AutomountServiceAccountToken: &automountToken,
	}

	return utils.CreateLogEntry("serviceaccount", utils.ExtractName(sa), utils.ExtractNamespace(sa), data)
}
