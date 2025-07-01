package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// ContainerHandler handles collection of container metrics
type ContainerHandler struct {
	utils.BaseHandler
}

// NewContainerHandler creates a new ContainerHandler
func NewContainerHandler(client kubernetes.Interface) *ContainerHandler {
	return &ContainerHandler{
		BaseHandler: utils.NewBaseHandler(client),
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

		// Create entries for each container
		for _, container := range pod.Status.ContainerStatuses {
			entry := h.createLogEntry(pod, &container, false)
			entry.Timestamp = listTime
			entries = append(entries, entry)
		}

		// Create entries for each init container
		for _, container := range pod.Status.InitContainerStatuses {
			entry := h.createLogEntry(pod, &container, true)
			entry.Timestamp = listTime
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// createLogEntry creates a ContainerData from a pod and container status
func (h *ContainerHandler) createLogEntry(pod *corev1.Pod, container *corev1.ContainerStatus, isInitContainer bool) types.ContainerData {
	// Handle nil container case
	if container == nil {
		return types.ContainerData{
			PodName: pod.Name,
			State:   "unknown",
		}
	}

	// Determine container state
	state := "unknown"
	var stateRunning, stateWaiting, stateTerminated *bool

	var waitingReason, waitingMessage string
	var startedAt, finishedAt, startedAtTerm *time.Time
	var exitCode int32
	var reason, message string
	var lastTerminatedReason string
	var lastTerminatedExitCode int32
	var lastTerminatedTimestamp *time.Time

	if container.State.Running != nil {
		state = "running"
		val := true
		stateRunning = &val
		if !container.State.Running.StartedAt.IsZero() {
			startedAt = &container.State.Running.StartedAt.Time
		}
	} else if container.State.Waiting != nil {
		state = "waiting"
		val := true
		stateWaiting = &val
		waitingReason = string(container.State.Waiting.Reason)
		waitingMessage = container.State.Waiting.Message
	} else if container.State.Terminated != nil {
		state = "terminated"
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
		ResourceType:            "container",
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
