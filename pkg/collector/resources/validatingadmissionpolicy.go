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
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ValidatingAdmissionPolicyHandler handles collection of validatingadmissionpolicy metrics
type ValidatingAdmissionPolicyHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewValidatingAdmissionPolicyHandler creates a new ValidatingAdmissionPolicyHandler
func NewValidatingAdmissionPolicyHandler(client *kubernetes.Clientset) *ValidatingAdmissionPolicyHandler {
	return &ValidatingAdmissionPolicyHandler{
		client: client,
	}
}

// SetupInformer sets up the validatingadmissionpolicy informer
func (h *ValidatingAdmissionPolicyHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create validatingadmissionpolicy informer
	h.informer = factory.Admissionregistration().V1beta1().ValidatingAdmissionPolicies().Informer()

	return nil
}

// Collect gathers validatingadmissionpolicy metrics from the cluster (uses cache)
func (h *ValidatingAdmissionPolicyHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all validatingadmissionpolicies from the cache
	vapList := utils.SafeGetStoreList(h.informer)

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
	// Extract basic metadata
	createdTimestamp := int64(0)
	if creationTime := vap.GetCreationTimestamp(); !creationTime.IsZero() {
		createdTimestamp = creationTime.Unix()
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(vap)

	// Extract basic fields
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

	// Create data structure
	// See: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
	data := types.ValidatingAdmissionPolicyData{
		CreatedTimestamp:   createdTimestamp,
		Labels:             vap.GetLabels(),
		Annotations:        vap.GetAnnotations(),
		FailurePolicy:      failurePolicy,
		MatchConstraints:   []string{}, // Simplified for now
		Validations:        []string{}, // Simplified for now
		AuditAnnotations:   []string{}, // Simplified for now
		MatchConditions:    []string{}, // Simplified for now
		Variables:          []string{}, // Simplified for now
		ParamKind:          paramKind,
		ObservedGeneration: observedGeneration,
		TypeChecking:       "",         // Simplified for now
		ExpressionWarnings: []string{}, // Simplified for now
		CreatedByKind:      createdByKind,
		CreatedByName:      createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "validatingadmissionpolicy",
		Name:         vap.GetName(),
		Namespace:    "", // ValidatingAdmissionPolicy is cluster-scoped
		Data:         utils.ConvertStructToMap(data),
	}
}
