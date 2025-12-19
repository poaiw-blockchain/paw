#!/bin/bash
# deploy-linkerd.sh - Deploy Linkerd service mesh for PAW
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

    if ! command -v linkerd &> /dev/null; then
        log_info "Installing Linkerd CLI..."
        curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install | sh
        export PATH=$HOME/.linkerd2/bin:$PATH
    fi

    # Verify cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log_success "Prerequisites met"
}

pre_install_check() {
    log_info "Running Linkerd pre-installation checks..."

    linkerd check --pre || {
        log_warn "Pre-check had warnings - proceeding anyway"
    }
}

install_linkerd_crds() {
    log_info "Installing Linkerd CRDs..."

    linkerd install --crds | kubectl apply -f -

    log_success "Linkerd CRDs installed"
}

install_linkerd() {
    log_info "Installing Linkerd control plane..."

    linkerd install \
        --set proxyInit.runAsRoot=true \
        --set proxy.resources.cpu.request=100m \
        --set proxy.resources.memory.request=64Mi \
        | kubectl apply -f -

    log_info "Waiting for Linkerd control plane..."
    linkerd check --wait 5m || log_warn "Linkerd check had issues"

    log_success "Linkerd control plane installed"
}

install_linkerd_viz() {
    log_info "Installing Linkerd Viz extension..."

    linkerd viz install | kubectl apply -f -

    log_info "Waiting for Linkerd Viz..."
    linkerd viz check --wait 3m || log_warn "Linkerd Viz check had issues"

    log_success "Linkerd Viz installed"
}

mesh_paw_namespace() {
    log_info "Meshing PAW namespace..."

    # Add injection annotation to namespace
    kubectl annotate namespace paw linkerd.io/inject=enabled --overwrite 2>/dev/null || log_warn "PAW namespace not found yet"

    # Restart pods to inject proxy
    kubectl rollout restart statefulset/paw-validator -n paw 2>/dev/null || log_warn "No validators to restart"
    kubectl rollout restart deployment -n paw 2>/dev/null || log_warn "No deployments to restart"

    log_success "PAW namespace meshed"
}

verify_mesh() {
    log_info "Verifying mesh..."

    echo ""
    echo "=== Linkerd Control Plane ==="
    kubectl get pods -n linkerd

    echo ""
    echo "=== Linkerd Viz ==="
    kubectl get pods -n linkerd-viz

    echo ""
    echo "=== PAW Mesh Status ==="
    linkerd check --proxy -n paw 2>/dev/null || echo "PAW namespace not ready for proxy check"

    echo ""
    echo "=== Meshed Pods ==="
    linkerd stat pods -n paw 2>/dev/null || echo "No meshed pods in PAW namespace yet"
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}Linkerd Service Mesh Deployed${NC}"
    echo "=============================================="
    echo ""
    echo "Features Enabled:"
    echo "  - Automatic mTLS between all meshed pods"
    echo "  - Traffic metrics and observability"
    echo "  - Retries and timeouts"
    echo ""
    echo "Commands:"
    echo "  Dashboard:     linkerd viz dashboard"
    echo "  Check:         linkerd check"
    echo "  Mesh status:   linkerd stat pods -n paw"
    echo "  Top traffic:   linkerd viz top deploy -n paw"
    echo "  Tap traffic:   linkerd viz tap deploy -n paw"
    echo ""
    echo "To mesh a new deployment:"
    echo "  kubectl annotate namespace <ns> linkerd.io/inject=enabled"
    echo "  kubectl rollout restart deploy/<name> -n <ns>"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}Linkerd Service Mesh Deployment${NC}"
    echo "=============================================="
    echo ""

    check_prerequisites
    pre_install_check
    install_linkerd_crds
    install_linkerd
    install_linkerd_viz
    mesh_paw_namespace
    verify_mesh
    print_summary
}

main
