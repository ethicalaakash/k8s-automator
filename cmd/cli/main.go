package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	"github.com/ethicalaakash/k8s-automator/internal/deployment"
	"github.com/ethicalaakash/k8s-automator/internal/health"
	"github.com/ethicalaakash/k8s-automator/internal/installer"
	"github.com/ethicalaakash/k8s-automator/internal/kubernetes"
)

const (
	defaultNamespace = "default"
)

func main() {
	// Define CLI flags
	kubeconfigPath := flag.String("kubeconfigPath", "", "Path to the kubeconfig file (defaults to ~/.kube/config or in-cluster config)")
	command := flag.String("command", "", "Command to execute (setup, deploy, status)")
	namespace := flag.String("namespace", defaultNamespace, "Kubernetes namespace")
	deploymentName := flag.String("name", "", "Deployment name")
	image := flag.String("image", "", "Container image (for deployment)")
	tag := flag.String("tag", "latest", "Container image tag (for deployment)")
	portsStr := flag.String("ports", "", "Comma-separated list of ports to expose (for deployment)")
	replicas := flag.Int("replicas", 1, "Number of replicas (for deployment)")
	cpuRequest := flag.String("cpu-request", "100m", "CPU request (for deployment)")
	cpuLimit := flag.String("cpu-limit", "200m", "CPU limit (for deployment)")
	memRequest := flag.String("mem-request", "128Mi", "Memory request (for deployment)")
	memLimit := flag.String("mem-limit", "256Mi", "Memory limit (for deployment)")
	enableAutoscaling := flag.Bool("enable-autoscaling", false, "Enable autoscaling (for deployment)")
	minReplicas := flag.Int("min-replicas", 1, "Minimum replicas for autoscaling (for deployment)")
	maxReplicas := flag.Int("max-replicas", 10, "Maximum replicas for autoscaling (for deployment)")
	cpuTarget := flag.Int("cpu-target", 80, "CPU target percentage for autoscaling (for deployment)")
	enableKEDA := flag.Bool("enable-keda", false, "Enable KEDA autoscaling (for deployment)")
	kedaScalerType := flag.String("keda-scaler", "cpu", "KEDA scaler type (for deployment)")
	kedaTriggerMetadata := flag.String("keda-metadata", "", "KEDA trigger metadata as key=value,key=value (for deployment)")
	outputFormat := flag.String("output", "text", "Output format (text, json)")

	flag.Parse()

	// Create context
	ctx := context.Background()

	// Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(*kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Process command
	switch *command {
	case "setup":
		// Initialize the installer
		inst := installer.NewInstaller(k8sClient)

		// Install necessary tools
		fmt.Println("Setting up Kubernetes cluster...")

		// Install Helm if not present
		if err := inst.InstallHelmIfNotPresent(); err != nil {
			log.Fatalf("Failed to install Helm: %v", err)
		}

		// Install KEDA
		if err := inst.InstallKEDA(ctx); err != nil {
			log.Fatalf("Failed to install KEDA: %v", err)
		}

		// Verify setup
		info, err := inst.VerifyClusterSetup(ctx)
		if err != nil {
			log.Fatalf("Failed to verify cluster setup: %v", err)
		}

		outputClusterInfo(info, *outputFormat)

	case "deploy":
		if *deploymentName == "" {
			log.Fatalf("Deployment name is required")
		}
		if *image == "" {
			log.Fatalf("Container image is required")
		}

		// Parse ports
		var ports []int32
		if *portsStr != "" {
			portStrings := strings.Split(*portsStr, ",")
			for _, p := range portStrings {
				port, err := strconv.Atoi(strings.TrimSpace(p))
				if err != nil {
					log.Fatalf("Invalid port number: %s", p)
				}
				ports = append(ports, int32(port))
			}
		}

		// Parse KEDA metadata
		kedaMetadata := make(map[string]string)
		if *kedaTriggerMetadata != "" {
			pairs := strings.Split(*kedaTriggerMetadata, ",")
			for _, pair := range pairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					kedaMetadata[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}

		// Create deployment config
		config := &common.DeploymentConfig{
			Name:                 *deploymentName,
			Namespace:            *namespace,
			Image:                *image,
			Tag:                  *tag,
			Ports:                ports,
			Replicas:             int32(*replicas),
			CPURequest:           *cpuRequest,
			CPULimit:             *cpuLimit,
			MemoryRequest:        *memRequest,
			MemoryLimit:          *memLimit,
			AutoscalingEnabled:   *enableAutoscaling,
			MinReplicas:          int32(*minReplicas),
			MaxReplicas:          int32(*maxReplicas),
			CPUTargetUtilization: int32(*cpuTarget),
			KEDAEnabled:          *enableKEDA,
			KEDAScalerType:       *kedaScalerType,
			KEDAScalerMetadata:   kedaMetadata,
			KEDAPollingInterval:  30,
			KEDACooldownPeriod:   300,
		}

		// Initialize deployment manager
		deployMgr := deployment.NewManager(k8sClient)

		// Create deployment
		fmt.Printf("Creating deployment %s in namespace %s...\n", *deploymentName, *namespace)
		if err := deployMgr.CreateDeployment(ctx, config); err != nil {
			log.Fatalf("Failed to create deployment: %v", err)
		}

		fmt.Printf("Deployment %s/%s created successfully\n", *namespace, *deploymentName)
		fmt.Printf("\nTo check the status of your deployment, run:\n")
		fmt.Printf("  ./bin/k8s-automator -command=status -name=%s -namespace=%s\n", *deploymentName, *namespace)

	case "status":
		if *deploymentName == "" {
			log.Fatalf("Deployment name is required")
		}

		// Initialize health monitor
		healthMon := health.NewMonitor(k8sClient)

		// Get deployment status
		status, err := healthMon.GetDeploymentStatus(ctx, *namespace, *deploymentName)
		if err != nil {
			log.Fatalf("Failed to get deployment status: %v", err)
		}

		outputDeploymentStatus(status, *outputFormat)

	default:
		fmt.Println("K8s Automator - A tool for automating Kubernetes operations")
		fmt.Println("\nUsage:")
		fmt.Println("  Setup the cluster:")
		fmt.Println("    k8s-automator -command=setup")
		fmt.Println("\n  Create a deployment:")
		fmt.Println("    k8s-automator -command=deploy -name=myapp -namespace=default -image=nginx -ports=80 -enable-autoscaling -min-replicas=2 -max-replicas=5")
		fmt.Println("\n  Get deployment status:")
		fmt.Println("    k8s-automator -command=status -name=myapp -namespace=default")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
}

// outputClusterInfo outputs the cluster information in the specified format
func outputClusterInfo(info *common.ClusterInfo, format string) {
	if format == "json" {
		jsonBytes, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonBytes))
		return
	}

	fmt.Printf("Kubernetes Cluster Info:\n")
	fmt.Printf("  Server Version: %s\n", info.ServerVersion)
	fmt.Printf("  Nodes: %d\n", info.Nodes)
	fmt.Printf("  Namespaces: %s\n", strings.Join(info.Namespaces, ", "))
	fmt.Printf("  Status: %s\n", info.Status)
	fmt.Printf("  Message: %s\n", info.Message)

	fmt.Printf("\nInstalled Tools:\n")
	for tool, version := range info.InstalledTools {
		fmt.Printf("  %s: %s\n", tool, version)
	}
}

