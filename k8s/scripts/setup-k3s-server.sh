#!/bin/bash
# setup-k3s-server.sh - Install k3s server for PAW multi-node testing
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration
K3S_VERSION="${K3S_VERSION:-v1.29.2+k3s1}"
CLUSTER_CIDR="${CLUSTER_CIDR:-10.42.0.0/16}"
SERVICE_CIDR="${SERVICE_CIDR:-10.43.0.0/16}"
INSTALL_INGRESS="${INSTALL_INGRESS:-true}"
TAILSCALE_INTERFACE="${TAILSCALE_INTERFACE:-}"

# Colors
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

    if [ "$(id -u)" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi

    # Check for existing k3s
    if command -v k3s &> /dev/null; then
        log_warn "k3s is already installed"
        read -p "Reinstall? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            log_info "Skipping installation"
            exit 0
        fi

        # Uninstall existing k3s
        /usr/local/bin/k3s-uninstall.sh 2>/dev/null || true
    fi

    # Check Docker (optional but recommended)
    if ! command -v docker &> /dev/null; then
        log_warn "Docker not installed - k3s will use containerd"
    fi

    log_success "Prerequisites check passed"
}

detect_network_interface() {
    log_info "Detecting network interface..."

    # Check for Tailscale
    if [ -n "$TAILSCALE_INTERFACE" ]; then
        log_info "Using specified Tailscale interface: $TAILSCALE_INTERFACE"
        return
    fi

    if ip link show tailscale0 &>/dev/null; then
        TAILSCALE_INTERFACE="tailscale0"
        log_info "Found Tailscale interface: tailscale0"
    else
        # Use default interface
        TAILSCALE_INTERFACE=$(ip route get 1 | awk '{print $5;exit}')
        log_info "Using default interface: $TAILSCALE_INTERFACE"
    fi
}

get_node_ip() {
    if [ "$TAILSCALE_INTERFACE" = "tailscale0" ]; then
        ip -4 addr show tailscale0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}'
    else
        hostname -I | awk '{print $1}'
    fi
}

prepare_directories() {
    log_info "Preparing directories..."

    mkdir -p /data/paw-validators
    mkdir -p /data/paw-nodes
    mkdir -p /data/paw-monitoring
    mkdir -p /etc/rancher/k3s

    log_success "Directories created"
}

install_k3s() {
    log_info "Installing k3s server..."

    local NODE_IP=$(get_node_ip)
    log_info "Node IP: $NODE_IP"

    local INSTALL_ARGS=(
        "--cluster-init"
        "--write-kubeconfig-mode=644"
        "--kubelet-arg=node-ip=$NODE_IP"
        "--cluster-cidr=$CLUSTER_CIDR"
        "--service-cidr=$SERVICE_CIDR"
        "--kube-apiserver-arg=enable-admission-plugins=NodeRestriction,PodSecurityPolicy"
        "--kube-apiserver-arg=audit-log-path=/var/log/kubernetes/audit.log"
        "--kube-apiserver-arg=audit-log-maxage=30"
        "--kube-apiserver-arg=audit-log-maxbackup=3"
        "--kube-apiserver-arg=audit-log-maxsize=100"
    )

    # Disable Traefik if we're installing nginx
    if [ "$INSTALL_INGRESS" = "true" ]; then
        INSTALL_ARGS+=("--disable=traefik")
    fi

    # Add Tailscale interface if detected
    if [ -n "$TAILSCALE_INTERFACE" ]; then
        INSTALL_ARGS+=("--flannel-iface=$TAILSCALE_INTERFACE")
        INSTALL_ARGS+=("--advertise-address=$NODE_IP")
        INSTALL_ARGS+=("--tls-san=$NODE_IP")
    fi

    # Create audit log directory
    mkdir -p /var/log/kubernetes

    # Install k3s
    curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="$K3S_VERSION" sh -s - server "${INSTALL_ARGS[@]}"

    # Wait for k3s to be ready
    log_info "Waiting for k3s to start..."
    sleep 10

    until kubectl get nodes &>/dev/null; do
        echo -n "."
        sleep 2
    done
    echo ""

    log_success "k3s server installed"
}

