#!/bin/bash
# deploy-vault.sh - Deploy HashiCorp Vault for PAW secrets management
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MODE="${MODE:-dev}"
VAULT_VERSION="${VAULT_VERSION:-1.15.0}"

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

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found"
        exit 1
    fi

    if ! command -v helm &> /dev/null; then
        log_error "helm not found"
        exit 1
    fi

    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log_success "Prerequisites met"
}

add_helm_repos() {
    log_info "Adding Helm repositories..."

    helm repo add hashicorp https://helm.releases.hashicorp.com 2>/dev/null || true
    helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
    helm repo update

    log_success "Helm repositories updated"
}

deploy_vault_dev() {
    log_info "Deploying Vault in DEV mode..."

    # Apply dev mode manifests
    kubectl apply -f "$PROJECT_ROOT/k8s/components/vault/vault-dev.yaml"

    log_info "Waiting for Vault to be ready..."
    kubectl wait --namespace vault \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/name=vault \
        --timeout=120s

    log_success "Vault deployed in dev mode"
    log_warn "DEV MODE: Root token is 'paw-dev-token' - DO NOT USE IN PRODUCTION"
}

deploy_vault_prod() {
    log_info "Deploying Vault in PRODUCTION mode..."

    # Create namespace
    kubectl create namespace vault --dry-run=client -o yaml | kubectl apply -f -

    # Deploy Vault with Helm
    helm upgrade --install vault hashicorp/vault \
        --namespace vault \
        --set "server.ha.enabled=true" \
        --set "server.ha.replicas=3" \
        --set "server.ha.raft.enabled=true" \
        --set "server.dataStorage.size=10Gi" \
        --set "server.auditStorage.enabled=true" \
        --set "server.auditStorage.size=10Gi" \
        --set "injector.enabled=false" \
        --set "csi.enabled=false" \
        --wait

    log_warn "Vault deployed in HA mode - manual initialization required"
    log_info "Run: kubectl exec -n vault vault-0 -- vault operator init"

    log_success "Vault deployed in production mode"
}

deploy_external_secrets() {
    log_info "Deploying External Secrets Operator..."

    # Create namespace
    kubectl create namespace external-secrets --dry-run=client -o yaml | kubectl apply -f -

    # Install ESO with Helm
    helm upgrade --install external-secrets external-secrets/external-secrets \
        --namespace external-secrets \
        --set "installCRDs=true" \
        --set "webhook.port=9443" \
        --wait

    log_info "Waiting for External Secrets Operator to be ready..."
    kubectl wait --namespace external-secrets \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/name=external-secrets \
        --timeout=120s

    log_success "External Secrets Operator deployed"
}

configure_vault_secrets() {
    log_info "Configuring Vault secrets for PAW..."

    local VAULT_ADDR="http://vault.vault.svc.cluster.local:8200"
    local VAULT_TOKEN="paw-dev-token"

    # Wait for Vault to be fully ready
    sleep 5

    # Configure secrets using kubectl exec
    kubectl exec -n vault deploy/vault -- sh -c "
        export VAULT_ADDR='http://127.0.0.1:8200'
        export VAULT_TOKEN='${VAULT_TOKEN}'

        # Enable KV secrets engine v2
        vault secrets enable -path=secret kv-v2 2>/dev/null || true

        # Create policy for PAW
        vault policy write paw-policy - <<'POLICY'
path \"secret/data/paw/*\" {
  capabilities = [\"read\", \"list\"]
}
path \"secret/metadata/paw/*\" {
  capabilities = [\"read\", \"list\"]
}
POLICY

        # Enable Kubernetes auth
        vault auth enable kubernetes 2>/dev/null || true

        # Configure Kubernetes auth (for production)
        vault write auth/kubernetes/config \\
            kubernetes_host=\"https://\${KUBERNETES_PORT_443_TCP_ADDR}:443\" \\
            token_reviewer_jwt=\"\$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)\" \\
            kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \\
            disable_local_ca_jwt=\"true\" 2>/dev/null || true

        # Create role for PAW namespace
        vault write auth/kubernetes/role/paw \\
            bound_service_account_names=paw-validator,paw-node,paw-api \\
            bound_service_account_namespaces=paw \\
            policies=paw-policy \\
            ttl=1h 2>/dev/null || true

        echo 'Vault policies and roles configured'
    "

    log_success "Vault secrets configured"
}

