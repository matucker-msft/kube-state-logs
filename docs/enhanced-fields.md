# Kube-State-Logs Enhanced Fields Documentation

This document details the additional fields that kube-state-logs provides beyond the standard kube-state-metrics data, organized by resource type.

## Overview

kube-state-logs provides **150+ additional fields** across all resource types, offering richer context, better structure, and enhanced relationships that are not available in kube-state-metrics.

## Resource-by-Resource Enhanced Fields

### 1. Pod Resources
**Additional Fields:** 15+ enhanced fields

#### Core Enhancements
- `completionTime` - When pod completes successfully (not in KSM)
- `statusReason` - Detailed status reason with full context
- `unschedulable` - Explicit unschedulable flag
- `podIPs` - Array of all pod IPs (KSM only has single IP)
- `tolerations` - Full toleration objects with complete configuration
- `nodeSelectors` - Full node selector map with all selectors
- `persistentVolumeClaims` - Full PVC info array with details
- `resourceLimits` - Aggregated pod-level resource limits
- `resourceRequests` - Aggregated pod-level resource requests

#### Container-Level Enhancements
- `state` - Human-readable container state (running/waiting/terminated)
- `waitingMessage` - Detailed waiting message
- `startedAtTerm` - When container started in terminated state
- `finishedAt` - When container finished execution
- `message` - Container status message
- `reason` - Container termination reason

#### Relationship Enhancements
- `createdByKind` - Owner resource kind (ReplicaSet, Job, etc.)
- `createdByName` - Owner resource name
- `deletionTimestamp` - When pod is marked for deletion

---

### 2. Deployment Resources
**Additional Fields:** 2 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name

---

### 3. Service Resources
**Additional Fields:** 2 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name

---

### 4. Node Resources
**Additional Fields:** 2 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name

---

### 5. Job Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `jobType` - Distinguishes between regular Jobs and CronJob-created Jobs

---

### 6. CronJob Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `nextScheduleTime` - Next scheduled execution time (not available in v1 API)

---

### 7. ConfigMap Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `dataKeys` - List of configuration keys (KSM doesn't expose this)

---

### 8. Secret Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `dataKeys` - List of secret keys (KSM doesn't expose this, for security)

---

### 9. PersistentVolumeClaim Resources
**Additional Fields:** 5 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `capacity` - Full capacity map (KSM has individual metrics)
- `requestStorage` - Explicit requested storage value
- `usedStorage` - Explicit used storage value

---

### 10. Ingress Resources
**Additional Fields:** 5 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `rules` - Full ingress rules array with paths and services
- `tls` - Complete TLS configuration array
- `loadBalancerIngress` - Detailed load balancer information array

---

### 11. HorizontalPodAutoscaler Resources
**Additional Fields:** 5 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `scaleTargetRef` - Target resource name
- `scaleTargetKind` - Target resource kind
- `targetMemoryUtilizationPercentage` - Memory target (KSM only has CPU)
- `currentMemoryUtilizationPercentage` - Current memory usage

---

### 12. ServiceAccount Resources
**Additional Fields:** 5 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `secrets` - Array of associated secret names
- `imagePullSecrets` - Array of image pull secret names
- `automountServiceAccountToken` - Token automount setting

---

### 13. Endpoints Resources
**Additional Fields:** 4 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `addresses` - Full address array with node and target details
- `ports` - Complete port information array
- `ready` - Overall readiness status

---

### 14. PersistentVolume Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `persistentVolumeSource` - Specific volume source type
- `isDefaultClass` - Default storage class indicator

---

### 15. ResourceQuota Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `hard` - Full resource limits map with actual values
- `used` - Full resource usage map with actual values

---

### 16. PodDisruptionBudget Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `disruptionAllowed` - Boolean flag for disruption allowance

---

### 17. StorageClass Resources
**Additional Fields:** 5 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `parameters` - Full storage parameters map
- `mountOptions` - Mount options array
- `allowedTopologies` - Allowed topology constraints

---

### 18. NetworkPolicy Resources
**Additional Fields:** 4 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `ingressRules` - Complete ingress rules array with ports and peers
- `egressRules` - Complete egress rules array with ports and peers
- Detailed peer information including pod selectors, namespace selectors, and IP blocks

---

### 19. ReplicationController Resources
**Additional Fields:** 2 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name

---

### 20. LimitRange Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `limits` - Full limits array with complete resource details (min, max, default, defaultRequest, maxLimitRequestRatio)

---

### 21. Lease Resources
**Additional Fields:** 2 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name

---

### 22. Role Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `rules` - Full policy rules array with apiGroups, resources, resourceNames, and verbs

