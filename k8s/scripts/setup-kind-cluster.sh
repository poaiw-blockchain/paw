#!/bin/bash
# setup-kind-cluster.sh - Create Kind cluster for PAW local development
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-paw-local}"
K8S_VERSION="${K8S_VERSION:-v1.29.2}"
REGISTRY_NAME="${REGISTRY_NAME:-paw-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5050}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_prerequisites() {
    log_info "Checking prerequisites..."

    local missing=()

    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi

    if ! command -v kind &> /dev/null; then
        missing+=("kind")
    fi

    if ! command -v kubectl &> /dev/null; then
        missing+=("kubectl")
    fi

    if ! command -v helm &> /dev/null; then
        missing+=("helm")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        echo "Install with:"
        echo "  docker: https://docs.docker.com/get-docker/"
        echo "  kind: go install sigs.k8s.io/kind@latest"
        echo "  kubectl: https://kubernetes.io/docs/tasks/tools/"
        echo "  helm: https://helm.sh/docs/intro/install/"
        exit 1
    fi

    # Check Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker is not running"
        exit 1
    fi

    log_success "All prerequisites met"
}

setup_local_registry() {
    log_info "Setting up local Docker registry..."

    if docker ps | grep -q "$REGISTRY_NAME"; then
        log_info "Registry already running"
        return
    fi

    # Create registry container
    docker run -d --restart=always \
        -p "127.0.0.1:${REGISTRY_PORT}:5000" \
        --name "$REGISTRY_NAME" \
        registry:2 || true

    log_success "Local registry running at localhost:${REGISTRY_PORT}"
}

create_kind_config() {
    log_info "Creating Kind cluster configuration..."

    cat > /tmp/kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      # PAW P2P
      - containerPort: 31656
        hostPort: 31656
        protocol: TCP
      # PAW RPC
      - containerPort: 31657
        hostPort: 31657
        protocol: TCP
      # PAW API
      - containerPort: 31317
        hostPort: 31317
        protocol: TCP
      # PAW gRPC
      - containerPort: 31090
        hostPort: 31090
        protocol: TCP
      # Grafana
      - containerPort: 31030
        hostPort: 31030
        protocol: TCP
      # Prometheus
      - containerPort: 31009
        hostPort: 31009
        protocol: TCP
      # Ingress HTTP
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      # Ingress HTTPS
      - containerPort: 443
        hostPort: 443
        protocol: TCP
    extraMounts:
      - hostPath: /data/paw-validators
        containerPath: /data/paw-validators
      - hostPath: /data/paw-nodes
        containerPath: /data/paw-nodes
      - hostPath: /data/paw-monitoring
        containerPath: /data/paw-monitoring
  - role: worker
    extraMounts:
      - hostPath: /data/paw-validators
        containerPath: /data/paw-validators
      - hostPath: /data/paw-nodes
        containerPath: /data/paw-nodes
  - role: worker
    extraMounts:
      - hostPath: /data/paw-validators
        containerPath: /data/paw-validators
      - hostPath: /data/paw-nodes
        containerPath: /data/paw-nodes
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
      endpoint = ["http://${REGISTRY_NAME}:5000"]
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
EOF

    log_success "Kind configuration created"
}

create_cluster() {
    log_info "Creating Kind cluster: ${CLUSTER_NAME}..."

    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        log_warn "Cluster ${CLUSTER_NAME} already exists"
        read -p "Delete and recreate? (y/N): " confirm
        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            kind delete cluster --name "$CLUSTER_NAME"
        else
            log_info "Using existing cluster"
            return
        fi
    fi

    # Create data directories
    sudo mkdir -p /data/paw-{validators,nodes,monitoring}
    sudo chown -R "$(id -u):$(id -g)" /data/paw-*

    # Create cluster
    kind create cluster --config /tmp/kind-config.yaml --image "kindest/node:${K8S_VERSION}"

    # Connect registry to cluster network
    if docker network inspect kind &>/dev/null; then
        docker network connect kind "$REGISTRY_NAME" 2>/dev/null || true
    fi

    log_success "Kind cluster created successfully"
}

install_ingress_nginx() {
    log_info "Installing NGINX Ingress Controller..."

    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

    log_info "Waiting for Ingress controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=180s || log_warn "Ingress controller not ready yet, continuing..."

    log_success "NGINX Ingress Controller installed"
}

install_local_path_provisioner() {
    log_info "Installing local-path-provisioner for storage..."

    kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.26/deploy/local-path-storage.yaml

    # Make it the default storage class
    kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'

    log_success "local-path-provisioner installed"
}

install_metrics_server() {
    log_info "Installing Metrics Server..."

    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

    # Patch for insecure TLS (required for Kind)
    kubectl patch deployment metrics-server -n kube-system \
        --type='json' \
        -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]' || true

    log_success "Metrics Server installed"
}

configure_kubectl() {
    log_info "Configuring kubectl context..."

    kubectl config use-context "kind-${CLUSTER_NAME}"
    kubectl cluster-info

    log_success "kubectl configured for cluster ${CLUSTER_NAME}"
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}PAW Kind Cluster Setup Complete${NC}"
    echo "=============================================="
    echo ""
    echo "Cluster: ${CLUSTER_NAME}"
    echo "Registry: localhost:${REGISTRY_PORT}"
    echo ""
    echo "Nodes:"
    kubectl get nodes -o wide
    echo ""
    echo "Port Mappings:"
    echo "  P2P:        localhost:31656"
    echo "  RPC:        localhost:31657"
    echo "  API:        localhost:31317"
    echo "  gRPC:       localhost:31090"
    echo "  Grafana:    localhost:31030"
    echo "  Prometheus: localhost:31009"
    echo "  HTTP:       localhost:80"
    echo "  HTTPS:      localhost:443"
    echo ""
    echo "Next steps:"
    echo "  1. Build PAW image: docker build -t localhost:${REGISTRY_PORT}/pawd:latest ."
    echo "  2. Push to registry: docker push localhost:${REGISTRY_PORT}/pawd:latest"
    echo "  3. Deploy PAW: ./k8s/scripts/deploy-local.sh"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kind Cluster Setup${NC}"
    echo "=============================================="
    echo ""

    check_prerequisites
    setup_local_registry
    create_kind_config
    create_cluster
    install_local_path_provisioner
    install_ingress_nginx
    install_metrics_server
    configure_kubectl
    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            CLUSTER_NAME="$2"
            shift 2
            ;;
        --k8s-version)
            K8S_VERSION="$2"
            shift 2
            ;;
        --registry-port)
            REGISTRY_PORT="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --name NAME          Cluster name (default: paw-local)"
            echo "  --k8s-version VER    Kubernetes version (default: v1.29.2)"
            echo "  --registry-port PORT Local registry port (default: 5050)"
            echo "  --help, -h           Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
