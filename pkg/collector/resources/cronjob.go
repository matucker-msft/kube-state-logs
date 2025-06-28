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

// CronJobHandler handles collection of cronjob metrics
type CronJobHandler struct {
	client   *kubernetes.Clientset
	informer cache.SharedIndexInformer
	logger   interfaces.Logger
}

// NewCronJobHandler creates a new CronJobHandler
func NewCronJobHandler(client *kubernetes.Clientset) *CronJobHandler {
	return &CronJobHandler{
		client: client,
	}
}

// SetupInformer sets up the cronjob informer
func (h *CronJobHandler) SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error {
	h.logger = logger

	// Create cronjob informer
	h.informer = factory.Batch().V1().CronJobs().Informer()

	return nil
}

// Collect gathers cronjob metrics from the cluster (uses cache)
func (h *CronJobHandler) Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error) {
	var entries []types.LogEntry

	// Get all cronjobs from the cache
	cronjobs := utils.SafeGetStoreList(h.informer)

	for _, obj := range cronjobs {
		cronjob, ok := obj.(*batchv1.CronJob)
		if !ok {
			continue
		}

		if !utils.ShouldIncludeNamespace(namespaces, cronjob.Namespace) {
			continue
		}

		entry := h.createLogEntry(cronjob)
		entries = append(entries, entry)
	}

	return entries, nil
}

// createLogEntry creates a LogEntry from a cronjob
func (h *CronJobHandler) createLogEntry(cronjob *batchv1.CronJob) types.LogEntry {

	// Get concurrency policy
	// Default is "Allow" when spec.concurrencyPolicy is not set
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#concurrency-policy
	concurrencyPolicy := string(cronjob.Spec.ConcurrencyPolicy)

	createdByKind, createdByName := utils.GetOwnerReferenceInfo(cronjob)

	// Get suspend status
	var suspend *bool
	if cronjob.Spec.Suspend != nil {
		suspend = cronjob.Spec.Suspend
	}

	// Get history limits
	var successfulJobsHistoryLimit *int32
	var failedJobsHistoryLimit *int32
	if cronjob.Spec.SuccessfulJobsHistoryLimit != nil {
		successfulJobsHistoryLimit = cronjob.Spec.SuccessfulJobsHistoryLimit
	}
	if cronjob.Spec.FailedJobsHistoryLimit != nil {
		failedJobsHistoryLimit = cronjob.Spec.FailedJobsHistoryLimit
	}

	// Get last and next schedule times
	var lastScheduleTime *time.Time
	if cronjob.Status.LastScheduleTime != nil {
		lastScheduleTime = &cronjob.Status.LastScheduleTime.Time
	}

	// Get condition active - CronJob doesn't have conditions in the same way
	// We'll determine if it's active based on whether it has active jobs
	conditionActive := len(cronjob.Status.Active) > 0

	data := types.CronJobData{
		CreatedTimestamp:           cronjob.CreationTimestamp.Unix(),
		Labels:                     cronjob.Labels,
		Annotations:                cronjob.Annotations,
		Schedule:                   cronjob.Spec.Schedule,
		ConcurrencyPolicy:          concurrencyPolicy,
		Suspend:                    suspend,
		SuccessfulJobsHistoryLimit: successfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     failedJobsHistoryLimit,
		ActiveJobsCount:            int32(len(cronjob.Status.Active)),
		LastScheduleTime:           lastScheduleTime,
		NextScheduleTime:           nil, // Not available in v1 API
		ConditionActive:            conditionActive,
		CreatedByKind:              createdByKind,
		CreatedByName:              createdByName,
	}

	return types.LogEntry{
		Timestamp:    time.Now(),
		ResourceType: "cronjob",
		Name:         cronjob.Name,
		Namespace:    cronjob.Namespace,
		Data:         h.convertToMap(data),
	}
}

// convertToMap converts a struct to map[string]any for JSON serialization
func (h *CronJobHandler) convertToMap(data any) map[string]any {
	return utils.ConvertStructToMap(data)
}
