package deployment

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	kmock "github.com/ethicalaakash/k8s-automator/mocks/kubernetes"
	"github.com/golang/mock/gomock"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
)

// TestNewManager tests the creation of a deployment manager
func TestNewManager(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock Kubernetes client
	mockClient := kmock.NewMockKubernetesClient(ctrl)

	// Create manager
	manager := NewManager(mockClient)

	// Verify manager was created
	if manager == nil {
		t.Error("Expected a non-nil DeploymentManager")
	}
}

func TestValidateDeploymentConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *common.DeploymentConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &common.DeploymentConfig{
				Name:  "test-app",
				Image: "nginx",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			config: &common.DeploymentConfig{
				Image: "nginx",
			},
			wantErr: true,
		},
		{
			name: "Empty image",
			config: &common.DeploymentConfig{
				Name: "test-app",
			},
			wantErr: true,
		},
		{
			name: "KEDA enabled but no scaler type",
			config: &common.DeploymentConfig{
				Name:        "test-app",
				Image:       "nginx",
				KEDAEnabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeploymentConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDeploymentConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildDeployment(t *testing.T) {
	// Test that the deployment is built correctly with minimal valid config
	config := &common.DeploymentConfig{
		Name:      "test-app",
		Namespace: "default",
		Image:     "nginx",
		Tag:       "latest",
		Ports:     []int32{80},
		Replicas:  1,
	}

	// Set default values before building
	_ = validateDeploymentConfig(config)

	deployment := buildDeployment(config)

	// Check basic properties
	if deployment.Name != config.Name {
		t.Errorf("Deployment name = %v, want %v", deployment.Name, config.Name)
	}

	if deployment.Namespace != config.Namespace {
		t.Errorf("Deployment namespace = %v, want %v", deployment.Namespace, config.Namespace)
	}

	if *deployment.Spec.Replicas != config.Replicas {
		t.Errorf("Deployment replicas = %v, want %v", *deployment.Spec.Replicas, config.Replicas)
	}

	// Check container
	container := deployment.Spec.Template.Spec.Containers[0]
	expectedImage := config.Image + ":" + config.Tag
	if container.Image != expectedImage {
		t.Errorf("Container image = %v, want %v", container.Image, expectedImage)
	}

	// Check ports
	if len(container.Ports) != len(config.Ports) {
		t.Errorf("Number of ports = %v, want %v", len(container.Ports), len(config.Ports))
	} else if container.Ports[0].ContainerPort != config.Ports[0] {
		t.Errorf("Port = %v, want %v", container.Ports[0].ContainerPort, config.Ports[0])
	}
}

func TestBuildService(t *testing.T) {
	// Test that the service is built correctly
	config := &common.DeploymentConfig{
		Name:      "test-app",
		Namespace: "default",
		Ports:     []int32{80, 443},
	}

	service := buildService(config)

	// Check basic properties
	if service.Name != config.Name {
		t.Errorf("Service name = %v, want %v", service.Name, config.Name)
	}

	if service.Namespace != config.Namespace {
		t.Errorf("Service namespace = %v, want %v", service.Namespace, config.Namespace)
	}

	// Check ports
	if len(service.Spec.Ports) != len(config.Ports) {
		t.Errorf("Number of service ports = %v, want %v", len(service.Spec.Ports), len(config.Ports))
	}

	// Check selector
	if service.Spec.Selector["app"] != config.Name {
		t.Errorf("Service selector app = %v, want %v", service.Spec.Selector["app"], config.Name)
	}
}

// TestCreateDeployment tests the CreateDeployment method
func TestCreateDeployment(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test basic deployment creation
	t.Run("Simple Deployment Success", func(t *testing.T) {
		// Create test configuration
		config := &common.DeploymentConfig{
			Name:      "test-app",
			Namespace: "test-ns",
			Image:     "nginx",
			Tag:       "latest",
			Ports:     []int32{80},
			Replicas:  2,
		}

		// Validate and set defaults (this is what the real method does)
		if err := validateDeploymentConfig(config); err != nil {
			t.Fatalf("Unexpected validation error: %v", err)
		}

		// Create the expected resources directly instead of mocking Kubernetes API
		deployment := buildDeployment(config)
		service := buildService(config)

		// Verify deployment was built correctly
		if deployment.Name != config.Name {
			t.Errorf("Expected deployment name %s, got %s", config.Name, deployment.Name)
		}
		if deployment.Namespace != config.Namespace {
			t.Errorf("Expected deployment namespace %s, got %s", config.Namespace, deployment.Namespace)
		}
		if *deployment.Spec.Replicas != config.Replicas {
			t.Errorf("Expected %d replicas, got %d", config.Replicas, *deployment.Spec.Replicas)
		}

		// Check container image
		expectedImage := fmt.Sprintf("%s:%s", config.Image, config.Tag)
		actualImage := deployment.Spec.Template.Spec.Containers[0].Image
		if actualImage != expectedImage {
			t.Errorf("Expected image %s, got %s", expectedImage, actualImage)
		}

		// Check ports
		if len(deployment.Spec.Template.Spec.Containers[0].Ports) != len(config.Ports) {
			t.Errorf("Expected %d ports, got %d", len(config.Ports), len(deployment.Spec.Template.Spec.Containers[0].Ports))
		}

		// Verify service was built correctly
		if service.Name != config.Name {
			t.Errorf("Expected service name %s, got %s", config.Name, service.Name)
		}
		if service.Namespace != config.Namespace {
			t.Errorf("Expected service namespace %s, got %s", config.Namespace, service.Namespace)
		}
		if len(service.Spec.Ports) != len(config.Ports) {
			t.Errorf("Expected %d ports, got %d", len(config.Ports), len(service.Spec.Ports))
		}
	})

	// Test autoscaling
	t.Run("Deployment with Autoscaling", func(t *testing.T) {
		// Setup test config with autoscaling
		config := &common.DeploymentConfig{
			Name:                 "test-app-hpa",
			Namespace:            "test-ns",
			Image:                "nginx",
			Ports:                []int32{80},
			AutoscalingEnabled:   true,
			MinReplicas:          2,
			MaxReplicas:          5,
			CPUTargetUtilization: 70,
		}

		// Validate and set defaults
		if err := validateDeploymentConfig(config); err != nil {
			t.Fatalf("Unexpected validation error: %v", err)
		}

		// Create the HPA directly
		hpa := buildHPA(config)

		// Verify HPA was built correctly
		if hpa.Name != config.Name {
			t.Errorf("Expected HPA name %s, got %s", config.Name, hpa.Name)
		}
		if hpa.Namespace != config.Namespace {
			t.Errorf("Expected HPA namespace %s, got %s", config.Namespace, hpa.Namespace)
		}
		if *hpa.Spec.MinReplicas != config.MinReplicas {
			t.Errorf("Expected min replicas %d, got %d", config.MinReplicas, *hpa.Spec.MinReplicas)
		}
		if hpa.Spec.MaxReplicas != config.MaxReplicas {
			t.Errorf("Expected max replicas %d, got %d", config.MaxReplicas, hpa.Spec.MaxReplicas)
		}
	})

	// Test for error cases
	t.Run("Error Cases", func(t *testing.T) {
		testCases := []struct {
			name     string
			config   *common.DeploymentConfig
			expected string
		}{
			{
				name: "Missing deployment name",
				config: &common.DeploymentConfig{
					Image: "nginx",
				},
				expected: "deployment name cannot be empty",
			},
			{
				name: "Missing image",
				config: &common.DeploymentConfig{
					Name: "test-app",
				},
				expected: "deployment image cannot be empty",
			},
			{
				name: "KEDA enabled without scaler type",
				config: &common.DeploymentConfig{
					Name:        "test-app",
					Image:       "nginx",
					KEDAEnabled: true,
				},
				expected: "KEDA scaler type cannot be empty",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Directly call validateDeploymentConfig
				err := validateDeploymentConfig(tc.config)

				// Verify the error
				if err == nil {
					t.Error("Expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tc.expected) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.expected, err.Error())
				}
			})
		}
	})
}

// TestCreateDeploymentWithMocks tests the CreateDeployment method with proper mocking of all Kubernetes clients
func TestCreateDeploymentWithMocks(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock Kubernetes client
	mockK8sClient := kmock.NewMockKubernetesClient(ctrl)

	// Create mock resource interfaces
	mockNamespaceClient := kmock.NewMockNamespaceInterface(ctrl)
	mockDeploymentClient := kmock.NewMockDeploymentInterface(ctrl)
	mockServiceClient := kmock.NewMockServiceInterface(ctrl)
	mockHpaClient := kmock.NewMockHorizontalPodAutoscalerInterface(ctrl)
	mockScaledObjectClient := kmock.NewMockScaledObjectInterface(ctrl)

	// Create test cases
	testCases := []struct {
		name          string
		config        *common.DeploymentConfig
		setupMocks    func()
		expectError   bool
		errorContains string
	}{
		{
			name: "Successful deployment with existing namespace",
			config: &common.DeploymentConfig{
				Name:      "test-app",
				Namespace: "existing-ns",
				Image:     "nginx",
				Tag:       "latest",
				Ports:     []int32{80},
				Replicas:  2,
			},
			setupMocks: func() {
				// Setup namespace client to find existing namespace
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "existing-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("existing-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, deployment *appsv1.Deployment) {
						// Verify deployment properties
						if deployment.Name != "test-app" {
							t.Errorf("Expected deployment name 'test-app', got '%s'", deployment.Name)
						}
						if deployment.Namespace != "existing-ns" {
							t.Errorf("Expected namespace 'existing-ns', got '%s'", deployment.Namespace)
						}
						if *deployment.Spec.Replicas != int32(2) {
							t.Errorf("Expected 2 replicas, got %d", *deployment.Spec.Replicas)
						}

						// Check container image
						containers := deployment.Spec.Template.Spec.Containers
						if len(containers) != 1 {
							t.Errorf("Expected 1 container, got %d", len(containers))
						} else if containers[0].Image != "nginx:latest" {
							t.Errorf("Expected image 'nginx:latest', got '%s'", containers[0].Image)
						}
					}).
					Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("existing-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, service *corev1.Service) {
						// Verify service properties
						if service.Name != "test-app" {
							t.Errorf("Expected service name 'test-app', got '%s'", service.Name)
						}
						if len(service.Spec.Ports) != 1 {
							t.Errorf("Expected 1 port, got %d", len(service.Spec.Ports))
						}
						if service.Spec.Ports[0].Port != int32(80) {
							t.Errorf("Expected port 80, got %d", service.Spec.Ports[0].Port)
						}
					}).
					Return(&corev1.Service{}, nil).Times(1)
			},
			expectError: false,
		},
		{
			name: "Create namespace and deployment",
			config: &common.DeploymentConfig{
				Name:      "test-app",
				Namespace: "new-ns",
				Image:     "nginx",
				Ports:     []int32{80},
			},
			setupMocks: func() {
				// Setup namespace client to not find namespace, then create it
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(2)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "new-ns").Return(nil, fmt.Errorf("namespace not found")).Times(1)
				mockNamespaceClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, namespace *corev1.Namespace) {
						if namespace.Name != "new-ns" {
							t.Errorf("Expected namespace name 'new-ns', got '%s'", namespace.Name)
						}
					}).
					Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("new-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("new-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&corev1.Service{}, nil).Times(1)
			},
			expectError: false,
		},
		{
			name: "Deployment with HPA",
			config: &common.DeploymentConfig{
				Name:                 "test-app-hpa",
				Namespace:            "test-ns",
				Image:                "nginx",
				Ports:                []int32{80},
				AutoscalingEnabled:   true,
				MinReplicas:          2,
				MaxReplicas:          5,
				CPUTargetUtilization: 70,
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("test-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&corev1.Service{}, nil).Times(1)

				// Setup HPA client
				mockK8sClient.EXPECT().HorizontalPodAutoscalers("test-ns").Return(mockHpaClient).Times(1)
				mockHpaClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) {
						// Verify HPA properties
						if hpa.Name != "test-app-hpa" {
							t.Errorf("Expected HPA name 'test-app-hpa', got '%s'", hpa.Name)
						}
						if *hpa.Spec.MinReplicas != int32(2) {
							t.Errorf("Expected min replicas 2, got %d", *hpa.Spec.MinReplicas)
						}
						if hpa.Spec.MaxReplicas != int32(5) {
							t.Errorf("Expected max replicas 5, got %d", hpa.Spec.MaxReplicas)
						}

						// Check metrics
						if len(hpa.Spec.Metrics) != 1 {
							t.Errorf("Expected 1 metric, got %d", len(hpa.Spec.Metrics))
						} else {
							metric := hpa.Spec.Metrics[0]
							if metric.Resource == nil || metric.Resource.Name != corev1.ResourceCPU {
								t.Errorf("Expected CPU resource metric")
							}
							if metric.Resource.Target.AverageUtilization == nil || *metric.Resource.Target.AverageUtilization != int32(70) {
								t.Errorf("Expected CPU target utilization 70, got %v",
									metric.Resource.Target.AverageUtilization)
							}
						}
					}).
					Return(&autoscalingv2.HorizontalPodAutoscaler{}, nil).Times(1)
			},
			expectError: false,
		},
		{
			name: "Deployment with KEDA CPU scaling",
			config: &common.DeploymentConfig{
				Name:                 "test-app-keda-cpu",
				Namespace:            "test-ns",
				Image:                "worker",
				Ports:                []int32{8080},
				KEDAEnabled:          true,
				KEDAScalerType:       "cpu",
				MinReplicas:          2,
				MaxReplicas:          10,
				CPUTargetUtilization: 50,
				KEDAPollingInterval:  30,
				KEDACooldownPeriod:   300,
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("test-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&corev1.Service{}, nil).Times(1)

				// Setup ScaledObject client for CPU scaling
				mockK8sClient.EXPECT().ScaledObjects("test-ns").Return(mockScaledObjectClient).Times(1)
				mockScaledObjectClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, scaledObject *kedav1alpha1.ScaledObject) {
						// Verify ScaledObject properties
						if scaledObject.Name != "test-app-keda-cpu" {
							t.Errorf("Expected ScaledObject name 'test-app-keda-cpu', got '%s'", scaledObject.Name)
						}
						if scaledObject.Namespace != "test-ns" {
							t.Errorf("Expected namespace 'test-ns', got '%s'", scaledObject.Namespace)
						}

						// Verify trigger type
						if len(scaledObject.Spec.Triggers) != 1 {
							t.Errorf("Expected 1 trigger, got %d", len(scaledObject.Spec.Triggers))
							return
						}

						if scaledObject.Spec.Triggers[0].Type != "cpu" {
							t.Errorf("Expected trigger type 'cpu', got '%s'", scaledObject.Spec.Triggers[0].Type)
						}

						// Verify CPU utilization value
						if value, ok := scaledObject.Spec.Triggers[0].Metadata["value"]; !ok || value != "50" {
							t.Errorf("Expected CPU target utilization '50', got '%s'", value)
						}
					}).
					Return(&kedav1alpha1.ScaledObject{}, nil).Times(1)
			},
			expectError: false,
		},
		{
			name: "Deployment with KEDA",
			config: &common.DeploymentConfig{
				Name:           "test-app-keda",
				Namespace:      "test-ns",
				Image:          "worker",
				Ports:          []int32{8080},
				KEDAEnabled:    true,
				KEDAScalerType: "prometheus",
				KEDAScalerMetadata: map[string]string{
					"serverAddress": "http://prometheus.monitoring",
					"metricName":    "http_requests",
					"threshold":     "100",
				},
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("test-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&corev1.Service{}, nil).Times(1)

				// Non-CPU KEDA is handled with a print statement, no API calls
			},
			expectError: false,
		},
		{
			name: "Namespace creation failure",
			config: &common.DeploymentConfig{
				Name:      "test-app",
				Namespace: "error-ns",
				Image:     "nginx",
			},
			setupMocks: func() {
				// Setup namespace client to fail on namespace creation
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(2)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "error-ns").Return(nil, fmt.Errorf("namespace not found")).Times(1)
				mockNamespaceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("permission denied")).Times(1)
			},
			expectError:   true,
			errorContains: "error creating namespace",
		},
		{
			name: "Deployment creation failure",
			config: &common.DeploymentConfig{
				Name:      "test-app",
				Namespace: "test-ns",
				Image:     "nginx",
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client to fail
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("deployment failed")).Times(1)
			},
			expectError:   true,
			errorContains: "error creating deployment",
		},
		{
			name: "Service creation failure",
			config: &common.DeploymentConfig{
				Name:      "test-app",
				Namespace: "test-ns",
				Image:     "nginx",
				Ports:     []int32{80},
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client to fail
				mockK8sClient.EXPECT().Services("test-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("service failed")).Times(1)
			},
			expectError:   true,
			errorContains: "error creating service",
		},
		{
			name: "HPA creation failure",
			config: &common.DeploymentConfig{
				Name:               "test-app",
				Namespace:          "test-ns",
				Image:              "nginx",
				Ports:              []int32{80},
				AutoscalingEnabled: true,
				MinReplicas:        2,
				MaxReplicas:        5,
			},
			setupMocks: func() {
				// Setup namespace client
				mockK8sClient.EXPECT().Namespaces().Return(mockNamespaceClient).Times(1)
				mockNamespaceClient.EXPECT().Get(gomock.Any(), "test-ns").Return(&corev1.Namespace{}, nil).Times(1)

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("test-ns").Return(mockDeploymentClient).Times(1)
				mockDeploymentClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&appsv1.Deployment{}, nil).Times(1)

				// Setup service client
				mockK8sClient.EXPECT().Services("test-ns").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&corev1.Service{}, nil).Times(1)

				// Setup HPA client to fail
				mockK8sClient.EXPECT().HorizontalPodAutoscalers("test-ns").Return(mockHpaClient).Times(1)
				mockHpaClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("hpa failed")).Times(1)
			},
			expectError:   true,
			errorContains: "error creating HPA",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tc.setupMocks()

			// Create manager and call CreateDeployment
			manager := NewManager(mockK8sClient)
			err := manager.CreateDeployment(context.Background(), tc.config)

			// Check error
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%s'", err.Error())
				}
			}
		})
	}
}
