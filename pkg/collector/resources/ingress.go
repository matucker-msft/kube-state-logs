package resources

import (
	"context"
	"strconv"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// IngressHandler handles collection of ingress metrics
type IngressHandler struct {
	utils.BaseHandler
}

// NewIngressHandler creates a new IngressHandler
func NewIngressHandler(client kubernetes.Interface) *IngressHandler {
	return &IngressHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the ingress informer
func (h *IngressHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create ingress informer
	informer := factory.Networking().V1().Ingresses().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers ingress metrics from the cluster (uses cache)
func (h *IngressHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all ingresses from the cache
	ingressList := utils.SafeGetStoreList(h.GetInformer())

	for _, obj := range ingressList {
		ingress, ok := obj.(*networkingv1.Ingress)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, ingress.Namespace) {
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
				// Default path type is "ImplementationSpecific" when not specified
				// See: https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types

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

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(ingress)

	// Create data structure
	data := types.IngressData{
		CreatedTimestamp:           utils.ExtractCreationTimestamp(ingress),
		Labels:                     utils.ExtractLabels(ingress),
		Annotations:                utils.ExtractAnnotations(ingress),
		IngressClassName:           ingressClassName,
		LoadBalancerIP:             "",
		LoadBalancerIngress:        loadBalancerIngress,
		Rules:                      rules,
		TLS:                        tls,
		ConditionLoadBalancerReady: conditionLoadBalancerReady,
		CreatedByKind:              createdByKind,
		CreatedByName:              createdByName,
	}

	return utils.CreateLogEntry("ingress", utils.ExtractName(ingress), utils.ExtractNamespace(ingress), data)
}
