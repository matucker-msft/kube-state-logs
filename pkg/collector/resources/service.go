package resources

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// ServiceHandler handles collection of service metrics
type ServiceHandler struct {
	client            *kubernetes.Clientset
	serviceInformer   cache.SharedIndexInformer
	endpointsInformer cache.SharedIndexInformer
	logger            interfaces.Logger
}

// NewServiceHandler creates a new ServiceHandler
func NewServiceHandler(client *kubernetes.Clientset) *ServiceHandler {
	return &ServiceHandler{
		client: client,
	}
}

// SetupInformer sets up the service informer
func (h *ServiceHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create service informer
	h.serviceInformer = factory.Core().V1().Services().Informer()

	// Create endpoints informer
	h.endpointsInformer = factory.Core().V1().Endpoints().Informer()

	// Add event handlers (no logging on events)
	h.serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

	// Add event handlers for endpoints (no logging on events)
	h.endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

// Collect gathers service metrics from the cluster (uses cache)
func (h *ServiceHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	services := safeGetStoreList(h.serviceInformer)

	for _, obj := range services {
		service, ok := obj.(*corev1.Service)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 && !slices.Contains(namespaces, service.Namespace) {
			continue
		}

		entry := h.createLogEntry(service)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a service
func (h *ServiceHandler) createLogEntry(service *corev1.Service) types.LogEntry {
	// Convert ports
	var ports []types.ServicePortData
	for _, port := range service.Spec.Ports {
		ports = append(ports, types.ServicePortData{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort.IntVal,
			NodePort:   port.NodePort,
		})
	}

	// Get external IP
	var externalIP string
	if len(service.Spec.ExternalIPs) > 0 {
		externalIP = service.Spec.ExternalIPs[0]
	}

	// Get load balancer info
	var loadBalancerIngress []types.LoadBalancerIngressData
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			loadBalancerIngress = append(loadBalancerIngress, types.LoadBalancerIngressData{
				IP:       ingress.IP,
				Hostname: ingress.Hostname,
			})
		}
	}

	// Count endpoints for this service
	endpointsCount := h.countEndpointsForService(service.Namespace, service.Name)

	// Get created by info
	createdByKind := ""
	createdByName := ""
	if len(service.OwnerReferences) > 0 {
		createdByKind = service.OwnerReferences[0].Kind
		createdByName = service.OwnerReferences[0].Name
	}

	// Get traffic policies
	internalTrafficPolicy := ""
	if service.Spec.InternalTrafficPolicy != nil {
		internalTrafficPolicy = string(*service.Spec.InternalTrafficPolicy)
	}
	externalTrafficPolicy := string(service.Spec.ExternalTrafficPolicy)

	// Get session affinity timeout
	sessionAffinityTimeout := int32(0)
	if service.Spec.SessionAffinityConfig != nil && service.Spec.SessionAffinityConfig.ClientIP != nil && service.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds != nil {
		sessionAffinityTimeout = *service.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds
	}

	data := types.ServiceData{
		Type:                                  string(service.Spec.Type),
		ClusterIP:                             service.Spec.ClusterIP,
		ExternalIP:                            externalIP,
		LoadBalancerIP:                        service.Spec.LoadBalancerIP,
		Ports:                                 ports,
		Selector:                              service.Spec.Selector,
		Labels:                                service.Labels,
		Annotations:                           service.Annotations,
		EndpointsCount:                        endpointsCount,
		LoadBalancerIngress:                   loadBalancerIngress,
		SessionAffinity:                       string(service.Spec.SessionAffinity),
		ExternalName:                          service.Spec.ExternalName,
		CreatedByKind:                         createdByKind,
		CreatedByName:                         createdByName,
		CreatedTimestamp:                      service.CreationTimestamp.Unix(),
		InternalTrafficPolicy:                 internalTrafficPolicy,
		ExternalTrafficPolicy:                 externalTrafficPolicy,
		SessionAffinityClientIPTimeoutSeconds: sessionAffinityTimeout,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "service",
		Name:         service.Name,
		Namespace:    service.Namespace,
		Data:         h.convertToMap(data),
	}
}

// countEndpointsForService counts the number of endpoints for a given service
func (h *ServiceHandler) countEndpointsForService(namespace, serviceName string) int {
	endpoints := h.endpointsInformer.GetStore().List()

	for _, obj := range endpoints {
		endpoint, ok := obj.(*corev1.Endpoints)
		if !ok {
			continue
		}

		// Check if this endpoint matches the service
		if endpoint.Namespace == namespace && endpoint.Name == serviceName {
			// Count all addresses across all subsets
			totalAddresses := 0
			for _, subset := range endpoint.Subsets {
				totalAddresses += len(subset.Addresses)
			}
			return totalAddresses
		}
	}

	return 0
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *ServiceHandler) convertToMap(data any) map[string]any {
	return convertStructToMap(data)
}
