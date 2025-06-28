package resources

import (
	"context"
	"testing"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestHorizontalPodAutoscalerHandler(t *testing.T) {
	minReplicas := int32(2)
	maxReplicas := int32(10)
	targetCPU := int32(80)
	currentCPU := int32(75)
	currentReplicas := int32(5)
	desiredReplicas := int32(6)

	hpa1 := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app-hpa",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "app-deployment",
				APIVersion: "apps/v1",
			},
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &targetCPU,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: currentReplicas,
			DesiredReplicas: desiredReplicas,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: "cpu",
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: &currentCPU,
						},
					},
				},
			},
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{
					Type:   autoscalingv2.AbleToScale,
					Status: "True",
				},
				{
					Type:   autoscalingv2.ScalingActive,
					Status: "True",
				},
			},
		},
	}

	hpa2 := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "memory-hpa",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 5,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       "db-statefulset",
				APIVersion: "apps/v1",
			},
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &targetCPU,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 3,
		},
	}

	hpaWithOwner := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-hpa",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Project", Name: "my-project"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 3,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "owned-deployment",
				APIVersion: "apps/v1",
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 1,
			DesiredReplicas: 1,
		},
	}

	hpaMinReplicasNil := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "nil-min-hpa",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 5,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "nil-min-deployment",
				APIVersion: "apps/v1",
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 1,
			DesiredReplicas: 1,
		},
	}

	tests := []struct {
		name           string
		hpas           []*autoscalingv2.HorizontalPodAutoscaler
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all horizontal pod autoscalers",
			hpas:          []*autoscalingv2.HorizontalPodAutoscaler{hpa1, hpa2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"app-hpa", "memory-hpa"},
		},
		{
			name:          "collect horizontal pod autoscalers from specific namespace",
			hpas:          []*autoscalingv2.HorizontalPodAutoscaler{hpa1, hpa2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"app-hpa"},
		},
		{
			name:          "collect horizontal pod autoscaler with owner reference",
			hpas:          []*autoscalingv2.HorizontalPodAutoscaler{hpaWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-hpa"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Project",
				"created_by_name": "my-project",
			},
		},
		{
			name:          "collect horizontal pod autoscaler with CPU metrics",
			hpas:          []*autoscalingv2.HorizontalPodAutoscaler{hpa1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"app-hpa"},
			expectedFields: map[string]interface{}{
				"scale_target_ref":  "app-deployment",
				"scale_target_kind": "Deployment",
			},
		},
		{
			name:          "collect horizontal pod autoscaler with nil min replicas",
			hpas:          []*autoscalingv2.HorizontalPodAutoscaler{hpaMinReplicasNil},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"nil-min-hpa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.hpas))
			for i, hpa := range tt.hpas {
				objects[i] = hpa
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewHorizontalPodAutoscalerHandler(client)
			factory := informers.NewSharedInformerFactory(client, time.Hour)
			err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
			if err != nil {
				t.Fatalf("Failed to setup informer: %v", err)
			}
			factory.Start(context.Background().Done())
			if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
				t.Fatal("Failed to sync cache")
			}
			entries, err := handler.Collect(context.Background(), tt.namespaces)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}
			if len(entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries))
			}
			entryNames := make([]string, len(entries))
			for i, entry := range entries {
				entryNames[i] = entry.Name
			}
			for _, expectedName := range tt.expectedNames {
				found := false
				for _, name := range entryNames {
					if name == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find horizontal pod autoscaler with name %s", expectedName)
				}
			}
			if tt.expectedFields != nil && len(entries) > 0 {
				entry := entries[0]
				for field, expectedValue := range tt.expectedFields {
					switch field {
					case "created_by_kind":
						if entry.Data["createdByKind"] != expectedValue.(string) {
							t.Errorf("Expected created_by_kind %s, got %v", expectedValue, entry.Data["createdByKind"])
						}
					case "created_by_name":
						if entry.Data["createdByName"] != expectedValue.(string) {
							t.Errorf("Expected created_by_name %s, got %v", expectedValue, entry.Data["createdByName"])
						}
					case "scale_target_ref":
						if entry.Data["scaleTargetRef"] != expectedValue.(string) {
							t.Errorf("Expected scale_target_ref %s, got %v", expectedValue, entry.Data["scaleTargetRef"])
						}
					case "scale_target_kind":
						if entry.Data["scaleTargetKind"] != expectedValue.(string) {
							t.Errorf("Expected scale_target_kind %s, got %v", expectedValue, entry.Data["scaleTargetKind"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "horizontalpodautoscaler" {
					t.Errorf("Expected resource type 'horizontalpodautoscaler', got %s", entry.ResourceType)
				}
				if entry.Name == "" {
					t.Error("Entry name should not be empty")
				}
				if entry.Namespace == "" {
					t.Error("Entry namespace should not be empty")
				}
				if entry.Data["createdTimestamp"] == nil {
					t.Error("Created timestamp should not be nil")
				}
				if entry.Data["minReplicas"] == nil {
					t.Error("minReplicas should not be nil")
				}
				if entry.Data["maxReplicas"] == nil {
					t.Error("maxReplicas should not be nil")
				}
				if entry.Data["currentReplicas"] == nil {
					t.Error("currentReplicas should not be nil")
				}
				if entry.Data["desiredReplicas"] == nil {
					t.Error("desiredReplicas should not be nil")
				}
			}
		})
	}
}

func TestHorizontalPodAutoscalerHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewHorizontalPodAutoscalerHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(context.Background().Done())
	if !cache.WaitForCacheSync(context.Background().Done(), handler.GetInformer().HasSynced) {
		t.Fatal("Failed to sync cache")
	}
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestHorizontalPodAutoscalerHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewHorizontalPodAutoscalerHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &v1.Pod{}
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}
