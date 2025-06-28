package types

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogEntry represents a single log entry for a Kubernetes resource
type LogEntry struct {
	Timestamp    time.Time      `json:"timestamp"`
	ResourceType string         `json:"resourceType"`
	Name         string         `json:"name"`
	Namespace    string         `json:"namespace"`
	Data         map[string]any `json:"data"`
}

// DeploymentData represents deployment-specific metrics (matching kube-state-metrics)
type DeploymentData struct {
	// Basic deployment info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Replica counts
	DesiredReplicas     int32 `json:"desiredReplicas"`
	CurrentReplicas     int32 `json:"currentReplicas"`
	ReadyReplicas       int32 `json:"readyReplicas"`
	AvailableReplicas   int32 `json:"availableReplicas"`
	UnavailableReplicas int32 `json:"unavailableReplicas"`
	UpdatedReplicas     int32 `json:"updatedReplicas"`

	// Deployment status
	ObservedGeneration  int64 `json:"observedGeneration"`
	ReplicasDesired     int32 `json:"replicasDesired"`
	ReplicasAvailable   int32 `json:"replicasAvailable"`
	ReplicasUnavailable int32 `json:"replicasUnavailable"`
	ReplicasUpdated     int32 `json:"replicasUpdated"`

	// Strategy info
	StrategyType                        string `json:"strategyType"`
	StrategyRollingUpdateMaxSurge       int32  `json:"strategyRollingUpdateMaxSurge"`
	StrategyRollingUpdateMaxUnavailable int32  `json:"strategyRollingUpdateMaxUnavailable"`

	// Conditions
	ConditionAvailable      bool `json:"conditionAvailable"`
	ConditionProgressing    bool `json:"conditionProgressing"`
	ConditionReplicaFailure bool `json:"conditionReplicaFailure"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Missing from KSM
	Paused             bool  `json:"paused"`
	MetadataGeneration int64 `json:"metadataGeneration"`
}

// PodData represents pod-specific metrics (matching kube-state-metrics)
type PodData struct {
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
	RestartCount  int32  `json:"restartCount"`
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Labels and annotations
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

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
	ResourceLimits         map[string]string `json:"resourceLimits"`
	ResourceRequests       map[string]string `json:"resourceRequests"`
}

// TolerationData represents pod toleration information
type TolerationData struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
	Operator string `json:"operator"`
}

// PVCData represents persistent volume claim information
type PVCData struct {
	ClaimName string `json:"claimName"`
	ReadOnly  bool   `json:"readOnly"`
}

// ContainerData represents container-specific metrics (matching kube-state-metrics)
type ContainerData struct {
	// Basic container info
	Name    string `json:"name"`
	Image   string `json:"image"`
	ImageID string `json:"imageID"`
	PodName string `json:"podName"`

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
	// Basic service info
	Type           string            `json:"type"`
	ClusterIP      string            `json:"clusterIP"`
	ExternalIP     string            `json:"externalIP"`
	LoadBalancerIP string            `json:"loadBalancerIP"`
	Ports          []ServicePortData `json:"ports"`
	Selector       map[string]string `json:"selector"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`

	// Service status
	EndpointsCount int `json:"endpointsCount"`

	// Load balancer info
	LoadBalancerIngress []LoadBalancerIngressData `json:"loadBalancerIngress"`

	// Session affinity
	SessionAffinity string `json:"sessionAffinity"`

	// External name
	ExternalName string `json:"externalName"`

	// Created by info
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Missing from KSM
	CreatedTimestamp                      int64  `json:"createdTimestamp"`
	InternalTrafficPolicy                 string `json:"internalTrafficPolicy"`
	ExternalTrafficPolicy                 string `json:"externalTrafficPolicy"`
	SessionAffinityClientIPTimeoutSeconds int32  `json:"sessionAffinityClientIPTimeoutSeconds"`
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

	// Node info
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Node addresses
	InternalIP string `json:"internalIP"`
	ExternalIP string `json:"externalIP"`
	Hostname   string `json:"hostname"`

	// Node status details
	Unschedulable bool `json:"unschedulable"`
	Ready         bool `json:"ready"`

	// Created by info
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Missing from KSM
	CreatedTimestamp  int64       `json:"createdTimestamp"`
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
	// Basic replicaset info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Replicaset specific
	IsCurrent bool `json:"isCurrent"` // Whether this is the current replicaset for its owner
}

// StatefulSetData represents statefulset-specific metrics (matching kube-state-metrics)
type StatefulSetData struct {
	// Basic statefulset info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Statefulset specific
	ServiceName         string `json:"serviceName"`
	PodManagementPolicy string `json:"podManagementPolicy"`
	UpdateStrategy      string `json:"updateStrategy"`
}

// DaemonSetData represents daemonset-specific metrics (matching kube-state-metrics)
type DaemonSetData struct {
	// Basic daemonset info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Daemonset specific
	UpdateStrategy string `json:"updateStrategy"`
}

// NamespaceData represents namespace-specific metrics (matching kube-state-metrics)
type NamespaceData struct {
	// Basic namespace info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Namespace status
	Phase string `json:"phase"`

	// Conditions
	ConditionActive      bool `json:"conditionActive"`
	ConditionTerminating bool `json:"conditionTerminating"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Namespace specific
	DeletionTimestamp *metav1.Time `json:"deletionTimestamp"`
}

