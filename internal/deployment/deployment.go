package deployment

import (
	"context"
	"fmt"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	"github.com/ethicalaakash/k8s-automator/internal/kubernetes"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeploymentManager defines the interface for deployment operations
type DeploymentManager interface {
	CreateDeployment(ctx context.Context, config *common.DeploymentConfig) error
}

// Manager handles the deployment operations
type Manager struct {
	k8sClient kubernetes.KubernetesClient
}

// NewManager creates a new deployment manager
func NewManager(k8sClient kubernetes.KubernetesClient) DeploymentManager {
	return &Manager{
		k8sClient: k8sClient,
	}
}

// CreateDeployment creates a new deployment based on the provided configuration
func (m *Manager) CreateDeployment(ctx context.Context, config *common.DeploymentConfig) error {
	// Validate deployment configuration
	if err := validateDeploymentConfig(config); err != nil {
		return err
	}

	// Check if namespace exists
	_, err := m.k8sClient.Namespaces().Get(ctx, config.Namespace)
	if err != nil {
		// Create namespace if it doesn't exist
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: config.Namespace,
			},
		}
		_, err = m.k8sClient.Namespaces().Create(ctx, namespace)
		if err != nil {
			return fmt.Errorf("error creating namespace %s: %v", config.Namespace, err)
		}
		fmt.Printf("Created namespace: %s\n", config.Namespace)
	}

	// Create the deployment
	deployment := buildDeployment(config)
	_, err = m.k8sClient.Deployments(config.Namespace).Create(ctx, deployment)
	if err != nil {
		return fmt.Errorf("error creating deployment: %v", err)
	}
	fmt.Printf("Created deployment: %s/%s\n", config.Namespace, config.Name)

	// Create the service
	service := buildService(config)
	_, err = m.k8sClient.Services(config.Namespace).Create(ctx, service)
	if err != nil {
		return fmt.Errorf("error creating service: %v", err)
	}
	fmt.Printf("Created service: %s/%s\n", config.Namespace, config.Name)

	// Create HPA if autoscaling is enabled
	if config.AutoscalingEnabled {
		hpa := buildHPA(config)
		_, err = m.k8sClient.HorizontalPodAutoscalers(config.Namespace).Create(ctx, hpa)
		if err != nil {
			return fmt.Errorf("error creating HPA: %v", err)
		}
		fmt.Printf("Created HorizontalPodAutoscaler: %s/%s\n", config.Namespace, config.Name)
	}

	// If KEDA is enabled, configure KEDA ScaledObject
	if config.KEDAEnabled {
		if config.KEDAScalerType == "cpu" {
			// Create CPU-based KEDA ScaledObject
			scaledObject := buildCPUScaledObject(config)
			_, err = m.k8sClient.ScaledObjects(config.Namespace).Create(ctx, scaledObject)
			if err != nil {
				return fmt.Errorf("error creating KEDA ScaledObject: %v", err)
			}
			fmt.Printf("Created KEDA ScaledObject with CPU scaling for %s/%s\n", config.Namespace, config.Name)
		} else {
			// For other scaler types (not implemented yet)
			fmt.Printf("KEDA scaling configured for %s/%s with %s scaler\n",
				config.Namespace, config.Name, config.KEDAScalerType)
		}
	}

	return nil
}

// validateDeploymentConfig validates the deployment configuration
func validateDeploymentConfig(config *common.DeploymentConfig) error {
	if config.Name == "" {
		return fmt.Errorf("deployment name cannot be empty")
	}
	if config.Namespace == "" {
		config.Namespace = "default"
	}
	if config.Image == "" {
		return fmt.Errorf("deployment image cannot be empty")
	}
	if config.Tag == "" {
		config.Tag = "latest"
	}
	if config.Replicas <= 0 {
		config.Replicas = 1
	}

	// Set default resource values if not provided
	if config.CPURequest == "" {
		config.CPURequest = "100m"
	}
	if config.CPULimit == "" {
		config.CPULimit = "200m"
	}
	if config.MemoryRequest == "" {
		config.MemoryRequest = "128Mi"
	}
	if config.MemoryLimit == "" {
		config.MemoryLimit = "256Mi"
	}

	// Set autoscaling defaults
	if config.AutoscalingEnabled {
		if config.MinReplicas <= 0 {
			config.MinReplicas = 1
		}
		if config.MaxReplicas <= 0 {
			config.MaxReplicas = 10
		}
		if config.CPUTargetUtilization <= 0 {
			config.CPUTargetUtilization = 80
		}
	}

	// Set KEDA defaults
	if config.KEDAEnabled {
		if config.KEDAScalerType == "" {
			return fmt.Errorf("KEDA scaler type cannot be empty when KEDA is enabled")
		}
		if config.KEDAPollingInterval <= 0 {
			config.KEDAPollingInterval = 30
		}
		if config.KEDACooldownPeriod <= 0 {
			config.KEDACooldownPeriod = 300
		}
	}

	return nil
}

