# Build parameters
BINARY_NAME := k8s-automator
GO := go
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOTEST := $(GO) test
GOMOD := $(GO) mod
GOGET := $(GO) get
MOCKGEN := mockgen
BIN_DIR := bin
SRC_DIR := .
CMD_DIR := cmd/cli
MOCKS_DIR := mocks

# Determine platform-specific binary name
ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(BINARY_NAME).exe
endif

# Default target
.PHONY: all
all: clean build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) $(SRC_DIR)/$(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "Dependencies installed"

# Generate mock files
.PHONY: generate
generate:
	@echo "Generating mocks..."
	@mkdir -p $(MOCKS_DIR)/kubernetes
	@# Install mockgen if not already installed
	@which mockgen > /dev/null || $(GOGET) github.com/golang/mock/mockgen@v1.6.0
	@# Generate Kubernetes client mocks
	@$(MOCKGEN) -destination=$(MOCKS_DIR)/kubernetes/mock_client.go \
		-package=kubernetes \
		github.com/ethicalaakash/k8s-automator/internal/kubernetes \
		KubernetesClient,NamespaceInterface,PodInterface,ServiceInterface,EventInterface,DeploymentInterface,HorizontalPodAutoscalerInterface,ScaledObjectInterface
	@echo "Mock generation complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@$(GOTEST) ./...
	@echo "Tests completed"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Install the binary to /usr/local/bin (requires sudo on Unix)
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Remove the binary from /usr/local/bin (requires sudo on Unix)
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from /usr/local/bin..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation complete"

# Run the application (for development)
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	@$(GO) run $(SRC_DIR)/$(CMD_DIR)

# Help message
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all       - Clean and build the application"
	@echo "  build     - Build the application"
	@echo "  deps      - Install dependencies"
	@echo "  generate  - Generate mock files for testing"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  install   - Install the binary to /usr/local/bin (may require sudo)"
	@echo "  uninstall - Remove the binary from /usr/local/bin (may require sudo)"
	@echo "  run       - Run the application directly (for development)"
	@echo "  help      - Display this help message" 