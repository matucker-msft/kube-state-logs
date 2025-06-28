package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// SecretHandler handles collection of secret metrics
type SecretHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewSecretHandler creates a new SecretHandler
func NewSecretHandler(client *kubernetes.Clientset) *SecretHandler {
	return &SecretHandler{
		client: client,
	}
}

// SetupInformer sets up the secret informer
func (h *SecretHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create secret informer
	h.informer = factory.Core().V1().Secrets().Informer()

	return nil
}

// Collect gathers secret metrics from the cluster (uses cache)
func (h *SecretHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all secrets from the cache
	secrets := utils.SafeGetStoreList(h.informer)

	for _, obj := range secrets {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, secret.Namespace) {
			continue
		}

		entry := h.createLogEntry(secret)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a secret
func (h *SecretHandler) createLogEntry(secret *corev1.Secret) types.LogEntry {
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(secret)

	// Get data keys (we don't expose the actual secret data, just the keys)
	// See: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types
	var dataKeys []string
	for key := range secret.Data {
		dataKeys = append(dataKeys, key)
	}

	data := types.SecretData{
		CreatedTimestamp: secret.CreationTimestamp.Unix(),
		Labels:           secret.Labels,
		Annotations:      secret.Annotations,
		Type:             string(secret.Type),
		DataKeys:         dataKeys,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "secret",
		Name:         secret.Name,
		Namespace:    secret.Namespace,
		Data:         h.convertToMap(data),
	}
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *SecretHandler) convertToMap(data any) map[string]any {
	return utils.ConvertStructToMap(data)
}
