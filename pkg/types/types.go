package types

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogEntryMetadata contains the common metadata for all log entries
type LogEntryMetadata struct {
	Timestamp        time.Time         `json:"timestamp"`
	ResourceType     string            `json:"resourceType"`
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`
	CreatedByKind    string            `json:"createdByKind"`
	CreatedByName    string            `json:"createdByName"`
}

// DeploymentData represents deployment-specific metrics (matching kube-state-metrics)
type DeploymentData struct {
	LogEntryMetadata
	// Replica counts (matching kube-state-metrics)
	DesiredReplicas     int32 `json:"desiredReplicas"`
	CurrentReplicas     int32 `json:"currentReplicas"`
	ReadyReplicas       int32 `json:"readyReplicas"`
	AvailableReplicas   int32 `json:"availableReplicas"`
	UnavailableReplicas int32 `json:"unavailableReplicas"`
	UpdatedReplicas     int32 `json:"updatedReplicas"`

	// Deployment status (matching kube-state-metrics)
	ObservedGeneration int64 `json:"observedGeneration"`
	CollisionCount     int32 `json:"collisionCount"`

	// Strategy info (matching kube-state-metrics)
	StrategyType                        string `json:"strategyType"`
	StrategyRollingUpdateMaxSurge       int32  `json:"strategyRollingUpdateMaxSurge"`
	StrategyRollingUpdateMaxUnavailable int32  `json:"strategyRollingUpdateMaxUnavailable"`

	// Conditions (matching kube-state-metrics)
	ConditionAvailable      bool `json:"conditionAvailable"`
	ConditionProgressing    bool `json:"conditionProgressing"`
	ConditionReplicaFailure bool `json:"conditionReplicaFailure"`

	// Spec fields (matching kube-state-metrics)
	Paused                  bool  `json:"paused"`
	MinReadySeconds         int32 `json:"minReadySeconds"`
	RevisionHistoryLimit    int32 `json:"revisionHistoryLimit"`
	ProgressDeadlineSeconds int32 `json:"progressDeadlineSeconds"`

	// Metadata (matching kube-state-metrics)
	MetadataGeneration int64 `json:"metadataGeneration"`
}

// PodData represents pod-specific metrics (matching kube-state-metrics)
type PodData struct {
	LogEntryMetadata
	// Basic pod info
	NodeName      string `json:"nodeName"`
	HostIP        string `json:"hostIP"`
	PodIP         string `json:"podIP"`
	Phase         string `json:"phase"`
	QoSClass      string `json:"qosClass"`
	PriorityClass string `json:"priorityClass"`

	// Pod conditions
	Ready           bool `json:"ready"`
	Initialized     bool `json:"initialized"`
	Scheduled       bool `json:"scheduled"`
	ContainersReady bool `json:"containersReady"`
	PodScheduled    bool `json:"podScheduled"`

	// Pod status
	RestartCount int32 `json:"restartCount"`

	// Missing from KSM
	DeletionTimestamp      *time.Time        `json:"deletionTimestamp"`
	StartTime              *time.Time        `json:"startTime"`
	InitializedTime        *time.Time        `json:"initializedTime"`
	ReadyTime              *time.Time        `json:"readyTime"`
	ScheduledTime          *time.Time        `json:"scheduledTime"`
	StatusReason           string            `json:"statusReason"`
	Unschedulable          bool              `json:"unschedulable"`
	RestartPolicy          string            `json:"restartPolicy"`
	ServiceAccount         string            `json:"serviceAccount"`
	SchedulerName          string            `json:"schedulerName"`
	OverheadCPUCores       string            `json:"overheadCPUCores"`
	OverheadMemoryBytes    string            `json:"overheadMemoryBytes"`
	RuntimeClassName       string            `json:"runtimeClassName"`
	PodIPs                 []string          `json:"podIPs"`
	Tolerations            []TolerationData  `json:"tolerations"`
	NodeSelectors          map[string]string `json:"nodeSelectors"`
	PersistentVolumeClaims []PVCData         `json:"persistentVolumeClaims"`
	CompletionTime         *time.Time        `json:"completionTime"`
}

// TolerationData represents pod toleration information
type TolerationData struct {
	Key               string `json:"key"`
	Value             string `json:"value"`
	Effect            string `json:"effect"`
	Operator          string `json:"operator"`
	TolerationSeconds string `json:"tolerationSeconds"`
}

// PVCData represents persistent volume claim information
type PVCData struct {
	ClaimName string `json:"claimName"`
	ReadOnly  bool   `json:"readOnly"`
}

// ContainerData represents container-specific metrics (matching kube-state-metrics)
type ContainerData struct {
	// Basic container info
	ResourceType string    `json:"resourceType"`
	Timestamp    time.Time `json:"timestamp"`
	Name         string    `json:"name"`
	Image        string    `json:"image"`
	ImageID      string    `json:"imageID"`
	PodName      string    `json:"podName"`
	Namespace    string    `json:"namespace"`

	// Container state
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`

	// Current state details
	StateRunning    bool `json:"stateRunning"`
	StateWaiting    bool `json:"stateWaiting"`
	StateTerminated bool `json:"stateTerminated"`

	// Waiting state details
	WaitingReason  string `json:"waitingReason"`
	WaitingMessage string `json:"waitingMessage"`

	// Running state details
	StartedAt *time.Time `json:"startedAt"`

	// Terminated state details
	ExitCode      int32      `json:"exitCode"`
	Reason        string     `json:"reason"`
	Message       string     `json:"message"`
	FinishedAt    *time.Time `json:"finishedAt"`
	StartedAtTerm *time.Time `json:"startedAtTerm"`

	// Resource requests/limits
	ResourceRequests map[string]string `json:"resourceRequests"`
	ResourceLimits   map[string]string `json:"resourceLimits"`

	// Missing from KSM
	LastTerminatedReason    string     `json:"lastTerminatedReason"`
	LastTerminatedExitCode  int32      `json:"lastTerminatedExitCode"`
	LastTerminatedTimestamp *time.Time `json:"lastTerminatedTimestamp"`
	StateStarted            *time.Time `json:"stateStarted"`
}

