package health_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethicalaakash/k8s-automator/internal/health"
	kmock "github.com/ethicalaakash/k8s-automator/mocks/kubernetes"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNewMonitor tests the creation of a health monitor
func TestNewMonitor(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock Kubernetes client
	mockClient := kmock.NewMockKubernetesClient(ctrl)

	// Create monitor
	monitor := health.NewMonitor(mockClient)

	// Verify monitor was created
	if monitor == nil {
		t.Error("Expected a non-nil HealthMonitor")
	}
}

// TestGetDeploymentStatus tests the GetDeploymentStatus method with proper mocking
func TestGetDeploymentStatus(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create test cases
	testCases := []struct {
		name           string
		namespace      string
		deploymentName string
		setupMocks     func(mockK8sClient *kmock.MockKubernetesClient,
			mockDeployClient *kmock.MockDeploymentInterface,
			mockPodClient *kmock.MockPodInterface,
			mockEventClient *kmock.MockEventInterface,
			mockServiceClient *kmock.MockServiceInterface)
		expectedStatus        string
		expectedAvailable     bool
		expectedReplicas      int32
		expectedReadyReplicas int32
		expectedError         bool
	}{
		{
			name:           "Healthy deployment",
			namespace:      "default",
			deploymentName: "test-app",
			setupMocks: func(mockK8sClient *kmock.MockKubernetesClient,
				mockDeployClient *kmock.MockDeploymentInterface,
				mockPodClient *kmock.MockPodInterface,
				mockEventClient *kmock.MockEventInterface,
				mockServiceClient *kmock.MockServiceInterface) {

				// Setup deployment client for healthy deployment
				mockK8sClient.EXPECT().Deployments("default").Return(mockDeployClient).Times(1)
				mockDeployClient.EXPECT().Get(gomock.Any(), "test-app").Return(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "test-app",
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:      3,
						ReadyReplicas: 3,
					},
				}, nil).Times(1)

				// Setup pod client
				mockK8sClient.EXPECT().Pods("default").Return(mockPodClient).Times(1)
				mockPodClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-1",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        true,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-2",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        true,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-3",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        true,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
					},
				}, nil).Times(1)

				// Setup events client
				mockK8sClient.EXPECT().Events("default").Return(mockEventClient).Times(3) // once for each pod
				mockEventClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.EventList{
					Items: []corev1.Event{},
				}, nil).Times(3)

				// Setup service client
				mockK8sClient.EXPECT().Services("default").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.ServiceList{
					Items: []corev1.Service{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app",
							},
							Spec: corev1.ServiceSpec{
								Type: corev1.ServiceTypeClusterIP,
								Ports: []corev1.ServicePort{
									{
										Port: 80,
									},
								},
							},
						},
					},
				}, nil).Times(1)
			},
			expectedStatus:        "Healthy",
			expectedAvailable:     true,
			expectedReplicas:      3,
			expectedReadyReplicas: 3,
			expectedError:         false,
		},
		{
			name:           "Degraded deployment",
			namespace:      "default",
			deploymentName: "test-app-degraded",
			setupMocks: func(mockK8sClient *kmock.MockKubernetesClient,
				mockDeployClient *kmock.MockDeploymentInterface,
				mockPodClient *kmock.MockPodInterface,
				mockEventClient *kmock.MockEventInterface,
				mockServiceClient *kmock.MockServiceInterface) {

				// Setup deployment client for degraded deployment
				mockK8sClient.EXPECT().Deployments("default").Return(mockDeployClient).Times(1)
				mockDeployClient.EXPECT().Get(gomock.Any(), "test-app-degraded").Return(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-degraded",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "test-app-degraded",
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:      3,
						ReadyReplicas: 1,
					},
				}, nil).Times(1)

				// Setup pod client
				mockK8sClient.EXPECT().Pods("default").Return(mockPodClient).Times(1)
				mockPodClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-1",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        true,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionTrue,
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-2",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodPending,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        false,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionFalse,
										Reason: "ContainersNotReady",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-3",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodPending,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        false,
										RestartCount: 0,
									},
								},
								Conditions: []corev1.PodCondition{
									{
										Type:   corev1.PodReady,
										Status: corev1.ConditionFalse,
										Reason: "ContainersNotReady",
									},
								},
							},
						},
					},
				}, nil).Times(1)

				// Setup events client
				mockK8sClient.EXPECT().Events("default").Return(mockEventClient).Times(3) // once for each pod
				mockEventClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.EventList{
					Items: []corev1.Event{
						{
							Type:    "Warning",
							Reason:  "FailedScheduling",
							Message: "0/3 nodes are available: insufficient memory",
						},
					},
				}, nil).Times(3)

				// Setup service client
				mockK8sClient.EXPECT().Services("default").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.ServiceList{
					Items: []corev1.Service{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-degraded",
							},
							Spec: corev1.ServiceSpec{
								Type: corev1.ServiceTypeClusterIP,
							},
						},
					},
				}, nil).Times(1)
			},
			expectedStatus:        "Degraded",
			expectedAvailable:     true,
			expectedReplicas:      3,
			expectedReadyReplicas: 1,
			expectedError:         false,
		},
		{
			name:           "Unhealthy deployment",
			namespace:      "default",
			deploymentName: "test-app-unhealthy",
			setupMocks: func(mockK8sClient *kmock.MockKubernetesClient,
				mockDeployClient *kmock.MockDeploymentInterface,
				mockPodClient *kmock.MockPodInterface,
				mockEventClient *kmock.MockEventInterface,
				mockServiceClient *kmock.MockServiceInterface) {

				// Setup deployment client for unhealthy deployment
				mockK8sClient.EXPECT().Deployments("default").Return(mockDeployClient).Times(1)
				mockDeployClient.EXPECT().Get(gomock.Any(), "test-app-unhealthy").Return(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app-unhealthy",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "test-app-unhealthy",
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:      3,
						ReadyReplicas: 0,
					},
				}, nil).Times(1)

				// Setup pod client
				mockK8sClient.EXPECT().Pods("default").Return(mockPodClient).Times(1)
				mockPodClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.PodList{
					Items: []corev1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-1",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodFailed,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        false,
										RestartCount: 5,
										LastTerminationState: corev1.ContainerState{
											Terminated: &corev1.ContainerStateTerminated{
												FinishedAt: metav1.Time{Time: time.Now().Add(-10 * time.Minute)},
												Reason:     "Error",
												ExitCode:   1,
											},
										},
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-pod-2",
							},
							Status: corev1.PodStatus{
								Phase: corev1.PodFailed,
								ContainerStatuses: []corev1.ContainerStatus{
									{
										Ready:        false,
										RestartCount: 5,
									},
								},
							},
						},
					},
				}, nil).Times(1)

				// Setup events client
				mockK8sClient.EXPECT().Events("default").Return(mockEventClient).Times(2) // once for each pod
				mockEventClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.EventList{
					Items: []corev1.Event{
						{
							Type:    "Warning",
							Reason:  "CrashLoopBackOff",
							Message: "Back-off restarting failed container",
						},
					},
				}, nil).Times(2)

				// Setup service client
				mockK8sClient.EXPECT().Services("default").Return(mockServiceClient).Times(1)
				mockServiceClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.ServiceList{
					Items: []corev1.Service{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-app-unhealthy",
							},
							Spec: corev1.ServiceSpec{
								Type: corev1.ServiceTypeClusterIP,
							},
						},
					},
				}, nil).Times(1)
			},
			expectedStatus:        "Unhealthy",
			expectedAvailable:     false,
			expectedReplicas:      3,
			expectedReadyReplicas: 0,
			expectedError:         false,
		},
		{
			name:           "Deployment not found",
			namespace:      "default",
			deploymentName: "nonexistent-app",
			setupMocks: func(mockK8sClient *kmock.MockKubernetesClient,
				mockDeployClient *kmock.MockDeploymentInterface,
				mockPodClient *kmock.MockPodInterface,
				mockEventClient *kmock.MockEventInterface,
				mockServiceClient *kmock.MockServiceInterface) {

				// Setup deployment client to return not found error
				mockK8sClient.EXPECT().Deployments("default").Return(mockDeployClient).Times(1)
				mockDeployClient.EXPECT().Get(gomock.Any(), "nonexistent-app").Return(nil,
					fmt.Errorf("deployment \"nonexistent-app\" not found")).Times(1)
			},
			expectedStatus:        "Unknown",
			expectedAvailable:     false,
			expectedReplicas:      0,
			expectedReadyReplicas: 0,
			expectedError:         true,
		},
		{
			name:           "No pods for deployment",
			namespace:      "default",
			deploymentName: "no-pods-app",
			setupMocks: func(mockK8sClient *kmock.MockKubernetesClient,
				mockDeployClient *kmock.MockDeploymentInterface,
				mockPodClient *kmock.MockPodInterface,
				mockEventClient *kmock.MockEventInterface,
				mockServiceClient *kmock.MockServiceInterface) {

				// Setup deployment client
				mockK8sClient.EXPECT().Deployments("default").Return(mockDeployClient).Times(1)
				mockDeployClient.EXPECT().Get(gomock.Any(), "no-pods-app").Return(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "no-pods-app",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "no-pods-app",
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						Replicas:      0,
						ReadyReplicas: 0,
					},
				}, nil).Times(1)

				// Setup pod client to return empty pod list
				mockK8sClient.EXPECT().Pods("default").Return(mockPodClient).Times(1)
				mockPodClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(&corev1.PodList{
					Items: []corev1.Pod{},
				}, nil).Times(1)

				// No events or services need to be checked since there are no pods
			},
			expectedStatus:        "Not Ready",
			expectedAvailable:     false,
			expectedReplicas:      0,
			expectedReadyReplicas: 0,
			expectedError:         false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock clients
			mockK8sClient := kmock.NewMockKubernetesClient(ctrl)
			mockDeployClient := kmock.NewMockDeploymentInterface(ctrl)
			mockPodClient := kmock.NewMockPodInterface(ctrl)
			mockEventClient := kmock.NewMockEventInterface(ctrl)
			mockServiceClient := kmock.NewMockServiceInterface(ctrl)

			// Setup test case mocks
			tc.setupMocks(mockK8sClient, mockDeployClient, mockPodClient, mockEventClient, mockServiceClient)

			// Create monitor and run GetDeploymentStatus
			monitor := health.NewMonitor(mockK8sClient)
			status, err := monitor.GetDeploymentStatus(context.Background(), tc.namespace, tc.deploymentName)

			// Check for expected error
			if tc.expectedError && err == nil {
				t.Error("Expected an error, but got nil")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// If we expect an error, we can stop here
			if tc.expectedError {
				return
			}

			// Check deployment status
			if status.Status != tc.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tc.expectedStatus, status.Status)
			}

			// Check availability
			if status.Available != tc.expectedAvailable {
				t.Errorf("Expected available=%v, got %v", tc.expectedAvailable, status.Available)
			}

			// Check replica counts
			if status.TotalReplicas != tc.expectedReplicas {
				t.Errorf("Expected %d total replicas, got %d", tc.expectedReplicas, status.TotalReplicas)
			}
			if status.ReadyReplicas != tc.expectedReadyReplicas {
				t.Errorf("Expected %d ready replicas, got %d", tc.expectedReadyReplicas, status.ReadyReplicas)
			}
		})
	}
}
