package installer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethicalaakash/k8s-automator/internal/common"
	"github.com/ethicalaakash/k8s-automator/internal/kubernetes"
	osCmd "github.com/ethicalaakash/k8s-automator/internal/os"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ToolInstaller defines the interface for installing and verifying tools
type ToolInstaller interface {
	InstallHelmIfNotPresent() error
	InstallKEDA(ctx context.Context) error
	VerifyClusterSetup(ctx context.Context) (*common.ClusterInfo, error)
}

// Installer handles the installation of tools like Helm and KEDA
type Installer struct {
	k8sClient kubernetes.KubernetesClient
	commander osCmd.Commander
}

// NewInstaller creates a new installer instance
func NewInstaller(k8sClient kubernetes.KubernetesClient) ToolInstaller {
	return &Installer{
		k8sClient: k8sClient,
		commander: osCmd.NewCommander(),
	}
}

// NewInstallerWithCommander creates a new installer with a custom command executor
func NewInstallerWithCommander(k8sClient kubernetes.KubernetesClient, commander osCmd.Commander) ToolInstaller {
	return &Installer{
		k8sClient: k8sClient,
		commander: commander,
	}
}

// InstallHelmIfNotPresent ensures Helm is installed on the system
func (i *Installer) InstallHelmIfNotPresent() error {
	// Check if Helm is already installed
	version, err := i.getHelmVersion()
	if err == nil && version != "" {
		fmt.Printf("Helm is already installed: %s\n", version)
		return nil
	}

	// Helm is not installed, try to install it
	fmt.Println("Installing Helm...")

	// Using the official installation script from https://helm.sh/docs/intro/install/
	// This will work on macOS and Linux
	output, err := i.commander.Execute("sh", "-c", "curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash")
	if err != nil {
		return fmt.Errorf("error installing Helm: %v\nOutput: %s", err, output)
	}

	// Verify installation
	version, err = i.getHelmVersion()
	if err != nil {
		return fmt.Errorf("error verifying Helm installation: %v", err)
	}

	fmt.Printf("Helm successfully installed: %s\n", version)
	return nil
}

// getHelmVersion returns the installed Helm version or an error if not installed
func (i *Installer) getHelmVersion() (string, error) {
	output, err := i.commander.Execute("helm", "version", "--short")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// InstallKEDA installs KEDA using Helm
func (i *Installer) InstallKEDA(ctx context.Context) error {
	if err := i.InstallHelmIfNotPresent(); err != nil {
		return fmt.Errorf("error ensuring Helm is installed: %v", err)
	}

	// Check if KEDA namespace exists
	_, err := i.k8sClient.Namespaces().Get(ctx, "keda")
	if err == nil {
		// Check if KEDA operator is running
		deployments, err := i.k8sClient.Deployments("keda").List(ctx, metav1.ListOptions{})
		if err == nil && len(deployments.Items) > 0 {
			for _, deployment := range deployments.Items {
				if strings.Contains(deployment.Name, "keda") && deployment.Status.ReadyReplicas > 0 {
					fmt.Println("KEDA is already installed and running")
					return nil
				}
			}
		}
	}

	// KEDA is not installed or not running properly, install it
	fmt.Println("Installing KEDA...")

	// Add the KEDA Helm repository
	addRepoOutput, err := i.commander.Execute("helm", "repo", "add", "kedacore", "https://kedacore.github.io/charts")
	if err != nil {
		return fmt.Errorf("error adding KEDA Helm repo: %v\nOutput: %s", err, addRepoOutput)
	}

	// Update Helm repositories
	updateRepoOutput, err := i.commander.Execute("helm", "repo", "update")
	if err != nil {
		return fmt.Errorf("error updating Helm repos: %v\nOutput: %s", err, updateRepoOutput)
	}

	// Create the keda namespace if it doesn't exist
	_, err = i.k8sClient.Namespaces().Get(ctx, "keda")
	if err != nil {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "keda",
			},
		}
		_, err = i.k8sClient.Namespaces().Create(ctx, namespace)
		if err != nil {
			return fmt.Errorf("error creating keda namespace: %v", err)
		}
	}

	// Install KEDA using Helm
	installOutput, err := i.commander.Execute("helm", "install", "keda", "kedacore/keda", "--namespace", "keda")
	if err != nil {
		return fmt.Errorf("error installing KEDA: %v\nOutput: %s", err, installOutput)
	}

	// Wait for KEDA operator to be ready
	fmt.Println("Waiting for KEDA operator to be ready...")
	err = i.waitForKEDAReady(ctx)
	if err != nil {
		return fmt.Errorf("error waiting for KEDA to be ready: %v", err)
	}

	fmt.Println("KEDA successfully installed")
	return nil
}

// waitForKEDAReady waits for the KEDA operator to be ready
func (i *Installer) waitForKEDAReady(ctx context.Context) error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		deployments, err := i.k8sClient.Deployments("keda").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, nil // Keep retrying
		}

		// Check KEDA operator and metrics server deployments
		for _, deployment := range deployments.Items {
			if strings.Contains(deployment.Name, "keda") && deployment.Status.ReadyReplicas > 0 {
				return true, nil
			}
		}

		return false, nil
	})
}

// VerifyClusterSetup verifies that the cluster has all necessary tools installed
func (i *Installer) VerifyClusterSetup(ctx context.Context) (*common.ClusterInfo, error) {
	clusterInfo, err := i.k8sClient.GetClusterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster info: %v", err)
	}

	// Check Helm
	helmVersion, err := i.getHelmVersion()
	if err == nil {
		clusterInfo.InstalledTools["helm"] = helmVersion
	} else {
		clusterInfo.InstalledTools["helm"] = "Not installed"
	}

	// Check KEDA
	kedaStatus := "Not installed"

	deployments, err := i.k8sClient.Deployments("keda").List(ctx, metav1.ListOptions{})
	if err == nil && len(deployments.Items) > 0 {
		for _, deployment := range deployments.Items {
			if strings.Contains(deployment.Name, "keda") && deployment.Status.ReadyReplicas > 0 {
				kedaStatus = fmt.Sprintf("Running (Deployment: %s, Replicas: %d/%d)",
					deployment.Name,
					deployment.Status.ReadyReplicas,
					deployment.Status.Replicas)
				break
			}
		}
	}
	clusterInfo.InstalledTools["keda"] = kedaStatus

	return clusterInfo, nil
}