// ServiceData represents service-specific metrics (matching kube-state-metrics)
type ServiceData struct {
	LogEntryMetadata
	// Basic service info
	Type           string            `json:"type"`
	ClusterIP      string            `json:"clusterIP"`
	ExternalIP     string            `json:"externalIP"`
	LoadBalancerIP string            `json:"loadBalancerIP"`
	Ports          []ServicePortData `json:"ports"`
	Selector       map[string]string `json:"selector"`

	// Service status
	EndpointsCount int `json:"endpointsCount"`

	// Load balancer info
	LoadBalancerIngress []LoadBalancerIngressData `json:"loadBalancerIngress"`

	// Session affinity
	SessionAffinity string `json:"sessionAffinity"`

	// External name
	ExternalName string `json:"externalName"`

	// Missing from KSM
	ExternalTrafficPolicy                 string `json:"externalTrafficPolicy"`
	SessionAffinityClientIPTimeoutSeconds int32  `json:"sessionAffinityClientIPTimeoutSeconds"`

	// Additional KSM fields we should track
	AllocateLoadBalancerNodePorts *bool    `json:"allocateLoadBalancerNodePorts"`
	LoadBalancerClass             *string  `json:"loadBalancerClass"`
	LoadBalancerSourceRanges      []string `json:"loadBalancerSourceRanges"`
}

// ServicePortData represents service port information
type ServicePortData struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
}

// LoadBalancerIngressData represents load balancer ingress information
type LoadBalancerIngressData struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

// NodeData represents node-specific metrics (matching kube-state-metrics)
type NodeData struct {
	LogEntryMetadata
	// Basic node info
	Architecture            string `json:"architecture"`
	OperatingSystem         string `json:"operatingSystem"`
	KernelVersion           string `json:"kernelVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	KubeProxyVersion        string `json:"kubeProxyVersion"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`

	// Node status
	Capacity    map[string]string `json:"capacity"`
	Allocatable map[string]string `json:"allocatable"`
	Conditions  map[string]bool   `json:"conditions"`
	Phase       string            `json:"phase"`

	// Node addresses
	InternalIP string `json:"internalIP"`
	ExternalIP string `json:"externalIP"`
	Hostname   string `json:"hostname"`

	// Node status details
	Unschedulable bool `json:"unschedulable"`
	Ready         bool `json:"ready"`

	// Missing from KSM
	Role              string      `json:"role"`
	Taints            []TaintData `json:"taints"`
	DeletionTimestamp *time.Time  `json:"deletionTimestamp"`
}

// TaintData represents node taint information
type TaintData struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// ReplicaSetData represents replicaset-specific metrics (matching kube-state-metrics)
type ReplicaSetData struct {
	LogEntryMetadata
	// Replica counts
	DesiredReplicas      int32 `json:"desiredReplicas"`
	CurrentReplicas      int32 `json:"currentReplicas"`
	ReadyReplicas        int32 `json:"readyReplicas"`
	AvailableReplicas    int32 `json:"availableReplicas"`
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas"`

	// Replicaset status
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions
	ConditionAvailable      bool `json:"conditionAvailable"`
	ConditionProgressing    bool `json:"conditionProgressing"`
	ConditionReplicaFailure bool `json:"conditionReplicaFailure"`

	// Replicaset specific
	IsCurrent bool `json:"isCurrent"` // Whether this is the current replicaset for its owner
}

