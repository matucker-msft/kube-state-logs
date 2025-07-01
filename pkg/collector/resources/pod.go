package resources

import (
	"context"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
)

// PodHandler handles collection of pod metrics
type PodHandler struct {
	utils.BaseHandler
}

// NewPodHandler creates a new PodHandler
func NewPodHandler(client kubernetes.Interface) *PodHandler {
	return &PodHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the pod informer
func (h *PodHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create pod informer
	informer := factory.Core().V1().Pods().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers pod metrics from the cluster (uses cache)
func (h *PodHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all pods from the cache
	pods := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pod.Namespace) {
			continue
		}

		entry := h.createLogEntry(pod)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a PodData from a pod
func (h *PodHandler) createLogEntry(pod *corev1.Pod) types.PodData {
	// Determine QoS class
	qosClass := string(pod.Status.QOSClass)
	if qosClass == "" {
		qosClass = "BestEffort" // Default QoS class when not set
		// See: https://kubernetes.io/docs/concepts/workloads/pods/pod-qos/#qos-classes
	}

	// Get priority class
	priorityClass := ""
	if pod.Spec.PriorityClassName != "" {
		priorityClass = pod.Spec.PriorityClassName
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(pod)

	// Check pod conditions
	ready := false
	initialized := false
	scheduled := false
	containersReady := false
	podScheduled := false

	for _, condition := range pod.Status.Conditions {
		switch condition.Type {
		case corev1.PodReady:
			ready = condition.Status == corev1.ConditionTrue
		case corev1.PodInitialized:
			initialized = condition.Status == corev1.ConditionTrue
		case corev1.PodScheduled:
			scheduled = condition.Status == corev1.ConditionTrue
		case corev1.ContainersReady:
			containersReady = condition.Status == corev1.ConditionTrue
		}
	}
	podScheduled = scheduled

	// Calculate total restart count
	var totalRestartCount int32
	for _, container := range pod.Status.ContainerStatuses {
		totalRestartCount += container.RestartCount
	}

	// Get timestamps
	var deletionTimestamp, startTime, initializedTime, readyTime, scheduledTime *time.Time
	if pod.DeletionTimestamp != nil {
		deletionTimestamp = &pod.DeletionTimestamp.Time
	}
	if pod.Status.StartTime != nil && !pod.Status.StartTime.IsZero() {
		startTime = &pod.Status.StartTime.Time
	}

	// Get condition timestamps
	for _, condition := range pod.Status.Conditions {
		switch condition.Type {
		case corev1.PodInitialized:
			if condition.Status == corev1.ConditionTrue && !condition.LastTransitionTime.IsZero() {
				initializedTime = &condition.LastTransitionTime.Time
			}
		case corev1.PodReady:
			if condition.Status == corev1.ConditionTrue && !condition.LastTransitionTime.IsZero() {
				readyTime = &condition.LastTransitionTime.Time
			}
		case corev1.PodScheduled:
			if condition.Status == corev1.ConditionTrue && !condition.LastTransitionTime.IsZero() {
				scheduledTime = &condition.LastTransitionTime.Time
			}
		}
	}

	// Get status reason - match kube-state-metrics logic
	// See: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase
	statusReason := ""
	if pod.Status.Reason != "" {
		statusReason = pod.Status.Reason
	} else {
		// Check conditions for reason
		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionFalse && condition.Reason != "" {
				statusReason = condition.Reason
				break
			}
		}
		// Check container statuses for terminated reasons
		if statusReason == "" {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
					statusReason = string(cs.State.Terminated.Reason)
					break
				}
			}
		}
	}

	// Get unschedulable status
	unschedulable := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			unschedulable = true
			break
		}
	}

	// Get pod IPs
	var podIPs []string
	if pod.Status.PodIP != "" {
		podIPs = append(podIPs, pod.Status.PodIP)
	}
	for _, ip := range pod.Status.PodIPs {
		if ip.IP != "" {
			podIPs = append(podIPs, ip.IP)
		}
	}

	// Get tolerations
	var tolerations []types.TolerationData
	for _, toleration := range pod.Spec.Tolerations {
		tolerationData := types.TolerationData{
			Key:      toleration.Key,
			Value:    toleration.Value,
			Effect:   string(toleration.Effect),
			Operator: string(toleration.Operator),
		}

		// Add toleration seconds if present
		if toleration.TolerationSeconds != nil {
			tolerationData.TolerationSeconds = strconv.FormatInt(*toleration.TolerationSeconds, 10)
		}

		tolerations = append(tolerations, tolerationData)
	}

	// Get PVC info
	var pvcs []types.PVCData
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			readOnly := false
			for _, mount := range pod.Spec.Containers {
				for _, volumeMount := range mount.VolumeMounts {
					if volumeMount.Name == volume.Name && volumeMount.ReadOnly {
						readOnly = true
						break
					}
				}
			}
			pvcs = append(pvcs, types.PVCData{
				ClaimName: volume.PersistentVolumeClaim.ClaimName,
				ReadOnly:  readOnly,
			})
		}
	}

	// Get overhead
	overheadCPUCores := ""
	overheadMemoryBytes := ""
	if pod.Spec.Overhead != nil {
		if cpu := pod.Spec.Overhead[corev1.ResourceCPU]; !cpu.IsZero() {
			overheadCPUCores = cpu.String()
		}
		if memory := pod.Spec.Overhead[corev1.ResourceMemory]; !memory.IsZero() {
			overheadMemoryBytes = memory.String()
		}
	}

	// Get runtime class name
	runtimeClassName := ""
	if pod.Spec.RuntimeClassName != nil {
		runtimeClassName = *pod.Spec.RuntimeClassName
	}

	// Get completion time (when pod phase is Succeeded)
	var completionTime *time.Time
	if pod.Status.Phase == corev1.PodSucceeded && pod.Status.StartTime != nil && !pod.Status.StartTime.IsZero() {
		completionTime = &pod.Status.StartTime.Time
	}

	// Aggregate pod-level resource requests and limits
	resourceRequests := make(map[string]string)
	resourceLimits := make(map[string]string)

	for _, container := range pod.Spec.Containers {
		for key, value := range container.Resources.Requests {
			resourceKey := string(key)
			if existing, exists := resourceRequests[resourceKey]; exists {
				// Parse existing value and add to it
				existingResource, err := resource.ParseQuantity(existing)
				if err == nil {
					// Add the resources properly
					existingResource.Add(value)
					resourceRequests[resourceKey] = existingResource.String()
				} else {
					// Fallback to string concatenation if parsing fails
					resourceRequests[resourceKey] = existing + " + " + value.String()
				}
			} else {
				resourceRequests[resourceKey] = value.String()
			}
		}
		for key, value := range container.Resources.Limits {
			resourceKey := string(key)
			if existing, exists := resourceLimits[resourceKey]; exists {
				// Parse existing value and add to it
				existingResource, err := resource.ParseQuantity(existing)
				if err == nil {
					// Add the resources properly
					existingResource.Add(value)
					resourceLimits[resourceKey] = existingResource.String()
				} else {
					// Fallback to string concatenation if parsing fails
					resourceLimits[resourceKey] = existing + " + " + value.String()
				}
			} else {
				resourceLimits[resourceKey] = value.String()
			}
		}
	}

	data := types.PodData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "pod",
			Name:             utils.ExtractName(pod),
			Namespace:        utils.ExtractNamespace(pod),
			CreatedTimestamp: utils.ExtractCreationTimestamp(pod),
			Labels:           utils.ExtractLabels(pod),
			Annotations:      utils.ExtractAnnotations(pod),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		NodeName:               pod.Spec.NodeName,
		HostIP:                 pod.Status.HostIP,
		PodIP:                  pod.Status.PodIP,
		Phase:                  string(pod.Status.Phase),
		QoSClass:               qosClass,
		PriorityClass:          priorityClass,
		Ready:                  ready,
		Initialized:            initialized,
		Scheduled:              scheduled,
		ContainersReady:        containersReady,
		PodScheduled:           podScheduled,
		RestartCount:           totalRestartCount,
		DeletionTimestamp:      deletionTimestamp,
		StartTime:              startTime,
		InitializedTime:        initializedTime,
		ReadyTime:              readyTime,
		ScheduledTime:          scheduledTime,
		StatusReason:           statusReason,
		Unschedulable:          unschedulable,
		RestartPolicy:          string(pod.Spec.RestartPolicy),
		ServiceAccount:         pod.Spec.ServiceAccountName,
		SchedulerName:          pod.Spec.SchedulerName,
		OverheadCPUCores:       overheadCPUCores,
		OverheadMemoryBytes:    overheadMemoryBytes,
		RuntimeClassName:       runtimeClassName,
		PodIPs:                 podIPs,
		Tolerations:            tolerations,
		NodeSelectors:          pod.Spec.NodeSelector,
		PersistentVolumeClaims: pvcs,
		CompletionTime:         completionTime,
		ResourceLimits:         resourceLimits,
		ResourceRequests:       resourceRequests,
	}

	return data
}
