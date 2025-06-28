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

// MutatingWebhookConfigurationHandler handles collection of mutatingwebhookconfiguration metrics
type MutatingWebhookConfigurationHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewMutatingWebhookConfigurationHandler creates a new MutatingWebhookConfigurationHandler
func NewMutatingWebhookConfigurationHandler(client *kubernetes.Clientset) *MutatingWebhookConfigurationHandler {
	return &MutatingWebhookConfigurationHandler{
		client: client,
	}
}

// SetupInformer sets up the mutatingwebhookconfiguration informer
func (h *MutatingWebhookConfigurationHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create mutatingwebhookconfiguration informer
	h.informer = factory.Admissionregistration().V1().MutatingWebhookConfigurations().Informer()

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

// Collect gathers mutatingwebhookconfiguration metrics from the cluster (uses cache)
func (h *MutatingWebhookConfigurationHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all mutatingwebhookconfigurations from the cache
	mwcList := safeGetStoreList(h.informer)

	for _, obj := range mwcList {
		mwc, ok := obj.(*admissionregistrationv1.MutatingWebhookConfiguration)
		if !ok {
			continue
		}

		entry := h.createLogEntry(mwc)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a mutatingwebhookconfiguration
func (h *MutatingWebhookConfigurationHandler) createLogEntry(mwc *admissionregistrationv1.MutatingWebhookConfiguration) types.LogEntry {
	// Extract webhooks
	var webhooks []types.WebhookData
	for _, webhook := range mwc.Webhooks {
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
	data := types.MutatingWebhookConfigurationData{
		CreatedTimestamp: mwc.CreationTimestamp.Unix(),
		Labels:           mwc.Labels,
		Annotations:      mwc.Annotations,
		Webhooks:         webhooks,
		CreatedByKind:    "",
		CreatedByName:    "",
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "mutatingwebhookconfiguration",
		Name:         mwc.Name,
		Namespace:    "", // Cluster-scoped resource
		Data:         convertStructToMap(data),
	}
}
