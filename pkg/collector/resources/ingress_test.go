package resources

import (
	"context"
	"testing"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestIngressHandler(t *testing.T) {
	pathTypePrefix := networkingv1.PathTypePrefix
	pathTypeExact := networkingv1.PathTypeExact
	ingressClassName := "nginx"

	ingress1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-ingress",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "web-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path:     "/api",
									PathType: &pathTypeExact,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "api-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"example.com"},
					SecretName: "tls-secret",
				},
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{
					{
						IP:       "192.168.1.100",
						Hostname: "example.com",
					},
				},
			},
		},
	}

	ingress2 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "api-ingress",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "api.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "api-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingressWithOwner := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-ingress",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Application", Name: "my-app"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "owned.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "owned-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingressNoLoadBalancer := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-lb-ingress",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "nolb.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "nolb-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		ingresses      []*networkingv1.Ingress
		namespaces     []string
		expectedCount  int
		expectedNames  []string
		expectedFields map[string]interface{}
	}{
		{
			name:          "collect all ingresses",
			ingresses:     []*networkingv1.Ingress{ingress1, ingress2},
			namespaces:    []string{},
			expectedCount: 2,
			expectedNames: []string{"web-ingress", "api-ingress"},
		},
		{
			name:          "collect ingresses from specific namespace",
			ingresses:     []*networkingv1.Ingress{ingress1, ingress2},
			namespaces:    []string{"default"},
			expectedCount: 1,
			expectedNames: []string{"web-ingress"},
		},
		{
			name:          "collect ingress with owner reference",
			ingresses:     []*networkingv1.Ingress{ingressWithOwner},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"owned-ingress"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Application",
				"created_by_name": "my-app",
			},
		},
		{
			name:          "collect ingress with load balancer",
			ingresses:     []*networkingv1.Ingress{ingress1},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"web-ingress"},
			expectedFields: map[string]interface{}{
				"condition_load_balancer_ready": true,
			},
		},
		{
			name:          "collect ingress without load balancer",
			ingresses:     []*networkingv1.Ingress{ingressNoLoadBalancer},
			namespaces:    []string{},
			expectedCount: 1,
			expectedNames: []string{"no-lb-ingress"},
			expectedFields: map[string]interface{}{
				"condition_load_balancer_ready": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.ingresses))
			for i, ing := range tt.ingresses {
				objects[i] = ing
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewIngressHandler(client)
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
					t.Errorf("Expected to find ingress with name %s", expectedName)
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
					case "condition_load_balancer_ready":
						if entry.Data["conditionLoadBalancerReady"] != expectedValue.(bool) {
							t.Errorf("Expected condition_load_balancer_ready %v, got %v", expectedValue, entry.Data["conditionLoadBalancerReady"])
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "ingress" {
					t.Errorf("Expected resource type 'ingress', got %s", entry.ResourceType)
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
				if entry.Data["rules"] == nil {
					t.Error("rules should not be nil")
				}
				if entry.Data["tls"] == nil {
					t.Error("tls should not be nil")
				}
				if entry.Data["loadBalancerIngress"] == nil {
					t.Error("loadBalancerIngress should not be nil")
				}
			}
		})
	}
}

func TestIngressHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewIngressHandler(client)
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

func TestIngressHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewIngressHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &networkingv1.NetworkPolicy{}
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}