install_local_path_provisioner() {
    log_info "Installing local-path-provisioner..."

    kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.26/deploy/local-path-storage.yaml

    # Make it the default storage class
    kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'

    log_success "local-path-provisioner installed"
}

install_nginx_ingress() {
    if [ "$INSTALL_INGRESS" != "true" ]; then
        return
    fi

    log_info "Installing NGINX Ingress Controller..."

    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.5/deploy/static/provider/baremetal/deploy.yaml

    log_success "NGINX Ingress Controller installed"
}

install_metrics_server() {
    log_info "Installing Metrics Server..."

    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

    # Patch for insecure TLS
    kubectl patch deployment metrics-server -n kube-system \
        --type='json' \
        -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]' || true

    log_success "Metrics Server installed"
}

setup_kubeconfig() {
    log_info "Setting up kubeconfig..."

    # Copy kubeconfig for non-root user
    local SUDO_USER_HOME=$(getent passwd "${SUDO_USER:-$USER}" | cut -d: -f6)

    mkdir -p "$SUDO_USER_HOME/.kube"
    cp /etc/rancher/k3s/k3s.yaml "$SUDO_USER_HOME/.kube/config"
    chown -R "${SUDO_USER:-$USER}:${SUDO_USER:-$USER}" "$SUDO_USER_HOME/.kube"

    # Update server address in kubeconfig
    local NODE_IP=$(get_node_ip)
    sed -i "s/127.0.0.1/$NODE_IP/g" "$SUDO_USER_HOME/.kube/config"

    log_success "Kubeconfig set up at $SUDO_USER_HOME/.kube/config"
}

get_join_token() {
    log_info "Getting join token for worker nodes..."

    local TOKEN=$(cat /var/lib/rancher/k3s/server/node-token)
    local NODE_IP=$(get_node_ip)

    echo ""
    echo "=============================================="
    echo "Worker Node Join Command:"
    echo "=============================================="
    echo ""
    echo "Run this on worker nodes to join the cluster:"
    echo ""
    echo "curl -sfL https://get.k3s.io | K3S_URL=https://${NODE_IP}:6443 K3S_TOKEN=${TOKEN} sh -s - agent --flannel-iface=<interface>"
    echo ""
    echo "Or use the setup script:"
    echo "./k8s/scripts/setup-k3s-agent.sh ${NODE_IP} ${TOKEN}"
    echo ""
}

verify_installation() {
    log_info "Verifying installation..."

    echo ""
    echo "Nodes:"
    kubectl get nodes -o wide

    echo ""
    echo "System pods:"
    kubectl get pods -n kube-system

    echo ""
    echo "Storage classes:"
    kubectl get storageclass
}

print_summary() {
    local NODE_IP=$(get_node_ip)

    echo ""
    echo "=============================================="
    echo -e "${GREEN}k3s Server Installation Complete${NC}"
    echo "=============================================="
    echo ""
    echo "Server: ${NODE_IP}:6443"
    echo "Kubeconfig: ~/.kube/config"
    echo ""
    echo "Features installed:"
    echo "  - k3s ${K3S_VERSION}"
    echo "  - local-path-provisioner (default storage)"
    echo "  - metrics-server"
    [ "$INSTALL_INGRESS" = "true" ] && echo "  - nginx-ingress"
    echo ""
    echo "Next steps:"
    echo "  1. Join worker nodes (see command above)"
    echo "  2. Deploy PAW: ./k8s/scripts/deploy-local.sh"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}k3s Server Installation${NC}"
    echo "=============================================="
    echo ""

    check_prerequisites
    detect_network_interface
    prepare_directories
    install_k3s
    install_local_path_provisioner
    install_nginx_ingress
    install_metrics_server
    setup_kubeconfig
    verify_installation
    get_join_token
    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            K3S_VERSION="$2"
            shift 2
            ;;
        --interface)
            TAILSCALE_INTERFACE="$2"
            shift 2
            ;;
        --no-ingress)
            INSTALL_INGRESS="false"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --version VER     k3s version (default: v1.29.2+k3s1)"
            echo "  --interface IFACE Network interface (default: auto-detect)"
            echo "  --no-ingress      Don't install nginx-ingress"
            echo "  --help, -h        Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
