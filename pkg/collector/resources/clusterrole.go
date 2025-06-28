package resources

import (
	"context"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// ClusterRoleHandler handles collection of clusterrole metrics
type ClusterRoleHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewClusterRoleHandler creates a new ClusterRoleHandler
func NewClusterRoleHandler(client *kubernetes.Clientset) *ClusterRoleHandler {
	return &ClusterRoleHandler{
		client: client,
	}
}

// SetupInformer sets up the clusterrole informer
func (h *ClusterRoleHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create clusterrole informer
	h.informer = factory.Rbac().V1().ClusterRoles().Informer()

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

// Collect gathers clusterrole metrics from the cluster (uses cache)
func (h *ClusterRoleHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all clusterroles from the cache
	crList := h.informer.GetStore().List()

	for _, obj := range crList {
		cr, ok := obj.(*rbacv1.ClusterRole)
		if !ok {
			continue
		}

		entry := h.createLogEntry(cr)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a clusterrole
func (h *ClusterRoleHandler) createLogEntry(cr *rbacv1.ClusterRole) types.LogEntry {
	// Convert rules
	var rules []types.PolicyRule
	for _, rule := range cr.Rules {
		policyRule := types.PolicyRule{
			APIGroups:     rule.APIGroups,
			Resources:     rule.Resources,
			ResourceNames: rule.ResourceNames,
			Verbs:         rule.Verbs,
		}
		rules = append(rules, policyRule)
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(cr.OwnerReferences) > 0 {
		createdByKind = cr.OwnerReferences[0].Kind
		createdByName = cr.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.ClusterRoleData{
		CreatedTimestamp: cr.CreationTimestamp.Unix(),
		Labels:           cr.Labels,
		Annotations:      cr.Annotations,
		Rules:            rules,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "clusterrole",
		Name:         cr.Name,
		Namespace:    "", // ClusterRoles are cluster-scoped
		Data:         convertStructToMap(data),
	}
}