// JobData represents job-specific metrics (matching kube-state-metrics)
type JobData struct {
	// Basic job info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// Job specific
	JobType string `json:"jobType"` // "Job" or "CronJob"
	Suspend *bool  `json:"suspend"`
}

// CronJobData represents cronjob-specific metrics (matching kube-state-metrics)
type CronJobData struct {
	// Basic cronjob info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// ConfigMapData represents configmap-specific metrics (matching kube-state-metrics)
type ConfigMapData struct {
	// Basic configmap info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ConfigMap specific
	DataKeys []string `json:"dataKeys"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// SecretData represents secret-specific metrics (matching kube-state-metrics)
type SecretData struct {
	// Basic secret info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Secret specific
	Type     string   `json:"type"`
	DataKeys []string `json:"dataKeys"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// PersistentVolumeClaimData represents persistentvolumeclaim-specific metrics (matching kube-state-metrics)
type PersistentVolumeClaimData struct {
	// Basic persistentvolumeclaim info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// PVC specific
	RequestStorage string `json:"requestStorage"`
	UsedStorage    string `json:"usedStorage"`
}

// IngressData represents ingress-specific metrics (matching kube-state-metrics)
type IngressData struct {
	// Basic ingress info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
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
	// Basic horizontalpodautoscaler info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// HPA specific
	ScaleTargetRef  string `json:"scaleTargetRef"`
	ScaleTargetKind string `json:"scaleTargetKind"`
}

// ServiceAccountData represents serviceaccount-specific metrics (matching kube-state-metrics)
type ServiceAccountData struct {
	// Basic serviceaccount info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ServiceAccount specific
	Secrets          []string `json:"secrets"`
	ImagePullSecrets []string `json:"imagePullSecrets"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// ServiceAccount specific
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken"`
}

// EndpointsData represents endpoints-specific metrics (matching kube-state-metrics)
type EndpointsData struct {
	// Basic endpoints info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Endpoints specific
	Addresses []EndpointAddressData `json:"addresses"`
	Ports     []EndpointPortData    `json:"ports"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

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
	// Basic persistentvolume info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// PersistentVolume specific
	CapacityBytes          int64  `json:"capacityBytes"`
	AccessModes            string `json:"accessModes"`
	ReclaimPolicy          string `json:"reclaimPolicy"`
	Status                 string `json:"status"`
	StorageClassName       string `json:"storageClassName"`
	VolumeMode             string `json:"volumeMode"`
	VolumePluginName       string `json:"volumePluginName"`
	PersistentVolumeSource string `json:"persistentVolumeSource"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// PersistentVolume specific
	IsDefaultClass bool `json:"isDefaultClass"`
}

// ResourceQuotaData represents resourcequota-specific metrics (matching kube-state-metrics)
type ResourceQuotaData struct {
	// Basic resourcequota info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ResourceQuota specific
	Hard map[string]int64 `json:"hard"`
	Used map[string]int64 `json:"used"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// ResourceQuota specific
	Scopes []string `json:"scopes"`
}

// PodDisruptionBudgetData represents poddisruptionbudget-specific metrics (matching kube-state-metrics)
type PodDisruptionBudgetData struct {
	// Basic poddisruptionbudget info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

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

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// CRDData represents generic CRD metrics (similar to kube-state-metrics)
type CRDData struct {
	// Basic CRD info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// CRD specific
	APIVersion   string                 `json:"apiVersion"`
	Kind         string                 `json:"kind"`
	Spec         map[string]interface{} `json:"spec"`
	Status       map[string]interface{} `json:"status"`
	CustomFields map[string]interface{} `json:"customFields"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// StorageClassData represents storageclass-specific metrics (matching kube-state-metrics)
type StorageClassData struct {
	// Basic storageclass info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// StorageClass specific
	Provisioner          string                 `json:"provisioner"`
	ReclaimPolicy        string                 `json:"reclaimPolicy"`
	VolumeBindingMode    string                 `json:"volumeBindingMode"`
	AllowVolumeExpansion bool                   `json:"allowVolumeExpansion"`
	Parameters           map[string]string      `json:"parameters"`
	MountOptions         []string               `json:"mountOptions"`
	AllowedTopologies    map[string]interface{} `json:"allowedTopologies"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// StorageClass specific
	IsDefaultClass bool `json:"isDefaultClass"`
}

// NetworkPolicyData represents networkpolicy-specific metrics (matching kube-state-metrics)
type NetworkPolicyData struct {
	// Basic networkpolicy info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// NetworkPolicy specific
	PolicyTypes  []string                   `json:"policyTypes"`
	IngressRules []NetworkPolicyIngressRule `json:"ingressRules"`
	EgressRules  []NetworkPolicyEgressRule  `json:"egressRules"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
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
	PodSelector       map[string]string      `json:"podSelector"`
	NamespaceSelector map[string]string      `json:"namespaceSelector"`
	IPBlock           map[string]interface{} `json:"ipBlock"`
}

// ReplicationControllerData represents replicationcontroller-specific metrics (matching kube-state-metrics)
type ReplicationControllerData struct {
	// Basic replicationcontroller info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ReplicationController specific
	DesiredReplicas      int32 `json:"desiredReplicas"`
	CurrentReplicas      int32 `json:"currentReplicas"`
	ReadyReplicas        int32 `json:"readyReplicas"`
	AvailableReplicas    int32 `json:"availableReplicas"`
	FullyLabeledReplicas int32 `json:"fullyLabeledReplicas"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`

	// ReplicationController specific
	ObservedGeneration int64 `json:"observedGeneration"`
}

// LimitRangeData represents limitrange-specific metrics (matching kube-state-metrics)
type LimitRangeData struct {
	// Basic limitrange info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// LimitRange specific
	Limits []LimitRangeItem `json:"limits"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
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

// LeaseData represents lease-specific metrics (matching kube-state-metrics)
type LeaseData struct {
	// Basic lease info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Lease specific
	HolderIdentity       string     `json:"holderIdentity"`
	LeaseDurationSeconds int32      `json:"leaseDurationSeconds"`
	RenewTime            *time.Time `json:"renewTime"`
	AcquireTime          *time.Time `json:"acquireTime"`
	LeaseTransitions     int32      `json:"leaseTransitions"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// RoleData represents role-specific metrics (matching kube-state-metrics)
type RoleData struct {
	// Basic role info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// Role specific
	Rules []PolicyRule `json:"rules"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// ClusterRoleData represents clusterrole-specific metrics (matching kube-state-metrics)
type ClusterRoleData struct {
	// Basic clusterrole info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ClusterRole specific
	Rules []PolicyRule `json:"rules"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// RoleBindingData represents rolebinding-specific metrics (matching kube-state-metrics)
type RoleBindingData struct {
	// Basic rolebinding info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// RoleBinding specific
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// ClusterRoleBindingData represents clusterrolebinding-specific metrics (matching kube-state-metrics)
type ClusterRoleBindingData struct {
	// Basic clusterrolebinding info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// ClusterRoleBinding specific
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// PolicyRule represents a policy rule in RBAC
type PolicyRule struct {
	APIGroups     []string `json:"apiGroups"`
	Resources     []string `json:"resources"`
	ResourceNames []string `json:"resourceNames"`
	Verbs         []string `json:"verbs"`
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
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	APIGroup  string `json:"apiGroup"`
}

// VolumeAttachmentData represents volumeattachment-specific metrics (matching kube-state-metrics)
type VolumeAttachmentData struct {
	// Basic volumeattachment info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// VolumeAttachment specific
	Attacher           string            `json:"attacher"`
	VolumeName         string            `json:"volumeName"`
	NodeName           string            `json:"nodeName"`
	Attached           bool              `json:"attached"`
	AttachmentMetadata map[string]string `json:"attachmentMetadata"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}

// CertificateSigningRequestData represents certificatesigningrequest-specific metrics (matching kube-state-metrics)
type CertificateSigningRequestData struct {
	// Basic certificatesigningrequest info
	CreatedTimestamp int64             `json:"createdTimestamp"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`

	// CertificateSigningRequest specific
	Status            string   `json:"status"`
	SignerName        string   `json:"signerName"`
	ExpirationSeconds *int32   `json:"expirationSeconds"`
	Usages            []string `json:"usages"`

	// Metadata
	CreatedByKind string `json:"createdByKind"`
	CreatedByName string `json:"createdByName"`
}
