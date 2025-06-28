package resources

import (
	"context"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// HorizontalPodAutoscalerHandler handles collection of horizontalpodautoscaler metrics
type HorizontalPodAutoscalerHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewHorizontalPodAutoscalerHandler creates a new HorizontalPodAutoscalerHandler
func NewHorizontalPodAutoscalerHandler(client *kubernetes.Clientset) *HorizontalPodAutoscalerHandler {
	return &HorizontalPodAutoscalerHandler{
		client: client,
	}
}

// SetupInformer sets up the horizontalpodautoscaler informer
func (h *HorizontalPodAutoscalerHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create horizontalpodautoscaler informer
	h.informer = factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer()

	return nil
}

// Collect gathers horizontalpodautoscaler metrics from the cluster (uses cache)
func (h *HorizontalPodAutoscalerHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all horizontalpodautoscalers from the cache
	hpas := safeGetStoreList(h.informer)

	for _, obj := range hpas {
		hpa, ok := obj.(*autoscalingv2.HorizontalPodAutoscaler)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, hpa.Namespace) {
			continue
		}

		entry := h.createLogEntry(hpa)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a horizontalpodautoscaler
func (h *HorizontalPodAutoscalerHandler) createLogEntry(hpa *autoscalingv2.HorizontalPodAutoscaler) types.LogEntry {
	// Extract target CPU and memory utilization
	var targetCPUUtilizationPercentage *int32
	var targetMemoryUtilizationPercentage *int32

	for _, metric := range hpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			if metric.Resource.Name == "cpu" && metric.Resource.Target.Type == autoscalingv2.UtilizationMetricType {
				targetCPUUtilizationPercentage = metric.Resource.Target.AverageUtilization
			}
			if metric.Resource.Name == "memory" && metric.Resource.Target.Type == autoscalingv2.UtilizationMetricType {
				targetMemoryUtilizationPercentage = metric.Resource.Target.AverageUtilization
			}
		}
	}

	// Extract current CPU and memory utilization
	var currentCPUUtilizationPercentage *int32
	var currentMemoryUtilizationPercentage *int32

	for _, metric := range hpa.Status.CurrentMetrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			if metric.Resource.Name == "cpu" {
				currentCPUUtilizationPercentage = metric.Resource.Current.AverageUtilization
			}
			if metric.Resource.Name == "memory" {
				currentMemoryUtilizationPercentage = metric.Resource.Current.AverageUtilization
			}
		}
	}

	// Determine conditions
	conditionAbleToScale := false
	conditionScalingActive := false
	conditionScalingLimited := false

	for _, condition := range hpa.Status.Conditions {
		switch condition.Type {
		case autoscalingv2.AbleToScale:
			conditionAbleToScale = condition.Status == "True"
		case autoscalingv2.ScalingActive:
			conditionScalingActive = condition.Status == "True"
		case autoscalingv2.ScalingLimited:
			conditionScalingLimited = condition.Status == "True"
		}
	}

	// Create data structure
	// Default min replicas is 1 when spec.minReplicas is nil
	// See: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
	minReplicas := int32(1)
	if hpa.Spec.MinReplicas != nil {
		minReplicas = *hpa.Spec.MinReplicas
	}

	data := types.HorizontalPodAutoscalerData{
		CreatedTimestamp:                   hpa.CreationTimestamp.Unix(),
		Labels:                             hpa.Labels,
		Annotations:                        hpa.Annotations,
		MinReplicas:                        &minReplicas,
		MaxReplicas:                        hpa.Spec.MaxReplicas,
		TargetCPUUtilizationPercentage:     targetCPUUtilizationPercentage,
		TargetMemoryUtilizationPercentage:  targetMemoryUtilizationPercentage,
		CurrentReplicas:                    hpa.Status.CurrentReplicas,
		DesiredReplicas:                    hpa.Status.DesiredReplicas,
		CurrentCPUUtilizationPercentage:    currentCPUUtilizationPercentage,
		CurrentMemoryUtilizationPercentage: currentMemoryUtilizationPercentage,
		ConditionAbleToScale:               conditionAbleToScale,
		ConditionScalingActive:             conditionScalingActive,
		ConditionScalingLimited:            conditionScalingLimited,
		CreatedByKind:                      "",
		CreatedByName:                      "",
		ScaleTargetRef:                     hpa.Spec.ScaleTargetRef.Name,
		ScaleTargetKind:                    hpa.Spec.ScaleTargetRef.Kind,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "horizontalpodautoscaler",
		Name:         hpa.Name,
		Namespace:    hpa.Namespace,
		Data:         convertStructToMap(data),
	}
}
