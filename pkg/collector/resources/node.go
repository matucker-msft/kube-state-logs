package resources

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"go.goms.io/aks/kube-state-logs/pkg/interfaces"
	"go.goms.io/aks/kube-state-logs/pkg/types"
	"go.goms.io/aks/kube-state-logs/pkg/utils"
)

// NodeHandler handles collection of node metrics
type NodeHandler struct {
	utils.BaseHandler
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(client kubernetes.Interface) *NodeHandler {
	return &NodeHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the node informer
func (h *NodeHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create node informer
	informer := factory.Core().V1().Nodes().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers node metrics from the cluster (uses cache)
func (h *NodeHandler) Collect(ctx context.Context, namespaces []string) ([]any, error) {
	var entries []any

	// Get all nodes from the cache
	nodes := utils.SafeGetStoreList(h.GetInformer())
	listTime := time.Now()

	for _, obj := range nodes {
		node, ok := obj.(*corev1.Node)
		if !ok {
			continue
		}

		entry := h.createLogEntry(node)
		entry.Timestamp = listTime
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a NodeData from a node
func (h *NodeHandler) createLogEntry(node *corev1.Node) types.NodeData {
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

	// Use resource utils for capacity and allocatable extraction
	capacity := utils.ExtractResourceMap(node.Status.Capacity)
	allocatable := utils.ExtractResourceMap(node.Status.Allocatable)

	// Get node conditions in a single loop
	var ready *bool
	conditions := make(map[string]*bool)
	unschedulable := node.Spec.Unschedulable

	for _, condition := range node.Status.Conditions {
		val := utils.ConvertCoreConditionStatus(condition.Status)

		if condition.Type == corev1.NodeReady {
			ready = val
		} else {
			// Add other conditions to the map
			conditions[string(condition.Type)] = val
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

	data := types.NodeData{
		LogEntryMetadata: types.LogEntryMetadata{
			Timestamp:        time.Now(),
			ResourceType:     "node",
			Name:             utils.ExtractName(node),
			Namespace:        utils.ExtractNamespace(node),
			CreatedTimestamp: utils.ExtractCreationTimestamp(node),
			Labels:           utils.ExtractLabels(node),
			Annotations:      utils.ExtractAnnotations(node),
			CreatedByKind:    createdByKind,
			CreatedByName:    createdByName,
		},
		Architecture:            node.Status.NodeInfo.Architecture,
		OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		KubeProxyVersion:        node.Status.NodeInfo.KubeProxyVersion,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		Capacity:                capacity,
		Allocatable:             allocatable,
		Ready:                   ready,
		Phase:                   phase,

		// All other conditions (excluding the top-level ones)
		Conditions:        conditions,
		InternalIP:        internalIP,
		ExternalIP:        externalIP,
		Hostname:          hostname,
		Unschedulable:     &unschedulable,
		Role:              nodeRole,
		Taints:            taints,
		DeletionTimestamp: utils.ExtractDeletionTimestamp(node),
	}

	return data
}
