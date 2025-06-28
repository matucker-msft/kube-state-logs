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
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
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

	return nil
}

// Collect gathers clusterrole metrics from the cluster (uses cache)
func (h *ClusterRoleHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all clusterroles from the cache
	crList := safeGetStoreList(h.informer)

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
	// See: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole
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

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(cr)

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
