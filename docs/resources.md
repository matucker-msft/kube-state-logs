# Kube-State-Logs Resource Examples

This document provides comprehensive examples of all resource types and their data fields that kube-state-logs collects from your Kubernetes cluster.

## Supported Resources

Kube-state-logs supports the following Kubernetes resource types:

- **Deployments** - Application deployment state and scaling information
- **Pods** - Pod lifecycle, scheduling, and status information  
- **Containers** - Individual container state and resource usage
- **Init Containers** - Initialization container state and completion status
- **Services** - Service configuration, endpoints, and load balancer information
- **Nodes** - Node hardware, capacity, and health information
- **ReplicaSets** - Replica set scaling and availability status
- **StatefulSets** - Stateful application deployment and storage information
- **DaemonSets** - Daemon set scheduling and node coverage information
- **Namespaces** - Namespace lifecycle and status information
- **Jobs** - Batch job execution and completion status
- **CronJobs** - Scheduled job configuration and execution history
- **ConfigMaps** - Configuration data and metadata
- **Secrets** - Sensitive data configuration and metadata
- **PersistentVolumeClaims** - Storage claims and their binding status
- **Ingresses** - External access configuration and load balancer status
- **HorizontalPodAutoscalers** - Autoscaling configuration and current metrics
- **ServiceAccounts** - RBAC service accounts and their associated secrets
- **PodDisruptionBudgets** - Pod disruption budget configuration and status
- **Endpoints** - Service endpoint addresses, ports, and readiness status
- **PersistentVolumes** - Storage volume capacity, access modes, and binding status
- **ResourceQuotas** - Namespace resource limits and current usage
- **StorageClasses** - Storage provisioner configuration and parameters
- **NetworkPolicies** - Network security rules and policy types
- **ReplicationControllers** - Legacy replication controller scaling and availability
- **LimitRanges** - Namespace resource limit ranges and defaults
- **Leases** - Leader election coordination and lease information
- **Roles** - Namespace-scoped RBAC permissions and rules
- **ClusterRoles** - Cluster-scoped RBAC permissions and rules
- **RoleBindings** - Namespace-scoped RBAC role assignments
- **ClusterRoleBindings** - Cluster-scoped RBAC role assignments
- **VolumeAttachments** - Storage volume attachment status and metadata
- **CertificateSigningRequests** - Certificate request status and signer information
- **MutatingWebhookConfigurations** - Admission control webhook configuration and rules
- **ValidatingWebhookConfigurations** - Admission control webhook configuration and rules
- **IngressClasses** - Ingress controller configuration and default status

## Resource Examples

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
        "annotations": {
            "deployment.kubernetes.io/revision": "1"
        },
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
        "createdByKind": "",
        "createdByName": "",
        "paused": false,
        "metadataGeneration": 1
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
        "hostIP": "192.168.1.100",
        "podIP": "10.244.0.5",
        "podIPs": ["10.244.0.5"],
        "phase": "Running",
        "qosClass": "Burstable",
        "priorityClass": "system-cluster-critical",
        "ready": true,
        "initialized": true,
        "scheduled": true,
        "containersReady": true,
        "podScheduled": true,
        "restartCount": 0,
        "createdByKind": "ReplicaSet",
        "createdByName": "sample-deployment-abc123",
        "labels": {
            "app": "sample-app"
        },
        "annotations": {
            "kubernetes.io/config.seen": "2024-01-15T10:30:00Z"
        },
        "deletionTimestamp": null,
        "startTime": "2024-01-15T10:30:00Z",
        "initializedTime": "2024-01-15T10:30:01Z",
        "readyTime": "2024-01-15T10:30:02Z",
        "scheduledTime": "2024-01-15T10:30:01Z",
        "statusReason": "",
        "unschedulable": false,
        "restartPolicy": "Always",
        "serviceAccount": "default",
        "schedulerName": "default-scheduler",
        "overheadCPUCores": "0",
        "overheadMemoryBytes": "0",
        "runtimeClassName": "",
        "tolerations": [
            {
                "key": "node.kubernetes.io/not-ready",
                "value": "",
                "effect": "NoExecute",
                "operator": "Exists"
            }
        ],
        "nodeSelectors": {},
        "persistentVolumeClaims": [],
        "completionTime": null,
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

