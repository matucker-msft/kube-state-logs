package resources

import (
	"context"
	"slices"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// NetworkPolicyHandler handles collection of networkpolicy metrics
type NetworkPolicyHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewNetworkPolicyHandler creates a new NetworkPolicyHandler
func NewNetworkPolicyHandler(client *kubernetes.Clientset) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{
		client: client,
	}
}

// SetupInformer sets up the networkpolicy informer
func (h *NetworkPolicyHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create networkpolicy informer
	h.informer = factory.Networking().V1().NetworkPolicies().Informer()

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

// Collect gathers networkpolicy metrics from the cluster (uses cache)
func (h *NetworkPolicyHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all networkpolicies from the cache
	npList := safeGetStoreList(h.informer)

	for _, obj := range npList {
		np, ok := obj.(*networkingv1.NetworkPolicy)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, np.Namespace) {
			continue
		}

		entry := h.createLogEntry(np)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a networkpolicy
func (h *NetworkPolicyHandler) createLogEntry(np *networkingv1.NetworkPolicy) types.LogEntry {
	// Get policy types
	policyTypes := make([]string, len(np.Spec.PolicyTypes))
	for i, policyType := range np.Spec.PolicyTypes {
		policyTypes[i] = string(policyType)
	}

	// Convert ingress rules
	var ingressRules []types.NetworkPolicyIngressRule
	for _, rule := range np.Spec.Ingress {
		ingressRule := types.NetworkPolicyIngressRule{
			Ports: h.convertPorts(rule.Ports),
			From:  h.convertPeers(rule.From),
		}
		ingressRules = append(ingressRules, ingressRule)
	}

	// Convert egress rules
	var egressRules []types.NetworkPolicyEgressRule
	for _, rule := range np.Spec.Egress {
		egressRule := types.NetworkPolicyEgressRule{
			Ports: h.convertPorts(rule.Ports),
			To:    h.convertPeers(rule.To),
		}
		egressRules = append(egressRules, egressRule)
	}

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(np.OwnerReferences) > 0 {
		createdByKind = np.OwnerReferences[0].Kind
		createdByName = np.OwnerReferences[0].Name
	}

	// Create data structure
	data := types.NetworkPolicyData{
		CreatedTimestamp: np.CreationTimestamp.Unix(),
		Labels:           np.Labels,
		Annotations:      np.Annotations,
		PolicyTypes:      policyTypes,
		IngressRules:     ingressRules,
		EgressRules:      egressRules,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "networkpolicy",
		Name:         np.Name,
		Namespace:    np.Namespace,
		Data:         convertStructToMap(data),
	}
}

// convertPorts converts networkingv1.NetworkPolicyPort to types.NetworkPolicyPort
func (h *NetworkPolicyHandler) convertPorts(ports []networkingv1.NetworkPolicyPort) []types.NetworkPolicyPort {
	var result []types.NetworkPolicyPort
	for _, port := range ports {
		npPort := types.NetworkPolicyPort{}
		if port.Protocol != nil {
			npPort.Protocol = string(*port.Protocol)
		}
		if port.Port != nil {
			npPort.Port = port.Port.IntVal
		}
		if port.EndPort != nil {
			npPort.EndPort = *port.EndPort
		}
		result = append(result, npPort)
	}
	return result
}

// convertPeers converts networkingv1.NetworkPolicyPeer to types.NetworkPolicyPeer
func (h *NetworkPolicyHandler) convertPeers(peers []networkingv1.NetworkPolicyPeer) []types.NetworkPolicyPeer {
	var result []types.NetworkPolicyPeer
	for _, peer := range peers {
		npPeer := types.NetworkPolicyPeer{}

		if peer.PodSelector != nil {
			npPeer.PodSelector = peer.PodSelector.MatchLabels
		}
		if peer.NamespaceSelector != nil {
			npPeer.NamespaceSelector = peer.NamespaceSelector.MatchLabels
		}
		if peer.IPBlock != nil {
			npPeer.IPBlock = map[string]any{
				"cidr":   peer.IPBlock.CIDR,
				"except": peer.IPBlock.Except,
			}
		}
		result = append(result, npPeer)
	}
	return result
}
