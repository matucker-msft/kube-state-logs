package resources

import (
	"context"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// ValidatingWebhookConfigurationHandler handles collection of validatingwebhookconfiguration metrics
type ValidatingWebhookConfigurationHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewValidatingWebhookConfigurationHandler creates a new ValidatingWebhookConfigurationHandler
func NewValidatingWebhookConfigurationHandler(client *kubernetes.Clientset) *ValidatingWebhookConfigurationHandler {
	return &ValidatingWebhookConfigurationHandler{
		client: client,
	}
}

// SetupInformer sets up the validatingwebhookconfiguration informer
func (h *ValidatingWebhookConfigurationHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create validatingwebhookconfiguration informer
	h.informer = factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer()

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

// Collect gathers validatingwebhookconfiguration metrics from the cluster (uses cache)
func (h *ValidatingWebhookConfigurationHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all validatingwebhookconfigurations from the cache
	vwcList := safeGetStoreList(h.informer)

	for _, obj := range vwcList {
		vwc, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration)
		if !ok {
			continue
		}

		entry := h.createLogEntry(vwc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a validatingwebhookconfiguration
func (h *ValidatingWebhookConfigurationHandler) createLogEntry(vwc *admissionregistrationv1.ValidatingWebhookConfiguration) types.LogEntry {
	// Extract webhooks
	// See: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook
	var webhooks []types.WebhookData
	for _, webhook := range vwc.Webhooks {
		// Extract client config
		var clientConfig types.WebhookClientConfigData
		if webhook.ClientConfig.URL != nil {
			clientConfig.URL = *webhook.ClientConfig.URL
		}
		if webhook.ClientConfig.Service != nil {
			path := ""
			if webhook.ClientConfig.Service.Path != nil {
				path = *webhook.ClientConfig.Service.Path
			}
			port := int32(0)
			if webhook.ClientConfig.Service.Port != nil {
				port = *webhook.ClientConfig.Service.Port
			}
			clientConfig.Service = &types.WebhookServiceData{
				Namespace: webhook.ClientConfig.Service.Namespace,
				Name:      webhook.ClientConfig.Service.Name,
				Path:      path,
				Port:      port,
			}
		}
		clientConfig.CABundle = webhook.ClientConfig.CABundle

		// Extract rules
		var rules []types.WebhookRuleData
		for _, rule := range webhook.Rules {
			rules = append(rules, types.WebhookRuleData{
				APIGroups:   rule.APIGroups,
				APIVersions: rule.APIVersions,
				Resources:   rule.Resources,
				Scope:       string(*rule.Scope),
			})
		}

		// Extract selectors
		var namespaceSelector map[string]string
		if webhook.NamespaceSelector != nil {
			namespaceSelector = webhook.NamespaceSelector.MatchLabels
		}

		var objectSelector map[string]string
		if webhook.ObjectSelector != nil {
			objectSelector = webhook.ObjectSelector.MatchLabels
		}

		webhooks = append(webhooks, types.WebhookData{
			Name:                    webhook.Name,
			ClientConfig:            clientConfig,
			Rules:                   rules,
			FailurePolicy:           string(*webhook.FailurePolicy),
			MatchPolicy:             string(*webhook.MatchPolicy),
			NamespaceSelector:       namespaceSelector,
			ObjectSelector:          objectSelector,
			SideEffects:             string(*webhook.SideEffects),
			TimeoutSeconds:          *webhook.TimeoutSeconds,
			AdmissionReviewVersions: webhook.AdmissionReviewVersions,
		})
	}

	// Create data structure
	data := types.ValidatingWebhookConfigurationData{
		CreatedTimestamp: vwc.CreationTimestamp.Unix(),
		Labels:           vwc.Labels,
		Annotations:      vwc.Annotations,
		Webhooks:         webhooks,
		CreatedByKind:    "",
		CreatedByName:    "",
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "validatingwebhookconfiguration",
		Name:         vwc.Name,
		Namespace:    "", // Cluster-scoped resource
		Data:         convertStructToMap(data),
	}
}
