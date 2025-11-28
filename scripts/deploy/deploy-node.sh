#!/bin/bash
# PAW Blockchain - Full Node Deployment Script
# Deploys a full node to Kubernetes cluster

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"

# Default values
NAMESPACE="${NAMESPACE:-paw-blockchain}"
REPLICAS="${REPLICAS:-3}"
CHAIN_ID="${CHAIN_ID:-paw-1}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
STORAGE_SIZE="${STORAGE_SIZE:-500Gi}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    # Check if connected to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Not connected to a Kubernetes cluster. Please configure kubectl."
        exit 1
    fi

    # Check if Docker image exists
    if ! docker images | grep -q "paw-chain/paw-node"; then
        log_warn "Docker image paw-chain/paw-node:$IMAGE_TAG not found locally."
        log_warn "Make sure to build and push the image before deploying."
        read -p "Continue anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    log_info "Prerequisites check passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating namespace $NAMESPACE..."

    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Namespace $NAMESPACE already exists"
    else
        kubectl apply -f "$K8S_DIR/namespace.yaml"
        log_info "Namespace $NAMESPACE created"
    fi
}

# Deploy storage resources
deploy_storage() {
    log_info "Deploying storage resources..."

    kubectl apply -f "$K8S_DIR/storage.yaml"
    log_info "Storage resources deployed"
}

# Deploy configuration
deploy_configuration() {
    log_info "Deploying configuration..."

    # Update ConfigMap with environment-specific values
    kubectl apply -f "$K8S_DIR/configmaps.yaml"

    log_info "Configuration deployed"
}

# Deploy services
deploy_services() {
    log_info "Deploying services..."

    kubectl apply -f "$K8S_DIR/services.yaml"
    log_info "Services deployed"
}

# Deploy node deployment
deploy_node() {
    log_info "Deploying PAW full nodes..."

    # Update deployment with custom values
    sed -e "s|replicas: 3|replicas: $REPLICAS|g" \
        -e "s|image: paw-chain/paw-node:latest|image: paw-chain/paw-node:$IMAGE_TAG|g" \
        -e "s|storage: 500Gi|storage: $STORAGE_SIZE|g" \
        "$K8S_DIR/node-deployment.yaml" | kubectl apply -f -

    log_info "Node deployment created"
}

# Wait for deployment to be ready
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."

    if kubectl rollout status deployment/paw-node -n "$NAMESPACE" --timeout=10m; then
        log_info "Deployment is ready"
    else
        log_error "Deployment failed to become ready"
        exit 1
    fi
}

# Display deployment info
display_info() {
    log_info "Deployment completed successfully!"
    echo ""
    log_info "Deployment Information:"
    echo "  Namespace: $NAMESPACE"
    echo "  Replicas: $REPLICAS"
    echo "  Chain ID: $CHAIN_ID"
    echo "  Image: paw-chain/paw-node:$IMAGE_TAG"
    echo ""

    log_info "Services:"
    kubectl get svc -n "$NAMESPACE" -l app=paw

    echo ""
    log_info "Pods:"
    kubectl get pods -n "$NAMESPACE" -l component=node

    echo ""
    log_info "To view logs, run:"
    echo "  kubectl logs -f -n $NAMESPACE -l component=node"

    echo ""
    log_info "To access RPC endpoint:"
    RPC_IP=$(kubectl get svc paw-rpc -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    echo "  RPC: http://$RPC_IP:26657"

    echo ""
    log_info "To access API endpoint:"
    API_IP=$(kubectl get svc paw-api -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    echo "  API: http://$API_IP:1317"
}

# Main deployment function
main() {
    log_info "Starting PAW full node deployment..."
    log_info "Configuration:"
    echo "  Namespace: $NAMESPACE"
    echo "  Replicas: $REPLICAS"
    echo "  Chain ID: $CHAIN_ID"
    echo "  Image Tag: $IMAGE_TAG"
    echo "  Storage Size: $STORAGE_SIZE"
    echo ""

    read -p "Proceed with deployment? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi

    check_prerequisites
    create_namespace
    deploy_storage
    deploy_configuration
    deploy_services
    deploy_node
    wait_for_deployment
    display_info

    log_info "Deployment script completed successfully"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy PAW blockchain full nodes to Kubernetes

OPTIONS:
    -n, --namespace     Kubernetes namespace (default: paw-blockchain)
    -r, --replicas      Number of replicas (default: 3)
    -c, --chain-id      Chain ID (default: paw-1)
    -t, --image-tag     Docker image tag (default: latest)
    -s, --storage       Storage size (default: 500Gi)
    -h, --help          Show this help message

EXAMPLES:
    # Deploy with defaults
    $0

    # Deploy with custom replicas
    $0 --replicas 5

    # Deploy with specific image tag
    $0 --image-tag v1.0.0

    # Deploy to custom namespace
    $0 --namespace paw-testnet

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--replicas)
            REPLICAS="$2"
            shift 2
            ;;
        -c|--chain-id)
            CHAIN_ID="$2"
            shift 2
            ;;
        -t|--image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -s|--storage)
            STORAGE_SIZE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
main
