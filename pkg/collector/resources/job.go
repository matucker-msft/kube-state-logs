package resources

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// JobHandler handles collection of job metrics
type JobHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(client *kubernetes.Clientset) *JobHandler {
	return &JobHandler{
		client: client,
	}
}

// SetupInformer sets up the job informer
func (h *JobHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create job informer
	h.informer = factory.Batch().V1().Jobs().Informer()

	return nil
}

// Collect gathers job metrics from the cluster (uses cache)
func (h *JobHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all jobs from the cache
	jobs := utils.SafeGetStoreList(h.informer)

	for _, obj := range jobs {
		job, ok := obj.(*batchv1.Job)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, job.Namespace) {
			continue
		}

		entry := h.createLogEntry(job)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a job
func (h *JobHandler) createLogEntry(job *batchv1.Job) types.LogEntry {

	// Determine job type
	jobType := "Job"
	if len(job.OwnerReferences) > 0 {
		jobType = job.OwnerReferences[0].Kind
	}

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(job)

	// Get job conditions
	conditionComplete := false
	conditionFailed := false
	for _, condition := range job.Status.Conditions {
		switch condition.Type {
		case batchv1.JobComplete:
			conditionComplete = condition.Status == "True"
		case batchv1.JobFailed:
			conditionFailed = condition.Status == "True"
		}
	}

	// Get suspend status
	var suspend *bool
	if job.Spec.Suspend != nil {
		suspend = job.Spec.Suspend
	}

	// Get active deadline seconds
	var activeDeadlineSeconds *int64
	if job.Spec.ActiveDeadlineSeconds != nil {
		activeDeadlineSeconds = job.Spec.ActiveDeadlineSeconds
	}

	// Get backoff limit
	// Default to 6 when spec.backoffLimit is nil (Kubernetes API default)
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/job/#pod-backoff-failure-policy
	backoffLimit := int32(6) // Default value
	if job.Spec.BackoffLimit != nil {
		backoffLimit = *job.Spec.BackoffLimit
	}

	data := types.JobData{
		CreatedTimestamp:      job.CreationTimestamp.Unix(),
		Labels:                job.Labels,
		Annotations:           job.Annotations,
		ActivePods:            job.Status.Active,
		SucceededPods:         job.Status.Succeeded,
		FailedPods:            job.Status.Failed,
		Completions:           job.Spec.Completions,
		Parallelism:           job.Spec.Parallelism,
		BackoffLimit:          backoffLimit,
		ActiveDeadlineSeconds: activeDeadlineSeconds,
		ConditionComplete:     conditionComplete,
		ConditionFailed:       conditionFailed,
		CreatedByKind:         createdByKind,
		CreatedByName:         createdByName,
		JobType:               jobType,
		Suspend:               suspend,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "job",
		Name:         job.Name,
		Namespace:    job.Namespace,
		Data:         h.convertToMap(data),
	}
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *JobHandler) convertToMap(data any) map[string]any {
	return utils.ConvertStructToMap(data)
}