### Container Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "container",
    "name": "app-container",
    "namespace": "default",
    "data": {
        "name": "app-container",
        "image": "nginx:latest",
        "imageID": "docker-pullable://nginx@sha256:abc123",
        "podName": "sample-pod-abc123",
        "ready": true,
        "restartCount": 0,
        "state": "running",
        "stateRunning": true,
        "stateWaiting": false,
        "stateTerminated": false,
        "waitingReason": "",
        "waitingMessage": "",
        "startedAt": "2024-01-15T10:30:02Z",
        "exitCode": 0,
        "reason": "",
        "message": "",
        "finishedAt": null,
        "startedAtTerm": null,
        "resourceRequests": {
            "cpu": "100m",
            "memory": "128Mi"
        },
        "resourceLimits": {
            "cpu": "500m",
            "memory": "512Mi"
        },
        "lastTerminatedReason": "",
        "lastTerminatedExitCode": 0,
        "lastTerminatedTimestamp": null,
        "stateStarted": "2024-01-15T10:30:02Z"
    }
}
```

### Init Container Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "init_container",
    "name": "init-container",
    "namespace": "default",
    "data": {
        "name": "init-container",
        "image": "busybox:latest",
        "imageID": "docker-pullable://busybox@sha256:def456",
        "podName": "sample-pod-abc123",
        "ready": true,
        "restartCount": 0,
        "state": "terminated",
        "stateRunning": false,
        "stateWaiting": false,
        "stateTerminated": true,
        "waitingReason": "",
        "waitingMessage": "",
        "startedAt": null,
        "exitCode": 0,
        "reason": "Completed",
        "message": "",
        "finishedAt": "2024-01-15T10:30:01Z",
        "startedAtTerm": "2024-01-15T10:30:00Z",
        "resourceRequests": {
            "cpu": "50m",
            "memory": "64Mi"
        },
        "resourceLimits": {
            "cpu": "100m",
            "memory": "128Mi"
        },
        "lastTerminatedReason": "Completed",
        "lastTerminatedExitCode": 0,
        "lastTerminatedTimestamp": "2024-01-15T10:30:01Z",
        "stateStarted": null
    }
}
```

### Service Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "service",
    "name": "sample-service",
    "namespace": "default",
    "data": {
        "type": "ClusterIP",
        "clusterIP": "10.96.0.10",
        "externalIP": "",
        "loadBalancerIP": "",
        "ports": [
            {
                "name": "http",
                "protocol": "TCP",
                "port": 80,
                "targetPort": 8080,
                "nodePort": 30080
            }
        ],
        "selector": {
            "app": "sample-app"
        },
        "labels": {
            "app": "sample-service"
        },
        "annotations": {},
        "endpointsCount": 3,
        "loadBalancerIngress": [],
        "sessionAffinity": "None",
        "externalName": "",
        "createdByKind": "",
        "createdByName": "",
        "createdTimestamp": 1718020800,
        "internalTrafficPolicy": "",
        "externalTrafficPolicy": "",
        "sessionAffinityClientIPTimeoutSeconds": 0
    }
}
```

### Node Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "node",
    "name": "worker-node-1",
    "namespace": "",
    "data": {
        "architecture": "amd64",
        "operatingSystem": "linux",
        "kernelVersion": "5.15.0-generic",
        "kubeletVersion": "v1.28.0",
        "kubeProxyVersion": "v1.28.0",
        "containerRuntimeVersion": "containerd://1.7.0",
        "capacity": {
            "cpu": "4",
            "memory": "8Gi",
            "pods": "110"
        },
        "allocatable": {
            "cpu": "3800m",
            "memory": "7Gi",
            "pods": "110"
        },
        "conditions": {
            "Ready": true,
            "MemoryPressure": false,
            "DiskPressure": false,
            "PIDPressure": false,
            "NetworkUnavailable": false
        },
        "labels": {
            "kubernetes.io/hostname": "worker-node-1",
            "node-role.kubernetes.io/worker": "true"
        },
        "annotations": {},
        "internalIP": "192.168.1.100",
        "externalIP": "203.0.113.1",
        "hostname": "worker-node-1",
        "unschedulable": false,
        "ready": true,
        "createdByKind": "",
        "createdByName": "",
        "createdTimestamp": 1718020800,
        "role": "worker",
        "taints": [
            {
                "key": "node.kubernetes.io/not-ready",
                "value": "",
                "effect": "NoExecute"
            }
        ],
        "deletionTimestamp": null
    }
}
```

