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
