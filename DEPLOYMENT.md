# Deployment Guide

This guide covers deploying kube-state-logs to your Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured to access your cluster
- Helm 3.x (for Helm deployment)
- Docker (for building images)

## Quick Start

### Using Helm (Recommended)

1. **Install using Helm**:
   ```bash
   helm install kube-state-logs ./charts/kube-state-logs \
     --namespace monitoring \
     --create-namespace
   ```

2. **Using GitHub Container Registry** (if available):
   ```bash
   helm install kube-state-logs ./charts/kube-state-logs \
     --namespace monitoring \
     --create-namespace \
     --set image.repository=ghcr.io/matucker-msft/kube-state-logs \
     --set image.tag=main
   ```

### Using Docker

1. **Build the Docker image**:
   ```bash
   docker build -t kube-state-logs:latest .
   ```

2. **Push to registry** (optional):
   ```bash
   docker tag kube-state-logs:latest your-registry/kube-state-logs:latest
   docker push your-registry/kube-state-logs:latest
   ```

3. **Run locally with Docker**:
   ```bash
   docker run --rm -v ~/.kube/config:/root/.kube/config kube-state-logs:latest
   ```

## Configuration

### Helm Values

Create a `values.yaml` file to customize the deployment:

```yaml
# Image configuration
image:
  repository: kube-state-logs
  tag: "0.1.0"

# Application configuration
config:
  logInterval: "1m"
  resources: "deployments,pods,services,nodes,replicasets,statefulsets,daemonsets,namespaces"
  namespaces: ""  # Empty for all namespaces
  logLevel: "info"

# Resource limits
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## Monitoring Specific Namespaces

To monitor only specific namespaces:

```yaml
config:
  namespaces: "kube-system,default,monitoring"
```

## Custom Resource Selection

To monitor only specific resources:

```yaml
config:
  resources: "deployments,pods,services"
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=kube-state-logs
```

### View Logs

```bash
kubectl logs -f deployment/kube-state-logs -n monitoring
```

### Check RBAC

```bash
kubectl auth can-i list deployments --as=system:serviceaccount:monitoring:kube-state-logs
kubectl auth can-i list pods --as=system:serviceaccount:monitoring:kube-state-logs
```

### Common Issues

1. **Permission Denied**: Ensure RBAC is properly configured
2. **No Logs**: Check if the application is running and has proper permissions
3. **High Resource Usage**: Adjust resource limits in values.yaml

## Development

### Local Development

1. **Run locally**:
   ```bash
   go run . --log-level=debug --log-interval=10s
   ```

2. **Build and test**:
   ```bash
   make build
   make test
   ```

3. **Docker build**:
   ```bash
   make docker-build
   ```

### Kind Cluster Testing

1. **Create kind cluster**:
   ```bash
   make kind-create
   ```

2. **Load image**:
   ```bash
   make kind-load
   ```

3. **Deploy**:
   ```bash
   make helm-install
   ```

## CI/CD

The project includes GitHub Actions workflows for:

- Go code formatting (`go fmt`)
- Go unit tests
- Helm chart linting
- Docker image building and pushing to GitHub Container Registry (on main branch)

### Available Images

Images are automatically built and pushed to GitHub Container Registry:
- `ghcr.io/matucker-msft/kube-state-logs:main` - Latest from main branch
- `ghcr.io/matucker-msft/kube-state-logs:sha-<commit>` - Specific commit

## Support

For issues and questions:

- GitHub Issues: [Create an issue](https://github.com/matucker-msft/kube-state-logs/issues)
- Documentation: Check the README.md
- Helm Chart: Check the values.yaml for all available options 