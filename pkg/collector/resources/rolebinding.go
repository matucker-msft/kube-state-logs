package resources

import (
	"context"
	"slices"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// RoleBindingHandler handles collection of rolebinding metrics
type RoleBindingHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewRoleBindingHandler creates a new RoleBindingHandler
func NewRoleBindingHandler(client *kubernetes.Clientset) *RoleBindingHandler {
	return &RoleBindingHandler{
		client: client,
	}
}

// SetupInformer sets up the rolebinding informer
func (h *RoleBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create rolebinding informer
	h.informer = factory.Rbac().V1().RoleBindings().Informer()

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

// Collect gathers rolebinding metrics from the cluster (uses cache)
func (h *RoleBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all rolebindings from the cache
	rbList := safeGetStoreList(h.informer)

	for _, obj := range rbList {
		rb, ok := obj.(*rbacv1.RoleBinding)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, rb.Namespace) {
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

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(rb.OwnerReferences) > 0 {
		createdByKind = rb.OwnerReferences[0].Kind
		createdByName = rb.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.RoleBindingData{
		CreatedTimestamp: rb.CreationTimestamp.Unix(),
		Labels:           rb.Labels,
		Annotations:      rb.Annotations,
		RoleRef:          roleRef,
		Subjects:         subjects,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "rolebinding",
		Name:         rb.Name,
		Namespace:    rb.Namespace,
		Data:         convertStructToMap(data),
	}
}
