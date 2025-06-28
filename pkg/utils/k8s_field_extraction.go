package utils

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExtractCreationTimestamp extracts the creation timestamp from a Kubernetes object
func ExtractCreationTimestamp(obj metav1.Object) int64 {
	if obj == nil {
		return 0
	}
	return obj.GetCreationTimestamp().Unix()
}

// ExtractLabels extracts labels from a Kubernetes object
func ExtractLabels(obj metav1.Object) map[string]string {
	if obj == nil {
		return nil
	}
	return obj.GetLabels()
}

// ExtractAnnotations extracts annotations from a Kubernetes object
func ExtractAnnotations(obj metav1.Object) map[string]string {
	if obj == nil {
		return nil
	}
	return obj.GetAnnotations()
}

// ExtractName extracts the name from a Kubernetes object
func ExtractName(obj metav1.Object) string {
	if obj == nil {
		return ""
	}
	return obj.GetName()
}

// ExtractNamespace extracts the namespace from a Kubernetes object
func ExtractNamespace(obj metav1.Object) string {
	if obj == nil {
		return ""
	}
	return obj.GetNamespace()
}

// ExtractGeneration extracts the generation from a Kubernetes object
func ExtractGeneration(obj metav1.Object) int64 {
	if obj == nil {
		return 0
	}
	return obj.GetGeneration()
}

// ExtractUID extracts the UID from a Kubernetes object
func ExtractUID(obj metav1.Object) string {
	if obj == nil {
		return ""
	}
	return string(obj.GetUID())
}

// ExtractResourceVersion extracts the resource version from a Kubernetes object
func ExtractResourceVersion(obj metav1.Object) string {
	if obj == nil {
		return ""
	}
	return obj.GetResourceVersion()
}

// ExtractDeletionTimestamp extracts the deletion timestamp from a Kubernetes object
func ExtractDeletionTimestamp(obj metav1.Object) *time.Time {
	if obj == nil || obj.GetDeletionTimestamp() == nil {
		return nil
	}
	return &obj.GetDeletionTimestamp().Time
}

// ExtractFinalizers extracts finalizers from a Kubernetes object
func ExtractFinalizers(obj metav1.Object) []string {
	if obj == nil {
		return nil
	}
	return obj.GetFinalizers()
}

// IsBeingDeleted checks if a Kubernetes object is being deleted
func IsBeingDeleted(obj metav1.Object) bool {
	if obj == nil {
		return false
	}
	return obj.GetDeletionTimestamp() != nil
}
