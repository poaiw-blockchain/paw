#!/bin/bash
# deploy-local.sh - Deploy PAW infrastructure to local Kind cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"
COMPONENTS="${COMPONENTS:-all}"
DRY_RUN="${DRY_RUN:-false}"
SKIP_BUILD="${SKIP_BUILD:-false}"
REGISTRY_PORT="${REGISTRY_PORT:-5050}"

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

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        log_info "Run ./k8s/scripts/setup-kind-cluster.sh first"
        exit 1
    fi

    # Check if using Kind cluster
    if kubectl config current-context | grep -q "kind"; then
        log_info "Using Kind cluster"
    else
        log_warn "Not using Kind cluster - some features may not work"
    fi

    log_success "Prerequisites met"
}

build_and_push_image() {
    if [ "$SKIP_BUILD" = "true" ]; then
        log_info "Skipping image build (--skip-build)"
        return
    fi

    log_info "Building PAW Docker image..."

    cd "$PROJECT_ROOT"

    # Build the image
    docker build -t "localhost:${REGISTRY_PORT}/pawd:latest" \
        -f Dockerfile \
        .

    # Push to local registry
    docker push "localhost:${REGISTRY_PORT}/pawd:latest"

    log_success "Image built and pushed to local registry"
}

apply_manifests() {
    local component=$1
    local path=$2

    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would apply: $path"
        kubectl apply -f "$path" --dry-run=client
    else
        log_info "Applying: $component"
        kubectl apply -f "$path"
    fi
}

deploy_base() {
    log_info "Deploying base infrastructure..."

    # Apply base manifests
    apply_manifests "Namespace" "$K8S_DIR/base/namespace.yaml"
    apply_manifests "RBAC" "$K8S_DIR/base/rbac.yaml"
    apply_manifests "Storage Classes" "$K8S_DIR/base/storage-class.yaml"
    apply_manifests "Network Policies" "$K8S_DIR/base/network-policies.yaml"

    log_success "Base infrastructure deployed"
}

deploy_vault() {
    log_info "Deploying Vault..."

    "$SCRIPT_DIR/deploy-vault.sh" --mode dev

    log_success "Vault deployed"
}

deploy_configmaps() {
    log_info "Deploying ConfigMaps..."

    # Create PAW config
    kubectl create configmap paw-config \
        --namespace paw \
        --from-literal=CHAIN_ID=paw-testnet-1 \
        --from-literal=MIN_GAS_PRICES=0.001upaw \
        --from-literal=LOG_LEVEL=info \
        --from-literal=PRUNING=custom \
        --from-literal=PRUNING_KEEP_RECENT=500000 \
        --from-literal=PRUNING_INTERVAL=10 \
        --dry-run=client -o yaml | kubectl apply -f -

    # Create genesis config (placeholder - update with real values)
    kubectl create configmap paw-genesis-config \
        --namespace paw \
        --from-literal=GENESIS_URL=https://raw.githubusercontent.com/paw-chain/networks/main/testnet/genesis.json \
        --from-literal=GENESIS_SHA256=placeholder_checksum \
        --dry-run=client -o yaml | kubectl apply -f -

    log_success "ConfigMaps deployed"
}

deploy_validators() {
    log_info "Deploying Validators..."

    apply_manifests "Validator StatefulSet" "$K8S_DIR/validators/statefulset.yaml"
    apply_manifests "Validator Services" "$K8S_DIR/validators/services.yaml"
    apply_manifests "Validator PDB" "$K8S_DIR/validators/pdb.yaml"

    # Wait for validators (optional)
    if [ "$DRY_RUN" != "true" ]; then
        log_info "Waiting for validators to be ready..."
        kubectl rollout status statefulset/paw-validator -n paw --timeout=300s || log_warn "Validators not ready yet"
    fi

    log_success "Validators deployed"
}

deploy_nodes() {
    log_info "Deploying Full Nodes..."

    if [ -f "$K8S_DIR/nodes/deployment.yaml" ]; then
        apply_manifests "Node Deployment" "$K8S_DIR/nodes/deployment.yaml"
        apply_manifests "Node Services" "$K8S_DIR/nodes/services.yaml"
    else
        log_warn "Node manifests not found - skipping"
    fi

    log_success "Full Nodes deployed"
}