// StatefulSetData represents statefulset-specific metrics (matching kube-state-metrics)
type StatefulSetData struct {
	LogEntryMetadata
	// Replica counts
	DesiredReplicas int32 `json:"desiredReplicas"`
	CurrentReplicas int32 `json:"currentReplicas"`
	ReadyReplicas   int32 `json:"readyReplicas"`
	UpdatedReplicas int32 `json:"updatedReplicas"`

	// Statefulset status
	ObservedGeneration int64  `json:"observedGeneration"`
	CurrentRevision    string `json:"currentRevision"`
	UpdateRevision     string `json:"updateRevision"`

	// Conditions
	ConditionAvailable      bool `json:"conditionAvailable"`
	ConditionProgressing    bool `json:"conditionProgressing"`
	ConditionReplicaFailure bool `json:"conditionReplicaFailure"`

	// Statefulset specific
	ServiceName         string `json:"serviceName"`
	PodManagementPolicy string `json:"podManagementPolicy"`
	UpdateStrategy      string `json:"updateStrategy"`
}

// DaemonSetData represents daemonset-specific metrics (matching kube-state-metrics)
type DaemonSetData struct {
	LogEntryMetadata
	// Replica counts
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32 `json:"currentNumberScheduled"`
	NumberReady            int32 `json:"numberReady"`
	NumberAvailable        int32 `json:"numberAvailable"`
	NumberUnavailable      int32 `json:"numberUnavailable"`
	NumberMisscheduled     int32 `json:"numberMisscheduled"`
	UpdatedNumberScheduled int32 `json:"updatedNumberScheduled"`

	// Daemonset status
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions
	ConditionAvailable      bool `json:"conditionAvailable"`
	ConditionProgressing    bool `json:"conditionProgressing"`
	ConditionReplicaFailure bool `json:"conditionReplicaFailure"`

	// Daemonset specific
	UpdateStrategy string `json:"updateStrategy"`
}

// NamespaceData represents namespace-specific metrics (matching kube-state-metrics)
type NamespaceData struct {
	LogEntryMetadata
	// Namespace status
	Phase string `json:"phase"`

	// Conditions
	ConditionActive      bool `json:"conditionActive"`
	ConditionTerminating bool `json:"conditionTerminating"`

	// Namespace specific
	DeletionTimestamp *metav1.Time `json:"deletionTimestamp"`
}

// JobData represents job-specific metrics (matching kube-state-metrics)
type JobData struct {
	LogEntryMetadata
	// Job status
	ActivePods    int32 `json:"activePods"`
	SucceededPods int32 `json:"succeededPods"`
	FailedPods    int32 `json:"failedPods"`

	// Job spec
	Completions           *int32 `json:"completions"`
	Parallelism           *int32 `json:"parallelism"`
	BackoffLimit          int32  `json:"backoffLimit"`
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds"`

	// Job conditions
	ConditionComplete bool `json:"conditionComplete"`
	ConditionFailed   bool `json:"conditionFailed"`

	// Job specific
	JobType string `json:"jobType"` // "Job" or "CronJob"
	Suspend *bool  `json:"suspend"`
}

// CronJobData represents cronjob-specific metrics (matching kube-state-metrics)
type CronJobData struct {
	LogEntryMetadata
	// CronJob spec
	Schedule                   string `json:"schedule"`
	ConcurrencyPolicy          string `json:"concurrencyPolicy"`
	Suspend                    *bool  `json:"suspend"`
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     *int32 `json:"failedJobsHistoryLimit"`

	// CronJob status
	ActiveJobsCount int32 `json:"activeJobsCount"`

	// Last execution info
	LastScheduleTime *time.Time `json:"lastScheduleTime"`
	NextScheduleTime *time.Time `json:"nextScheduleTime"`

	// Conditions
	ConditionActive bool `json:"conditionActive"`
}

