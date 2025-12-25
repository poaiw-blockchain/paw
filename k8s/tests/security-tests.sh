#!/bin/bash
# security-tests.sh - Security validation tests for PAW Kubernetes deployment
set -u  # Keep unset variable check, but allow command failures

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-paw}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASSED=0
FAILED=0
WARNINGS=0

log_test() { echo -e "${BLUE}[TEST]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; ((WARNINGS++)); }

test_pod_security_context() {
    log_test "Checking pod security contexts..."

    local pods=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    local violations=0

    for pod in $pods; do
        # Check if running as root
        local run_as_user=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.runAsUser}' 2>/dev/null)
        if [ "$run_as_user" = "0" ]; then
            log_warn "Pod $pod running as root"
            ((violations++))
        fi

        # Check privilege escalation
        local allow_priv=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.allowPrivilegeEscalation}' 2>/dev/null)
        if [ "$allow_priv" = "true" ]; then
            log_warn "Pod $pod allows privilege escalation"
            ((violations++))
        fi

        # Check capabilities
        local caps=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.capabilities.add}' 2>/dev/null)
        if [ -n "$caps" ] && [ "$caps" != "[]" ]; then
            log_warn "Pod $pod has additional capabilities: $caps"
            ((violations++))
        fi
    done

    if [ "$violations" -eq 0 ]; then
        log_pass "All pods have secure security contexts"
    else
        log_fail "$violations security context violations found"
    fi
}

test_network_policies() {
    log_test "Checking network policies..."

    local policy_count=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$policy_count" -eq 0 ]; then
        log_fail "No network policies found - all traffic is allowed"
        return 1
    fi

    # Check for default deny policy
    local default_deny=$(kubectl get networkpolicies -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' | grep -c "default-deny" || true)

    if [ "$default_deny" -gt 0 ]; then
        log_pass "Default deny network policy exists"
    else
        log_warn "No default deny network policy found"
    fi

    log_pass "$policy_count network policies configured"
}

test_rbac_configuration() {
    log_test "Checking RBAC configuration..."

    # Check for service accounts
    local sa_count=$(kubectl get serviceaccounts -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$sa_count" -le 1 ]; then
        log_warn "Only default service account found - consider using dedicated SAs"
    else
        log_pass "$sa_count service accounts configured"
    fi

    # Check for role bindings
    local rb_count=$(kubectl get rolebindings -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$rb_count" -gt 0 ]; then
        log_pass "$rb_count role bindings configured"
    else
        log_warn "No role bindings found"
    fi

    # Check pods are using non-default service accounts
    local pods_default_sa=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{range .items[*]}{.metadata.name}:{.spec.serviceAccountName}{"\n"}{end}' | grep ":default$" | wc -l)

    if [ "$pods_default_sa" -gt 0 ]; then
        log_warn "$pods_default_sa pods using default service account"
    else
        log_pass "All pods using dedicated service accounts"
    fi
}

test_secret_management() {
    log_test "Checking secret management..."

    local secrets=$(kubectl get secrets -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    local issues=0

    for secret in $secrets; do
        # Check if secret is mounted as environment variable (less secure)
        local env_refs=$(kubectl get pods -n "$NAMESPACE" -o jsonpath="{.items[*].spec.containers[*].env[?(@.valueFrom.secretKeyRef.name=='$secret')].name}" 2>/dev/null)

        if [ -n "$env_refs" ]; then
            log_warn "Secret $secret is exposed as environment variable"
            ((issues++))
        fi
    done

    if [ "$issues" -eq 0 ]; then
        log_pass "Secrets properly managed (mounted as volumes or external secrets)"
    fi

    # Check for external secrets
    local external_secrets=$(kubectl get externalsecrets -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$external_secrets" -gt 0 ]; then
        log_pass "$external_secrets external secrets configured (Vault integration)"
    else
        log_warn "No external secrets found - consider using Vault"
    fi
}

test_resource_limits() {
    log_test "Checking resource limits..."

    local pods_without_limits=0
    local pods=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')

    for pod in $pods; do
        local containers=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}')

        for container in $containers; do
            local cpu_limit=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath="{.spec.containers[?(@.name=='$container')].resources.limits.cpu}" 2>/dev/null)
            local mem_limit=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath="{.spec.containers[?(@.name=='$container')].resources.limits.memory}" 2>/dev/null)

            if [ -z "$cpu_limit" ] || [ -z "$mem_limit" ]; then
                log_warn "Container $container in $pod missing resource limits"
                ((pods_without_limits++))
            fi
        done
    done

    if [ "$pods_without_limits" -eq 0 ]; then
        log_pass "All containers have resource limits set"
    else
        log_fail "$pods_without_limits containers missing resource limits"
    fi
}

test_pod_disruption_budget() {
    log_test "Checking Pod Disruption Budgets..."

    local pdb_count=$(kubectl get pdb -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$pdb_count" -gt 0 ]; then
        log_pass "$pdb_count Pod Disruption Budgets configured"
    else
        log_warn "No Pod Disruption Budgets found"
    fi
}

test_image_security() {
    log_test "Checking image security..."

    local pods=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    local issues=0

    for pod in $pods; do
        local images=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].image}')

        for image in $images; do
            # Check for :latest tag
            if echo "$image" | grep -q ":latest"; then
                log_warn "Pod $pod uses :latest tag: $image"
                ((issues++))
            fi

            # Check for missing tag
            if ! echo "$image" | grep -q ":"; then
                log_warn "Pod $pod uses untagged image: $image"
                ((issues++))
            fi
        done

        # Check imagePullPolicy
        local pull_policy=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].imagePullPolicy}' 2>/dev/null)
        if [ "$pull_policy" = "Always" ]; then
            # This is actually good for production
            :
        fi
    done

    if [ "$issues" -eq 0 ]; then
        log_pass "All images use explicit tags"
    else
        log_warn "$issues image tagging issues found"
    fi
}

