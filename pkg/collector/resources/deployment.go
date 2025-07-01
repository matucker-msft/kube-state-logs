package resources

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// DeploymentHandler handles collection of deployment metrics
type DeploymentHandler struct {
	utils.BaseHandler
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(client kubernetes.Interface) *DeploymentHandler {
	return &DeploymentHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the deployment informer
func (h *DeploymentHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create deployment informer
	informer := factory.Apps().V1().Deployments().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers deployment metrics from the cluster (uses cache)
func (h *DeploymentHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all deployments from the cache
	deployments := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range deployments {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, deployment.Namespace) {
			continue
		}

		entry := h.createLogEntry(deployment)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a DeploymentData from a deployment
func (h *DeploymentHandler) createLogEntry(deployment *appsv1.Deployment) types.DeploymentData {
	// Get strategy info
	strategyType := string(deployment.Spec.Strategy.Type)
	strategyRollingUpdateMaxSurge := int32(0)
	strategyRollingUpdateMaxUnavailable := int32(0)

	if deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			strategyRollingUpdateMaxSurge = deployment.Spec.Strategy.RollingUpdate.MaxSurge.IntVal
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			strategyRollingUpdateMaxUnavailable = deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal
		}
	}

	// Get desired replicas
	desiredReplicas := int32(1) // Default to 1 when spec.replicas is nil
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}

	// Get current replicas
	currentReplicas := deployment.Status.Replicas
	readyReplicas := deployment.Status.ReadyReplicas
	availableReplicas := deployment.Status.AvailableReplicas
	unavailableReplicas := deployment.Status.UnavailableReplicas
	updatedReplicas := deployment.Status.UpdatedReplicas

	// Check conditions
	conditionAvailable := false
	conditionProgressing := false
	conditionReplicaFailure := false

	for _, condition := range deployment.Status.Conditions {
		switch condition.Type {
		case appsv1.DeploymentAvailable:
			conditionAvailable = condition.Status == corev1.ConditionTrue
		case appsv1.DeploymentProgressing:
			conditionProgressing = condition.Status == corev1.ConditionTrue
		case appsv1.DeploymentReplicaFailure:
			conditionReplicaFailure = condition.Status == corev1.ConditionTrue
		}
	}

	observedGeneration := deployment.Status.ObservedGeneration
	createdByKind, createdByName := utils.GetOwnerReferenceInfo(deployment)

	return types.DeploymentData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "deployment",
			Name:             utils.ExtractName(deployment),
			Namespace:        utils.ExtractNamespace(deployment),
			CreatedTimestamp: utils.ExtractCreationTimestamp(deployment),
			Labels:           utils.ExtractLabels(deployment),
			Annotations:      utils.ExtractAnnotations(deployment),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		DesiredReplicas:                     desiredReplicas,
		CurrentReplicas:                     currentReplicas,
		ReadyReplicas:                       readyReplicas,
		AvailableReplicas:                   availableReplicas,
		UnavailableReplicas:                 unavailableReplicas,
		UpdatedReplicas:                     updatedReplicas,
		ObservedGeneration:                  observedGeneration,
		ReplicasDesired:                     desiredReplicas,
		ReplicasAvailable:                   availableReplicas,
		ReplicasUnavailable:                 unavailableReplicas,
		ReplicasUpdated:                     updatedReplicas,
		StrategyType:                        strategyType,
		StrategyRollingUpdateMaxSurge:       strategyRollingUpdateMaxSurge,
		StrategyRollingUpdateMaxUnavailable: strategyRollingUpdateMaxUnavailable,
		ConditionAvailable:                  conditionAvailable,
		ConditionProgressing:                conditionProgressing,
		ConditionReplicaFailure:             conditionReplicaFailure,
		Paused:                              deployment.Spec.Paused,
		MetadataGeneration:                  utils.ExtractGeneration(deployment),
	}

}

// getConditionStatus checks if a condition is true
func (h *DeploymentHandler) getConditionStatus(conditions []appsv1.DeploymentCondition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == appsv1.DeploymentConditionType(conditionType) {
			return condition.Status == "True"
		}
	}
	return false
}