// ConfigMapData represents configmap-specific metrics (matching kube-state-metrics)
type ConfigMapData struct {
	LogEntryMetadata
	// ConfigMap specific
	DataKeys []string `json:"dataKeys"`
}

// SecretData represents secret-specific metrics (matching kube-state-metrics)
type SecretData struct {
	LogEntryMetadata
	// Secret specific
	Type     string   `json:"type"`
	DataKeys []string `json:"dataKeys"`
}

// PersistentVolumeClaimData represents persistentvolumeclaim-specific metrics (matching kube-state-metrics)
type PersistentVolumeClaimData struct {
	LogEntryMetadata
	// PVC spec
	AccessModes      []string `json:"accessModes"`
	StorageClassName *string  `json:"storageClassName"`
	VolumeName       string   `json:"volumeName"`

	// PVC status
	Phase    string            `json:"phase"`
	Capacity map[string]string `json:"capacity"`

	// Conditions
	ConditionPending bool `json:"conditionPending"`
	ConditionBound   bool `json:"conditionBound"`
	ConditionLost    bool `json:"conditionLost"`

	// PVC specific
	RequestStorage string `json:"requestStorage"`
	UsedStorage    string `json:"usedStorage"`
}

// IngressData represents ingress-specific metrics (matching kube-state-metrics)
type IngressData struct {
	LogEntryMetadata
	// Ingress spec
	IngressClassName *string `json:"ingressClassName"`
	LoadBalancerIP   string  `json:"loadBalancerIP"`

	// Ingress status
	LoadBalancerIngress []LoadBalancerIngressData `json:"loadBalancerIngress"`

	// Ingress rules
	Rules []IngressRuleData `json:"rules"`

	// TLS configuration
	TLS []IngressTLSData `json:"tls"`

	// Conditions
	ConditionLoadBalancerReady bool `json:"conditionLoadBalancerReady"`
}

// IngressRuleData represents ingress rule information
type IngressRuleData struct {
	Host  string            `json:"host"`
	Paths []IngressPathData `json:"paths"`
}

// IngressPathData represents ingress path information
type IngressPathData struct {
	Path     string `json:"path"`
	PathType string `json:"pathType"`
	Service  string `json:"service"`
	Port     string `json:"port"`
}

// IngressTLSData represents ingress TLS configuration
type IngressTLSData struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

// HorizontalPodAutoscalerData represents horizontalpodautoscaler-specific metrics (matching kube-state-metrics)
type HorizontalPodAutoscalerData struct {
	LogEntryMetadata
	// HPA spec
	MinReplicas                       *int32 `json:"minReplicas"`
	MaxReplicas                       int32  `json:"maxReplicas"`
	TargetCPUUtilizationPercentage    *int32 `json:"targetCPUUtilizationPercentage"`
	TargetMemoryUtilizationPercentage *int32 `json:"targetMemoryUtilizationPercentage"`

	// HPA status
	CurrentReplicas                    int32  `json:"currentReplicas"`
	DesiredReplicas                    int32  `json:"desiredReplicas"`
	CurrentCPUUtilizationPercentage    *int32 `json:"currentCPUUtilizationPercentage"`
	CurrentMemoryUtilizationPercentage *int32 `json:"currentMemoryUtilizationPercentage"`

	// Conditions
	ConditionAbleToScale    bool `json:"conditionAbleToScale"`
	ConditionScalingActive  bool `json:"conditionScalingActive"`
	ConditionScalingLimited bool `json:"conditionScalingLimited"`

	// HPA specific
	ScaleTargetRef  string `json:"scaleTargetRef"`
	ScaleTargetKind string `json:"scaleTargetKind"`
}

// ServiceAccountData represents serviceaccount-specific metrics (matching kube-state-metrics)
type ServiceAccountData struct {
	LogEntryMetadata
	// ServiceAccount specific
	Secrets          []string `json:"secrets"`
	ImagePullSecrets []string `json:"imagePullSecrets"`

	// ServiceAccount specific
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken"`
}