### ReplicaSet Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "replicaset",
    "name": "sample-deployment-abc123",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app",
            "pod-template-hash": "abc123"
        },
        "annotations": {},
        "desiredReplicas": 3,
        "currentReplicas": 3,
        "readyReplicas": 3,
        "availableReplicas": 3,
        "fullyLabeledReplicas": 3,
        "observedGeneration": 1,
        "conditionAvailable": true,
        "conditionProgressing": false,
        "conditionReplicaFailure": false,
        "createdByKind": "Deployment",
        "createdByName": "sample-deployment",
        "isCurrent": true
    }
}
```

### StatefulSet Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "statefulset",
    "name": "sample-statefulset",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-statefulset"
        },
        "annotations": {},
        "desiredReplicas": 3,
        "currentReplicas": 3,
        "readyReplicas": 3,
        "updatedReplicas": 3,
        "observedGeneration": 1,
        "currentRevision": "sample-statefulset-1",
        "updateRevision": "sample-statefulset-1",
        "conditionAvailable": true,
        "conditionProgressing": false,
        "conditionReplicaFailure": false,
        "createdByKind": "",
        "createdByName": "",
        "serviceName": "sample-statefulset",
        "podManagementPolicy": "OrderedReady",
        "updateStrategy": "RollingUpdate"
    }
}
```

### DaemonSet Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "daemonset",
    "name": "sample-daemonset",
    "namespace": "kube-system",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-daemonset"
        },
        "annotations": {},
        "desiredNumberScheduled": 3,
        "currentNumberScheduled": 3,
        "numberReady": 3,
        "numberAvailable": 3,
        "numberUnavailable": 0,
        "numberMisscheduled": 0,
        "updatedNumberScheduled": 3,
        "observedGeneration": 1,
        "conditionAvailable": true,
        "conditionProgressing": false,
        "conditionReplicaFailure": false,
        "createdByKind": "",
        "createdByName": "",
        "updateStrategy": "RollingUpdate"
    }
}
```

### Namespace Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "namespace",
    "name": "default",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "kubernetes.io/metadata.name": "default"
        },
        "annotations": {},
        "phase": "Active",
        "conditionActive": true,
        "conditionTerminating": false,
        "createdByKind": "",
        "createdByName": "",
        "deletionTimestamp": null
    }
}
```

### Job Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "job",
    "name": "sample-job",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-job"
        },
        "annotations": {},
        "activePods": 0,
        "succeededPods": 1,
        "failedPods": 0,
        "completions": 1,
        "parallelism": 1,
        "backoffLimit": 6,
        "activeDeadlineSeconds": null,
        "conditionComplete": true,
        "conditionFailed": false,
        "createdByKind": "",
        "createdByName": "",
        "jobType": "Job",
        "suspend": null
    }
}
```

### CronJob Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "cronjob",
    "name": "sample-cronjob",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-cronjob"
        },
        "annotations": {},
        "schedule": "0 0 * * *",
        "concurrencyPolicy": "Allow",
        "suspend": null,
        "successfulJobsHistoryLimit": 3,
        "failedJobsHistoryLimit": 1,
        "activeJobsCount": 0,
        "lastScheduleTime": "2024-01-15T00:00:00Z",
        "nextScheduleTime": null,
        "conditionActive": false,
        "createdByKind": "",
        "createdByName": ""
    }
}
```

### ConfigMap Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "configmap",
    "name": "sample-configmap",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "dataKeys": ["config.json", "environment"],
        "createdByKind": "",
        "createdByName": ""
    }
}
```

### Secret Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "secret",
    "name": "sample-secret",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "type": "Opaque",
        "dataKeys": ["username", "password", "api-key"],
        "createdByKind": "",
        "createdByName": ""
    }
}
```

### PersistentVolumeClaim Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "persistentvolumeclaim",
    "name": "sample-pvc",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "accessModes": ["ReadWriteOnce"],
        "capacity": {
            "storage": "1Gi"
        },
        "condition": "Bound",
        "phase": "Bound",
        "volumeName": "sample-pv"
    }
}
```

### PersistentVolume Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "persistentvolume",
    "name": "sample-pv",
    "namespace": "",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "capacity": {
            "storage": "1Gi"
        },
        "accessModes": ["ReadWriteOnce"],
        "phase": "Bound",
        "claimRef": {
            "apiVersion": "v1",
            "kind": "PersistentVolumeClaim",
            "name": "sample-pvc",
            "namespace": "default"
        },
        "storageClassName": "standard",
        "volumeMode": "Filesystem",
        "nodeAffinity": {
            "required": {
                "nodeSelectorTerms": [
                    {
                        "matchExpressions": [
                            {
                                "key": "kubernetes.io/hostname",
                                "operator": "In",
                                "values": ["worker-node-1"]
                            }
                        ]
                    }
                ]
            }
        }
    }
}
```

