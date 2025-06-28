package resources

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	testutils "github.com/matucker-msft/kube-state-logs/pkg/collector/testutils"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// createTestJob creates a test Job with various configurations
func createTestJob(name, namespace string, active, succeeded, failed int32) *batchv1.Job {
	now := metav1.Now()
	completions := int32(1)
	parallelism := int32(1)
	backoffLimit := int32(6)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     name,
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "test job",
			},
			CreationTimestamp: now,
			Generation:        1,
		},
		Spec: batchv1.JobSpec{
			Completions:  &completions,
			Parallelism:  &parallelism,
			BackoffLimit: &backoffLimit,
		},
		Status: batchv1.JobStatus{
			Active:    active,
			Succeeded: succeeded,
			Failed:    failed,
		},
	}
	return job
}

func TestNewJobHandler(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.BaseHandler == (utils.BaseHandler{}) {
		t.Error("Expected BaseHandler to be embedded")
	}
}

func TestJobHandler_SetupInformer(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)
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

func TestJobHandler_Collect(t *testing.T) {
	job1 := createTestJob("test-job-1", "default", 1, 0, 0)
	job2 := createTestJob("test-job-2", "kube-system", 0, 1, 0)
	client := fake.NewSimpleClientset(job1, job2)
	handler := NewJobHandler(client)
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

func TestJobHandler_createLogEntry(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)
	job := createTestJob("test-job", "default", 1, 0, 0)
	entry := handler.createLogEntry(job)
	if entry.ResourceType != "job" {
		t.Errorf("Expected resource type 'job', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-job" {
		t.Errorf("Expected name 'test-job', got '%s'", entry.Name)
	}
	if entry.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", entry.Namespace)
	}
	data := entry.Data
	val, ok := data["activePods"]
	if !ok || val == nil {
		t.Fatalf("activePods missing or nil")
	}
	if val.(int32) != 1 {
		t.Errorf("Expected active pods 1, got %d", val.(int32))
	}
	val, ok = data["succeededPods"]
	if !ok || val == nil {
		t.Fatalf("succeededPods missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected succeeded pods 0, got %d", val.(int32))
	}
	val, ok = data["failedPods"]
	if !ok || val == nil {
		t.Fatalf("failedPods missing or nil")
	}
	if val.(int32) != 0 {
		t.Errorf("Expected failed pods 0, got %d", val.(int32))
	}
	val, ok = data["backoffLimit"]
	if !ok || val == nil {
		t.Fatalf("backoffLimit missing or nil")
	}
	if val.(int32) != 6 {
		t.Errorf("Expected backoff limit 6, got %d", val.(int32))
	}
	val, ok = data["jobType"]
	if !ok || val == nil {
		t.Fatalf("jobType missing or nil")
	}
	if val.(string) != "Job" {
		t.Errorf("Expected job type 'Job', got '%s'", val.(string))
	}
}

func TestJobHandler_createLogEntry_WithOwnerReference(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)
	job := createTestJob("test-job", "default", 1, 0, 0)
	job.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "batch/v1",
			Kind:       "CronJob",
			Name:       "test-cronjob",
			UID:        "test-uid",
		},
	}
	entry := handler.createLogEntry(job)
	data := entry.Data
	val, ok := data["createdByKind"]
	if !ok || val == nil {
		t.Fatalf("createdByKind missing or nil")
	}
	if val.(string) != "CronJob" {
		t.Errorf("Expected created by kind 'CronJob', got '%s'", val.(string))
	}
	val, ok = data["createdByName"]
	if !ok || val == nil {
		t.Fatalf("createdByName missing or nil")
	}
	if val.(string) != "test-cronjob" {
		t.Errorf("Expected created by name 'test-cronjob', got '%s'", val.(string))
	}
	val, ok = data["jobType"]
	if !ok || val == nil {
		t.Fatalf("jobType missing or nil")
	}
	if val.(string) != "CronJob" {
		t.Errorf("Expected job type 'CronJob', got '%s'", val.(string))
	}
}

func TestJobHandler_createLogEntry_WithNilPointers(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)

	// Create a job with Suspend and ActiveDeadlineSeconds explicitly set to nil
	job := createTestJob("test-job", "default", 1, 0, 0)
	job.Spec.Suspend = nil
	job.Spec.ActiveDeadlineSeconds = nil

	entry := handler.createLogEntry(job)

	// Verify the entry is created successfully even with nil pointers
	if entry.ResourceType != "job" {
		t.Errorf("Expected resource type 'job', got '%s'", entry.ResourceType)
	}
	if entry.Name != "test-job" {
		t.Errorf("Expected name 'test-job', got '%s'", entry.Name)
	}

	// Check that the data is still properly structured
	data := entry.Data
	if data == nil {
		t.Fatal("Expected data to not be nil")
	}

	// Verify that suspend and activeDeadlineSeconds are present but nil in the data
	suspendVal, ok := data["suspend"]
	if !ok {
		t.Error("Expected suspend to be present in data")
	}
	if suspendVal != nil {
		t.Errorf("Expected suspend to be nil, got %v", suspendVal)
	}

	activeDeadlineVal, ok := data["activeDeadlineSeconds"]
	if !ok {
		t.Error("Expected activeDeadlineSeconds to be present in data")
	}
	if activeDeadlineVal != nil {
		t.Errorf("Expected activeDeadlineSeconds to be nil, got %v", activeDeadlineVal)
	}
}

func TestJobHandler_createLogEntry_WithNonNilPointers(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)

	// Create a job with Suspend and ActiveDeadlineSeconds set to non-nil values
	job := createTestJob("test-job", "default", 1, 0, 0)
	suspend := true
	activeDeadlineSeconds := int64(3600)
	job.Spec.Suspend = &suspend
	job.Spec.ActiveDeadlineSeconds = &activeDeadlineSeconds

	entry := handler.createLogEntry(job)

	// Verify the entry is created successfully
	if entry.ResourceType != "job" {
		t.Errorf("Expected resource type 'job', got '%s'", entry.ResourceType)
	}

	// Check that the data includes the non-nil values
	data := entry.Data
	suspendVal, ok := data["suspend"]
	if !ok {
		t.Fatal("Expected suspend to be present in data")
	}
	if suspendVal.(bool) != true {
		t.Errorf("Expected suspend to be true, got %v", suspendVal)
	}

	activeDeadlineVal, ok := data["activeDeadlineSeconds"]
	if !ok {
		t.Fatal("Expected activeDeadlineSeconds to be present in data")
	}
	if activeDeadlineVal.(int64) != 3600 {
		t.Errorf("Expected activeDeadlineSeconds to be 3600, got %v", activeDeadlineVal)
	}
}

func TestJobHandler_Collect_EmptyCache(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := NewJobHandler(client)
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

func TestJobHandler_Collect_NamespaceFiltering(t *testing.T) {
	job1 := createTestJob("test-job-1", "default", 1, 0, 0)
	job2 := createTestJob("test-job-2", "kube-system", 0, 1, 0)
	job3 := createTestJob("test-job-3", "monitoring", 0, 0, 1)
	client := fake.NewSimpleClientset(job1, job2, job3)
	handler := NewJobHandler(client)
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