// EndpointsData represents endpoints-specific metrics (matching kube-state-metrics)
type EndpointsData struct {
	LogEntryMetadata
	// Endpoints specific
	Addresses []EndpointAddressData `json:"addresses"`
	Ports     []EndpointPortData    `json:"ports"`

	// Endpoints specific
	Ready bool `json:"ready"`
}

// EndpointAddressData represents endpoint address information
type EndpointAddressData struct {
	IP        string `json:"ip"`
	Hostname  string `json:"hostname"`
	NodeName  string `json:"nodeName"`
	TargetRef string `json:"targetRef"`
}

// EndpointPortData represents endpoint port information
type EndpointPortData struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
}

// PersistentVolumeData represents persistentvolume-specific metrics (matching kube-state-metrics)
type PersistentVolumeData struct {
	LogEntryMetadata
	// PersistentVolume specific
	CapacityBytes          int64  `json:"capacityBytes"`
	AccessModes            string `json:"accessModes"`
	ReclaimPolicy          string `json:"reclaimPolicy"`
	Status                 string `json:"status"`
	StorageClassName       string `json:"storageClassName"`
	VolumeMode             string `json:"volumeMode"`
	VolumePluginName       string `json:"volumePluginName"`
	PersistentVolumeSource string `json:"persistentVolumeSource"`

	// PersistentVolume specific
	IsDefaultClass bool `json:"isDefaultClass"`
}

// ResourceQuotaData represents resourcequota-specific metrics (matching kube-state-metrics)
type ResourceQuotaData struct {
	LogEntryMetadata
	// ResourceQuota specific
	Hard map[string]int64 `json:"hard"`
	Used map[string]int64 `json:"used"`

	// ResourceQuota specific
	Scopes []string `json:"scopes"`
}

// PodDisruptionBudgetData represents poddisruptionbudget-specific metrics (matching kube-state-metrics)
type PodDisruptionBudgetData struct {
	LogEntryMetadata
	// PodDisruptionBudget specific
	MinAvailable             int32 `json:"minAvailable"`
	MaxUnavailable           int32 `json:"maxUnavailable"`
	CurrentHealthy           int32 `json:"currentHealthy"`
	DesiredHealthy           int32 `json:"desiredHealthy"`
	ExpectedPods             int32 `json:"expectedPods"`
	DisruptionsAllowed       int32 `json:"disruptionsAllowed"`
	TotalReplicas            int32 `json:"totalReplicas"`
	DisruptionAllowed        bool  `json:"disruptionAllowed"`
	StatusCurrentHealthy     int32 `json:"statusCurrentHealthy"`
	StatusDesiredHealthy     int32 `json:"statusDesiredHealthy"`
	StatusExpectedPods       int32 `json:"statusExpectedPods"`
	StatusDisruptionsAllowed int32 `json:"statusDisruptionsAllowed"`
	StatusTotalReplicas      int32 `json:"statusTotalReplicas"`
	StatusDisruptionAllowed  bool  `json:"statusDisruptionAllowed"`
}

// CRDData represents CRD-specific metrics
type CRDData struct {
	LogEntryMetadata
	// CRD specific
	APIVersion   string         `json:"apiVersion"`
	Kind         string         `json:"kind"`
	Spec         map[string]any `json:"spec"`
	Status       map[string]any `json:"status"`
	CustomFields map[string]any `json:"customFields"`
}

// StorageClassData represents storageclass-specific metrics (matching kube-state-metrics)
type StorageClassData struct {
	LogEntryMetadata
	// StorageClass specific
	Provisioner          string            `json:"provisioner"`
	ReclaimPolicy        string            `json:"reclaimPolicy"`
	VolumeBindingMode    string            `json:"volumeBindingMode"`
	AllowVolumeExpansion bool              `json:"allowVolumeExpansion"`
	Parameters           map[string]string `json:"parameters"`
	MountOptions         []string          `json:"mountOptions"`
	AllowedTopologies    map[string]any    `json:"allowedTopologies"`

	// StorageClass specific
	IsDefaultClass bool `json:"isDefaultClass"`
}

// NetworkPolicyData represents networkpolicy-specific metrics (matching kube-state-metrics)
type NetworkPolicyData struct {
	LogEntryMetadata
	// NetworkPolicy specific
	PolicyTypes  []string                   `json:"policyTypes"`
	IngressRules []NetworkPolicyIngressRule `json:"ingressRules"`
	EgressRules  []NetworkPolicyEgressRule  `json:"egressRules"`
}

