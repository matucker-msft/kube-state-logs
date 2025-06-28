package resources

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
)

// JobHandler handles collection of job metrics
type JobHandler struct {
	utils.BaseHandler
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(client *kubernetes.Clientset) *JobHandler {
	return &JobHandler{
		BaseHandler: utils.NewBaseHandler(client),
	}
}

// SetupInformer sets up the job informer
func (h *JobHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	// Create job informer
	informer := factory.Batch().V1().Jobs().Informer()
	h.SetupBaseInformer(informer, logger)
	return nil
}

// Collect gathers job metrics from the cluster (uses cache)
func (h *JobHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all jobs from the cache
	jobs := utils.SafeGetStoreList(h.GetInformer())

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
	conditionComplete := utils.GetConditionStatusGeneric(job.Status.Conditions, string(batchv1.JobComplete))
	conditionFailed := utils.GetConditionStatusGeneric(job.Status.Conditions, string(batchv1.JobFailed))

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
	backoffLimit := int32(6)
	if job.Spec.BackoffLimit != nil {
		backoffLimit = *job.Spec.BackoffLimit
	}

	data := types.JobData{
		CreatedTimestamp:      utils.ExtractCreationTimestamp(job),
		Labels:                utils.ExtractLabels(job),
		Annotations:           utils.ExtractAnnotations(job),
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

	return utils.CreateLogEntry("job", utils.ExtractName(job), utils.ExtractNamespace(job), data)
}
