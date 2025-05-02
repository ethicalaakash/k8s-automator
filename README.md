# Kubernetes Cluster Automator

A Go-based CLI tool that automates operations on Kubernetes clusters. This tool can connect to Kubernetes clusters, install necessary tooling, create deployments with event-driven scaling using KEDA, and provide health status for deployments.

## Features

- **Cluster Setup**: Connect to Kubernetes clusters and install necessary tools like Helm and KEDA
- **Deployment Management**: Create deployments with specified resource requirements and port configurations
- **Autoscaling**: Configure standard HPA (Horizontal Pod Autoscaler) and KEDA-based event-driven autoscaling
- **Health Monitoring**: Retrieve detailed health status for deployments
- **Testability**: Uses interfaces and mocks to enable unit testing without a real Kubernetes cluster

## Prerequisites

- Go 1.17+
- Access to a Kubernetes cluster with `kubectl` configured
- Admin privileges on the cluster for installing tools

## Installation

Clone the repo:
```
git clone git@github.com:ethicalaakash/k8s-automator.git
```

Build the CLI tool:
```
make build
```

## Usage

### Setting Up a Cluster

```bash
# Setup a cluster with Helm and KEDA
./bin/k8s-automator -command=setup
```

### Creating a Deployment

```bash
# Create a basic deployment
./bin/k8s-automator -command=deploy -name=myapp -namespace=default -image=nginx -ports=80

# Create a deployment with autoscaling
./bin/k8s-automator -command=deploy -name=myapp2 -namespace=default -image=nginx -ports=80 \
  -enable-autoscaling -min-replicas=2 -max-replicas=5 -cpu-target=70

# Create a deployment with KEDA autoscaling
./bin/k8s-automator -command=deploy -name=myapp3 -namespace=default -image=nginx -ports=80 \
  -enable-keda -keda-scaler=cpu -cpu-target=50 -min-replicas=2 -max-replicas=5


### Checking Deployment Status

```bash
# Get the status of a deployment
./bin/k8s-automator -command=status -name=myapp -namespace=default

```

## Available Commands

- `setup`: Setup the Kubernetes cluster with required tools
- `deploy`: Create a new deployment
- `status`: Get the status of a deployment

## Command-Line Arguments

### Global Arguments
- `-kubeconfig`: Path to the kubeconfig file (defaults to `~/.kube/config` or in-cluster config)
- `-command`: Command to execute (`setup`, `deploy`, `status`)
- `-output`: Output format (`text`, `json`)

### Deployment Arguments
- `-name`: Deployment name
- `-namespace`: Kubernetes namespace (defaults to "default")
- `-image`: Container image
- `-tag`: Container image tag (defaults to "latest")
- `-ports`: Comma-separated list of ports to expose
- `-replicas`: Number of replicas
- `-cpu-request`: CPU request (defaults to "100m")
- `-cpu-limit`: CPU limit (defaults to "200m")
- `-mem-request`: Memory request (defaults to "128Mi")
- `-mem-limit`: Memory limit (defaults to "256Mi")

### Autoscaling Arguments
- `-enable-autoscaling`: Enable autoscaling
- `-min-replicas`: Minimum replicas for autoscaling
- `-max-replicas`: Maximum replicas for autoscaling
- `-cpu-target`: CPU target percentage for autoscaling

### KEDA Arguments
- `-enable-keda`: Enable KEDA autoscaling
- `-keda-scaler`: KEDA scaler type (supported: `cpu` for CPU-based scaling, others require custom metadata)
- `-keda-metadata`: KEDA trigger metadata as key=value,key=value (required for non-CPU scalers)
- `-cpu-target`: CPU target percentage (used for both standard HPA and KEDA CPU scaler)

## Architecture

The tool is organized into several packages:

- `cmd/cli`: Main CLI application
- `internal/common`: Common types and utilities
- `internal/kubernetes`: Kubernetes client operations
- `internal/installer`: Installation of Helm and KEDA
- `internal/deployment`: Deployment management
- `internal/health`: Health status monitoring

### Design Patterns

This project uses several design patterns to enhance maintainability and testability:

- **Interface-based design**: Components interact through interfaces rather than concrete implementations
- **Dependency Injection**: Dependencies are injected rather than created directly
- **Resource-Specific Interfaces**: The Kubernetes client provides type-safe interfaces for each resource type (Pods, Deployments, Services, etc.)

### Testing

The project is designed to be testable without requiring a real Kubernetes cluster. 

The mocking approach uses GoMock to create mock implementations of all interfaces.

To run the tests:

```bash
make test
```