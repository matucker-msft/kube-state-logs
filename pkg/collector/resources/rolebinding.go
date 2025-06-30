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
func (h *RoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all rolebindings from the cache
	rbs := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range rbs {
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

// createLogEntry creates a RoleBindingData from a rolebinding
func (h *RoleBindingHandler) createLogEntry(rb *rbacv1.RoleBinding) types.RoleBindingData {
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
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "rolebinding",
			Name:             utils.ExtractName(rb),
			Namespace:        utils.ExtractNamespace(rb),
			CreatedTimestamp: utils.ExtractCreationTimestamp(rb),
			Labels:           utils.ExtractLabels(rb),
			Annotations:      utils.ExtractAnnotations(rb),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		RoleRef:  roleRef,
		Subjects: subjects,
	}

	return data
}
