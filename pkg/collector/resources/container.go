package resources

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// Container state constants
const (
	ContainerStateRunning    = "running"
	ContainerStateWaiting    = "waiting"
	ContainerStateTerminated = "terminated"
	ContainerStateUnknown    = "unknown"
)

// ContainerHandler handles collection of container metrics
type ContainerHandler struct {
	utils.BaseHandler
	stateCache cache.ThreadSafeStore
}

// NewContainerHandler creates a new ContainerHandler
func NewContainerHandler(client kubernetes.Interface) *ContainerHandler {
	return &ContainerHandler{
		BaseHandler: utils.NewBaseHandler(client),
		stateCache:  cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{}),
	}
}

// SetupInformer sets up the pod informer (containers are accessed through pods)
func (h *ContainerHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create pod informer (containers are accessed through pods)
	informer := factory.Core().V1().Pods().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers container metrics from the cluster (uses cache)
func (h *ContainerHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	// Get all pods from the cache
	pods := utils.SafeGetStoreList(h.GetInformer())
	return h.processPods(pods, namespaces)
}

// processPods processes a list of pods and returns container entries
func (h *ContainerHandler) processPods(pods []any, namespaces []string) ([]any, error) {
	var entries []any
	currentStates := make(map[string]string)
	listTime := time.Now()

	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pod.Namespace) {
			continue
		}

		// Process regular containers
		for _, container := range pod.Status.ContainerStatuses {
			containerKey := h.getContainerKey(pod.Namespace, pod.Name, container.Name, false)
			currentState := h.getContainerState(&container)
			currentStates[containerKey] = currentState

			// Always log running containers
			if currentState == ContainerStateRunning {
				entry := h.createLogEntry(pod, &container, false)
				entry.Timestamp = listTime
				entries = append(entries, entry)
			}

			// Log newly terminated containers
			if h.isNewlyTerminated(containerKey, currentState, &container) {
				entry := h.createLogEntry(pod, &container, false)
				entry.Timestamp = listTime
				entries = append(entries, entry)
			}
		}

		// Process init containers
		for _, container := range pod.Status.InitContainerStatuses {
			containerKey := h.getContainerKey(pod.Namespace, pod.Name, container.Name, true)
			currentState := h.getContainerState(&container)
			currentStates[containerKey] = currentState

			// Always log running init containers
			if currentState == ContainerStateRunning {
				entry := h.createLogEntry(pod, &container, true)
				entry.Timestamp = listTime
				entries = append(entries, entry)
			}

			// Log newly terminated init containers
			if h.isNewlyTerminated(containerKey, currentState, &container) {
				entry := h.createLogEntry(pod, &container, true)
				entry.Timestamp = listTime
				entries = append(entries, entry)
			}
		}
	}

	// Update state cache and cleanup deleted containers
	h.updateStateCache(currentStates)
	return entries, nil
}

// getContainerKey creates a unique key for a container
func (h *ContainerHandler) getContainerKey(namespace, podName, containerName string, isInit bool) string {
	prefix := h.getResourceType(isInit)
	return fmt.Sprintf("%s/%s/%s/%s", namespace, podName, containerName, prefix)
}

// getContainerState determines the current state of a container
func (h *ContainerHandler) getContainerState(container *corev1.ContainerStatus) string {
	if container.State.Running != nil {
		return ContainerStateRunning
	} else if container.State.Waiting != nil {
		return ContainerStateWaiting
	} else if container.State.Terminated != nil {
		return ContainerStateTerminated
	}
	return ContainerStateUnknown
}

// getResourceType determines the resource type based on container type
func (h *ContainerHandler) getResourceType(isInitContainer bool) string {
	if isInitContainer {
		return "init_container"
	}
	return "container"
}

// isNewlyTerminated checks if a container should be logged as terminated
func (h *ContainerHandler) isNewlyTerminated(containerKey, currentState string, container *corev1.ContainerStatus) bool {
	if currentState != ContainerStateTerminated {
		return false
	}

	// Check if container terminated within the last hour
	if container != nil && container.State.Terminated != nil {
		if container.State.Terminated.FinishedAt.IsZero() {
			return false // No finish time, skip
		}

		// Only log if terminated within the last hour
		oneHourAgo := time.Now().Add(-1 * time.Hour)
		if container.State.Terminated.FinishedAt.Time.Before(oneHourAgo) {
			return false // Too old, skip
		}
	}

	// Get previous state from cache
	if previousStateObj, exists := h.stateCache.Get(containerKey); exists {
		previousState := previousStateObj.(string)
		// Log if it transitioned from running to terminated
		return previousState == ContainerStateRunning
	}

	// Log if we haven't seen this container before (first time seeing a terminated container)
	return true
}

