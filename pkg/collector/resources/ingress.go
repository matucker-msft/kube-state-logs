package resources

import (
	"context"
	"slices"
	"strconv"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// IngressHandler handles collection of ingress metrics
type IngressHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewIngressHandler creates a new IngressHandler
func NewIngressHandler(client *kubernetes.Clientset) *IngressHandler {
	return &IngressHandler{
		client: client,
	}
}

// SetupInformer sets up the ingress informer
func (h *IngressHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create ingress informer
	h.informer = factory.Networking().V1().Ingresses().Informer()

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

// Collect gathers ingress metrics from the cluster (uses cache)
func (h *IngressHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all ingresses from the cache
	ingresses := h.informer.GetStore().List()

	for _, obj := range ingresses {
		ingress, ok := obj.(*networkingv1.Ingress)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, ingress.Namespace) {
			continue
		}

		entry := h.createLogEntry(ingress)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from an ingress
func (h *IngressHandler) createLogEntry(ingress *networkingv1.Ingress) types.LogEntry {
	// Extract ingress class name
	var ingressClassName *string
	if ingress.Spec.IngressClassName != nil {
		ingressClassName = ingress.Spec.IngressClassName
	}

	// Extract load balancer ingress
	var loadBalancerIngress []types.LoadBalancerIngressData
	for _, lb := range ingress.Status.LoadBalancer.Ingress {
		loadBalancerIngress = append(loadBalancerIngress, types.LoadBalancerIngressData{
			IP:       lb.IP,
			Hostname: lb.Hostname,
		})
	}

	// Extract rules
	var rules []types.IngressRuleData
	for _, rule := range ingress.Spec.Rules {
		var paths []types.IngressPathData
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				pathType := ""
				if path.PathType != nil {
					pathType = string(*path.PathType)
				}

				port := ""
				if path.Backend.Service != nil {
					port = strconv.FormatInt(int64(path.Backend.Service.Port.Number), 10)
				}

				paths = append(paths, types.IngressPathData{
					Path:     path.Path,
					PathType: pathType,
					Service:  path.Backend.Service.Name,
					Port:     port,
				})
			}
		}

		rules = append(rules, types.IngressRuleData{
			Host:  rule.Host,
			Paths: paths,
		})
	}

	// Extract TLS configuration
	var tls []types.IngressTLSData
	for _, tlsConfig := range ingress.Spec.TLS {
		tls = append(tls, types.IngressTLSData{
			Hosts:      tlsConfig.Hosts,
			SecretName: tlsConfig.SecretName,
		})
	}

	// Determine conditions
	conditionLoadBalancerReady := false
	// Note: Ingress doesn't have conditions in the same way as other resources
	// We'll check if load balancer ingress is available as a proxy for readiness
	conditionLoadBalancerReady = len(ingress.Status.LoadBalancer.Ingress) > 0

	// Create data structure
	data := types.IngressData{
		CreatedTimestamp:           ingress.CreationTimestamp.Unix(),
		Labels:                     ingress.Labels,
		Annotations:                ingress.Annotations,
		IngressClassName:           ingressClassName,
		LoadBalancerIP:             "",
		LoadBalancerIngress:        loadBalancerIngress,
		Rules:                      rules,
		TLS:                        tls,
		ConditionLoadBalancerReady: conditionLoadBalancerReady,
		CreatedByKind:              "",
		CreatedByName:              "",
	}

	// Convert to map[string]any for the LogEntry
	dataMap := map[string]any{
		"createdTimestamp":           data.CreatedTimestamp,
		"labels":                     data.Labels,
		"annotations":                data.Annotations,
		"ingressClassName":           data.IngressClassName,
		"loadBalancerIP":             data.LoadBalancerIP,
		"loadBalancerIngress":        data.LoadBalancerIngress,
		"rules":                      data.Rules,
		"tls":                        data.TLS,
		"conditionLoadBalancerReady": data.ConditionLoadBalancerReady,
		"createdByKind":              data.CreatedByKind,
		"createdByName":              data.CreatedByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "ingress",
		Name:         ingress.Name,
		Namespace:    ingress.Namespace,
		Data:         dataMap,
	}
}
