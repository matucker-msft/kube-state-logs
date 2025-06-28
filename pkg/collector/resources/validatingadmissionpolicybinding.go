package resources

import (
	"context"
	"time"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// ValidatingAdmissionPolicyBindingHandler handles collection of validatingadmissionpolicybinding metrics
type ValidatingAdmissionPolicyBindingHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewValidatingAdmissionPolicyBindingHandler creates a new ValidatingAdmissionPolicyBindingHandler
func NewValidatingAdmissionPolicyBindingHandler(client *kubernetes.Clientset) *ValidatingAdmissionPolicyBindingHandler {
	return &ValidatingAdmissionPolicyBindingHandler{
		client: client,
	}
}

// SetupInformer sets up the validatingadmissionpolicybinding informer
func (h *ValidatingAdmissionPolicyBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create validatingadmissionpolicybinding informer
	h.informer = factory.Admissionregistration().V1beta1().ValidatingAdmissionPolicyBindings().Informer()

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

// Collect gathers validatingadmissionpolicybinding metrics from the cluster (uses cache)
func (h *ValidatingAdmissionPolicyBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all validatingadmissionpolicybindings from the cache
	vapbList := safeGetStoreList(h.informer)

	for _, obj := range vapbList {
		vapb, ok := obj.(*admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding)
		if !ok {
			continue
		}

		entry := h.createLogEntry(vapb)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a ValidatingAdmissionPolicyBinding
func (h *ValidatingAdmissionPolicyBindingHandler) createLogEntry(vapb *admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding) types.LogEntry {
	// Extract basic metadata
	createdTimestamp := int64(0)
	if creationTime := vapb.GetCreationTimestamp(); !creationTime.IsZero() {
		createdTimestamp = creationTime.Unix()
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if ownerRefs := vapb.GetOwnerReferences(); len(ownerRefs) > 0 {
		createdByKind = ownerRefs[0].Kind
		createdByName = ownerRefs[0].Name
	}

	// Extract basic fields
	policyName := ""
	if vapb.Spec.PolicyName != "" {
		policyName = vapb.Spec.PolicyName
	}

	paramRef := ""
	if vapb.Spec.ParamRef != nil {
		paramRef = vapb.Spec.ParamRef.Name
	}

	observedGeneration := int64(0)
	// Status field not available in this API version

	// Create data structure
	// See: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
	data := types.ValidatingAdmissionPolicyBindingData{
		CreatedTimestamp:   createdTimestamp,
		Labels:             vapb.GetLabels(),
		Annotations:        vapb.GetAnnotations(),
		PolicyName:         policyName,
		ParamRef:           paramRef,
		MatchResources:     []string{}, // Simplified for now
		ValidationActions:  []string{}, // Simplified for now
		ObservedGeneration: observedGeneration,
		CreatedByKind:      createdByKind,
		CreatedByName:      createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "validatingadmissionpolicybinding",
		Name:         vapb.GetName(),
		Namespace:    "", // ValidatingAdmissionPolicyBinding is cluster-scoped
		Data:         convertStructToMap(data),
	}
}
