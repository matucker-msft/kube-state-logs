package resources

import (
	"context"
	"time"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"go.goms.io/aks/kube-state-logs/pkg/interfaces"
	"go.goms.io/aks/kube-state-logs/pkg/types"
	"go.goms.io/aks/kube-state-logs/pkg/utils"
)

// ValidatingAdmissionPolicyBindingHandler handles collection of validatingadmissionpolicybinding metrics
type ValidatingAdmissionPolicyBindingHandler struct {
	utils.BaseHandler
}

// NewValidatingAdmissionPolicyBindingHandler creates a new ValidatingAdmissionPolicyBindingHandler
func NewValidatingAdmissionPolicyBindingHandler(client kubernetes.Interface) *ValidatingAdmissionPolicyBindingHandler {
	return &ValidatingAdmissionPolicyBindingHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the validatingadmissionpolicybinding informer
func (h *ValidatingAdmissionPolicyBindingHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create validatingadmissionpolicybinding informer
	informer := factory.Admissionregistration().V1beta1().ValidatingAdmissionPolicyBindings().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers validatingadmissionpolicybinding metrics from the cluster (uses cache)
func (h *ValidatingAdmissionPolicyBindingHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all validatingadmissionpolicybindings from the cache
	bindings := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range bindings {
		binding, ok := obj.(*admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, binding.Namespace) {
			continue
		}

		entry := h.createLogEntry(binding)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a ValidatingAdmissionPolicyBindingData from a validatingadmissionpolicybinding
func (h *ValidatingAdmissionPolicyBindingHandler) createLogEntry(binding *admissionregistrationv1beta1.ValidatingAdmissionPolicyBinding) types.ValidatingAdmissionPolicyBindingData {
	createdTimestamp := utils.ExtractCreationTimestamp(binding)
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(binding)

	policyName := ""
	if binding.Spec.PolicyName != "" {
		policyName = binding.Spec.PolicyName
	}

	paramRef := ""
	if binding.Spec.ParamRef != nil {
		paramRef = binding.Spec.ParamRef.Name
	}

	observedGeneration := int64(0)

	data := types.ValidatingAdmissionPolicyBindingData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "validatingadmissionpolicybinding",
			Name:             utils.ExtractName(binding),
			Namespace:        utils.ExtractNamespace(binding),
			CreatedTimestamp: createdTimestamp,
			Labels:           binding.GetLabels(),
			Annotations:      binding.GetAnnotations(),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		PolicyName:         policyName,
		ParamRef:           paramRef,
		MatchResources:     []string{},
		ValidationActions:  []string{},
		ObservedGeneration: observedGeneration,
	}

	return data
}
