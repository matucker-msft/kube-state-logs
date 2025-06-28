package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/matucker-msft/kube-state-logs/pkg/collector/resources"
	"github.com/matucker-msft/kube-state-logs/pkg/config"
	"github.com/matucker-msft/kube-state-logs/pkg/interfaces"
	"github.com/matucker-msft/kube-state-logs/pkg/types"
)

// Collector handles the collection and logging of Kubernetes resource state
type Collector struct {
	config   *config.Config
	client   *kubernetes.Clientset
	logger   interfaces.Logger
	handlers map[string]interfaces.ResourceHandler
	factory  informers.SharedInformerFactory
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// New creates a new Collector instance
func New(cfg *config.Config) (*Collector, error) {
	// Create Kubernetes client
	var kubeConfig *rest.Config
	var err error

	if cfg.Kubeconfig != "" {
		// Use kubeconfig file
		klog.Infof("Using kubeconfig file: %s", cfg.Kubeconfig)
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig file: %w", err)
		}
	} else {
		// Use in-cluster config
		klog.Info("Using in-cluster config")
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Create logger
	logger := NewLogger()

	// Create shared informer factory with no resync (0 means no resync)
	factory := informers.NewSharedInformerFactory(client, 0)

	// Create collector
	c := &Collector{
		config:   cfg,
		client:   client,
		logger:   logger,
		handlers: make(map[string]interfaces.ResourceHandler),
		factory:  factory,
		stopCh:   make(chan struct{}),
	}

	// Register resource handlers
	c.registerHandlers()

	return c, nil
}

// registerHandlers registers all available resource handlers
func (c *Collector) registerHandlers() {
	// Register resource handlers
	handlers := map[string]interfaces.ResourceHandler{
		"pod":                              resources.NewPodHandler(c.client),
		"service":                          resources.NewServiceHandler(c.client),
		"node":                             resources.NewNodeHandler(c.client),
		"deployment":                       resources.NewDeploymentHandler(c.client),
		"job":                              resources.NewJobHandler(c.client),
		"cronjob":                          resources.NewCronJobHandler(c.client),
		"configmap":                        resources.NewConfigMapHandler(c.client),
		"secret":                           resources.NewSecretHandler(c.client),
		"persistentvolumeclaim":            resources.NewPersistentVolumeClaimHandler(c.client),
		"ingress":                          resources.NewIngressHandler(c.client),
		"horizontalpodautoscaler":          resources.NewHorizontalPodAutoscalerHandler(c.client),
		"serviceaccount":                   resources.NewServiceAccountHandler(c.client),
		"endpoints":                        resources.NewEndpointsHandler(c.client),
		"persistentvolume":                 resources.NewPersistentVolumeHandler(c.client),
		"resourcequota":                    resources.NewResourceQuotaHandler(c.client),
		"poddisruptionbudget":              resources.NewPodDisruptionBudgetHandler(c.client),
		"storageclass":                     resources.NewStorageClassHandler(c.client),
		"networkpolicy":                    resources.NewNetworkPolicyHandler(c.client),
		"replicationcontroller":            resources.NewReplicationControllerHandler(c.client),
		"limitrange":                       resources.NewLimitRangeHandler(c.client),
		"lease":                            resources.NewLeaseHandler(c.client),
		"role":                             resources.NewRoleHandler(c.client),
		"clusterrole":                      resources.NewClusterRoleHandler(c.client),
		"rolebinding":                      resources.NewRoleBindingHandler(c.client),
		"clusterrolebinding":               resources.NewClusterRoleBindingHandler(c.client),
		"volumeattachment":                 resources.NewVolumeAttachmentHandler(c.client),
		"certificatesigningrequest":        resources.NewCertificateSigningRequestHandler(c.client),
		"namespace":                        resources.NewNamespaceHandler(c.client),
		"daemonset":                        resources.NewDaemonSetHandler(c.client),
		"statefulset":                      resources.NewStatefulSetHandler(c.client),
		"replicaset":                       resources.NewReplicaSetHandler(c.client),
		"mutatingwebhookconfiguration":     resources.NewMutatingWebhookConfigurationHandler(c.client),
		"validatingwebhookconfiguration":   resources.NewValidatingWebhookConfigurationHandler(c.client),
		"ingressclass":                     resources.NewIngressClassHandler(c.client),
		"priorityclass":                    resources.NewPriorityClassHandler(c.client),
		"runtimeclass":                     resources.NewRuntimeClassHandler(c.client),
		"validatingadmissionpolicy":        resources.NewValidatingAdmissionPolicyHandler(c.client),
		"validatingadmissionpolicybinding": resources.NewValidatingAdmissionPolicyBindingHandler(c.client),
	}

	for resourceName, handler := range handlers {
		c.handlers[resourceName] = handler
	}
}

// Run starts the informers and collection loop
func (c *Collector) Run(ctx context.Context) error {
	klog.Info("Starting kube-state-logs with individual tickers...")

	// Setup informers for each configured resource type
	for _, resourceType := range c.config.Resources {
		handler, exists := c.handlers[resourceType]
		if !exists {
			klog.Warningf("No handler found for resource type: %s", resourceType)
			continue
		}

		// Setup informer with no resync period
		if err := handler.SetupInformer(c.factory, c.logger, 0); err != nil {
			klog.Errorf("Failed to setup informer for %s: %v", resourceType, err)
			continue
		}
	}

	// Start the informer factory
	c.factory.Start(c.stopCh)

	// Wait for all informers to sync
	klog.Info("Waiting for informers to sync...")
	synced := c.factory.WaitForCacheSync(c.stopCh)
	for resourceType, isSynced := range synced {
		if !isSynced {
			return fmt.Errorf("failed to sync informer for %v", resourceType)
		}
	}

	klog.Info("All informers synced successfully")

	// Start individual tickers for each resource
	c.startResourceTickers(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	close(c.stopCh)
	c.wg.Wait()
	return ctx.Err()
}

// startResourceTickers starts individual tickers for each resource based on their configured intervals
func (c *Collector) startResourceTickers(ctx context.Context) {
	// Create a map of resource names to their intervals
	resourceIntervals := make(map[string]time.Duration)

	// First, populate with specific resource configs
	for _, resourceConfig := range c.config.ResourceConfigs {
		resourceIntervals[resourceConfig.Name] = resourceConfig.Interval
	}

	// Then, ensure all resources in the Resources list have an interval (use default if not specified)
	for _, resourceName := range c.config.Resources {
		if _, exists := resourceIntervals[resourceName]; !exists {
			resourceIntervals[resourceName] = c.config.LogInterval
		}
	}

	// Start tickers for all resources
	for resourceName, interval := range resourceIntervals {
		// Check if we have a handler for this resource
		handler, exists := c.handlers[resourceName]
		if !exists {
			klog.Warningf("No handler found for resource type: %s", resourceName)
			continue
		}

		klog.Infof("Starting ticker for %s with interval %v", resourceName, interval)

		c.wg.Add(1)
		go func(name string, tickerInterval time.Duration, h interfaces.ResourceHandler) {
			defer c.wg.Done()

			ticker := time.NewTicker(tickerInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := c.collectAndLogResource(ctx, name, h); err != nil {
						klog.Errorf("Collection failed for %s: %v", name, err)
					}
				}
			}
		}(resourceName, interval, handler)
	}
}

// collectAndLogResource collects and logs data for a specific resource
func (c *Collector) collectAndLogResource(ctx context.Context, resourceName string, handler interfaces.ResourceHandler) error {
	entries, err := handler.Collect(ctx, c.config.Namespaces)
	if err != nil {
		return fmt.Errorf("failed to collect %s: %w", resourceName, err)
	}

	// Log all collected entries
	for _, entry := range entries {
		if err := c.logger.Log(entry); err != nil {
			klog.Errorf("Failed to log entry for %s: %v", resourceName, err)
		}
	}

	klog.V(2).Infof("Collected and logged %d entries for %s", len(entries), resourceName)
	return nil
}

// collectAndLog collects data from all configured resources and logs them
// This is now mainly used for initial collection or manual triggers
func (c *Collector) collectAndLog(ctx context.Context) error {
	var allEntries []types.LogEntry

	// Collect from each configured resource type
	for _, resourceType := range c.config.Resources {
		handler, exists := c.handlers[resourceType]
		if !exists {
			klog.Warningf("No handler found for resource type: %s", resourceType)
			continue
		}

		entries, err := handler.Collect(ctx, c.config.Namespaces)
		if err != nil {
			klog.Errorf("Failed to collect %s: %v", resourceType, err)
			continue
		}

		allEntries = append(allEntries, entries...)
	}

	// Log all collected entries
	for _, entry := range allEntries {
		if err := c.logger.Log(entry); err != nil {
			klog.Errorf("Failed to log entry: %v", err)
		}
	}

	klog.V(2).Infof("Collected and logged %d entries", len(allEntries))
	return nil
}
