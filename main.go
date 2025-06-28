package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/matucker-msft/kube-state-logs/pkg/collector"
	"github.com/matucker-msft/kube-state-logs/pkg/config"
)

func main() {
	// Parse command line flags
	var (
		logInterval     = flag.Duration("log-interval", 1*time.Minute, "Default interval between log outputs")
		resources       = flag.String("resources", "pod,service,node,deployment,job,cronjob,configmap,secret,persistentvolumeclaim,ingress,horizontalpodautoscaler,serviceaccount,endpoints,persistentvolume,resourcequota,poddisruptionbudget,storageclass,networkpolicy,replicationcontroller,limitrange,lease,role,clusterrole,rolebinding,clusterrolebinding,volumeattachment,certificatesigningrequest,mutatingwebhookconfiguration,validatingwebhookconfiguration,ingressclass", "Comma-separated list of resources to monitor")
		resourceConfigs = flag.String("resource-configs", "", "Comma-separated list of resource:interval pairs (e.g., 'deployments:5m,pods:1m,services:2m'). If not specified, uses log-interval for all resources.")
		namespaces      = flag.String("namespaces", "", "Comma-separated list of namespaces to monitor (empty for all)")
		logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		kubeconfig      = flag.String("kubeconfig", "", "Path to kubeconfig file (empty for in-cluster config)")
	)
	flag.Parse()

	// Set log level
	if err := config.SetLogLevel(*logLevel); err != nil {
		klog.Fatalf("Failed to set log level: %v", err)
	}

	klog.Info("Starting kube-state-logs...")

	// Parse resource configurations
	resourceConfigsList := config.ParseResourceConfigs(*resourceConfigs, *logInterval)

	// If no specific resource configs provided, create default ones from resources list
	if len(resourceConfigsList) == 0 {
		resourcesList := config.ParseResourceList(*resources)
		for _, resource := range resourcesList {
			resourceConfigsList = append(resourceConfigsList, config.ResourceConfig{
				Name:     resource,
				Interval: *logInterval,
			})
		}
	}

	// Create configuration
	cfg := &config.Config{
		LogInterval:     *logInterval,
		Resources:       config.ParseResourceList(*resources),
		ResourceConfigs: resourceConfigsList,
		Namespaces:      config.ParseNamespaceList(*namespaces),
		Kubeconfig:      *kubeconfig,
	}

	// Create collector
	collector, err := collector.New(cfg)
	if err != nil {
		klog.Fatalf("Failed to create collector: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		klog.Infof("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Start the collector
	if err := collector.Run(ctx); err != nil {
		klog.Fatalf("Collector failed: %v", err)
	}

	klog.Info("kube-state-logs stopped")
}
