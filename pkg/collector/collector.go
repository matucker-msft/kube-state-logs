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
	handlers map[string]ResourceHandler
	factory  informers.SharedInformerFactory
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// ResourceHandler defines the interface for resource-specific collectors
type ResourceHandler interface {
	SetupInformer(factory informers.SharedInformerFactory, logger interfaces.Logger, resyncPeriod time.Duration) error
	Collect(ctx context.Context, namespaces []string) ([]types.LogEntry, error)
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
		handlers: make(map[string]ResourceHandler),
		factory:  factory,
		stopCh:   make(chan struct{}),
	}

	// Register resource handlers
	c.registerHandlers()

	return c, nil
}

// registerHandlers registers all available resource handlers
func (c *Collector) registerHandlers() {
	c.handlers["deployments"] = resources.NewDeploymentHandler(c.client)
	c.handlers["pods"] = resources.NewPodHandler(c.client)
	c.handlers["services"] = resources.NewServiceHandler(c.client)
	c.handlers["nodes"] = resources.NewNodeHandler(c.client)
	c.handlers["replicasets"] = resources.NewReplicaSetHandler(c.client)
	c.handlers["statefulsets"] = resources.NewStatefulSetHandler(c.client)
	c.handlers["daemonsets"] = resources.NewDaemonSetHandler(c.client)
	c.handlers["namespaces"] = resources.NewNamespaceHandler(c.client)
	c.handlers["jobs"] = resources.NewJobHandler(c.client)
	c.handlers["cronjobs"] = resources.NewCronJobHandler(c.client)
	c.handlers["configmaps"] = resources.NewConfigMapHandler(c.client)
	c.handlers["secrets"] = resources.NewSecretHandler(c.client)
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
		go func(name string, tickerInterval time.Duration, h ResourceHandler) {
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
func (c *Collector) collectAndLogResource(ctx context.Context, resourceName string, handler ResourceHandler) error {
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