// updateStateCache updates the state cache with current states and cleans up deleted containers
func (h *ContainerHandler) updateStateCache(currentStates map[string]string) {
	// Get all keys in cache
	cacheKeys := h.stateCache.ListKeys()

	// Remove containers that no longer exist
	for _, key := range cacheKeys {
		if _, exists := currentStates[key]; !exists {
			h.stateCache.Delete(key)
		}
	}

	// Update with current states
	for key, state := range currentStates {
		h.stateCache.Add(key, state)
	}
}

// createLogEntry creates a ContainerData from a pod and container status
func (h *ContainerHandler) createLogEntry(pod *corev1.Pod, container *corev1.ContainerStatus, isInitContainer bool) types.ContainerData {
	// Handle nil container case
	if container == nil {
		return types.ContainerData{
			ResourceType: h.getResourceType(isInitContainer),
			Timestamp:    time.Now(),
			PodName:      pod.Name,
			Namespace:    pod.Namespace,
			State:        ContainerStateUnknown,
		}
	}

	// Determine container state
	state := ContainerStateUnknown
	var stateRunning, stateWaiting, stateTerminated *bool

	var waitingReason, waitingMessage string
	var startedAt, finishedAt, startedAtTerm *time.Time
	var exitCode int32
	var reason, message string
	var lastTerminatedReason string
	var lastTerminatedExitCode int32
	var lastTerminatedTimestamp *time.Time

	if container.State.Running != nil {
		state = ContainerStateRunning
		val := true
		stateRunning = &val
		if !container.State.Running.StartedAt.IsZero() {
			startedAt = &container.State.Running.StartedAt.Time
		}
	} else if container.State.Waiting != nil {
		state = ContainerStateWaiting
		val := true
		stateWaiting = &val
		waitingReason = string(container.State.Waiting.Reason)
		waitingMessage = container.State.Waiting.Message
	} else if container.State.Terminated != nil {
		state = ContainerStateTerminated
		val := true
		stateTerminated = &val
		exitCode = container.State.Terminated.ExitCode
		reason = string(container.State.Terminated.Reason)
		message = container.State.Terminated.Message
		if !container.State.Terminated.FinishedAt.IsZero() {
			finishedAt = &container.State.Terminated.FinishedAt.Time
		}
		if !container.State.Terminated.StartedAt.IsZero() {
			startedAtTerm = &container.State.Terminated.StartedAt.Time
		}
	}

	// Get last terminated state
	if container.LastTerminationState.Terminated != nil {
		lastTerminatedReason = string(container.LastTerminationState.Terminated.Reason)
		lastTerminatedExitCode = container.LastTerminationState.Terminated.ExitCode
		if !container.LastTerminationState.Terminated.FinishedAt.IsZero() {
			lastTerminatedTimestamp = &container.LastTerminationState.Terminated.FinishedAt.Time
		}
	}

	// Extract resource requests and limits from pod spec
	var resourceRequests, resourceLimits map[string]string
	if isInitContainer {
		// Look in init containers for init container resources
		for _, containerSpec := range pod.Spec.InitContainers {
			if containerSpec.Name == container.Name {
				resourceRequests = utils.ExtractResourceMap(containerSpec.Resources.Requests)
				resourceLimits = utils.ExtractResourceMap(containerSpec.Resources.Limits)
				break
			}
		}
	} else {
		// Look in regular containers for regular container resources
		for _, containerSpec := range pod.Spec.Containers {
			if containerSpec.Name == container.Name {
				resourceRequests = utils.ExtractResourceMap(containerSpec.Resources.Requests)
				resourceLimits = utils.ExtractResourceMap(containerSpec.Resources.Limits)
				break
			}
		}
	}

	imageID := container.ImageID

	// Get state started time (when container first started)
	var stateStarted *time.Time
	if container.State.Running != nil && !container.State.Running.StartedAt.IsZero() {
		stateStarted = &container.State.Running.StartedAt.Time
	}

	data := types.ContainerData{
		ResourceType:            h.getResourceType(isInitContainer),
		Timestamp:               time.Now(),
		Name:                    container.Name,
		Image:                   container.Image,
		ImageID:                 imageID,
		PodName:                 pod.Name,
		Namespace:               pod.Namespace,
		Ready:                   &container.Ready,
		RestartCount:            container.RestartCount,
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

	return data
}
