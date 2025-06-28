package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition represents a Kubernetes condition
type Condition struct {
	Type    string
	Status  metav1.ConditionStatus
	Reason  string
	Message string
}

// ConditionInterface defines the interface for Kubernetes conditions
type ConditionInterface interface {
	GetType() string
	GetStatus() metav1.ConditionStatus
	GetReason() string
	GetMessage() string
}

// GetConditionStatus checks if a specific condition type is true
func GetConditionStatus(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}

// GetConditionStatusGeneric checks if a specific condition type is true for any condition type
func GetConditionStatusGeneric(conditions any, conditionType string) bool {
	switch typedConditions := conditions.(type) {
	case []metav1.Condition:
		return GetConditionStatus(typedConditions, conditionType)
	case []corev1.NodeCondition:
		for _, condition := range typedConditions {
			if string(condition.Type) == conditionType {
				return condition.Status == corev1.ConditionTrue
			}
		}
		return false
	default:
		// For other condition types, we'll need to handle them specifically
		// This is a fallback for now
		return false
	}
}

// GetConditionReason gets the reason for a specific condition type
func GetConditionReason(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Reason
		}
	}
	return ""
}

// GetConditionMessage gets the message for a specific condition type
func GetConditionMessage(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Message
		}
	}
	return ""
}

// GetConditionByType gets a specific condition by type
func GetConditionByType(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// HasCondition checks if an object has a specific condition type
func HasCondition(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// CountConditionsByStatus counts conditions by their status
func CountConditionsByStatus(conditions []metav1.Condition, status metav1.ConditionStatus) int {
	count := 0
	for _, condition := range conditions {
		if condition.Status == status {
			count++
		}
	}
	return count
}
