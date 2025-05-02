package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedautoscalingv2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesClient defines the interface for Kubernetes operations
type KubernetesClient interface {
	// Cluster operations
	GetClusterInfo(ctx context.Context) (*common.ClusterInfo, error)

	// Core API groups
	Namespaces() NamespaceInterface
	Pods(namespace string) PodInterface
	Services(namespace string) ServiceInterface
	Events(namespace string) EventInterface

	// Apps API group
	Deployments(namespace string) DeploymentInterface

	// Autoscaling API group
	HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerInterface

	// KEDA ScaledObjects (Custom Resources)
	ScaledObjects(namespace string) ScaledObjectInterface

	// Legacy method - for backward compatibility, consider removing later
	GetClientset() *kubernetes.Clientset
	GetConfig() *rest.Config
}

// Client represents a Kubernetes client wrapper
type Client struct {
	clientset     *kubernetes.Clientset
	config        *rest.Config
	dynamicClient dynamic.Interface
	runtimeScheme *runtime.Scheme
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (KubernetesClient, error) {
	var err error
	var config *rest.Config

	// If kubeconfigPath is provided, use it
	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("error building kubeconfig from path %s: %v", kubeconfigPath, err)
		}
	} else {
		// Try to use in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// If not in cluster, try to use default kubeconfig
			home := homedir.HomeDir()
			if home == "" {
				return nil, fmt.Errorf("unable to locate kubeconfig: home directory not found and not running in-cluster")
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			if err != nil {
				return nil, fmt.Errorf("error building kubeconfig: %v", err)
			}
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes clientset: %v", err)
	}

	// Create dynamic client for custom resources
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating dynamic client: %v", err)
	}

	// Create runtime scheme and register KEDA types
	runtimeScheme := runtime.NewScheme()
	scheme.AddToScheme(runtimeScheme)
	kedav1alpha1.AddToScheme(runtimeScheme)

	return &Client{
		clientset:     clientset,
		config:        config,
		dynamicClient: dynamicClient,
		runtimeScheme: runtimeScheme,
	}, nil
}

// GetClientset returns the underlying Kubernetes clientset
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

// GetConfig returns the Kubernetes rest config
func (c *Client) GetConfig() *rest.Config {
	return c.config
}

// GetClusterInfo retrieves information about the Kubernetes cluster
func (c *Client) GetClusterInfo(ctx context.Context) (*common.ClusterInfo, error) {
	info := &common.ClusterInfo{
		InstalledTools: make(map[string]string),
		Nodes:          0,
		Namespaces:     []string{},
		Status:         "Connected",
		Message:        "Successfully connected to Kubernetes cluster",
	}

	// Get version info
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error getting server version: %v", err)
	}
	info.ServerVersion = version.String()

	// Get node count
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %v", err)
	}
	info.Nodes = len(nodes.Items)

	// Get namespaces
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing namespaces: %v", err)
	}
	for _, ns := range namespaces.Items {
		info.Namespaces = append(info.Namespaces, ns.Name)
	}

	return info, nil
}

// Namespaces returns a namespace client
func (c *Client) Namespaces() NamespaceInterface {
	return &namespaceClient{client: c.clientset.CoreV1().Namespaces()}
}

// Pods returns a pod client for the specified namespace
func (c *Client) Pods(namespace string) PodInterface {
	return &podClient{client: c.clientset.CoreV1().Pods(namespace)}
}

// Services returns a service client for the specified namespace
func (c *Client) Services(namespace string) ServiceInterface {
	return &serviceClient{client: c.clientset.CoreV1().Services(namespace)}
}

// Events returns an event client for the specified namespace
func (c *Client) Events(namespace string) EventInterface {
	return &eventClient{client: c.clientset.CoreV1().Events(namespace)}
}

// Deployments returns a deployment client for the specified namespace
func (c *Client) Deployments(namespace string) DeploymentInterface {
	return &deploymentClient{client: c.clientset.AppsV1().Deployments(namespace)}
}

// HorizontalPodAutoscalers returns an HPA client for the specified namespace
func (c *Client) HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerInterface {
	return &hpaClient{client: c.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace)}
}

// ScaledObjects returns a KEDA ScaledObject client for the specified namespace
func (c *Client) ScaledObjects(namespace string) ScaledObjectInterface {
	return &scaledObjectClient{
		client: c.dynamicClient.Resource(
			schema.GroupVersionResource{
				Group:    "keda.sh",
				Version:  "v1alpha1",
				Resource: "scaledobjects",
			}).Namespace(namespace),
	}
}

