package health

import (
	"context"
	"fmt"
	"time"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	k8s "github.com/ethicalaakash/k8s-automator/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealthMonitor defines the interface for health monitoring operations
type HealthMonitor interface {
	GetDeploymentStatus(ctx context.Context, namespace, name string) (*common.DeploymentStatus, error)
}

// Monitor handles the health monitoring of deployments
type Monitor struct {
	k8sClient k8s.KubernetesClient
}

// NewMonitor creates a new health monitor
func NewMonitor(k8sClient k8s.KubernetesClient) HealthMonitor {
	return &Monitor{
		k8sClient: k8sClient,
	}
}

// GetDeploymentStatus retrieves the health status of a deployment
func (m *Monitor) GetDeploymentStatus(ctx context.Context, namespace, name string) (*common.DeploymentStatus, error) {
	// Initialize response
	status := &common.DeploymentStatus{
		Name:      name,
		Namespace: namespace,
		Status:    "Unknown",
		Message:   "Status could not be determined",
	}

	// Get deployment
	deployment, err := m.k8sClient.Deployments(namespace).Get(ctx, name)
	if err != nil {
		return status, fmt.Errorf("error getting deployment: %v", err)
	}

	// Get basic deployment status
	status.TotalReplicas = deployment.Status.Replicas
	status.ReadyReplicas = deployment.Status.ReadyReplicas
	status.Available = deployment.Status.ReadyReplicas > 0

	// Get pods for this deployment
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	pods, err := m.k8sClient.Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return status, fmt.Errorf("error getting pods: %v", err)
	}

	// Analyze pod status and collect metrics
	restartCount := int32(0)
	var lastRestartTime time.Time

	// If we have no pods, set not ready status
	if len(pods.Items) == 0 {
		status.Status = "Not Ready"
		status.Message = "No pods found for deployment"
		return status, nil
	}

	// Check pod statuses and collect metrics
	for _, pod := range pods.Items {
		// Check if pod is running and ready
		podReady := isPodReady(&pod)
		if !podReady {
			// Add information about non-ready pods
			status.Events = append(status.Events,
				fmt.Sprintf("Pod %s is not ready: %s", pod.Name, getPodStatusReason(&pod)))
		}

		// Get restart counts
		for _, containerStatus := range pod.Status.ContainerStatuses {
			restartCount += containerStatus.RestartCount

			// Check for last restart time
			if containerStatus.LastTerminationState.Terminated != nil {
				terminatedTime := containerStatus.LastTerminationState.Terminated.FinishedAt.Time
				if terminatedTime.After(lastRestartTime) {
					lastRestartTime = terminatedTime
				}
			}
		}

		// Collect pod events
		events, err := m.k8sClient.Events(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", pod.Name),
		})
		if err == nil {
			for _, event := range events.Items {
				if event.Type == "Warning" {
					status.Events = append(status.Events, fmt.Sprintf("%s: %s", event.Reason, event.Message))
				}
			}
		}
	}

	// Update status with restart information
	status.RestartCount = restartCount
	if !lastRestartTime.IsZero() {
		status.LastRestartTime = lastRestartTime.Format(time.RFC3339)
	}

	// For this example, we'll skip metrics since k8s.io/metrics is not part of the standard client
	// In a production environment, you would configure this to use metrics-server
	status.CPUUsage = "Metrics unavailable"
	status.MemoryUsage = "Metrics unavailable"

	// Get service endpoints
	services, err := m.k8sClient.Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", name),
	})
	if err == nil {
		for _, service := range services.Items {
			if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				for _, ingress := range service.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						for _, port := range service.Spec.Ports {
							status.ServiceEndpoints = append(
								status.ServiceEndpoints,
								fmt.Sprintf("http://%s:%d", ingress.IP, port.Port),
							)
						}
					}
				}
			} else if service.Spec.Type == corev1.ServiceTypeNodePort {
				for _, port := range service.Spec.Ports {
					status.ServiceEndpoints = append(
						status.ServiceEndpoints,
						fmt.Sprintf("<node-ip>:%d", port.NodePort),
					)
				}
			} else {
				status.ServiceEndpoints = append(
					status.ServiceEndpoints,
					fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
				)
			}
		}
	}

	// Set overall status based on collected information
	if status.ReadyReplicas == status.TotalReplicas && status.ReadyReplicas > 0 {
		status.Status = "Healthy"
		status.Message = "Deployment is running normally"
	} else if status.ReadyReplicas > 0 {
		status.Status = "Degraded"
		status.Message = fmt.Sprintf("Only %d/%d replicas are ready", status.ReadyReplicas, status.TotalReplicas)
	} else {
		status.Status = "Unhealthy"
		status.Message = "No replicas are ready"
	}

	return status, nil
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// getPodStatusReason returns a human readable explanation why a pod is not ready
func getPodStatusReason(pod *corev1.Pod) string {
	if pod.Status.Phase != corev1.PodRunning {
		return string(pod.Status.Phase)
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
			return condition.Reason
		}
	}

	return "Unknown reason"
}
