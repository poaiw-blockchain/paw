#!/bin/bash
# Helm Chart Validation Script for PAW Blockchain
# This script validates the Helm chart structure and generates test manifests

set -e

CHART_DIR="$(dirname "$0")/paw"

echo "=== PAW Helm Chart Validation ==="
echo ""

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo "WARNING: Helm is not installed. Skipping helm lint."
    echo "Install Helm from: https://helm.sh/docs/intro/install/"
    echo ""
else
    echo "1. Running helm lint..."
    helm lint "$CHART_DIR"
    echo "   ✓ Lint passed"
    echo ""
fi

# Validate chart structure
echo "2. Validating chart structure..."

required_files=(
    "Chart.yaml"
    "values.yaml"
    "templates/_helpers.tpl"
    "templates/deployment.yaml"
    "templates/service.yaml"
    "templates/pvc.yaml"
    "templates/serviceaccount.yaml"
)

for file in "${required_files[@]}"; do
    if [ -f "$CHART_DIR/$file" ]; then
        echo "   ✓ $file exists"
    else
        echo "   ✗ $file is missing"
        exit 1
    fi
done
echo ""

# Validate YAML syntax
echo "3. Validating YAML syntax..."
if command -v yamllint &> /dev/null; then
    yamllint -d relaxed "$CHART_DIR"
    echo "   ✓ YAML syntax is valid"
elif command -v python3 &> /dev/null; then
    python3 -c "import yaml, sys, glob; [yaml.safe_load(open(f)) for f in glob.glob('$CHART_DIR/**/*.yaml', recursive=True) if not f.endswith('values.yaml')]"
    echo "   ✓ YAML syntax is valid (basic check)"
else
    echo "   ⚠ Skipping YAML validation (yamllint/python3 not found)"
fi
echo ""

# Generate test manifests
if command -v helm &> /dev/null; then
    echo "4. Generating test manifests..."

    mkdir -p "$CHART_DIR/test-output"

    # Test with default values
    echo "   - Generating with default values..."
    helm template test-release "$CHART_DIR" > "$CHART_DIR/test-output/default-manifest.yaml"

    # Test with production values
    if [ -f "$CHART_DIR/values-production.yaml" ]; then
        echo "   - Generating with production values..."
        helm template test-release "$CHART_DIR" -f "$CHART_DIR/values-production.yaml" > "$CHART_DIR/test-output/production-manifest.yaml"
    fi

    # Test with dev values
    if [ -f "$CHART_DIR/values-dev.yaml" ]; then
        echo "   - Generating with dev values..."
        helm template test-release "$CHART_DIR" -f "$CHART_DIR/values-dev.yaml" > "$CHART_DIR/test-output/dev-manifest.yaml"
    fi

    echo "   ✓ Test manifests generated in $CHART_DIR/test-output/"
    echo ""
fi

# Validate generated manifests with kubectl
if command -v kubectl &> /dev/null && command -v helm &> /dev/null; then
    echo "5. Validating manifests with kubectl..."

    for manifest in "$CHART_DIR"/test-output/*.yaml; do
        if [ -f "$manifest" ]; then
            echo "   - Validating $(basename "$manifest")..."
            kubectl apply --dry-run=client -f "$manifest" > /dev/null 2>&1
            if [ $? -eq 0 ]; then
                echo "     ✓ Valid Kubernetes manifest"
            else
                echo "     ✗ Invalid Kubernetes manifest"
                exit 1
            fi
        fi
    done
    echo ""
fi

echo "=== Validation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Install the chart: helm install my-paw-node ./helm/paw"
echo "  2. Check status: helm status my-paw-node"
echo "  3. View manifests: helm get manifest my-paw-node"
echo ""
echo "For production deployment:"
echo "  helm install my-paw-node ./helm/paw -f ./helm/paw/values-production.yaml"
