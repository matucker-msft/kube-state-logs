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
func NewContainerHandler(client *kubernetes.Clientset) *ContainerHandler {
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
func (h *ContainerHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all pods from the cache
	pods := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, pod.Namespace) {
			continue
		}

		// Create separate log entries for each container
		containerEntries := h.createContainerLogEntries(pod)
		entries = append(entries, containerEntries...)
	}

	return entries, nil
}

// createContainerLogEntries creates LogEntry for each container in a pod
func (h *ContainerHandler) createContainerLogEntries(pod *corev1.Pod) []types.LogEntry {
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
func (h *ContainerHandler) createContainerLogEntry(pod *corev1.Pod, containerSpec *corev1.Container, isInit bool) types.LogEntry {
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

	// Extract resource requests and limits
	resourceRequests := utils.ExtractResourceMap(containerSpec.Resources.Requests)
	resourceLimits := utils.ExtractResourceMap(containerSpec.Resources.Limits)

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

	return utils.CreateLogEntry("container", utils.ExtractName(pod)+"-"+containerSpec.Name, utils.ExtractNamespace(pod), data)
}
