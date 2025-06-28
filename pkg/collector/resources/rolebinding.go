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

// RoleBindingHandler handles collection of rolebinding metrics
type RoleBindingHandler struct {
	utils.BaseHandler
}

// NewRoleBindingHandler creates a new RoleBindingHandler
func NewRoleBindingHandler(client kubernetes.Interface) *RoleBindingHandler {
	return &RoleBindingHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the rolebinding informer
func (h *RoleBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create rolebinding informer
	informer := factory.Rbac().V1().RoleBindings().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers rolebinding metrics from the cluster (uses cache)
func (h *RoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all rolebindings from the cache
	rbList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range rbList {
		rb, ok := obj.(*rbacv1.RoleBinding)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, rb.Namespace) {
			continue
		}

		entry := h.createLogEntry(rb)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a rolebinding
func (h *RoleBindingHandler) createLogEntry(rb *rbacv1.RoleBinding) types.LogEntry {
	// Convert role ref
	roleRef := types.RoleRef{
		APIGroup: rb.RoleRef.APIGroup,
		Kind:     rb.RoleRef.Kind,
		Name:     rb.RoleRef.Name,
	}

	// Convert subjects
	var subjects []types.Subject
	for _, subject := range rb.Subjects {
		subj := types.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		}
		subjects = append(subjects, subj)
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(rb)

	// Create data structure
	data := types.RoleBindingData{
		CreatedTimestamp: utils.ExtractCreationTimestamp(rb),
		Labels:           utils.ExtractLabels(rb),
		Annotations:      utils.ExtractAnnotations(rb),
		RoleRef:          roleRef,
		Subjects:         subjects,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return utils.CreateLogEntry("rolebinding", utils.ExtractName(rb), utils.ExtractNamespace(rb), data)
}