test_service_mesh() {
    log_test "Checking service mesh (mTLS)..."

    # Check if Linkerd is installed
    if kubectl get namespace linkerd &>/dev/null; then
        log_pass "Linkerd service mesh installed"

        # Check if namespace is meshed
        local inject_annotation=$(kubectl get namespace "$NAMESPACE" -o jsonpath='{.metadata.annotations.linkerd\.io/inject}' 2>/dev/null)

        if [ "$inject_annotation" = "enabled" ]; then
            log_pass "Namespace is meshed (mTLS enabled)"
        else
            log_warn "Namespace not meshed - consider enabling Linkerd injection"
        fi
    else
        log_warn "Linkerd not installed - no automatic mTLS"
    fi
}

test_audit_logging() {
    log_test "Checking audit logging..."

    # Check if audit logs are enabled (cluster-level)
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[0].spec.containers[0].command}' 2>/dev/null | grep -q "audit-log-path"; then
        log_pass "Kubernetes audit logging is enabled"
    else
        log_warn "Kubernetes audit logging may not be enabled (check cluster config)"
    fi
}

test_sensitive_data_exposure() {
    log_test "Checking for sensitive data exposure..."

    local issues=0

    # Check for hardcoded secrets in configmaps
    local configmaps=$(kubectl get configmaps -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')

    for cm in $configmaps; do
        local data=$(kubectl get configmap "$cm" -n "$NAMESPACE" -o jsonpath='{.data}' 2>/dev/null)

        # Check for common secret patterns
        if echo "$data" | grep -qiE "(password|secret|key|token|apikey).*="; then
            log_warn "ConfigMap $cm may contain sensitive data"
            ((issues++))
        fi
    done

    if [ "$issues" -eq 0 ]; then
        log_pass "No obvious sensitive data in ConfigMaps"
    fi
}

print_summary() {
    echo ""
    echo "=============================================="
    echo "SECURITY TEST RESULTS"
    echo "=============================================="
    echo -e "Passed:   ${GREEN}$PASSED${NC}"
    echo -e "Failed:   ${RED}$FAILED${NC}"
    echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
    echo "=============================================="

    if [ "$FAILED" -gt 0 ]; then
        echo -e "${RED}SECURITY TESTS FAILED${NC}"
        echo ""
        echo "Critical issues found. Please address before production deployment."
        exit 1
    elif [ "$WARNINGS" -gt 0 ]; then
        echo -e "${YELLOW}SECURITY TESTS PASSED WITH WARNINGS${NC}"
        echo ""
        echo "Consider addressing warnings before production deployment."
        exit 0
    else
        echo -e "${GREEN}ALL SECURITY TESTS PASSED${NC}"
        exit 0
    fi
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Kubernetes Security Tests${NC}"
    echo "=============================================="
    echo "Namespace: $NAMESPACE"
    echo "Time: $(date)"
    echo ""

    test_pod_security_context
    test_network_policies
    test_rbac_configuration
    test_secret_management
    test_resource_limits
    test_pod_disruption_budget
    test_image_security
    test_service_mesh
    test_audit_logging
    test_sensitive_data_exposure

    print_summary
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace|-n)
            NAMESPACE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --namespace, -n NAME  Namespace to test (default: paw)"
            echo "  --help, -h            Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
