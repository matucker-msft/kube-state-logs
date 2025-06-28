package resources

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
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

	// Convert capacity and allocatable to string maps
	capacity := make(map[string]string)
	allocatable := make(map[string]string)

	for key, value := range node.Status.Capacity {
		capacity[string(key)] = value.String()
	}
	for key, value := range node.Status.Allocatable {
		allocatable[string(key)] = value.String()
	}

	// Check node conditions
	conditions := make(map[string]bool)
	ready := false
	for _, condition := range node.Status.Conditions {
		conditions[string(condition.Type)] = condition.Status == corev1.ConditionTrue
		if condition.Type == corev1.NodeReady {
			ready = condition.Status == corev1.ConditionTrue
		}
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(node.OwnerReferences) > 0 {
		createdByKind = node.OwnerReferences[0].Kind
		createdByName = node.OwnerReferences[0].Name
	}

	// Get node role
	role := ""
	if node.Labels["node-role.kubernetes.io/control-plane"] == "true" || node.Labels["node-role.kubernetes.io/master"] == "true" {
		role = "master"
	} else {
		role = "worker"
	}

	// Get taints
	var taints []types.TaintData
	for _, taint := range node.Spec.Taints {
		taints = append(taints, types.TaintData{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		})
	}

	// Get deletion timestamp
	var deletionTimestamp *time.Time
	if node.DeletionTimestamp != nil {
		deletionTimestamp = &node.DeletionTimestamp.Time
	}

	data := types.NodeData{
		Architecture:            node.Status.NodeInfo.Architecture,
		OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		KubeProxyVersion:        node.Status.NodeInfo.KubeProxyVersion,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
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
		Role:                    role,
		Taints:                  taints,
		DeletionTimestamp:       deletionTimestamp,
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
