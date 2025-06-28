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

// ValidatingAdmissionPolicyHandler handles collection of validatingadmissionpolicy metrics
type ValidatingAdmissionPolicyHandler struct {
	utils.BaseHandler
}

// NewValidatingAdmissionPolicyHandler creates a new ValidatingAdmissionPolicyHandler
func NewValidatingAdmissionPolicyHandler(client *kubernetes.Clientset) *ValidatingAdmissionPolicyHandler {
	return &ValidatingAdmissionPolicyHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the validatingadmissionpolicy informer
func (h *ValidatingAdmissionPolicyHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create validatingadmissionpolicy informer
	informer := factory.Admissionregistration().V1beta1().ValidatingAdmissionPolicies().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers validatingadmissionpolicy metrics from the cluster (uses cache)
func (h *ValidatingAdmissionPolicyHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all validatingadmissionpolicies from the cache
	vapList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range vapList {
		vap, ok := obj.(*admissionregistrationv1beta1.ValidatingAdmissionPolicy)
		if !ok {
			continue
		}

		entry := h.createLogEntry(vap)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a ValidatingAdmissionPolicy
func (h *ValidatingAdmissionPolicyHandler) createLogEntry(vap *admissionregistrationv1beta1.ValidatingAdmissionPolicy) types.LogEntry {
	createdTimestamp := utils.ExtractCreationTimestamp(vap)
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(vap)

	failurePolicy := ""
	if vap.Spec.FailurePolicy != nil {
		failurePolicy = string(*vap.Spec.FailurePolicy)
	}

	paramKind := ""
	if vap.Spec.ParamKind != nil {
		paramKind = vap.Spec.ParamKind.Kind
	}

	observedGeneration := int64(0)
	if vap.Status.ObservedGeneration != 0 {
		observedGeneration = vap.Status.ObservedGeneration
	}

	data := types.ValidatingAdmissionPolicyData{
		CreatedTimestamp:   createdTimestamp,
		Labels:             vap.GetLabels(),
		Annotations:        vap.GetAnnotations(),
		FailurePolicy:      failurePolicy,
		MatchConstraints:   []string{},
		Validations:        []string{},
		AuditAnnotations:   []string{},
		MatchConditions:    []string{},
		Variables:          []string{},
		ParamKind:          paramKind,
		ObservedGeneration: observedGeneration,
		TypeChecking:       "",
		ExpressionWarnings: []string{},
		CreatedByKind:      createdByKind,
		CreatedByName:      createdByName,
	}

	return utils.CreateLogEntry("validatingadmissionpolicy", utils.ExtractName(vap), utils.ExtractNamespace(vap), data)
}
