package resources

import (
	"context"
	"slices"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// DeploymentHandler handles collection of deployment metrics
type DeploymentHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(client *kubernetes.Clientset) *DeploymentHandler {
	return &DeploymentHandler{
		client: client,
	}
}

// SetupInformer sets up the deployment informer
func (h *DeploymentHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create deployment informer with resync period
	h.informer = factory.Apps().V1().Deployments().Informer()

	// Add event handler that logs on resync (periodic full state)
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

// Collect gathers deployment metrics from the cluster (uses cache)
func (h *DeploymentHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	deployments := safeGetStoreList(h.informer)

	for _, obj := range deployments {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, deployment.Namespace) {
			continue
		}

		entry := h.createLogEntry(deployment)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a deployment
func (h *DeploymentHandler) createLogEntry(deployment *appsv1.Deployment) types.LogEntry {
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

	// Get desired replicas with nil check
	desiredReplicas := int32(1) // Default value per Kubernetes API
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(deployment)

	// Get status fields with nil checks
	currentReplicas := deployment.Status.Replicas
	readyReplicas := deployment.Status.ReadyReplicas
	availableReplicas := deployment.Status.AvailableReplicas
	unavailableReplicas := deployment.Status.UnavailableReplicas
	updatedReplicas := deployment.Status.UpdatedReplicas
	observedGeneration := deployment.Status.ObservedGeneration

	data := types.DeploymentData{
		CreatedTimestamp:                    deployment.CreationTimestamp.Unix(),
		Labels:                              deployment.Labels,
		Annotations:                         deployment.Annotations,
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
		ConditionAvailable:                  h.getConditionStatus(deployment.Status.Conditions, "Available"),
		ConditionProgressing:                h.getConditionStatus(deployment.Status.Conditions, "Progressing"),
		ConditionReplicaFailure:             h.getConditionStatus(deployment.Status.Conditions, "ReplicaFailure"),
		CreatedByKind:                       createdByKind,
		CreatedByName:                       createdByName,
		Paused:                              deployment.Spec.Paused,
		MetadataGeneration:                  deployment.ObjectMeta.Generation,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "deployment",
		Name:         deployment.Name,
		Namespace:    deployment.Namespace,
		Data:         h.convertToMap(data),
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

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *DeploymentHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
