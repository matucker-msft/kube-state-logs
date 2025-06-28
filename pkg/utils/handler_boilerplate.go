package utils

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// BaseHandler provides common fields and methods for resource handlers
type BaseHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewBaseHandler creates a new BaseHandler
func NewBaseHandler(client *kubernetes.Clientset) BaseHandler {
	return BaseHandler{
		client: client,
	}
}

// SetupBaseInformer sets up the base informer with common configuration
func (h *BaseHandler) SetupBaseInformer(informer cache.SharedIndexInformer, logger interfaces.Logger) {
	h.informer = informer
	h.logger = logger
}

// GetClient returns the Kubernetes client
func (h *BaseHandler) GetClient() *kubernetes.Clientset {
	return h.client
}

// GetInformer returns the informer
func (h *BaseHandler) GetInformer() cache.SharedIndexInformer {
	return h.informer
}

// GetLogger returns the logger
func (h *BaseHandler) GetLogger() interfaces.Logger {
	return h.logger
}

// CreateLogEntry creates a standard LogEntry with common fields
func CreateLogEntry(resourceType, name, namespace string, data any) types.LogEntry {
	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: resourceType,
		Name:         name,
		Namespace:    namespace,
		Data:         ConvertStructToMap(data),
	}
}

// CreateClusterScopedLogEntry creates a LogEntry for cluster-scoped resources
func CreateClusterScopedLogEntry(resourceType, name string, data any) types.LogEntry {
	return CreateLogEntry(resourceType, name, "", data)
}