deploy_monitoring() {
    log_info "Deploying Monitoring Stack..."

    "$SCRIPT_DIR/deploy-monitoring.sh" || log_warn "Monitoring deployment had issues"

    log_success "Monitoring deployed"
}

deploy_linkerd() {
    log_info "Deploying Linkerd Service Mesh..."

    "$SCRIPT_DIR/deploy-linkerd.sh" || log_warn "Linkerd deployment had issues"

    log_success "Linkerd deployed"
}

patch_for_local() {
    log_info "Applying local development patches..."

    # Patch validator StatefulSet to use local registry
    kubectl patch statefulset paw-validator -n paw \
        --type='json' \
        -p="[{\"op\": \"replace\", \"path\": \"/spec/template/spec/containers/0/image\", \"value\": \"localhost:${REGISTRY_PORT}/pawd:latest\"}]" \
        2>/dev/null || true

    # Patch storage class to use local-path
    kubectl patch statefulset paw-validator -n paw \
        --type='json' \
        -p='[{"op": "replace", "path": "/spec/volumeClaimTemplates/0/spec/storageClassName", "value": "local-path"}]' \
        2>/dev/null || true

    # Reduce resource requests for local dev
    kubectl patch statefulset paw-validator -n paw \
        --type='json' \
        -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "500m"}]' \
        2>/dev/null || true

    kubectl patch statefulset paw-validator -n paw \
        --type='json' \
        -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "1Gi"}]' \
        2>/dev/null || true

    log_success "Local patches applied"
}

verify_deployment() {
    log_info "Verifying deployment..."

    echo ""
    echo "=== Namespace ==="
    kubectl get namespace paw

    echo ""
    echo "=== Pods ==="
    kubectl get pods -n paw -o wide

    echo ""
    echo "=== Services ==="
    kubectl get services -n paw

    echo ""
    echo "=== PVCs ==="
    kubectl get pvc -n paw

    echo ""
    echo "=== External Secrets ==="
    kubectl get externalsecrets -n paw 2>/dev/null || echo "None"
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}PAW Local Deployment Complete${NC}"
    echo "=============================================="
    echo ""
    echo "Access Points:"
    echo "  P2P:        localhost:31656"
    echo "  RPC:        localhost:31657"
    echo "  API:        localhost:31317"
    echo "  gRPC:       localhost:31090"
    echo "  Grafana:    localhost:31030"
    echo "  Prometheus: localhost:31009"
    echo ""
    echo "Commands:"
    echo "  Check status:  kubectl get pods -n paw"
    echo "  View logs:     kubectl logs -f paw-validator-0 -n paw"
    echo "  Exec into pod: kubectl exec -it paw-validator-0 -n paw -- sh"
    echo "  Run tests:     ./k8s/tests/smoke-tests.sh"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Local Deployment${NC}"
    echo "=============================================="
    echo ""

    check_prerequisites

    case "$COMPONENTS" in
        all)
            build_and_push_image
            deploy_base
            deploy_vault
            deploy_configmaps
            deploy_validators
            deploy_nodes
            deploy_monitoring
            # deploy_linkerd  # Optional - can be slow
            patch_for_local
            ;;
        base)
            deploy_base
            ;;
        vault)
            deploy_vault
            ;;
        validators)
            deploy_validators
            ;;
        nodes)
            deploy_nodes
            ;;
        monitoring)
            deploy_monitoring
            ;;
        linkerd)
            deploy_linkerd
            ;;
        *)
            log_error "Unknown component: $COMPONENTS"
            exit 1
            ;;
    esac

    verify_deployment
    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --components)
            COMPONENTS="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --skip-build)
            SKIP_BUILD="true"
            shift
            ;;
        --registry-port)
            REGISTRY_PORT="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --components COMP   Components to deploy: all, base, vault, validators, nodes, monitoring, linkerd"
            echo "  --dry-run           Show what would be applied without applying"
            echo "  --skip-build        Skip Docker image build"
            echo "  --registry-port PORT Local registry port (default: 5050)"
            echo "  --help, -h          Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
