# Kube-State-Logs

A Kubernetes operator that generates structured logs containing cluster state metrics, similar to kube-state-metrics but outputting logs instead of Prometheus metrics.

## Overview

Kube-State-Logs is designed to provide the same rich Kubernetes resource state information that kube-state-metrics offers, but in a log-based format. Instead of exposing Prometheus metrics, it periodically logs the current state of Kubernetes resources with calculated metrics and conditions.

This approach is particularly useful for:
- Log aggregation systems that don't support Prometheus metrics
- Environments where you want to correlate Kubernetes state with application logs
- Systems that prefer structured JSON logs over time-series metrics
- Debugging and monitoring scenarios where log-based analysis is preferred

## Features

- **Comprehensive Resource Coverage**: Monitors all major Kubernetes resources including:
  - Deployments, ReplicaSets (current only), StatefulSets, DaemonSets
  - Pods, Services, Nodes, Namespaces
  - Container-level metrics and states

- **Rich State Information**: Each log entry includes:
  - Resource metadata (name, namespace, labels, annotations)
  - Current state metrics (replicas, conditions, status)
  - Calculated metrics (availability, readiness, health)
  - Timestamps and generation information

- **Configurable Logging**: 
  - Adjustable logging intervals
  - Resource type filtering
  - Namespace filtering
  - Custom log levels

- **Structured JSON Output**: Machine-readable logs that integrate easily with log aggregation systems

- **Smart ReplicaSet Filtering**: Only logs current ReplicaSets to reduce noise from deployment revisions

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
  logInterval: "1m"
  resources: "deployments,pods,services,nodes,replicasets,statefulsets,daemonsets,namespaces"
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
  --resources=deployments,pods,services,nodes \
  --namespaces=default,kube-system \
  --log-level=info \
  --kubeconfig=/path/to/kubeconfig
```

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
    "data": {
    "createdTimestamp": 1718020800,
    "labels": {
        "app": "sample-deployment"
    },
    "desiredReplicas": 3,
    "currentReplicas": 3,
    "readyReplicas": 3,
    "availableReplicas": 3,
    "unavailableReplicas": 0,
    "updatedReplicas": 3,
    "observedGeneration": 8,
    "conditionAvailable": true,
    "conditionProgressing": true,
        "conditionReplicaFailure": false,
        "strategyType": "RollingUpdate"
}
}
```

### Pod Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "pod",
    "name": "sample-pod-abc123",
    "namespace": "default",
    "data": {
        "nodeName": "worker-node-1",
        "phase": "Running",
        "ready": true,
        "restartCount": 0,
        "podIP": "10.244.0.5",
        "qosClass": "Burstable",
        "conditionAvailable": true,
        "conditionProgressing": false,
        "labels": {
            "app": "sample-app"
        }
    }
}
```

### Container Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "container",
    "name": "app-container",
    "namespace": "default",
    "data": {
        "podName": "sample-pod-abc123",
        "image": "nginx:latest",
        "ready": true,
        "restartCount": 0,
        "state": "running",
        "stateRunning": true,
        "resourceRequests": {
            "cpu": "100m",
            "memory": "128Mi"
        },
        "resourceLimits": {
            "cpu": "500m",
            "memory": "512Mi"
        }
    }
}
```

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
git clone https://github.com/matucker-msft/kube-state-logs.git
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

For detailed deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md).

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) - The original Prometheus metrics exporter
- [kubernetes](https://github.com/kubernetes/kubernetes) - The Kubernetes project
