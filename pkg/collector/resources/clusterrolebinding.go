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

// ClusterRoleBindingHandler handles collection of clusterrolebinding metrics
type ClusterRoleBindingHandler struct {
	utils.BaseHandler
}

// NewClusterRoleBindingHandler creates a new ClusterRoleBindingHandler
func NewClusterRoleBindingHandler(client *kubernetes.Clientset) *ClusterRoleBindingHandler {
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
func (h *ClusterRoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all clusterrolebindings from the cache
	crbList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range crbList {
		crb, ok := obj.(*rbacv1.ClusterRoleBinding)
		if !ok {
			continue
		}

		entry := h.createLogEntry(crb)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a clusterrolebinding
func (h *ClusterRoleBindingHandler) createLogEntry(crb *rbacv1.ClusterRoleBinding) types.LogEntry {
	// Convert role ref
	roleRef := types.RoleRef{
		APIGroup: crb.RoleRef.APIGroup,
		Kind:     crb.RoleRef.Kind,
		Name:     crb.RoleRef.Name,
	}

	// Convert subjects
	var subjects []types.Subject
	for _, subject := range crb.Subjects {
		subj := types.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		}
		subjects = append(subjects, subj)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(crb)

	// Create data structure
	data := types.ClusterRoleBindingData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(crb),
		Labels:           utils.ExtractLabels(crb),
		Annotations:      utils.ExtractAnnotations(crb),
		RoleRef:          roleRef,
		Subjects:         subjects,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("clusterrolebinding", utils.ExtractName(crb), utils.ExtractNamespace(crb), data)
}
