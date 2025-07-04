package resources

import (
	"context"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"go.goms.io/aks/kube-state-logs/pkg/interfaces"
	"go.goms.io/aks/kube-state-logs/pkg/types"
	"go.goms.io/aks/kube-state-logs/pkg/utils"
)

// ClusterRoleBindingHandler handles collection of clusterrolebinding metrics
type ClusterRoleBindingHandler struct {
	utils.BaseHandler
}

// NewClusterRoleBindingHandler creates a new ClusterRoleBindingHandler
func NewClusterRoleBindingHandler(client kubernetes.Interface) *ClusterRoleBindingHandler {
	return &ClusterRoleBindingHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the clusterrolebinding informer
func (h *ClusterRoleBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create clusterrolebinding informer
	informer := factory.Rbac().V1().ClusterRoleBindings().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers clusterrolebinding metrics from the cluster (uses cache)
func (h *ClusterRoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all clusterrolebindings from the cache
	clusterrolebindings := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range clusterrolebindings {
		clusterrolebinding, ok := obj.(*rbacv1.ClusterRoleBinding)
		if !ok {
			continue
		}

		entry := h.createLogEntry(clusterrolebinding)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a ClusterRoleBindingData from a clusterrolebinding
func (h *ClusterRoleBindingHandler) createLogEntry(binding *rbacv1.ClusterRoleBinding) types.ClusterRoleBindingData {
	// Convert role ref
	roleRef := types.RoleRef{
		APIGroup: binding.RoleRef.APIGroup,
		Kind:     binding.RoleRef.Kind,
		Name:     binding.RoleRef.Name,
	}

	// Convert subjects
	var subjects []types.Subject
	for _, subject := range binding.Subjects {
		subj := types.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		}
		subjects = append(subjects, subj)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(binding)

	// Create data structure
	data := types.ClusterRoleBindingData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "clusterrolebinding",
			Name:             utils.ExtractName(binding),
			Namespace:        utils.ExtractNamespace(binding),
			CreatedTimestamp: utils.ExtractCreationTimestamp(binding),
			Labels:           utils.ExtractLabels(binding),
			Annotations:      utils.ExtractAnnotations(binding),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		RoleRef:  roleRef,
		Subjects: subjects,
	}

	return data
}
