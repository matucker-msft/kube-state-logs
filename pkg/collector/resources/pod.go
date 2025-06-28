package resources

import (
	"context"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// PodHandler handles collection of pod metrics
type PodHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewPodHandler creates a new PodHandler
func NewPodHandler(client *kubernetes.Clientset) *PodHandler {
	return &PodHandler{
		client: client,
	}
}

// SetupInformer sets up the pod informer
func (h *PodHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create pod informer
	h.informer = factory.Core().V1().Pods().Informer()

	return nil
}

// Collect gathers pod metrics from the cluster (uses cache)
func (h *PodHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all pods from the cache
	pods := utils.SafeGetStoreList(h.informer)

	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pod.Namespace) {
			continue
		}

		// Create pod log entry
		podEntry := h.createPodLogEntry(pod)
		entries = append(entries, podEntry)

		// Create separate log entries for each container
		containerEntries := h.createContainerLogEntries(pod)
		entries = append(entries, containerEntries...)
	}

	return entries, nil
}

// createPodLogEntry creates a LogEntry from a pod
func (h *PodHandler) createPodLogEntry(pod *corev1.Pod) types.LogEntry {
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
				// Add to existing value (simplified - in reality would need proper resource arithmetic)
				resourceRequests[resourceKey] = existing + " + " + value.String()
			} else {
				resourceRequests[resourceKey] = value.String()
			}
		}
		for key, value := range container.Resources.Limits {
			resourceKey := string(key)
			if existing, exists := resourceLimits[resourceKey]; exists {
				// Add to existing value (simplified - in reality would need proper resource arithmetic)
				resourceLimits[resourceKey] = existing + " + " + value.String()
			} else {
				resourceLimits[resourceKey] = value.String()
			}
		}
	}

	data := types.PodData{
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
		CreatedByKind:          createdByKind,
		CreatedByName:          createdByName,
		Labels:                 pod.Labels,
		Annotations:            pod.Annotations,
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

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "pod",
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Data:         utils.ConvertStructToMap(data),
	}
}

// createContainerLogEntries creates LogEntry for each container in a pod
func (h *PodHandler) createContainerLogEntries(pod *corev1.Pod) []types.LogEntry {
	var entries []types.LogEntry

	// Process all containers (including init containers)
	for _, container := range pod.Spec.Containers {
		entry := h.createContainerLogEntry(pod, &container, false)
		entries = append(entries, entry)
	}

	for _, container := range pod.Spec.InitContainers {
		entry := h.createContainerLogEntry(pod, &container, true)
		entries = append(entries, entry)
	}

	return entries
}

// createContainerLogEntry creates a LogEntry for a specific container
func (h *PodHandler) createContainerLogEntry(pod *corev1.Pod, containerSpec *corev1.Container, isInit bool) types.LogEntry {
	// Find container status
	var containerStatus *corev1.ContainerStatus
	statuses := pod.Status.ContainerStatuses
	if isInit {
		statuses = pod.Status.InitContainerStatuses
	}

	for _, status := range statuses {
		if status.Name == containerSpec.Name {
			containerStatus = &status
			break
		}
	}

	// Determine container state
	state := "unknown"
	stateRunning := false
	stateWaiting := false
	stateTerminated := false
	var waitingReason, waitingMessage string
	var startedAt *time.Time
	var exitCode int32
	var reason, message string
	var finishedAt, startedAtTerm *time.Time

	// Last terminated state info
	var lastTerminatedReason string
	var lastTerminatedExitCode int32
	var lastTerminatedTimestamp *time.Time

	if containerStatus != nil {
		if containerStatus.State.Running != nil {
			state = "running"
			stateRunning = true
			if !containerStatus.State.Running.StartedAt.IsZero() {
				startedAt = &containerStatus.State.Running.StartedAt.Time
			}
		} else if containerStatus.State.Waiting != nil {
			state = "waiting"
			stateWaiting = true
			waitingReason = string(containerStatus.State.Waiting.Reason)
			waitingMessage = containerStatus.State.Waiting.Message
		} else if containerStatus.State.Terminated != nil {
			state = "terminated"
			stateTerminated = true
			exitCode = containerStatus.State.Terminated.ExitCode
			reason = string(containerStatus.State.Terminated.Reason)
			message = containerStatus.State.Terminated.Message
			if !containerStatus.State.Terminated.FinishedAt.IsZero() {
				finishedAt = &containerStatus.State.Terminated.FinishedAt.Time
			}
			if !containerStatus.State.Terminated.StartedAt.IsZero() {
				startedAtTerm = &containerStatus.State.Terminated.StartedAt.Time
			}
		}

		// Get last terminated state
		if containerStatus.LastTerminationState.Terminated != nil {
			lastTerminatedReason = string(containerStatus.LastTerminationState.Terminated.Reason)
			lastTerminatedExitCode = containerStatus.LastTerminationState.Terminated.ExitCode
			if !containerStatus.LastTerminationState.Terminated.FinishedAt.IsZero() {
				lastTerminatedTimestamp = &containerStatus.LastTerminationState.Terminated.FinishedAt.Time
			}
		}
	}

	// Convert resource requests/limits to string maps
	resourceRequests := make(map[string]string)
	resourceLimits := make(map[string]string)

	for key, value := range containerSpec.Resources.Requests {
		resourceRequests[string(key)] = value.String()
	}
	for key, value := range containerSpec.Resources.Limits {
		resourceLimits[string(key)] = value.String()
	}

	imageID := ""
	if containerStatus != nil {
		imageID = containerStatus.ImageID
	}

	// Get state started time (when container first started)
	var stateStarted *time.Time
	if containerStatus != nil && containerStatus.State.Running != nil && !containerStatus.State.Running.StartedAt.IsZero() {
		stateStarted = &containerStatus.State.Running.StartedAt.Time
	}

	data := types.ContainerData{
		Name:    containerSpec.Name,
		Image:   containerSpec.Image,
		ImageID: imageID,
		PodName: pod.Name,
		Ready:   containerStatus != nil && containerStatus.Ready,
		RestartCount: func() int32 {
			if containerStatus != nil {
				return containerStatus.RestartCount
			}
			return 0
		}(),
		State:                   state,
		StateRunning:            stateRunning,
		StateWaiting:            stateWaiting,
		StateTerminated:         stateTerminated,
		WaitingReason:           waitingReason,
		WaitingMessage:          waitingMessage,
		StartedAt:               startedAt,
		ExitCode:                exitCode,
		Reason:                  reason,
		Message:                 message,
		FinishedAt:              finishedAt,
		StartedAtTerm:           startedAtTerm,
		ResourceRequests:        resourceRequests,
		ResourceLimits:          resourceLimits,
		LastTerminatedReason:    lastTerminatedReason,
		LastTerminatedExitCode:  lastTerminatedExitCode,
		LastTerminatedTimestamp: lastTerminatedTimestamp,
		StateStarted:            stateStarted,
	}

	resourceType := "container"
	if isInit {
		resourceType = "init_container"
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: resourceType,
		Name:         containerSpec.Name,
		Namespace:    pod.Namespace,
		Data:         utils.ConvertStructToMap(data),
	}
}
