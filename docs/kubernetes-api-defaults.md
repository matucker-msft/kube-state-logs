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

### ReplicationController
- **Default replicas**: `1` when `spec.replicas` is not specified
- **Reference**: [ReplicationController](https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller/#replicationcontroller)
- **Implementation**: `pkg/collector/resources/replicationcontroller.go`

## Job and CronJob Behavior

### Job Backoff Limit
- **Default backoff limit**: `6` when `spec.backoffLimit` is not specified
- **Reference**: [Pod Backoff Failure Policy](https://kubernetes.io/docs/concepts/workloads/controllers/job/#pod-backoff-failure-policy)
- **Implementation**: `pkg/collector/resources/job.go`

### CronJob Concurrency Policy
- **Default concurrency policy**: `Allow` when `spec.concurrencyPolicy` is not set
- **Reference**: [Concurrency Policy](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#concurrency-policy)
- **Implementation**: `pkg/collector/resources/cronjob.go`

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

## Storage Behavior

### PersistentVolumeClaim Access Modes
- **Access modes**: `ReadWriteOnce`, `ReadOnlyMany`, `ReadWriteMany`
- **Reference**: [Access Modes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes)
- **Implementation**: `pkg/collector/resources/persistentvolumeclaim.go`

### PersistentVolumeClaim Phase
- **Phases**: `Pending`, `Bound`, `Lost`
- **Reference**: [PVC Phase](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#phase)
- **Implementation**: `pkg/collector/resources/persistentvolumeclaim.go`

### StorageClass Defaults
- **Reclaim Policy**: `Delete` when `reclaimPolicy` is nil
- **Volume Binding Mode**: `Immediate` when `volumeBindingMode` is nil
- **Allow Volume Expansion**: `false` when `allowVolumeExpansion` is nil
- **Reference**: [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/)
- **Implementation**: `pkg/collector/resources/storageclass.go`

### VolumeAttachment
- **Attachment metadata**: Contains provider-specific information
- **Reference**: [Volume Attachments](https://kubernetes.io/docs/concepts/storage/volume-attachments/)
- **Implementation**: `pkg/collector/resources/volumeattachment.go`

## Networking Behavior

### Ingress Path Type
- **Default path type**: `ImplementationSpecific` when `pathType` is not specified
- **Reference**: [Path Types](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types)
- **Implementation**: `pkg/collector/resources/ingress.go`

### IngressClass Default
- **Default class**: Determined by `ingressclass.kubernetes.io/is-default-class` annotation
- **Reference**: [Default Ingress Class](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-ingress-class)
- **Implementation**: `pkg/collector/resources/ingressclass.go`

### NetworkPolicy Policy Types
- **Default policy types**: Includes `Ingress` when `policyTypes` is not specified
- **Reference**: [Default Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/#default-policies)
- **Implementation**: `pkg/collector/resources/networkpolicy.go`

### Secret Types
- **Secret types**: `Opaque`, `kubernetes.io/service-account-token`, `kubernetes.io/dockercfg`, etc.
- **Reference**: [Secret Types](https://kubernetes.io/docs/concepts/configuration/secret/#secret-types)
- **Implementation**: `pkg/collector/resources/secret.go`

## Autoscaling Behavior

### HorizontalPodAutoscaler Min Replicas
- **Default min replicas**: `1` when `spec.minReplicas` is not specified
- **Reference**: [Horizontal Pod Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- **Implementation**: `pkg/collector/resources/horizontalpodautoscaler.go`

## Authentication and Authorization

### ServiceAccount Automount Token
- **Default automount**: `true` when `automountServiceAccountToken` is not specified
- **Reference**: [Service Account Token](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server)
- **Implementation**: `pkg/collector/resources/serviceaccount.go`

### Role and ClusterRole Rules
- **Rules**: Define permissions for resources and verbs
- **Reference**: [Role and ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
- **Implementation**: `pkg/collector/resources/role.go`, `pkg/collector/resources/clusterrole.go`

### RoleBinding and ClusterRoleBinding
- **Role references**: Link roles to subjects (users, groups, service accounts)
- **Reference**: [RoleBinding and ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding)
- **Implementation**: `pkg/collector/resources/rolebinding.go`, `pkg/collector/resources/clusterrolebinding.go`

### CertificateSigningRequest
- **Usages**: Define how the certificate can be used
- **Reference**: [Certificate Signing Requests](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#kubernetes-signers)
- **Implementation**: `pkg/collector/resources/certificatesigningrequest.go`

## Policy and Quotas

### ResourceQuota Scopes
- **Scopes**: `BestEffort`, `NotBestEffort`, `Terminating`, `NotTerminating`, etc.
- **Reference**: [Quota Scopes](https://kubernetes.io/docs/concepts/policy/resource-quotas/#quota-scopes)
- **Implementation**: `pkg/collector/resources/resourcequota.go`

### PodDisruptionBudget Min Available/Max Unavailable
- **MinAvailable/MaxUnavailable**: Can be specified as absolute numbers or percentages
- **Reference**: [Pod Disruption Budgets](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets)
- **Implementation**: `pkg/collector/resources/poddisruptionbudget.go`

### LimitRange
- **Limit types**: `Container`, `Pod`, `PersistentVolumeClaim`
- **Reference**: [Limit Range](https://kubernetes.io/docs/concepts/policy/limit-range/#limit-range)
- **Implementation**: `pkg/collector/resources/limitrange.go`

## Scheduling and Coordination

### PriorityClass Preemption Policy
- **Default preemption policy**: `PreemptLowerOrEqual` when not specified
- **Reference**: [Preemption Policies](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#preemption-policies)
- **Implementation**: `pkg/collector/resources/priorityclass.go`

### Lease Duration
- **Default lease duration**: `15` seconds when `leaseDurationSeconds` is nil
- **Reference**: [Leases](https://kubernetes.io/docs/concepts/architecture/leases/)
- **Implementation**: `pkg/collector/resources/lease.go`

### RuntimeClass
- **Handler**: Specifies the container runtime implementation
- **Reference**: [Runtime Class](https://kubernetes.io/docs/concepts/containers/runtime-class/)
- **Implementation**: `pkg/collector/resources/runtimeclass.go`

## Admission Control

### MutatingWebhookConfiguration
- **Webhooks**: Modify objects during admission
- **Reference**: [Mutating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
- **Implementation**: `pkg/collector/resources/mutatingwebhookconfiguration.go`

### ValidatingWebhookConfiguration
- **Webhooks**: Validate objects during admission
- **Reference**: [Validating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook)
- **Implementation**: `pkg/collector/resources/validatingwebhookconfiguration.go`

### ValidatingAdmissionPolicy
- **Policy rules**: Define validation rules using CEL expressions
- **Reference**: [Validating Admission Policy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/)
- **Implementation**: `pkg/collector/resources/validatingadmissionpolicy.go`

### ValidatingAdmissionPolicyBinding
- **Policy binding**: Links policies to resources
- **Reference**: [Validating Admission Policy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/)
- **Implementation**: `pkg/collector/resources/validatingadmissionpolicybinding.go`

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