init_paw_secrets() {
    log_info "Initializing PAW secrets in Vault..."

    local VAULT_TOKEN="paw-dev-token"

    # Generate sample secrets
    JWT_SECRET=$(openssl rand -base64 32)

    # Store secrets in Vault
    kubectl exec -n vault deploy/vault -- sh -c "
        export VAULT_ADDR='http://127.0.0.1:8200'
        export VAULT_TOKEN='${VAULT_TOKEN}'

        # Store API secrets
        vault kv put secret/paw/api jwt_secret='${JWT_SECRET}'

        # Placeholder for validator keys (will be replaced with real keys)
        vault kv put secret/paw/validators \\
            priv_validator_key='{}' \\
            node_key='{}'

        # Placeholder for genesis
        vault kv put secret/paw/genesis genesis_json='{}'

        echo 'PAW secrets initialized'
    "

    log_success "PAW secrets initialized in Vault"
}

setup_external_secrets_store() {
    log_info "Setting up External Secrets ClusterSecretStore..."

    # Apply External Secrets configuration
    kubectl apply -f "$PROJECT_ROOT/k8s/components/vault/external-secrets.yaml"

    log_info "Waiting for ClusterSecretStore to be ready..."
    sleep 5

    # Verify the store
    if kubectl get clustersecretstore vault-backend -o jsonpath='{.status.conditions[0].status}' 2>/dev/null | grep -q "True"; then
        log_success "ClusterSecretStore is ready"
    else
        log_warn "ClusterSecretStore may not be ready yet - check with: kubectl get clustersecretstore"
    fi
}

verify_setup() {
    log_info "Verifying Vault setup..."

    echo ""
    echo "Vault Status:"
    kubectl get pods -n vault

    echo ""
    echo "External Secrets Status:"
    kubectl get pods -n external-secrets

    echo ""
    echo "ClusterSecretStore Status:"
    kubectl get clustersecretstore vault-backend -o wide 2>/dev/null || echo "Not found yet"

    echo ""
    echo "ExternalSecrets in PAW namespace:"
    kubectl get externalsecrets -n paw 2>/dev/null || echo "None yet (namespace may not exist)"
}

print_summary() {
    echo ""
    echo "=============================================="
    echo -e "${GREEN}Vault Setup Complete${NC}"
    echo "=============================================="
    echo ""
    echo "Mode: ${MODE}"
    echo ""
    if [ "$MODE" = "dev" ]; then
        echo "Vault Address: http://localhost:8200 (port-forward required)"
        echo "Root Token: paw-dev-token"
        echo ""
        echo "Port forward: kubectl port-forward -n vault svc/vault 8200:8200"
        echo ""
    fi
    echo "Next steps:"
    echo "  1. Create PAW namespace: kubectl apply -f k8s/base/namespace.yaml"
    echo "  2. Store real validator keys in Vault"
    echo "  3. Verify ExternalSecrets sync: kubectl get externalsecrets -n paw"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Vault Deployment${NC}"
    echo "=============================================="
    echo ""

    check_prerequisites
    add_helm_repos

    if [ "$MODE" = "dev" ]; then
        deploy_vault_dev
    else
        deploy_vault_prod
    fi

    deploy_external_secrets

    if [ "$MODE" = "dev" ]; then
        configure_vault_secrets
        init_paw_secrets
    fi

    setup_external_secrets_store
    verify_setup
    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            MODE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --mode MODE    Deployment mode: dev or prod (default: dev)"
            echo "  --help, -h     Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
