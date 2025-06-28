# Kubernetes API Defaults and Behaviors

This document explains the Kubernetes API defaults and behaviors that kube-state-logs implements to match the official Kubernetes API specification.

## Resource Replicas Defaults

### ReplicaSet
- **Default replicas**: `1` when `spec.replicas` is not specified
- **Reference**: [ReplicaSet Basics](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#replicaset-basics)
- **Implementation**: `pkg/collector/resources/replicaset.go`

### Deployment
- **Default replicas**: `1` when `spec.replicas` is not specified
- **Reference**: [Creating a Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#creating-a-deployment)
- **Implementation**: `pkg/collector/resources/deployment.go`

### StatefulSet
- **Default replicas**: `1` when `spec.replicas` is not specified
- **Reference**: [Creating a StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#creating-a-statefulset)
- **Implementation**: `pkg/collector/resources/statefulset.go`

## Pod Behavior

### QoS Class Default
- **Default QoS**: `BestEffort` when `status.qosClass` is not set
- **Reference**: [Pod QoS Classes](https://kubernetes.io/docs/concepts/workloads/pods/pod-qos/#qos-classes)
- **Implementation**: `pkg/collector/resources/pod.go`

### Status Reason Logic
- **Priority order**:
  1. `status.reason` (if set)
  2. First condition with `status: False` and non-empty `reason`
  3. First container with terminated state and non-empty `reason`
- **Reference**: [Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase)
- **Implementation**: `pkg/collector/resources/pod.go`

## Service Behavior

### Target Port Handling
- **IntOrString behavior**: 
  - Integer values: use the specified port
  - String values: default to `0` (port name resolution not implemented)
- **Reference**: [Defining a Service](https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service)
- **Implementation**: `pkg/collector/resources/service.go`

## Node Behavior

### Node Phase
- **Default phase**: `Unknown` when `status.phase` is not set
- **Valid phases**: `Pending`, `Running`, `Terminated`
- **Reference**: [Node Status](https://kubernetes.io/docs/concepts/architecture/nodes/#node-status)
- **Implementation**: `pkg/collector/resources/node.go`

## Why These Defaults Matter

1. **API Consistency**: Matching Kubernetes API defaults ensures our logs accurately reflect the cluster state
2. **Predictable Behavior**: Users can rely on consistent behavior regardless of how resources are created
3. **Debugging**: Understanding defaults helps with troubleshooting resource issues
4. **Compliance**: Following official Kubernetes specifications ensures compatibility

## Implementation Notes

- All defaults are documented with links to official Kubernetes documentation
- Nil checks are implemented to prevent panics when optional fields are not set
- Default values are applied consistently across all resource handlers
- Comments in code reference the relevant Kubernetes documentation sections 