### Ingress Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "ingress",
    "name": "sample-ingress",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "spec": {
            "rules": [
                {
                    "host": "sample.example.com",
                    "http": {
                        "paths": [
                            {
                                "path": "/",
                                "pathType": "Prefix",
                                "backend": {
                                    "service": {
                                        "name": "sample-service",
                                        "port": {
                                            "number": 80
                                        }
                                    }
                                }
                            }
                        ]
                    }
                }
            ]
        },
        "status": {
            "loadBalancer": {
                "ingress": [
                    {
                        "ip": "192.168.1.100"
                    }
                ]
            }
        }
    }
}
```

### HorizontalPodAutoscaler Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "horizontalpodautoscaler",
    "name": "sample-hpa",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "spec": {
            "scaleTargetRef": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "sample-deployment"
            },
            "minReplicas": 1,
            "maxReplicas": 5,
            "targetCPUUtilizationPercentage": 80
        },
        "status": {
            "currentReplicas": 3,
            "desiredReplicas": 3,
            "currentCPUUtilizationPercentage": 70
        }
    }
}
```

### ServiceAccount Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "serviceaccount",
    "name": "sample-sa",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "imagePullSecrets": [
            {
                "name": "sample-secret"
            }
        ],
        "automountServiceAccountToken": true
    }
}
```

### Endpoints Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "endpoints",
    "name": "sample-service",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-service"
        },
        "annotations": {},
        "subsets": [
            {
                "addresses": [
                    {
                        "ip": "10.244.0.5"
                    }
                ],
                "ports": [
                    {
                        "name": "http",
                        "port": 8080
                    }
                ]
            }
        ]
    }
}
```

### ResourceQuota Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "resourcequota",
    "name": "sample-quota",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "hard": {
            "pods": "10",
            "memory": "500Mi",
            "cpu": "500m"
        },
        "used": {
            "pods": "3",
            "memory": "128Mi",
            "cpu": "250m"
        }
    }
}
```

### PodDisruptionBudget Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "poddisruptionbudget",
    "name": "sample-pdb",
    "namespace": "default",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-app"
        },
        "annotations": {},
        "spec": {
            "minAvailable": 1,
            "selector": {
                "matchLabels": {
                    "app": "sample-app"
                }
            }
        },
        "status": {
            "currentHealthy": 2,
            "desiredHealthy": 1,
            "disruptionsAllowed": 1,
            "expectedPods": 3
        }
    }
}
```

### MutatingWebhookConfiguration Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "mutatingwebhookconfiguration",
    "name": "sample-mutating-webhook",
    "namespace": "",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-webhook"
        },
        "annotations": {},
        "webhooks": [
            {
                "name": "sample-webhook.example.com",
                "clientConfig": {
                    "url": "https://webhook.example.com/mutate",
                    "service": {
                        "namespace": "webhook-system",
                        "name": "sample-webhook-service",
                        "path": "/mutate",
                        "port": 443
                    },
                    "caBundle": "LS0tLS1CRUdJTi..."
                },
                "rules": [
                    {
                        "apiGroups": ["apps"],
                        "apiVersions": ["v1"],
                        "resources": ["deployments"],
                        "scope": "Namespaced"
                    }
                ],
                "failurePolicy": "Ignore",
                "matchPolicy": "Equivalent",
                "namespaceSelector": {},
                "objectSelector": {},
                "sideEffects": "None",
                "timeoutSeconds": 30,
                "admissionReviewVersions": ["v1"]
            }
        ]
    }
}
```

### ValidatingWebhookConfiguration Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "validatingwebhookconfiguration",
    "name": "sample-validating-webhook",
    "namespace": "",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "sample-webhook"
        },
        "annotations": {},
        "webhooks": [
            {
                "name": "sample-webhook.example.com",
                "clientConfig": {
                    "url": "https://webhook.example.com/validate",
                    "service": {
                        "namespace": "webhook-system",
                        "name": "sample-webhook-service",
                        "path": "/validate",
                        "port": 443
                    },
                    "caBundle": "LS0tLS1CRUdJTi..."
                },
                "rules": [
                    {
                        "apiGroups": ["apps"],
                        "apiVersions": ["v1"],
                        "resources": ["deployments"],
                        "scope": "Namespaced"
                    }
                ],
                "failurePolicy": "Fail",
                "matchPolicy": "Equivalent",
                "namespaceSelector": {},
                "objectSelector": {},
                "sideEffects": "None",
                "timeoutSeconds": 30,
                "admissionReviewVersions": ["v1"]
            }
        ]
    }
}
```

