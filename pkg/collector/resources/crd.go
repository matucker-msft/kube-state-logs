package resources

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// CRDHandler handles collection of generic CRD metrics
type CRDHandler struct {
	client       dynamic.Interface
	informer     cache.SharedIndexInformer
	logger       interfaces.Logger
	gvr          schema.GroupVersionResource
	resourceName string
	customFields []string // JSONPath-like field paths to extract
}

// NewCRDHandler creates a new CRDHandler for a specific CRD
func NewCRDHandler(client dynamic.Interface, gvr schema.GroupVersionResource, resourceName string, customFields []string) *CRDHandler {
	return &CRDHandler{
		client:       client,
		gvr:          gvr,
		resourceName: resourceName,
		customFields: customFields,
	}
}

// SetupInformer sets up the CRD informer
func (h *CRDHandler) SetupInformer(factory dynamicinformer.DynamicSharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create dynamic informer for the CRD
	h.informer = factory.ForResource(h.gvr).Informer()

	return nil
}

// Collect gathers CRD metrics from the cluster (uses cache)
func (h *CRDHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all CRD resources from the cache
	crdList := safeGetStoreList(h.informer)

	for _, obj := range crdList {
		unstructuredObj, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		// Filter by namespace if specified
		if len(namespaces) > 0 {
			namespace := unstructuredObj.GetNamespace()
			if namespace != "" && !contains(namespaces, namespace) {
				continue
			}
		}

		entry := h.createLogEntry(unstructuredObj)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a CRD resource
func (h *CRDHandler) createLogEntry(obj *unstructured.Unstructured) types.LogEntry {
	// Extract basic metadata
	createdTimestamp := int64(0)
	if creationTime := obj.GetCreationTimestamp(); !creationTime.IsZero() {
		createdTimestamp = creationTime.Unix()
	}

	// Extract spec and status
	spec := make(map[string]any)
	if specObj, exists, _ := unstructured.NestedMap(obj.Object, "spec"); exists {
		spec = specObj
	}

	status := make(map[string]any)
	if statusObj, exists, _ := unstructured.NestedMap(obj.Object, "status"); exists {
		status = statusObj
	}

	// Extract custom fields based on JSONPath-like paths
	customFields := make(map[string]any)
	for _, fieldPath := range h.customFields {
		if value := h.extractField(obj.Object, fieldPath); value != nil {
			customFields[fieldPath] = value
		}
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(obj)

	// Create data structure
	data := types.CRDData{
		CreatedTimestamp: createdTimestamp,
		Labels:           obj.GetLabels(),
		Annotations:      obj.GetAnnotations(),
		APIVersion:       obj.GetAPIVersion(),
		Kind:             obj.GetKind(),
		Spec:             spec,
		Status:           status,
		CustomFields:     customFields,
		CreatedByKind:    createdByKind,
		CreatedByName:    createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: h.resourceName,
		Name:         obj.GetName(),
		Namespace:    obj.GetNamespace(),
		Data:         convertStructToMap(data),
	}
}

// extractField extracts a field from an object using a dot-separated path
func (h *CRDHandler) extractField(obj map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		if current == nil {
			return nil
		}

		if i == len(parts)-1 {
			// Last part, return the value
			return current[part]
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
