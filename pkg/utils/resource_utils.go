package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ExtractResourceQuantity extracts a resource quantity as a string
func ExtractResourceQuantity(quantity *resource.Quantity) string {
	if quantity == nil {
		return ""
	}
	return quantity.String()
}

// ExtractResourceQuantityAsInt64 extracts a resource quantity as int64
func ExtractResourceQuantityAsInt64(quantity *resource.Quantity) int64 {
	if quantity == nil {
		return 0
	}
	return quantity.Value()
}

// ExtractResourceQuantityAsFloat64 extracts a resource quantity as float64
func ExtractResourceQuantityAsFloat64(quantity *resource.Quantity) float64 {
	if quantity == nil {
		return 0.0
	}
	return quantity.AsApproximateFloat64()
}

// ExtractResourceMap extracts a resource map as string map
func ExtractResourceMap(resourceList corev1.ResourceList) map[string]string {
	if resourceList == nil {
		return nil
	}

	result := make(map[string]string)
	for resourceName, quantity := range resourceList {
		result[string(resourceName)] = quantity.String()
	}
	return result
}

// ExtractResourceRequests extracts resource requests as string map
func ExtractResourceRequests(requirements *corev1.ResourceRequirements) map[string]string {
	if requirements == nil || requirements.Requests == nil {
		return nil
	}
	return ExtractResourceMap(requirements.Requests)
}

// ExtractResourceLimits extracts resource limits as string map
func ExtractResourceLimits(requirements *corev1.ResourceRequirements) map[string]string {
	if requirements == nil || requirements.Limits == nil {
		return nil
	}
	return ExtractResourceMap(requirements.Limits)
}

// ExtractSpecificResource extracts a specific resource quantity
func ExtractSpecificResource(resourceList corev1.ResourceList, resourceName corev1.ResourceName) string {
	if resourceList == nil {
		return ""
	}
	if quantity, exists := resourceList[resourceName]; exists {
		return quantity.String()
	}
	return ""
}

// ExtractCPU extracts CPU resource as string
func ExtractCPU(resourceList corev1.ResourceList) string {
	return ExtractSpecificResource(resourceList, corev1.ResourceCPU)
}

// ExtractMemory extracts memory resource as string
func ExtractMemory(resourceList corev1.ResourceList) string {
	return ExtractSpecificResource(resourceList, corev1.ResourceMemory)
}

// ExtractStorage extracts storage resource as string
func ExtractStorage(resourceList corev1.ResourceList) string {
	return ExtractSpecificResource(resourceList, corev1.ResourceStorage)
}

// ExtractEphemeralStorage extracts ephemeral storage resource as string
func ExtractEphemeralStorage(resourceList corev1.ResourceList) string {
	return ExtractSpecificResource(resourceList, corev1.ResourceEphemeralStorage)
}

// ParseQuantity safely parses a resource quantity string
func ParseQuantity(quantityStr string) *resource.Quantity {
	if quantityStr == "" {
		return nil
	}
	quantity, err := resource.ParseQuantity(quantityStr)
	if err != nil {
		return nil
	}
	return &quantity
}

// ConvertToBytes converts a resource quantity to bytes
func ConvertToBytes(quantity *resource.Quantity) int64 {
	if quantity == nil {
		return 0
	}
	return quantity.Value()
}

// ConvertToMillicores converts a CPU resource quantity to millicores
func ConvertToMillicores(quantity *resource.Quantity) int64 {
	if quantity == nil {
		return 0
	}
	return quantity.MilliValue()
}

// IsZeroQuantity checks if a resource quantity is zero
func IsZeroQuantity(quantity *resource.Quantity) bool {
	if quantity == nil {
		return true
	}
	return quantity.IsZero()
}

// CompareQuantities compares two resource quantities
func CompareQuantities(q1, q2 *resource.Quantity) int {
	if q1 == nil && q2 == nil {
		return 0
	}
	if q1 == nil {
		return -1
	}
	if q2 == nil {
		return 1
	}
	return q1.Cmp(*q2)
}
