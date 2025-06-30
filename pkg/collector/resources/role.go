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

// RoleHandler handles collection of role metrics
type RoleHandler struct {
	utils.BaseHandler
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(client kubernetes.Interface) *RoleHandler {
	return &RoleHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the role informer
func (h *RoleHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create role informer
	informer := factory.Rbac().V1().Roles().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers role metrics from the cluster (uses cache)
func (h *RoleHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all roles from the cache
	roles := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range roles {
		role, ok := obj.(*rbacv1.Role)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, role.Namespace) {
			continue
		}

		entry := h.createLogEntry(role)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a RoleData from a role
func (h *RoleHandler) createLogEntry(role *rbacv1.Role) types.RoleData {
	// Convert rules
	// See: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole
	var rules []types.PolicyRule
	for _, rule := range role.Rules {
		policyRule := types.PolicyRule{
			APIGroups:     rule.APIGroups,
			Resources:     rule.Resources,
			ResourceNames: rule.ResourceNames,
			Verbs:         rule.Verbs,
		}
		rules = append(rules, policyRule)
	}

	// Get created by info
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(role)

	// Create data structure
	data := types.RoleData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "role",
			Name:             utils.ExtractName(role),
			Namespace:        utils.ExtractNamespace(role),
			CreatedTimestamp: utils.ExtractCreationTimestamp(role),
			Labels:           utils.ExtractLabels(role),
			Annotations:      utils.ExtractAnnotations(role),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		Rules: rules,
	}

	return data
}
