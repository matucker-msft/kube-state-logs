package resources

import (
	"context"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ClusterRoleHandler handles collection of clusterrole metrics
type ClusterRoleHandler struct {
	utils.BaseHandler
}

// NewClusterRoleHandler creates a new ClusterRoleHandler
func NewClusterRoleHandler(client kubernetes.Interface) *ClusterRoleHandler {
	return &ClusterRoleHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the clusterrole informer
func (h *ClusterRoleHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create clusterrole informer
	informer := factory.Rbac().V1().ClusterRoles().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers clusterrole metrics from the cluster (uses cache)
func (h *ClusterRoleHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all clusterroles from the cache
	crList := utils.SafeGetStoreList(h.GetInformer())

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
		CreatedTimestamp: utils.ExtractCreationTimestamp(cr),
		Labels:           utils.ExtractLabels(cr),
		Annotations:      utils.ExtractAnnotations(cr),
		Rules:            rules,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("clusterrole", utils.ExtractName(cr), utils.ExtractNamespace(cr), data)
}