### IngressClass Log Entry

```json
{
    "timestamp": "2024-01-15T10:30:00Z",
    "resourceType": "ingressclass",
    "name": "nginx",
    "namespace": "",
    "data": {
        "createdTimestamp": 1718020800,
        "labels": {
            "app": "nginx-ingress"
        },
        "annotations": {
            "ingressclass.kubernetes.io/is-default-class": "true"
        },
        "controller": "k8s.io/ingress-nginx",
        "isDefault": true
    }
}
```

## Field Descriptions

### Common Fields

All log entries include these common fields:

- **timestamp**: ISO 8601 timestamp when the log was generated
- **resourceType**: Type of Kubernetes resource (e.g., "deployment", "pod", "service")
- **name**: Name of the resource
- **namespace**: Namespace where the resource exists (empty for cluster-scoped resources like nodes)
- **data**: Resource-specific data fields

### Resource-Specific Fields

Each resource type includes fields that match the corresponding kube-state-metrics, plus additional useful information:

- **Deployments**: Replica counts, strategy information, conditions, and scaling status
- **Pods**: Lifecycle state, scheduling information, resource usage, and container status
- **Containers**: Individual container state, resource requests/limits, and restart information
- **Services**: Network configuration, endpoints, load balancer status, and session affinity
- **Nodes**: Hardware information, capacity, health conditions, and scheduling status
- **ReplicaSets**: Scaling status, availability, and relationship to parent deployments
- **StatefulSets**: Stateful application deployment, storage, and update strategy
- **DaemonSets**: Node coverage, scheduling status, and update strategy
- **Namespaces**: Lifecycle state and conditions
- **Jobs**: Batch execution status, completion tracking, and failure handling
- **CronJobs**: Schedule configuration, execution history, and concurrency policies
- **ConfigMaps**: Configuration data keys and metadata
- **Secrets**: Sensitive data keys, types, and metadata (without exposing actual values)
- **PersistentVolumeClaims**: Storage requests, capacity, access modes, and binding status
- **Ingresses**: Host rules, TLS configuration, load balancer status, and path mappings
- **HorizontalPodAutoscalers**: Target metrics, current utilization, scaling conditions, and replica ranges
- **ServiceAccounts**: Associated secrets, image pull secrets, and token mounting configuration
- **PodDisruptionBudgets**: Configuration and status of pod disruption budgets
- **Endpoints**: Service endpoint addresses, ports, and readiness status
- **PersistentVolumes**: Storage volume capacity, access modes, and binding status
- **ResourceQuotas**: Namespace resource limits and current usage
- **StorageClasses**: Storage provisioner configuration and parameters
- **NetworkPolicies**: Network security rules and policy types
- **ReplicationControllers**: Legacy replication controller scaling and availability
- **LimitRanges**: Namespace resource limit ranges and defaults
- **Leases**: Leader election coordination and lease information
- **Roles**: Namespace-scoped RBAC permissions and rules
- **ClusterRoles**: Cluster-scoped RBAC permissions and rules
- **RoleBindings**: Namespace-scoped RBAC role assignments
- **ClusterRoleBindings**: Cluster-scoped RBAC role assignments
- **VolumeAttachments**: Storage volume attachment status and metadata
- **CertificateSigningRequests**: Certificate request status and signer information
- **MutatingWebhookConfigurations**: Admission control webhook configuration and rules
- **ValidatingWebhookConfigurations**: Admission control webhook configuration and rules
- **IngressClasses**: Ingress controller configuration and default status

For detailed field descriptions and their meanings, refer to the [Kubernetes API documentation](https://kubernetes.io/docs/reference/kubernetes-api/). 

## Generic CRD Support

kube-state-logs can log any Kubernetes Custom Resource Definition (CRD) generically. You can specify which CRDs to log and which fields to extract using the `--crd-configs` flag or Helm values.

### What is logged for CRDs?
- Metadata: name, namespace, labels, annotations, creation timestamp
- `spec` and `status` fields (full objects)
- Any custom fields you specify (dot-separated paths)

### Example CLI usage

```
--crd-configs="mygroup.example.com/v1:widgets:spec.size|spec.color,anothergroup.io/v1:foos:spec.enabled"
```

### Example Helm values

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