// outputDeploymentStatus outputs the deployment status in the specified format
func outputDeploymentStatus(status *common.DeploymentStatus, format string) {
	if format == "json" {
		jsonBytes, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonBytes))
		return
	}

	// Default to text output
	fmt.Printf("Deployment Status:\n")
	fmt.Printf("  Name: %s\n", status.Name)
	fmt.Printf("  Namespace: %s\n", status.Namespace)
	fmt.Printf("  Status: %s\n", status.Status)
	fmt.Printf("  Message: %s\n", status.Message)
	fmt.Printf("  Replicas: %d/%d ready\n", status.ReadyReplicas, status.TotalReplicas)
	fmt.Printf("  CPU Usage: %s\n", status.CPUUsage)
	fmt.Printf("  Memory Usage: %s\n", status.MemoryUsage)

	if status.RestartCount > 0 {
		fmt.Printf("  Restart Count: %d\n", status.RestartCount)
		if status.LastRestartTime != "" {
			fmt.Printf("  Last Restart: %s\n", status.LastRestartTime)
		}
	}

	if len(status.ServiceEndpoints) > 0 {
		fmt.Printf("  Service Endpoints:\n")
		for _, endpoint := range status.ServiceEndpoints {
			fmt.Printf("    - %s\n", endpoint)
		}
	}

	if len(status.Events) > 0 {
		fmt.Printf("  Events:\n")
		for _, event := range status.Events {
			fmt.Printf("    - %s\n", event)
		}
	}
}
