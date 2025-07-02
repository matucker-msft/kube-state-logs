# Kube-State-Logs

A Kubernetes operator that generates structured logs containing cluster state metrics, similar to kube-state-metrics but outputting logs instead of Prometheus metrics.

## 🤖 AI-Assisted Development Notice

**Transparency Notice**: This project was primarily developed with the assistance of AI tools. While the core concepts, architecture decisions, and requirements were human-defined, the majority of the implementation code, documentation, and testing was generated with AI assistance. We believe in being transparent about this development approach and welcome contributions from both human developers and AI-assisted workflows.

## Overview

Kube-State-Logs is designed to provide the same rich Kubernetes resource state information that kube-state-metrics offers, but in a log-based format. Instead of exposing Prometheus metrics, it periodically logs the current state of Kubernetes resources with calculated metrics and conditions.

This approach is particularly useful for:
- Log aggregation systems that don't support Prometheus metrics
- Environments where you want to correlate Kubernetes state with application logs
- Systems that prefer structured JSON logs over time-series metrics
- Debugging and monitoring scenarios where log-based analysis is preferred

## Inspired by kube-state-metrics

🚀 **This project is heavily inspired by [kube-state-metrics](https://go.goms.io/aks/kube-state-metrics)**, the official Kubernetes project that exposes cluster state as Prometheus metrics. 

**Key differences:**
- **kube-state-metrics**: Exposes Prometheus metrics via HTTP endpoint
- **kube-state-logs**: Outputs structured JSON logs to stdout/stderr

**What we share:**
- ✅ **100% resource coverage** - All resources supported by kube-state-metrics
- ✅ **Same calculated metrics** - Replica counts, conditions, status fields
- ✅ **Same filtering capabilities** - Namespace and resource type filtering
- ✅ **Same RBAC requirements** - Identical permissions needed

**What we enhance:**
- 📊 **Richer data structure** - JSON objects instead of flat metrics
- 🔗 **Better relationships** - Owner references and object links
- ⏰ **Enhanced timestamps** - Creation, modification, and deletion times
- 📝 **Additional context** - Labels, annotations, and metadata arrays

For a detailed comparison, see [docs/comparison.md](docs/comparison.md).

## Documentation

📚 **Comprehensive documentation is available in the [docs/](docs/) directory:**

- **[📋 Resource Coverage](docs/resources.md)** - Complete list of all supported Kubernetes resources
- **[🚀 Deployment Guide](docs/deployment.md)** - Installation and configuration instructions
- **[📊 Comparison with kube-state-metrics](docs/comparison.md)** - Detailed comparison showing 100% coverage
- **[✨ Enhanced Features](docs/enhanced-fields.md)** - Additional fields and capabilities beyond kube-state-metrics
- **[📖 Quick Comparison](docs/comparison-summary.md)** - Summary comparison table

## Features

- **Comprehensive Resource Coverage**: Monitors all major Kubernetes resources including:
  - Deployments, ReplicaSets (current only), StatefulSets, DaemonSets
  - Pods, Services, Nodes, Namespaces
  - Jobs, CronJobs, ConfigMaps, Secrets
  - PersistentVolumeClaims, Ingresses, HorizontalPodAutoscalers, ServiceAccounts
  - Container-level metrics and states
  - RBAC resources (Roles, ClusterRoles, RoleBindings, ClusterRoleBindings)
  - Storage resources (PersistentVolumes, StorageClasses, VolumeAttachments)
  - Network resources (Endpoints, NetworkPolicies, IngressClasses)
  - Admission control resources (MutatingWebhookConfigurations, ValidatingWebhookConfigurations)
  - Security resources (CertificateSigningRequests, PodDisruptionBudgets)
  - Resource management (ResourceQuotas, LimitRanges, Leases)

- **Rich State Information**: Each log entry includes:
  - Resource metadata (name, namespace, labels, annotations)
  - Current state metrics (replicas, conditions, status)
  - Calculated metrics (availability, readiness, health)
  - Timestamps and generation information

- **Configurable Logging**: 
  - Adjustable logging intervals
  - **Individual resource intervals** - Different intervals for different resource types
  - Resource type filtering
  - Namespace filtering
  - Custom log levels

- **Structured JSON Output**: Machine-readable logs that integrate easily with log aggregation systems

- **Smart ReplicaSet Filtering**: Only logs current ReplicaSets to reduce noise from deployment revisions

## Supported Resources

kube-state-logs supports logging for the following Kubernetes resources:

- Pod
- Service
- Node
- Deployment
- Job
- CronJob
- ConfigMap
- Secret
- PersistentVolumeClaim
- PersistentVolume
- Ingress
- HorizontalPodAutoscaler
- ServiceAccount
- Endpoints
- ResourceQuota
- PodDisruptionBudget
- StorageClass
- NetworkPolicy
- ReplicationController
- LimitRange
- Lease
- Role
- ClusterRole
- RoleBinding
- ClusterRoleBinding
- VolumeAttachment
- CertificateSigningRequest
- Namespace
- DaemonSet
- StatefulSet
- ReplicaSet
- MutatingWebhookConfiguration
- ValidatingWebhookConfiguration
- IngressClass
- PriorityClass
- RuntimeClass
- ValidatingAdmissionPolicy
- ValidatingAdmissionPolicyBinding

### CRD Logging (Generic Custom Resource Support)

kube-state-logs can log any Kubernetes Custom Resource Definition (CRD) generically. You can specify which CRDs to log and which fields to extract using the `--crd-configs` flag or Helm values.

#### Example CLI usage:

```bash
--crd-configs="mygroup.example.com/v1:widgets:spec.size|spec.color,anothergroup.io/v1:foos:spec.enabled"
```

- This will log all CRD objects for the specified GVRs, including their metadata, spec, status, and any custom fields you list (dot-separated paths).

#### Example Helm values:

```yaml
config:
  crdConfigs:
    - apiVersion: mygroup.example.com/v1
      resource: widgets
      customFields:
        - spec.size
        - spec.color
    - apiVersion: anothergroup.io/v1
      resource: foos
      customFields:
        - spec.enabled
```

#### What is logged for CRDs?
- Metadata: name, namespace, labels, annotations, creation timestamp
- `spec` and `status` fields (full objects)
- Any custom fields you specify

See [docs/resources.md](docs/resources.md) for more details.

## Installation

### Using Helm (Recommended)

```bash
# Install from local chart
helm install kube-state-logs ./charts/kube-state-logs \
  --namespace monitoring \
  --create-namespace
```

### Using Docker

```bash
# Build the image
docker build -t kube-state-logs:latest .

# Run locally
docker run --rm -v ~/.kube/config:/root/.kube/config kube-state-logs:latest
```

## Configuration

The operator can be configured via command-line flags or Helm values:

```yaml
# values.yaml
image:
  repository: kube-state-logs
  tag: "0.1.0"

config:
  logInterval: "1m"  # Default interval for resources without specific configs
  resources: "deployments,pods,services,nodes,replicasets,statefulsets,daemonsets,namespaces,jobs,cronjobs,configmaps,secrets,persistentvolumeclaims,ingresses,horizontalpodautoscalers,serviceaccounts"
  resourceConfigs: "deployments:5m,pods:1m,services:2m"  # Individual resource intervals
  namespaces: ""  # Empty for all namespaces
  logLevel: "info"

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### Command-line Options

```bash
./kube-state-logs \
  --log-interval=1m \
  --resources=deployments,pods,services,nodes,replicasets,statefulsets,daemonsets,namespaces,jobs,cronjobs,configmaps,secrets,persistentvolumeclaims,ingresses,horizontalpodautoscalers,serviceaccounts \
  --resource-configs=deployments:5m,pods:1m,services:2m \
  --namespaces=default,kube-system \
  --log-level=info \
  --kubeconfig=/path/to/kubeconfig
```

### Individual Resource Intervals

You can specify different logging intervals for different resource types using the `--resource-configs` flag:

```bash
# Format: resource:interval,resource:interval
--resource-configs=deployments:5m,pods:1m,services:2m,namespaces:10m
```

**Examples:**
- `--resource-configs=deployments:5m,pods:1m` - Deployments every 5 minutes, pods every 1 minute
- `--resource-configs=namespaces:10m` - Only namespaces every 10 minutes
- `--resource-configs=pods:30s,services:2m` - Pods every 30 seconds, services every 2 minutes
- `--resource-configs=horizontalpodautoscalers:30s,persistentvolumeclaims:5m` - HPAs every 30 seconds, PVCs every 5 minutes
- `--resource-configs=ingresses:1m,serviceaccounts:10m` - Ingresses every 1 minute, service accounts every 10 minutes

**Rules:**
- Resources not specified in `--resource-configs` use the `--log-interval` value
- All resources listed in `--resources` will be monitored
- Intervals can use standard time units: `s`, `m`, `h` (e.g., `30s`, `5m`, `2h`)

## Usage

Once deployed, kube-state-logs will start generating logs at the configured interval. You can view the logs using:

```bash
kubectl logs -f deployment/kube-state-logs -n monitoring
```

## Example Output

### Deployment Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "deployment",
    "name": "sample-deployment",
    "namespace": "my-sample-namespace",
    "createdTimestamp": 1718020800,
    "labels": {
        "app": "sample-deployment"
    },
    "annotations": {
        "deployment.kubernetes.io/revision": "1"
    },
    "createdByKind": "",
    "createdByName": "",
    "desiredReplicas": 3,
    "currentReplicas": 3,
    "readyReplicas": 3,
    "availableReplicas": 3,
    "unavailableReplicas": 0,
    "updatedReplicas": 3,
    "observedGeneration": 8,
    "replicasDesired": 3,
    "replicasAvailable": 3,
    "replicasUnavailable": 0,
    "replicasUpdated": 3,
    "strategyType": "RollingUpdate",
    "strategyRollingUpdateMaxSurge": 1,
    "strategyRollingUpdateMaxUnavailable": 1,
    "conditionAvailable": true,
    "conditionProgressing": true,
    "conditionReplicaFailure": false,
    "paused": false,
    "metadataGeneration": 1
}
```

For comprehensive examples of all supported resource types (pods, containers, services, nodes, replicasets, statefulsets, daemonsets, namespaces), see [docs/resources.md](docs/resources.md).

## Integration

### Log Aggregation Systems

Kube-State-Logs integrates seamlessly with popular log aggregation systems:

- **ELK Stack**: Use Logstash to parse and index the JSON logs
- **Fluentd/Fluent Bit**: Configure parsers for structured JSON input
- **Splunk**: Use JSON extraction for field parsing
- **Datadog**: Automatic JSON log parsing and correlation

### Monitoring and Alerting

Use your existing log-based monitoring tools to:
- Set up alerts based on resource state changes
- Create dashboards showing cluster health
- Correlate application logs with Kubernetes state
- Track resource availability over time

## Development

### Building from Source

```bash
git clone https://go.goms.io/aks/kube-state-metrics.git
cd kube-state-logs
make build
```

### Running Locally

```bash
make run
```

### Testing

```bash
make test
```

## Deployment

For detailed deployment instructions, see [docs/deployment.md](docs/deployment.md).

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Related Projects

- **[kube-state-metrics](https://go.goms.io/aks/kube-state-metrics)** - The original Prometheus metrics exporter that inspired this project. This is the official Kubernetes project that exposes cluster state as Prometheus metrics.
- [kubernetes](https://github.com/kubernetes/kubernetes) - The Kubernetes project
