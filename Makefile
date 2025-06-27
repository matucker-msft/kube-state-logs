.PHONY: build run test clean docker-build docker-push helm-package helm-install helm-uninstall help

# Variables
IMAGE_NAME ?= kube-state-logs
IMAGE_TAG ?= latest
REGISTRY ?= 
HELM_RELEASE_NAME ?= kube-state-logs
HELM_NAMESPACE ?= monitoring

# Build the application
build:
	go build -o bin/kube-state-logs .

# Build for multiple platforms
build-multi:
	GOOS=linux GOARCH=amd64 go build -o bin/kube-state-logs-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/kube-state-logs-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/kube-state-logs-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/kube-state-logs-darwin-arm64 .

# Run the application locally
run:
	go run .

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf coverage.out

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build Docker image
docker-build:
	docker build -t $(REGISTRY)$(IMAGE_NAME):$(IMAGE_TAG) .

# Build Docker image for multiple platforms
docker-build-multi:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(REGISTRY)$(IMAGE_NAME):$(IMAGE_TAG) .

# Push Docker image
docker-push:
	docker push $(REGISTRY)$(IMAGE_NAME):$(IMAGE_TAG)

# Run with specific flags
run-debug:
	go run . --log-level=debug --log-interval=10s

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate manifests
manifests:
	mkdir -p deploy
	kustomize build config/default > deploy/manifests.yaml

# Helm commands
helm-package:
	helm package charts/kube-state-logs

helm-install:
	helm install $(HELM_RELEASE_NAME) charts/kube-state-logs \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--wait

helm-upgrade:
	helm upgrade $(HELM_RELEASE_NAME) charts/kube-state-logs \
		--namespace $(HELM_NAMESPACE) \
		--wait

helm-uninstall:
	helm uninstall $(HELM_RELEASE_NAME) --namespace $(HELM_NAMESPACE)

helm-test:
	helm test $(HELM_RELEASE_NAME) --namespace $(HELM_NAMESPACE)

helm-lint:
	helm lint charts/kube-state-logs

helm-template:
	helm template $(HELM_RELEASE_NAME) charts/kube-state-logs \
		--namespace $(HELM_NAMESPACE) \
		--debug

# Kubernetes commands
k8s-apply:
	kubectl apply -f deploy/manifests.yaml

k8s-delete:
	kubectl delete -f deploy/manifests.yaml

k8s-logs:
	kubectl logs -f deployment/$(HELM_RELEASE_NAME) -n $(HELM_NAMESPACE)

k8s-status:
	kubectl get pods -n $(HELM_NAMESPACE) -l app.kubernetes.io/name=kube-state-logs

# Development helpers
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install sigs.k8s.io/kind@latest
	go install helm.sh/helm/v3/cmd/helm@latest

# Create kind cluster for testing
kind-create:
	kind create cluster --name kube-state-logs-test

kind-delete:
	kind delete cluster --name kube-state-logs-test

# Load image into kind cluster
kind-load:
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name kube-state-logs-test

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  build-multi        - Build for multiple platforms"
	@echo "  run                - Run the application locally"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-build-multi - Build Docker image for multiple platforms"
	@echo "  docker-push        - Push Docker image"
	@echo "  run-debug          - Run with debug flags"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  helm-package       - Package Helm chart"
	@echo "  helm-install       - Install Helm chart"
	@echo "  helm-upgrade       - Upgrade Helm chart"
	@echo "  helm-uninstall     - Uninstall Helm chart"
	@echo "  helm-test          - Test Helm release"
	@echo "  helm-lint          - Lint Helm chart"
	@echo "  helm-template      - Template Helm chart"
	@echo "  k8s-apply          - Apply Kubernetes manifests"
	@echo "  k8s-delete         - Delete Kubernetes manifests"
	@echo "  k8s-logs           - Show application logs"
	@echo "  k8s-status         - Show application status"
	@echo "  dev-setup          - Setup development environment"
	@echo "  kind-create        - Create kind cluster"
	@echo "  kind-delete        - Delete kind cluster"
	@echo "  kind-load          - Load image into kind cluster"
	@echo "  help               - Show this help" 