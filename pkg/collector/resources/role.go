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

// RoleHandler handles collection of role metrics
type RoleHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(client *kubernetes.Clientset) *RoleHandler {
	return &RoleHandler{
		client: client,
	}
}

// SetupInformer sets up the role informer
func (h *RoleHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create role informer
	h.informer = factory.Rbac().V1().Roles().Informer()

	return nil
}

// Collect gathers role metrics from the cluster (uses cache)
func (h *RoleHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all roles from the cache
	roleList := safeGetStoreList(h.informer)

	for _, obj := range roleList {
		role, ok := obj.(*rbacv1.Role)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if !utils.ShouldIncludeNamespace(namespaces, role.Namespace) {
			continue
		}

		entry := h.createLogEntry(role)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a role
func (h *RoleHandler) createLogEntry(role *rbacv1.Role) types.LogEntry {
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
		CreatedTimestamp: role.CreationTimestamp.Unix(),
		Labels:           role.Labels,
		Annotations:      role.Annotations,
		Rules:            rules,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "role",
		Name:         role.Name,
		Namespace:    role.Namespace,
		Data:         convertStructToMap(data),
	}
}