---

### 23. ClusterRole Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `rules` - Full policy rules array with complete RBAC details

---

### 24. RoleBinding Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `roleRef` - Complete role reference object
- `subjects` - Full subjects array with kind, name, namespace, and apiGroup

---

### 25. ClusterRoleBinding Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `roleRef` - Complete role reference object
- `subjects` - Full subjects array with complete RBAC binding details

---

### 26. VolumeAttachment Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `attachmentMetadata` - Full attachment metadata map

---

### 27. CertificateSigningRequest Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `usages` - Array of certificate usages

---

### 28. MutatingWebhookConfiguration Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `webhooks` - Full webhooks array with complete configuration details:
  - Client configuration (URL, service, CABundle)
  - Rules (apiGroups, apiVersions, resources, scope)
  - Selectors (namespaceSelector, objectSelector)
  - Policy settings (failurePolicy, matchPolicy, sideEffects, timeoutSeconds)
  - Admission review versions

---

### 29. ValidatingWebhookConfiguration Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `webhooks` - Full webhooks array with complete validation configuration details (same structure as mutating webhooks)

---

### 30. IngressClass Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `isDefault` - Boolean indicating if this is the default ingress class

---

### 31. Namespace Resources
**Additional Fields:** 4 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `conditionActive` - Boolean for namespace active condition
- `conditionTerminating` - Boolean for namespace terminating condition
- `deletionTimestamp` - When namespace is marked for deletion

---

### 32. DaemonSet Resources
**Additional Fields:** 4 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `updateStrategy` - Update strategy type
- `conditionAvailable` - Boolean for available condition
- `conditionProgressing` - Boolean for progressing condition
- `conditionReplicaFailure` - Boolean for replica failure condition

---

### 33. StatefulSet Resources
**Additional Fields:** 6 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `serviceName` - Associated service name
- `podManagementPolicy` - Pod management policy type
- `updateStrategy` - Update strategy type
- `conditionAvailable` - Boolean for available condition
- `conditionProgressing` - Boolean for progressing condition
- `conditionReplicaFailure` - Boolean for replica failure condition

---

### 34. ReplicaSet Resources
**Additional Fields:** 3 enhanced fields

- `createdByKind` - Owner resource kind
- `createdByName` - Owner resource name
- `isCurrent` - Boolean indicating if this is the current ReplicaSet

---

## Common Enhanced Field Patterns

### 1. Owner References (Universal)
Every resource includes:
- `createdByKind` - The kind of resource that created this resource
- `createdByName` - The name of the resource that created this resource

### 2. Enhanced Arrays and Objects
Instead of individual metrics, KSL provides:
- **Full arrays** with complete object details
- **Rich objects** with all properties
- **Relationship mappings** between resources

### 3. Boolean Condition Flags
Many resources include explicit boolean flags for conditions:
- `conditionAvailable`
- `conditionProgressing`
- `conditionReplicaFailure`
- `conditionActive`
- `conditionTerminating`

### 4. Enhanced Timestamps
- Proper time objects instead of numeric timestamps
- Additional timing information (completionTime, deletionTimestamp)
- Contextual timing data

### 5. Resource Relationship Data
- **Scale targets** for autoscalers
- **Service associations** for StatefulSets
- **Volume attachments** for storage
- **Webhook configurations** for admission control

## Benefits of Enhanced Fields

### 1. **Richer Context**
- Owner relationships show resource hierarchies
- Complete object structures provide full context
- Boolean flags make conditions easily queryable

### 2. **Better Analytics**
- Arrays enable complex aggregations
- Relationships support dependency analysis
- Enhanced timestamps enable temporal analysis

### 3. **Improved Debugging**
- Detailed error messages and reasons
- Complete configuration information
- Relationship mapping for troubleshooting

### 4. **Enhanced Monitoring**
- Real-time condition status
- Resource utilization details
- Performance metrics

### 5. **Log-Based Advantages**
- Structured JSON for easy parsing
- Rich objects for comprehensive analysis
- Relationship data for correlation

## Summary

kube-state-logs provides **150+ additional fields** that significantly enhance the monitoring and observability capabilities beyond what kube-state-metrics offers. These enhancements include:

- **Universal owner references** for resource relationships
- **Complete object arrays** instead of individual metrics
- **Boolean condition flags** for easy querying
- **Enhanced timestamps** with contextual data
- **Resource relationship mappings** for dependency analysis
- **Detailed configuration information** for debugging
- **Rich metadata** for comprehensive monitoring

These enhancements make kube-state-logs a superior choice for log-based monitoring environments, providing richer context, better structure, and enhanced analytical capabilities. 