// Define interfaces for Kubernetes resource operations

// NamespaceInterface defines operations on Kubernetes namespaces
type NamespaceInterface interface {
	Create(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error)
	Get(ctx context.Context, name string) (*corev1.Namespace, error)
	List(ctx context.Context) (*corev1.NamespaceList, error)
}

// PodInterface defines operations on Kubernetes pods
type PodInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
	Get(ctx context.Context, name string) (*corev1.Pod, error)
}

// ServiceInterface defines operations on Kubernetes services
type ServiceInterface interface {
	Create(ctx context.Context, service *corev1.Service) (*corev1.Service, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error)
}

// EventInterface defines operations on Kubernetes events
type EventInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error)
}

// DeploymentInterface defines operations on Kubernetes deployments
type DeploymentInterface interface {
	Create(ctx context.Context, deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	Get(ctx context.Context, name string) (*appsv1.Deployment, error)
	List(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error)
}

// HorizontalPodAutoscalerInterface defines operations on Kubernetes HPAs
type HorizontalPodAutoscalerInterface interface {
	Create(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error)
}

// ScaledObjectInterface defines operations on KEDA ScaledObjects
type ScaledObjectInterface interface {
	Create(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (*kedav1alpha1.ScaledObject, error)
}

// Implementation of Kubernetes resource interfaces

type namespaceClient struct {
	client typedcorev1.NamespaceInterface
}

func (c *namespaceClient) Create(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	return c.client.Create(ctx, namespace, metav1.CreateOptions{})
}

func (c *namespaceClient) Get(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.client.Get(ctx, name, metav1.GetOptions{})
}

func (c *namespaceClient) List(ctx context.Context) (*corev1.NamespaceList, error) {
	return c.client.List(ctx, metav1.ListOptions{})
}

type podClient struct {
	client typedcorev1.PodInterface
}

func (c *podClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	return c.client.List(ctx, opts)
}

func (c *podClient) Get(ctx context.Context, name string) (*corev1.Pod, error) {
	return c.client.Get(ctx, name, metav1.GetOptions{})
}

type serviceClient struct {
	client typedcorev1.ServiceInterface
}

func (c *serviceClient) Create(ctx context.Context, service *corev1.Service) (*corev1.Service, error) {
	return c.client.Create(ctx, service, metav1.CreateOptions{})
}

func (c *serviceClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error) {
	return c.client.List(ctx, opts)
}

type eventClient struct {
	client typedcorev1.EventInterface
}

func (c *eventClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error) {
	return c.client.List(ctx, opts)
}

type deploymentClient struct {
	client typedappsv1.DeploymentInterface
}

func (c *deploymentClient) Create(ctx context.Context, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.client.Create(ctx, deployment, metav1.CreateOptions{})
}

func (c *deploymentClient) Get(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return c.client.Get(ctx, name, metav1.GetOptions{})
}

func (c *deploymentClient) List(ctx context.Context, opts metav1.ListOptions) (*appsv1.DeploymentList, error) {
	return c.client.List(ctx, opts)
}

type hpaClient struct {
	client typedautoscalingv2.HorizontalPodAutoscalerInterface
}

func (c *hpaClient) Create(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	return c.client.Create(ctx, hpa, metav1.CreateOptions{})
}

// scaledObjectClient implements the ScaledObjectInterface
type scaledObjectClient struct {
	client dynamic.ResourceInterface
}

func (c *scaledObjectClient) Create(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (*kedav1alpha1.ScaledObject, error) {
	// Convert the typed object to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(scaledObject)
	if err != nil {
		return nil, fmt.Errorf("error converting ScaledObject to unstructured: %v", err)
	}

	unstructured := &unstructured.Unstructured{Object: unstructuredObj}
	unstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "keda.sh",
		Version: "v1alpha1",
		Kind:    "ScaledObject",
	})

	// Create the resource
	result, err := c.client.Create(ctx, unstructured, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Convert the result back to a typed object
	resultScaledObject := &kedav1alpha1.ScaledObject{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, resultScaledObject)
	if err != nil {
		return nil, fmt.Errorf("error converting unstructured to ScaledObject: %v", err)
	}

	return resultScaledObject, nil
}
