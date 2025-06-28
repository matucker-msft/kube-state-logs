package resources

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestCronJob creates a test CronJob with various configurations
func createTestCronJob(name, namespace, schedule string) *batchv1.CronJob {
	now := metav1.Now()
	suspend := false
	successfulJobsHistoryLimit := int32(3)
	failedJobsHistoryLimit := int32(1)
	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test cronjob",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   schedule,
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			Suspend:                    &suspend,
			SuccessfulJobsHistoryLimit: &successfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     &failedJobsHistoryLimit,
		},
		Status: batchv1.CronJobStatus{
			Active: []corev1.ObjectReference{
				{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "test-job-1",
					Namespace:  namespace,
				},
			},
			LastScheduleTime: &now,
		},
	}
	return cronjob
}

func TestNewCronJobHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCronJobHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestCronJobHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCronJobHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if handler.GetInformer() == nil {
		t.Error("Expected informer to be set up")
	}
}

func TestCronJobHandler_Collect(t *testing.T) {
	cronjob1 := createTestCronJob("test-cronjob-1", "default", "*/5 * * * *")
	cronjob2 := createTestCronJob("test-cronjob-2", "kube-system", "0 0 * * *")
	client := fake.NewSimpleClientset(cronjob1, cronjob2)
	handler := NewCronJobHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	entries, err = handler.Collect(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for default namespace, got %d", len(entries))
	}
	if entries[0].Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entries[0].Namespace)
	}
}

func TestCronJobHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCronJobHandler(client)
	cronjob := createTestCronJob("test-cronjob", "default", "*/5 * * * *")
	entry := handler.createLogEntry(cronjob)
	if entry.ResourceType != "cronjob" {
		t.Errorf("Expected resource type 'cronjob', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-cronjob" {
		t.Errorf("Expected name 'test-cronjob', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["schedule"]
	if !ok || val == nil {
		t.Fatalf("schedule missing or nil")
	}
	if val.(string) != "*/5 * * * *" {
		t.Errorf("Expected schedule '*/5 * * * *', got '%s'", val.(string))
	}
	val, ok = data["concurrencyPolicy"]
	if !ok || val == nil {
		t.Fatalf("concurrencyPolicy missing or nil")
	}
	if val.(string) != "Forbid" {
		t.Errorf("Expected concurrency policy 'Forbid', got '%s'", val.(string))
	}
	val, ok = data["activeJobsCount"]
	if !ok || val == nil {
		t.Fatalf("activeJobsCount missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected active jobs count 1, got %d", val.(int32))
	}
	val, ok = data["conditionActive"]
	if !ok || val == nil {
		t.Fatalf("conditionActive missing or nil")
	}
	if val.(bool) != true {
		t.Errorf("Expected condition active true, got %t", val.(bool))
	}
	val, ok = data["successfulJobsHistoryLimit"]
	if !ok || val == nil {
		t.Fatalf("successfulJobsHistoryLimit missing or nil")
	}
	if val.(int32) != 3 {
		t.Errorf("Expected successful jobs history limit 3, got %d", val.(int32))
	}
	val, ok = data["failedJobsHistoryLimit"]
	if !ok || val == nil {
		t.Fatalf("failedJobsHistoryLimit missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected failed jobs history limit 1, got %d", val.(int32))
	}
}

func TestCronJobHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCronJobHandler(client)
	cronjob := createTestCronJob("test-cronjob", "default", "*/5 * * * *")
	cronjob.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-deploy",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(cronjob)
	data := entry.Data
	val, ok := data["createdByKind"]
	if !ok || val == nil {
		t.Fatalf("createdByKind missing or nil")
	}
	if val.(string) != "Deployment" {
		t.Errorf("Expected created by kind 'Deployment', got '%s'", val.(string))
	}
	val, ok = data["createdByName"]
	if !ok || val == nil {
		t.Fatalf("createdByName missing or nil")
	}
	if val.(string) != "test-deploy" {
		t.Errorf("Expected created by name 'test-deploy', got '%s'", val.(string))
	}
}

func TestCronJobHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewCronJobHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for empty cache, got %d", len(entries))
	}
}

func TestCronJobHandler_Collect_NamespaceFiltering(t *testing.T) {
	cronjob1 := createTestCronJob("test-cronjob-1", "default", "*/5 * * * *")
	cronjob2 := createTestCronJob("test-cronjob-2", "kube-system", "0 0 * * *")
	cronjob3 := createTestCronJob("test-cronjob-3", "monitoring", "0 */6 * * *")
	client := fake.NewSimpleClientset(cronjob1, cronjob2, cronjob3)
	handler := NewCronJobHandler(client)
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	logger := &testutils.MockLogger{}
	err := handler.SetupInformer(factory, logger, time.Hour)
	if err != nil {
		t.Fatalf("Failed to setup informer: %v", err)
	}
	factory.Start(nil)
	factory.WaitForCacheSync(nil)
	ctx := context.Background()
	entries, err := handler.Collect(ctx, []string{"default", "monitoring"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for default and monitoring namespaces, got %d", len(entries))
	}
	namespaces := make(map[string]bool)
	for _, entry := range entries {
		namespaces[entry.Namespace] = true
	}
	if !namespaces["default"] {
		t.Error("Expected entry from default namespace")
	}
	if !namespaces["monitoring"] {
		t.Error("Expected entry from monitoring namespace")
	}
	if namespaces["kube-system"] {
		t.Error("Did not expect entry from kube-system namespace")
	}
}
