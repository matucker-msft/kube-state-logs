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

// ClusterRoleBindingHandler handles collection of clusterrolebinding metrics
type ClusterRoleBindingHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewClusterRoleBindingHandler creates a new ClusterRoleBindingHandler
func NewClusterRoleBindingHandler(client *kubernetes.Clientset) *ClusterRoleBindingHandler {
	return &ClusterRoleBindingHandler{
		client: client,
	}
}

// SetupInformer sets up the clusterrolebinding informer
func (h *ClusterRoleBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create clusterrolebinding informer
	h.informer = factory.Rbac().V1().ClusterRoleBindings().Informer()

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

// Collect gathers clusterrolebinding metrics from the cluster (uses cache)
func (h *ClusterRoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all clusterrolebindings from the cache
	crbList := safeGetStoreList(h.informer)

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
	// See: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding
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

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(crb.OwnerReferences) > 0 {
		createdByKind = crb.OwnerReferences[0].Kind
		createdByName = crb.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.ClusterRoleBindingData{
		CreatedTimestamp: crb.CreationTimestamp.Unix(),
		Labels:           crb.Labels,
		Annotations:      crb.Annotations,
		RoleRef:          roleRef,
		Subjects:         subjects,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "clusterrolebinding",
		Name:         crb.Name,
		Namespace:    "", // ClusterRoleBindings are cluster-scoped
		Data:         convertStructToMap(data),
	}
}