// NetworkPolicyIngressRule represents an ingress rule in a network policy
type NetworkPolicyIngressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	From  []NetworkPolicyPeer `json:"from"`
}

// NetworkPolicyEgressRule represents an egress rule in a network policy
type NetworkPolicyEgressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	To    []NetworkPolicyPeer `json:"to"`
}

// NetworkPolicyPort represents a port in a network policy rule
type NetworkPolicyPort struct {
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
	EndPort  int32  `json:"endPort"`
}

// NetworkPolicyPeer represents a peer in a network policy rule
type NetworkPolicyPeer struct {
	PodSelector       map[string]string `json:"podSelector"`
	NamespaceSelector map[string]string `json:"namespaceSelector"`
	IPBlock           map[string]any    `json:"ipBlock"`
}

// ReplicationControllerData represents replicationcontroller-specific metrics (matching kube-state-metrics)
type ReplicationControllerData struct {
	LogEntryMetadata
	// ReplicationController specific
	DesiredReplicas      int32 `json:"desiredReplicas"`
	CurrentReplicas      int32 `json:"currentReplicas"`
	ReadyReplicas        int32 `json:"readyReplicas"`
	AvailableReplicas    int32 `json:"availableReplicas"`
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas"`

	// ReplicationController specific
	ObservedGeneration int64 `json:"observedGeneration"`
}

// LimitRangeData represents limitrange-specific metrics (matching kube-state-metrics)
type LimitRangeData struct {
	LogEntryMetadata
	// LimitRange specific
	Limits []LimitRangeItem `json:"limits"`
}

// LimitRangeItem represents a limit range item
type LimitRangeItem struct {
	Type                 string            `json:"type"`
	ResourceType         string            `json:"resourceType"`
	ResourceName         string            `json:"resourceName"`
	Min                  map[string]string `json:"min"`
	Max                  map[string]string `json:"max"`
	Default              map[string]string `json:"default"`
	DefaultRequest       map[string]string `json:"defaultRequest"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio"`
}

// CertificateSigningRequestData represents certificatesigningrequest-specific metrics
type CertificateSigningRequestData struct {
	LogEntryMetadata
	// CertificateSigningRequest specific
	Status            string   `json:"status"`
	SignerName        string   `json:"signerName"`
	ExpirationSeconds *int32   `json:"expirationSeconds"`
	Usages            []string `json:"usages"`
}

// PolicyRule represents a policy rule in RBAC
type PolicyRule struct {
	APIGroups       []string `json:"apiGroups"`
	Resources       []string `json:"resources"`
	Verbs           []string `json:"verbs"`
	ResourceNames   []string `json:"resourceNames"`
	NonResourceURLs []string `json:"nonResourceURLs"`
}

// RoleData represents role-specific metrics
type RoleData struct {
	LogEntryMetadata
	// Role specific
	Rules []PolicyRule `json:"rules"`
}

// ClusterRoleData represents clusterrole-specific metrics
type ClusterRoleData struct {
	LogEntryMetadata
	// ClusterRole specific
	Rules []PolicyRule `json:"rules"`
}

