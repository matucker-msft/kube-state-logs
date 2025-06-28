package resources

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// NodeHandler handles collection of node metrics
type NodeHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(client *kubernetes.Clientset) *NodeHandler {
	return &NodeHandler{
		client: client,
	}
}

// SetupInformer sets up the node informer
func (h *NodeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create node informer
	h.informer = factory.Core().V1().Nodes().Informer()

	// Add event handlers (no logging on events)
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

// Collect gathers node metrics from the cluster (uses cache)
func (h *NodeHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	nodes := safeGetStoreList(h.informer)

	for _, obj := range nodes {
		node, ok := obj.(*corev1.Node)
		if !ok {
			continue
		}

		entry := h.createLogEntry(node)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a node
func (h *NodeHandler) createLogEntry(node *corev1.Node) types.LogEntry {
	// Get node addresses
	var internalIP, externalIP, hostname string
	if node.Status.Addresses != nil {
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				internalIP = addr.Address
			case corev1.NodeExternalIP:
				externalIP = addr.Address
			case corev1.NodeHostName:
				hostname = addr.Address
			}
		}
	}

	// Convert capacity and allocatable to string maps
	capacity := make(map[string]string)
	allocatable := make(map[string]string)

	if node.Status.Capacity != nil {
		for key, value := range node.Status.Capacity {
			capacity[string(key)] = value.String()
		}
	}
	if node.Status.Allocatable != nil {
		for key, value := range node.Status.Allocatable {
			allocatable[string(key)] = value.String()
		}
	}

	// Check node conditions
	conditions := make(map[string]bool)
	ready := false
	if node.Status.Conditions != nil {
		for _, condition := range node.Status.Conditions {
			conditions[string(condition.Type)] = condition.Status == corev1.ConditionTrue
			if condition.Type == corev1.NodeReady {
				ready = condition.Status == corev1.ConditionTrue
			}
		}
	}

	// Determine node phase
	// See: https://kubernetes.io/docs/concepts/architecture/nodes/#node-status
	phase := "Unknown"
	if node.Status.Phase != "" {
		phase = string(node.Status.Phase)
	}

	// Get node role
	nodeRole := ""
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if role != "" {
				nodeRole = role
				break
			}
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(node)

	// Get taints
	var taints []types.TaintData
	if node.Spec.Taints != nil {
		for _, taint := range node.Spec.Taints {
			taints = append(taints, types.TaintData{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			})
		}
	}

	// Get deletion timestamp
	var deletionTimestamp *time.Time
	if node.DeletionTimestamp != nil {
		deletionTimestamp = &node.DeletionTimestamp.Time
	}

	// Get node info with nil checks
	architecture := ""
	operatingSystem := ""
	kernelVersion := ""
	kubeletVersion := ""
	kubeProxyVersion := ""
	containerRuntimeVersion := ""

	// NodeSystemInfo is a struct, not a pointer, so we can access it directly
	architecture = node.Status.NodeInfo.Architecture
	operatingSystem = node.Status.NodeInfo.OperatingSystem
	kernelVersion = node.Status.NodeInfo.KernelVersion
	kubeletVersion = node.Status.NodeInfo.KubeletVersion
	kubeProxyVersion = node.Status.NodeInfo.KubeProxyVersion
	containerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion

	data := types.NodeData{
		Architecture:            architecture,
		OperatingSystem:         operatingSystem,
		KernelVersion:           kernelVersion,
		KubeletVersion:          kubeletVersion,
		KubeProxyVersion:        kubeProxyVersion,
		ContainerRuntimeVersion: containerRuntimeVersion,
		Capacity:                capacity,
		Allocatable:             allocatable,
		Conditions:              conditions,
		Labels:                  node.Labels,
		Annotations:             node.Annotations,
		InternalIP:              internalIP,
		ExternalIP:              externalIP,
		Hostname:                hostname,
		Unschedulable:           node.Spec.Unschedulable,
		Ready:                   ready,
		CreatedByKind:           createdByKind,
		CreatedByName:           createdByName,
		CreatedTimestamp:        node.CreationTimestamp.Unix(),
		Role:                    nodeRole,
		Taints:                  taints,
		DeletionTimestamp:       deletionTimestamp,
		Phase:                   phase,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "node",
		Name:         node.Name,
		Namespace:    "", // Nodes don't have namespaces
		Data:         h.convertToMap(data),
	}
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *NodeHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
