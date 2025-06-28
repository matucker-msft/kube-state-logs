package resources

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
)

func TestNetworkPolicyHandler(t *testing.T) {
	protocolTCP := corev1.ProtocolTCP
	protocolUDP := corev1.ProtocolUDP

	np1 := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-policy",
			Namespace:         "default",
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"purpose": "test"},
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "web"},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 80},
						},
						{
							Protocol: &protocolTCP,
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
						},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "frontend"},
							},
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "frontend"},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 53},
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR:   "10.0.0.0/8",
								Except: []string{"10.0.0.0/24"},
							},
						},
					},
				},
			},
		},
	}

	np2 := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "api-policy",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolUDP,
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
							EndPort:  func() *int32 { v := int32(8089); return &v }(),
						},
					},
				},
			},
		},
	}

	npWithOwner := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "owned-policy",
			Namespace:         "default",
			OwnerReferences:   []metav1.OwnerReference{{Kind: "Application", Name: "my-app"}},
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "owned"},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "database"},
							},
						},
					},
				},
			},
		},
	}

	npDefault := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default-policy",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "default"},
			},
			// No PolicyTypes specified, should default to Ingress
		},
	}

	tests := []struct {
		name            string
		networkPolicies []*networkingv1.NetworkPolicy
		namespaces      []string
		expectedCount   int
		expectedNames   []string
		expectedFields  map[string]interface{}
	}{
		{
			name:            "collect all network policies",
			networkPolicies: []*networkingv1.NetworkPolicy{np1, np2},
			namespaces:      []string{},
			expectedCount:   2,
			expectedNames:   []string{"web-policy", "api-policy"},
		},
		{
			name:            "collect network policies from specific namespace",
			networkPolicies: []*networkingv1.NetworkPolicy{np1, np2},
			namespaces:      []string{"default"},
			expectedCount:   1,
			expectedNames:   []string{"web-policy"},
		},
		{
			name:            "collect network policy with owner reference",
			networkPolicies: []*networkingv1.NetworkPolicy{npWithOwner},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"owned-policy"},
			expectedFields: map[string]interface{}{
				"created_by_kind": "Application",
				"created_by_name": "my-app",
			},
		},
		{
			name:            "collect network policy with ingress and egress rules",
			networkPolicies: []*networkingv1.NetworkPolicy{np1},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"web-policy"},
			expectedFields: map[string]interface{}{
				"policy_types": []string{"Ingress", "Egress"},
			},
		},
		{
			name:            "collect network policy with default policy types",
			networkPolicies: []*networkingv1.NetworkPolicy{npDefault},
			namespaces:      []string{},
			expectedCount:   1,
			expectedNames:   []string{"default-policy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.networkPolicies))
			for i, np := range tt.networkPolicies {
				objects[i] = np
			}
			client := fake.NewSimpleClientset(objects...)
			handler := NewNetworkPolicyHandler(client)
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
					t.Errorf("Expected to find network policy with name %s", expectedName)
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
					case "policy_types":
						expectedTypes := expectedValue.([]string)
						types, ok := entry.Data["policyTypes"].([]string)
						if !ok {
							t.Errorf("Expected policyTypes to be []string, got %T", entry.Data["policyTypes"])
						} else if len(types) != len(expectedTypes) {
							t.Errorf("Expected %d policy types, got %d", len(expectedTypes), len(types))
						}
					}
				}
			}
			for _, entry := range entries {
				if entry.ResourceType != "networkpolicy" {
					t.Errorf("Expected resource type 'networkpolicy', got %s", entry.ResourceType)
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
				if entry.Data["policyTypes"] == nil {
					t.Error("policyTypes should not be nil")
				}
				if entry.Data["ingressRules"] == nil {
					t.Error("ingressRules should not be nil")
				}
				if entry.Data["egressRules"] == nil {
					t.Error("egressRules should not be nil")
				}
			}
		})
	}
}

func TestNetworkPolicyHandler_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNetworkPolicyHandler(client)
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

func TestNetworkPolicyHandler_InvalidObject(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewNetworkPolicyHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	err := handler.SetupInformer(factory, &testutils.MockLogger{}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	invalidObj := &networkingv1.Ingress{}
	handler.GetInformer().GetStore().Add(invalidObj)
	entries, err := handler.Collect(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries with invalid object, got %d", len(entries))
	}
}

func createTestNetworkPolicy(name, namespace string) *networkingv1.NetworkPolicy {
	port := int32(80)
	protocol := corev1.ProtocolTCP
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test-app",
			},
			Annotations: map[string]string{
				"test-annotation": "test-value",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocol,
							Port:     &intstr.IntOrString{IntVal: port},
						},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "test-client",
								},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "kube-system",
								},
							},
						},
					},
				},
			},
		},
	}
}
