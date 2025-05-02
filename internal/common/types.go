package common

// DeploymentConfig holds the configuration for a deployment
type DeploymentConfig struct {
	// Deployment information
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Image     string            `json:"image"`
	Tag       string            `json:"tag"`
	Ports     []int32           `json:"ports"`
	Replicas  int32             `json:"replicas"`
	Labels    map[string]string `json:"labels"`
	EnvVars   map[string]string `json:"envVars"`

	// Resource specifications
	CPURequest    string `json:"cpuRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryRequest string `json:"memoryRequest"`
	MemoryLimit   string `json:"memoryLimit"`

	// Autoscaling configuration
	AutoscalingEnabled      bool  `json:"autoscalingEnabled"`
	MinReplicas             int32 `json:"minReplicas"`
	MaxReplicas             int32 `json:"maxReplicas"`
	CPUTargetUtilization    int32 `json:"cpuTargetUtilization"`
	MemoryTargetUtilization int32 `json:"memoryTargetUtilization"`

	// KEDA specific configuration
	KEDAEnabled         bool              `json:"kedaEnabled"`
	KEDAScalerType      string            `json:"kedaScalerType"`
	KEDAScalerMetadata  map[string]string `json:"kedaScalerMetadata"`
	KEDAPollingInterval int32             `json:"kedaPollingInterval"`
	KEDACooldownPeriod  int32             `json:"kedaCooldownPeriod"`
}

// DeploymentStatus holds the status information for a deployment
type DeploymentStatus struct {
	Name             string   `json:"name"`
	Namespace        string   `json:"namespace"`
	Available        bool     `json:"available"`
	ReadyReplicas    int32    `json:"readyReplicas"`
	TotalReplicas    int32    `json:"totalReplicas"`
	CPUUsage         string   `json:"cpuUsage"`
	MemoryUsage      string   `json:"memoryUsage"`
	ServiceEndpoints []string `json:"serviceEndpoints"`
	Events           []string `json:"events"`
	LastRestartTime  string   `json:"lastRestartTime"`
	RestartCount     int32    `json:"restartCount"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
}

// ClusterInfo holds information about the connected Kubernetes cluster
type ClusterInfo struct {
	ServerVersion  string            `json:"serverVersion"`
	Nodes          int               `json:"nodes"`
	Namespaces     []string          `json:"namespaces"`
	InstalledTools map[string]string `json:"installedTools"`
	Message        string            `json:"message"`
	Status         string            `json:"status"`
}
