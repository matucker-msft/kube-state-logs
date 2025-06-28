package resources

import (
	"context"
	"time"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ValidatingAdmissionPolicyBindingHandler handles collection of validatingadmissionpolicybinding metrics
type ValidatingAdmissionPolicyBindingHandler struct {
	utils.BaseHandler
}

// NewValidatingAdmissionPolicyBindingHandler creates a new ValidatingAdmissionPolicyBindingHandler
func NewValidatingAdmissionPolicyBindingHandler(client *kubernetes.Clientset) *ValidatingAdmissionPolicyBindingHandler {
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
func (h *ValidatingAdmissionPolicyBindingHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all validatingadmissionpolicybindings from the cache
	vapbList := utils.SafeGetStoreList(h.GetInformer())

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
	createdTimestamp := utils.ExtractCreationTimestamp(vapb)
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(vapb)

	policyName := ""
	if vapb.Spec.PolicyName != "" {
		policyName = vapb.Spec.PolicyName
	}

	paramRef := ""
	if vapb.Spec.ParamRef != nil {
		paramRef = vapb.Spec.ParamRef.Name
	}

	observedGeneration := int64(0)

	data := types.ValidatingAdmissionPolicyBindingData{
		CreatedTimestamp:   createdTimestamp,
		Labels:             vapb.GetLabels(),
		Annotations:        vapb.GetAnnotations(),
		PolicyName:         policyName,
		ParamRef:           paramRef,
		MatchResources:     []string{},
		ValidationActions:  []string{},
		ObservedGeneration: observedGeneration,
		CreatedByKind:      createdByKind,
		CreatedByName:      createdByName,
	}

	return utils.CreateLogEntry("validatingadmissionpolicybinding", utils.ExtractName(vapb), utils.ExtractNamespace(vapb), data)
}
