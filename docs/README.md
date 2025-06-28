# kube-state-logs Documentation

This directory contains comprehensive documentation for kube-state-logs, a Kubernetes logging tool that outputs structured JSON logs instead of Prometheus metrics.

## Documentation Index

### Core Documentation
- **[Resource Coverage](RESOURCES.md)** - Complete list of all Kubernetes resources supported by kube-state-logs
- **[Deployment Guide](DEPLOYMENT.md)** - How to deploy and configure kube-state-logs in your Kubernetes cluster
- **[ksm-comparison.md**](ksm-comparison.md)** - High-level comparison with kube-state-metrics
- **[ksm-detailed-comparison.md**](ksm-detailed-comparison.md)** - Detailed resource-by-resource comparison with kube-state-metrics
- **[ksl-enhanced-fields.md**](ksl-enhanced-fields.md)** - Documentation of enhanced fields provided by kube-state-logs

### Comparison with kube-state-metrics
- **[Detailed Comparison](KSM_DETAILED_COMPARISON.md)** - Comprehensive comparison of kube-state-logs vs kube-state-metrics metrics
- **[Resource Comparison](KSM_COMPARISON.md)** - Summary comparison showing 100% coverage of kube-state-metrics resources

### Enhanced Features
- **[Enhanced Fields](KSL_ENHANCED_FIELDS.md)** - Detailed documentation of additional fields and enhancements provided by kube-state-logs

### Implementation Details
- **[kubernetes-api-defaults.md**](kubernetes-api-defaults.md)** - Kubernetes API defaults and behaviors implemented in kube-state-logs

## Quick Start

1. **Deployment**: See [Deployment Guide](DEPLOYMENT.md) for installation instructions
2. **Resource Coverage**: Check [Resource Coverage](RESOURCES.md) to see all supported Kubernetes resources
3. **Comparison**: Review [Detailed Comparison](KSM_DETAILED_COMPARISON.md) to understand how kube-state-logs relates to kube-state-metrics
4. **Features**: Explore [Enhanced Fields](KSL_ENHANCED_FIELDS.md) to see additional capabilities

## Key Benefits

- **100% Coverage**: Complete parity with kube-state-metrics resource coverage
- **Structured Logs**: JSON output instead of Prometheus metrics
- **Enhanced Data**: Richer context and additional fields
- **Better Relationships**: Improved object relationship tracking
- **Modern Structure**: Enhanced timestamps and metadata
- **Kubernetes API Compliance**: Follows official Kubernetes specifications and defaults 