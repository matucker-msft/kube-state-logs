package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matucker-msft/kube-state-logs/pkg/collector"
	"github.com/matucker-msft/kube-state-logs/pkg/config"
)

func main() {
	// Parse command line flags
	var (
		kubeconfig      = flag.String("kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not specified)")
		logInterval     = flag.Duration("log-interval", 30*time.Second, "Interval between log outputs")
		namespaces      = flag.String("namespaces", "", "Comma-separated list of namespaces to monitor (empty for all)")
		resources       = flag.String("resources", "", "Comma-separated list of resources to collect (empty for all)")
		resourceConfigs = flag.String("resource-configs", "", "Comma-separated list of resource:interval pairs (e.g., 'pods:30s,services:60s')")
		crdConfigs      = flag.String("crd-configs", "", "Comma-separated list of CRD configurations (e.g., 'apps/v1:deployments:spec.replicas|spec.template.spec.containers')")
		logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Default resources to collect
	defaultResources := []string{
		"pods",
		"services",
		"endpoints",
		"nodes",
		"deployments",
		"jobs",
		"cronjobs",
		"configmaps",
		"secrets",
		"persistentvolumeclaims",
		"persistentvolumes",
		"resourcequotas",
		"poddisruptionbudgets",
		"ingresses",
		"horizontalpodautoscalers",
		"serviceaccounts",
	}

	// Set log level
	if err := config.SetLogLevel(*logLevel); err != nil {
		log.Fatalf("Failed to set log level: %v", err)
	}

	// Parse configuration
	cfg := &config.Config{
		LogInterval:     *logInterval,
		Resources:       config.ParseResourceList(*resources),
		ResourceConfigs: config.ParseResourceConfigs(*resourceConfigs, *logInterval),
		CRDs:            config.ParseCRDConfigs(*crdConfigs),
		Namespaces:      config.ParseNamespaceList(*namespaces),
		Kubeconfig:      *kubeconfig,
	}

	// If no resources specified, use defaults
	if len(cfg.Resources) == 0 {
		cfg.Resources = defaultResources
	}

	// Create collector
	c, err := collector.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run in a goroutine
	go func() {
		if err := c.Run(ctx); err != nil {
			log.Printf("Collector error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()

	// Give some time for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	log.Println("Shutdown complete")
}