// RoleRef represents a role reference in RBAC
type RoleRef struct {
	APIGroup string `json:"apiGroup"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

// Subject represents a subject in RBAC
type Subject struct {
	Kind      string `json:"kind"`
	APIGroup  string `json:"apiGroup"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// RoleBindingData represents rolebinding-specific metrics
type RoleBindingData struct {
	LogEntryMetadata
	// RoleBinding specific
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`
}

// ClusterRoleBindingData represents clusterrolebinding-specific metrics
type ClusterRoleBindingData struct {
	LogEntryMetadata
	// ClusterRoleBinding specific
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`
}

// IngressClassData represents ingressclass-specific metrics
type IngressClassData struct {
	LogEntryMetadata
	// IngressClass specific
	Controller string `json:"controller"`
	IsDefault  bool   `json:"isDefault"`
}

// LeaseData represents lease-specific metrics
type LeaseData struct {
	LogEntryMetadata
	// Lease specific
	HolderIdentity       string     `json:"holderIdentity"`
	LeaseDurationSeconds int32      `json:"leaseDurationSeconds"`
	RenewTime            *time.Time `json:"renewTime"`
	AcquireTime          *time.Time `json:"acquireTime"`
	LeaseTransitions     int32      `json:"leaseTransitions"`
}

// WebhookData represents webhook-specific metrics
type WebhookData struct {
	LogEntryMetadata
	// Webhook specific
	Name                    string                  `json:"name"`
	ClientConfig            WebhookClientConfigData `json:"clientConfig"`
	Rules                   []WebhookRuleData       `json:"rules"`
	FailurePolicy           string                  `json:"failurePolicy"`
	MatchPolicy             string                  `json:"matchPolicy"`
	NamespaceSelector       map[string]string       `json:"namespaceSelector"`
	ObjectSelector          map[string]string       `json:"objectSelector"`
	SideEffects             string                  `json:"sideEffects"`
	TimeoutSeconds          *int32                  `json:"timeoutSeconds"`
	AdmissionReviewVersions []string                `json:"admissionReviewVersions"`
}

// WebhookClientConfigData represents webhook client config-specific metrics
type WebhookClientConfigData struct {
	LogEntryMetadata
	// WebhookClientConfig specific
	URL      string              `json:"url"`
	Service  *WebhookServiceData `json:"service"`
	CABundle []byte              `json:"caBundle"`
}

// WebhookServiceData represents webhook service-specific metrics
type WebhookServiceData struct {
	LogEntryMetadata
	// WebhookService specific
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Port      int32  `json:"port"`
}

// WebhookRuleData represents webhook rule-specific metrics
type WebhookRuleData struct {
	LogEntryMetadata
	// WebhookRule specific
	APIGroups   []string `json:"apiGroups"`
	APIVersions []string `json:"apiVersions"`
	Resources   []string `json:"resources"`
	Scope       string   `json:"scope"`
}

// MutatingWebhookConfigurationData represents mutatingwebhookconfiguration-specific metrics
type MutatingWebhookConfigurationData struct {
	LogEntryMetadata
	// MutatingWebhookConfiguration specific
	Webhooks []WebhookData `json:"webhooks"`
}

// PriorityClassData represents priorityclass-specific metrics
type PriorityClassData struct {
	LogEntryMetadata
	// PriorityClass specific
	Value            int32  `json:"value"`
	GlobalDefault    bool   `json:"globalDefault"`
	Description      string `json:"description"`
	PreemptionPolicy string `json:"preemptionPolicy"`
}

// RuntimeClassData represents runtimeclass-specific metrics
type RuntimeClassData struct {
	LogEntryMetadata
	// RuntimeClass specific
	Handler string `json:"handler"`
}

// VolumeAttachmentData represents volumeattachment-specific metrics
type VolumeAttachmentData struct {
	LogEntryMetadata
	// VolumeAttachment specific
	Attacher   string `json:"attacher"`
	VolumeName string `json:"volumeName"`
	NodeName   string `json:"nodeName"`
	Attached   bool   `json:"attached"`
}

// ValidatingAdmissionPolicyData represents validatingadmissionpolicy-specific metrics
type ValidatingAdmissionPolicyData struct {
	LogEntryMetadata
	FailurePolicy      string   `json:"failurePolicy"`
	MatchConstraints   []string `json:"matchConstraints"`
	Validations        []string `json:"validations"`
	AuditAnnotations   []string `json:"auditAnnotations"`
	MatchConditions    []string `json:"matchConditions"`
	Variables          []string `json:"variables"`
	ParamKind          string   `json:"paramKind"`
	ObservedGeneration int64    `json:"observedGeneration"`
	TypeChecking       string   `json:"typeChecking"`
	ExpressionWarnings []string `json:"expressionWarnings"`
}

// ValidatingAdmissionPolicyBindingData represents validatingadmissionpolicybinding-specific metrics
type ValidatingAdmissionPolicyBindingData struct {
	LogEntryMetadata
	PolicyName         string   `json:"policyName"`
	ParamRef           string   `json:"paramRef"`
	MatchResources     []string `json:"matchResources"`
	ValidationActions  []string `json:"validationActions"`
	ObservedGeneration int64    `json:"observedGeneration"`
}

// ValidatingWebhookConfigurationData represents validatingwebhookconfiguration-specific metrics
type ValidatingWebhookConfigurationData struct {
	LogEntryMetadata
	// ValidatingWebhookConfiguration specific
	Webhooks []WebhookData `json:"webhooks"`
}