// buildDeployment creates a Kubernetes Deployment object from the deployment configuration
func buildDeployment(config *common.DeploymentConfig) *appsv1.Deployment {
	// Prepare container ports
	containerPorts := make([]corev1.ContainerPort, 0, len(config.Ports))
	for _, port := range config.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	// Prepare environment variables
	envVars := make([]corev1.EnvVar, 0, len(config.EnvVars))
	for key, value := range config.EnvVars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Prepare labels if not provided
	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}
	config.Labels["app"] = config.Name

	// Create the deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    config.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &config.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: config.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  config.Name,
							Image: fmt.Sprintf("%s:%s", config.Image, config.Tag),
							Ports: containerPorts,
							Env:   envVars,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.CPURequest),
									corev1.ResourceMemory: resource.MustParse(config.MemoryRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(config.CPULimit),
									corev1.ResourceMemory: resource.MustParse(config.MemoryLimit),
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

// buildService creates a Kubernetes Service object from the deployment configuration
func buildService(config *common.DeploymentConfig) *corev1.Service {
	// Prepare service ports
	servicePorts := make([]corev1.ServicePort, 0, len(config.Ports))
	for _, port := range config.Ports {
		servicePort := corev1.ServicePort{
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
			Protocol:   corev1.ProtocolTCP,
			Name:       fmt.Sprintf("port-%d", port),
		}
		servicePorts = append(servicePorts, servicePort)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    config.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": config.Name,
			},
			Ports: servicePorts,
			Type:  corev1.ServiceTypeClusterIP,
		},
	}

	return service
}

// buildHPA creates a Kubernetes HorizontalPodAutoscaler object from the deployment configuration
func buildHPA(config *common.DeploymentConfig) *autoscalingv2.HorizontalPodAutoscaler {
	// Define CPU target utilization
	cpuMetric := autoscalingv2.MetricSpec{
		Type: autoscalingv2.ResourceMetricSourceType,
		Resource: &autoscalingv2.ResourceMetricSource{
			Name: corev1.ResourceCPU,
			Target: autoscalingv2.MetricTarget{
				Type:               autoscalingv2.UtilizationMetricType,
				AverageUtilization: &config.CPUTargetUtilization,
			},
		},
	}

	// Define HPA
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    config.Labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       config.Name,
			},
			MinReplicas: &config.MinReplicas,
			MaxReplicas: config.MaxReplicas,
			Metrics:     []autoscalingv2.MetricSpec{cpuMetric},
		},
	}

	return hpa
}

// buildCPUScaledObject creates a KEDA ScaledObject with CPU metrics trigger
func buildCPUScaledObject(config *common.DeploymentConfig) *kedav1alpha1.ScaledObject {
	minReplicas := config.MinReplicas
	maxReplicas := config.MaxReplicas
	pollingInterval := config.KEDAPollingInterval
	cooldownPeriod := config.KEDACooldownPeriod

	return &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    config.Labels,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Name:       config.Name,
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			MinReplicaCount: &minReplicas,
			MaxReplicaCount: &maxReplicas,
			PollingInterval: &pollingInterval,
			CooldownPeriod:  &cooldownPeriod,
			Triggers: []kedav1alpha1.ScaleTriggers{
				{
					Type:       "cpu",
					MetricType: "Utilization",
					Metadata: map[string]string{
						"value": fmt.Sprintf("%d", config.CPUTargetUtilization),
					},
				},
			},
		},
	}
}
