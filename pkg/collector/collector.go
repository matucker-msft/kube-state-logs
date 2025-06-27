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

	// Create shared informer factory with resync period from command line flag
	// This ensures the cache is refreshed every interval to keep data current
	factory := informers.NewSharedInformerFactory(client, cfg.LogInterval)

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
	klog.Info("Starting kube-state-logs with informers...")

	// Setup informers for each configured resource type
	for _, resourceType := range c.config.Resources {
		handler, exists := c.handlers[resourceType]
		if !exists {
			klog.Warningf("No handler found for resource type: %s", resourceType)
			continue
		}

		if err := handler.SetupInformer(c.factory, c.logger, c.config.LogInterval); err != nil {
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

	// Start periodic collection loop
	ticker := time.NewTicker(c.config.LogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(c.stopCh)
			c.wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			if err := c.collectAndLog(ctx); err != nil {
				klog.Errorf("Collection failed: %v", err)
			}
		}
	}
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
