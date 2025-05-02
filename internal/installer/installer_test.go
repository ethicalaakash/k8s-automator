package installer_test

import (
	"fmt"
	"testing"

	"github.com/ethicalaakash/k8s-automator/internal/installer"
	kmock "github.com/ethicalaakash/k8s-automator/mocks/kubernetes"
	osmock "github.com/ethicalaakash/k8s-automator/mocks/os"
	"github.com/golang/mock/gomock"
)

// TestInstallHelmIfNotPresent tests the InstallHelmIfNotPresent method
func TestInstallHelmIfNotPresent(t *testing.T) {
	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Kubernetes client
	mockK8sClient := kmock.NewMockKubernetesClient(ctrl)

	// Test case 1: Helm is already installed
	t.Run("Helm already installed", func(t *testing.T) {
		// Create a mock commander
		mockCmd := osmock.NewMockCommander()
		mockCmd.AddResponse("helm version --short", []byte("v3.11.1"), nil)

		// Create a new installer with the mock commander
		inst := installer.NewInstallerWithCommander(mockK8sClient, mockCmd)

		// Call the method
		err := inst.InstallHelmIfNotPresent()

		// Check results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the correct command was executed
		if len(mockCmd.ExecutedCommands) != 1 {
			t.Errorf("Expected 1 command execution, got %d", len(mockCmd.ExecutedCommands))
		}
		if mockCmd.ExecutedCommands[0] != "helm version --short" {
			t.Errorf("Expected 'helm version --short', got '%s'", mockCmd.ExecutedCommands[0])
		}
	})

	// Test case 2: Helm is not installed
	t.Run("Helm not installed", func(t *testing.T) {
		// Create a mock commander
		mockCmd := osmock.NewMockCommander()

		// Track the number of helm version calls
		helmVersionCalls := 0

		// Add a handler for "helm version" command
		mockCmd.AddHandler("helm version", func(name string, args ...string) ([]byte, error) {
			helmVersionCalls++

			if helmVersionCalls == 1 {
				// First call - simulate Helm not installed
				return nil, fmt.Errorf("helm not found")
			}
			// Second call - simulate successful installation
			return []byte("v3.11.1"), nil
		})

		// Add a response for the installation script
		mockCmd.AddResponse("sh -c curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash",
			[]byte("Helm installed successfully"), nil)

		// Create a new installer with the mock commander
		inst := installer.NewInstallerWithCommander(mockK8sClient, mockCmd)

		// Call the method
		err := inst.InstallHelmIfNotPresent()

		// Check results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the number of calls to helm version
		if helmVersionCalls != 2 {
			t.Errorf("Expected 2 calls to 'helm version', got %d", helmVersionCalls)
		}

		// Verify the correct commands were executed
		if len(mockCmd.ExecutedCommands) != 3 {
			t.Errorf("Expected 3 command executions, got %d", len(mockCmd.ExecutedCommands))
		}
	})
}

// TestVerifyClusterSetup tests the VerifyClusterSetup method
func TestVerifyClusterSetup(t *testing.T) {
	// This would now use our new mock commander approach
	t.Skip("Skipping test as it requires mocking Kubernetes clientset")
}

// TestInstallKEDA tests the InstallKEDA method
func TestInstallKEDA(t *testing.T) {
	// This would now use our new mock commander approach
	t.Skip("Skipping test as it requires mocking Kubernetes clientset and Helm")
}
