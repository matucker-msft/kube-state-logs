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
)

// ServiceAccountHandler handles collection of serviceaccount metrics
type ServiceAccountHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewServiceAccountHandler creates a new ServiceAccountHandler
func NewServiceAccountHandler(client *kubernetes.Clientset) *ServiceAccountHandler {
	return &ServiceAccountHandler{
		client: client,
	}
}

// SetupInformer sets up the serviceaccount informer
func (h *ServiceAccountHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create serviceaccount informer
	h.informer = factory.Core().V1().ServiceAccounts().Informer()

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

// Collect gathers serviceaccount metrics from the cluster (uses cache)
func (h *ServiceAccountHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all serviceaccounts from the cache
	serviceAccounts := h.informer.GetStore().List()

	for _, obj := range serviceAccounts {
		sa, ok := obj.(*corev1.ServiceAccount)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, sa.Namespace) {
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

	// Create data structure
	data := types.ServiceAccountData{
		CreatedTimestamp:             sa.CreationTimestamp.Unix(),
		Labels:                       sa.Labels,
		Annotations:                  sa.Annotations,
		Secrets:                      secrets,
		ImagePullSecrets:             imagePullSecrets,
		CreatedByKind:                "",
		CreatedByName:                "",
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
	}

	// Convert to map[string]any for the LogEntry
	dataMap := map[string]any{
		"createdTimestamp":             data.CreatedTimestamp,
		"labels":                       data.Labels,
		"annotations":                  data.Annotations,
		"secrets":                      data.Secrets,
		"imagePullSecrets":             data.ImagePullSecrets,
		"createdByKind":                data.CreatedByKind,
		"createdByName":                data.CreatedByName,
		"automountServiceAccountToken": data.AutomountServiceAccountToken,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "serviceaccount",
		Name:         sa.Name,
		Namespace:    sa.Namespace,
		Data:         dataMap,
	}
}
