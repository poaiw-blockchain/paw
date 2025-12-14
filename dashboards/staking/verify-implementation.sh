#!/bin/bash

echo "=========================================="
echo "PAW Staking Dashboard - Implementation Verification"
echo "=========================================="
echo ""

# Check for required files
echo "📁 Checking file structure..."
FILES=(
    "index.html"
    "app.js"
    "package.json"
    "README.md"
    "styles/main.css"
    "components/ValidatorList.js"
    "components/ValidatorComparison.js"
    "components/StakingCalculator.js"
    "components/DelegationPanel.js"
    "components/RewardsPanel.js"
    "components/PortfolioView.js"
    "services/stakingAPI.js"
    "utils/ui.js"
    "tests/stakingAPI.test.js"
    "tests/calculator.test.js"
    "tests/e2e.test.js"
    "tests/run-tests.js"
)

missing=0
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ MISSING: $file"
        ((missing++))
    fi
done

echo ""
echo "📊 Statistics:"
echo "  Total required files: ${#FILES[@]}"
echo "  Missing files: $missing"
echo ""

# Count lines of code
echo "📝 Lines of Code:"
find . -name "*.js" -o -name "*.html" -o -name "*.css" | grep -v node_modules | xargs wc -l | tail -1
echo ""

# Run tests
echo "🧪 Running test suite..."
node tests/run-tests.js

echo ""
echo "=========================================="
echo "✅ Verification Complete"
echo "=========================================="
