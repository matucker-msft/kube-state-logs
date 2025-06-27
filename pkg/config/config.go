package config

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Config holds the configuration for kube-state-logs
type Config struct {
	LogInterval time.Duration
	Resources   []string
	Namespaces  []string
	Kubeconfig  string
}

// ParseResourceList parses a comma-separated string into a slice of resource types
func ParseResourceList(resources string) []string {
	if resources == "" {
		return []string{}
	}
	return strings.Split(resources, ",")
}

// ParseNamespaceList parses a comma-separated string into a slice of namespace names
func ParseNamespaceList(namespaces string) []string {
	if namespaces == "" {
		return []string{}
	}
	return strings.Split(namespaces, ",")
}

// SetLogLevel sets the klog verbosity level
func SetLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug":
		klog.InitFlags(nil)
		// Set to maximum verbosity for debug
		klog.V(10).Info("Debug logging enabled")
	case "info":
		klog.InitFlags(nil)
		// Default level
	case "warn":
		klog.InitFlags(nil)
		// Reduce verbosity for warnings only
	case "error":
		klog.InitFlags(nil)
		// Only show errors
	default:
		return fmt.Errorf("invalid log level: %s", level)
	}
	return nil
}
