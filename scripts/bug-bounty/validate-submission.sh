#!/bin/bash

################################################################################
# PAW Blockchain Bug Bounty Submission Validator
#
# This script validates bug bounty submissions for completeness and quality.
# It checks that all required fields are present and provides a quality score
# to help prioritize triage.
#
# Usage: ./validate-submission.sh <submission-file.md>
#
# Exit Codes:
#   0 - Submission is valid
#   1 - Submission has errors
#   2 - Invalid arguments
################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
REQUIRED_MISSING=0
OPTIONAL_MISSING=0
QUALITY_SCORE=0
MAX_QUALITY_SCORE=100

# Function to print colored output
print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

# Function to check if a section exists and has content
check_required_field() {
    local field_name=$1
    local field_pattern=$2
    local points=${3:-5}

    if grep -q "$field_pattern" "$SUBMISSION_FILE"; then
        # Check if there's actual content after the field marker
        local content=$(sed -n "/$field_pattern/,/^##/p" "$SUBMISSION_FILE" | grep -v "^##" | grep -v "^\s*$" | wc -l)
        if [ "$content" -gt 2 ]; then
            print_success "$field_name: Present"
            QUALITY_SCORE=$((QUALITY_SCORE + points))
            return 0
        else
            print_warning "$field_name: Header present but appears empty"
            REQUIRED_MISSING=$((REQUIRED_MISSING + 1))
            return 1
        fi
    else
        print_error "$field_name: Missing"
        REQUIRED_MISSING=$((REQUIRED_MISSING + 1))
        return 1
    fi
}

# Function to check optional fields
check_optional_field() {
    local field_name=$1
    local field_pattern=$2
    local points=${3:-3}

    if grep -q "$field_pattern" "$SUBMISSION_FILE"; then
        local content=$(sed -n "/$field_pattern/,/^##/p" "$SUBMISSION_FILE" | grep -v "^##" | grep -v "^\s*$" | wc -l)
        if [ "$content" -gt 2 ]; then
            print_success "$field_name: Present (bonus)"
            QUALITY_SCORE=$((QUALITY_SCORE + points))
            return 0
        fi
    fi
    print_warning "$field_name: Not provided"
    OPTIONAL_MISSING=$((OPTIONAL_MISSING + 1))
    return 1
}

# Function to check for proof of concept
check_poc() {
    print_header "Checking Proof of Concept"

    local has_code=false
    local has_steps=false

    # Check for code blocks
    if grep -q '```' "$SUBMISSION_FILE"; then
        print_success "Code blocks found"
        has_code=true
        QUALITY_SCORE=$((QUALITY_SCORE + 10))
    fi

    # Check for reproduction steps (numbered lists)
    if grep -q '^[0-9]\+\.' "$SUBMISSION_FILE"; then
        print_success "Step-by-step instructions found"
        has_steps=true
        QUALITY_SCORE=$((QUALITY_SCORE + 10))
    fi

    if [ "$has_code" = false ] && [ "$has_steps" = false ]; then
        print_error "No clear proof of concept or reproduction steps"
        REQUIRED_MISSING=$((REQUIRED_MISSING + 1))
        return 1
    fi

    return 0
}

# Function to check CVSS scoring
check_cvss() {
    print_header "Checking CVSS Assessment"

    if grep -qi "CVSS" "$SUBMISSION_FILE"; then
        if grep -q "CVSS:3.1/" "$SUBMISSION_FILE"; then
            print_success "CVSS 3.1 vector string found"
            QUALITY_SCORE=$((QUALITY_SCORE + 8))
            return 0
        else
            print_warning "CVSS mentioned but no vector string found"
            QUALITY_SCORE=$((QUALITY_SCORE + 3))
            return 1
        fi
    else
        print_warning "No CVSS assessment provided (optional but recommended)"
        OPTIONAL_MISSING=$((OPTIONAL_MISSING + 1))
        return 1
    fi
}

# Function to estimate submission quality
assess_quality() {
    print_header "Quality Assessment"

    local word_count=$(wc -w < "$SUBMISSION_FILE")
    print_info "Word count: $word_count"

    if [ "$word_count" -lt 200 ]; then
        print_warning "Submission appears very brief (< 200 words)"
    elif [ "$word_count" -lt 500 ]; then
        print_info "Submission is moderate length (200-500 words)"
        QUALITY_SCORE=$((QUALITY_SCORE + 3))
    else
        print_success "Submission is detailed (> 500 words)"
        QUALITY_SCORE=$((QUALITY_SCORE + 5))
    fi

    # Check for external links (references, PoC repos, etc.)
    local link_count=$(grep -o 'https\?://[^ ]*' "$SUBMISSION_FILE" | wc -l)
    if [ "$link_count" -gt 0 ]; then
        print_info "External links found: $link_count"
        QUALITY_SCORE=$((QUALITY_SCORE + 2))
    fi

    # Check for images/screenshots
    if grep -q '!\[.*\]' "$SUBMISSION_FILE"; then
        print_success "Images/screenshots included"
        QUALITY_SCORE=$((QUALITY_SCORE + 3))
    fi
}

# Function to check for red flags
check_red_flags() {
    print_header "Checking for Red Flags"

    local red_flags=0

    # Check for mainnet testing admission
    if grep -qi "mainnet" "$SUBMISSION_FILE" && grep -qi "tested" "$SUBMISSION_FILE"; then
        print_error "WARNING: Report may mention testing on mainnet"
        red_flags=$((red_flags + 1))
    fi

    # Check for data exfiltration mention
    if grep -qi "exfiltrat\|download.*data\|access.*database" "$SUBMISSION_FILE"; then
        print_warning "WARNING: Report mentions data access/exfiltration"
        red_flags=$((red_flags + 1))
    fi

    # Check for extortion language
    if grep -qi "must pay\|demand\|unless you\|or else" "$SUBMISSION_FILE"; then
        print_error "WARNING: Report may contain extortion language"
        red_flags=$((red_flags + 1))
    fi

    # Check for public disclosure
    if grep -qi "published\|disclosed.*public\|blog post\|twitter" "$SUBMISSION_FILE"; then
        print_warning "WARNING: Report mentions public disclosure"
        red_flags=$((red_flags + 1))
    fi

    if [ "$red_flags" -eq 0 ]; then
        print_success "No red flags detected"
    else
        print_warning "Detected $red_flags potential red flag(s) - manual review required"
    fi

    return $red_flags
}

# Main validation function
validate_submission() {
    print_header "PAW Bug Bounty Submission Validator"
    print_info "Validating: $SUBMISSION_FILE"

    # Check required fields
    print_header "Required Fields"
    check_required_field "Report Title" "Report Title.*:" 5
    check_required_field "Researcher Contact" "Contact Email.*:" 5
    check_required_field "Severity Assessment" "Your Severity Assessment.*:" 5
    check_required_field "Executive Summary" "## Executive Summary" 10
    check_required_field "Vulnerability Type" "### 1\. Vulnerability Type" 5
    check_required_field "Affected Components" "### 2\. Affected Components" 8
    check_required_field "Root Cause Analysis" "### 4\. Root Cause Analysis" 10
    check_required_field "Technical Description" "### 5\. Technical Description" 10
    check_required_field "Reproduction Steps" "### 8\. Step-by-Step Reproduction" 10
    check_required_field "Impact Description" "### 11\. Impact Description" 8

    # Check PoC
    check_poc

    # Check optional but valuable fields
    print_header "Optional Fields (Improve Reward Potential)"
    check_optional_field "CVSS Assessment" "### 15\. CVSS 3.1 Assessment" 8
    check_optional_field "Proposed Fix" "### 18\. Proposed Fix" 5
    check_optional_field "Attack Scenario" "### 10\. Attack Scenario" 5
    check_optional_field "Mitigation Suggestions" "### 19\. Mitigation/Workarounds" 3

    # Additional CVSS check
    check_cvss

    # Quality assessment
    assess_quality

    # Check for red flags
    check_red_flags
    RED_FLAGS=$?

    # Final summary
    print_header "Validation Summary"
    echo -e "${BLUE}Quality Score:${NC} $QUALITY_SCORE / $MAX_QUALITY_SCORE"
    echo -e "${BLUE}Required Fields Missing:${NC} $REQUIRED_MISSING"
    echo -e "${BLUE}Optional Fields Missing:${NC} $OPTIONAL_MISSING"
    echo -e "${BLUE}Red Flags:${NC} $RED_FLAGS"

    # Determine quality tier
    if [ $QUALITY_SCORE -ge 80 ]; then
        echo -e "\n${GREEN}Quality Tier: EXCELLENT${NC}"
        echo "This submission is highly detailed and complete. Eligible for bonus rewards."
    elif [ $QUALITY_SCORE -ge 60 ]; then
        echo -e "\n${GREEN}Quality Tier: GOOD${NC}"
        echo "This submission is well-prepared and complete."
    elif [ $QUALITY_SCORE -ge 40 ]; then
        echo -e "\n${YELLOW}Quality Tier: ADEQUATE${NC}"
        echo "This submission meets minimum requirements but could be improved."
    else
        echo -e "\n${YELLOW}Quality Tier: NEEDS IMPROVEMENT${NC}"
        echo "This submission is incomplete or lacks detail."
    fi

    # Recommendations
    if [ $REQUIRED_MISSING -gt 0 ] || [ $QUALITY_SCORE -lt 60 ]; then
        print_header "Recommendations for Improvement"

        if [ $REQUIRED_MISSING -gt 0 ]; then
            echo "- Fill in all required fields"
        fi

        if ! grep -q "CVSS:3.1/" "$SUBMISSION_FILE"; then
            echo "- Add CVSS 3.1 scoring (use https://www.first.org/cvss/calculator/3.1)"
        fi

        if ! grep -q "### 18\. Proposed Fix" "$SUBMISSION_FILE"; then
            echo "- Consider adding remediation suggestions"
        fi

        if ! grep -q '!\[.*\]' "$SUBMISSION_FILE"; then
            echo "- Consider adding screenshots or diagrams"
        fi

        local word_count=$(wc -w < "$SUBMISSION_FILE")
        if [ "$word_count" -lt 500 ]; then
            echo "- Provide more technical detail and context"
        fi
    fi

    # Exit code
    echo ""
    if [ $REQUIRED_MISSING -gt 0 ]; then
        print_error "Validation FAILED: Missing required fields"
        return 1
    elif [ $RED_FLAGS -gt 2 ]; then
        print_warning "Validation PASSED with WARNINGS: Multiple red flags detected"
        return 0
    else
        print_success "Validation PASSED: Submission is complete"
        return 0
    fi
}

# Script entry point
main() {
    # Check arguments
    if [ $# -ne 1 ]; then
        echo "Usage: $0 <submission-file.md>"
        echo ""
        echo "Example:"
        echo "  $0 vulnerability-report.md"
        exit 2
    fi

    SUBMISSION_FILE="$1"

    # Check if file exists
    if [ ! -f "$SUBMISSION_FILE" ]; then
        print_error "File not found: $SUBMISSION_FILE"
        exit 2
    fi

    # Check if file is markdown
    if [[ ! "$SUBMISSION_FILE" =~ \.md$ ]]; then
        print_warning "File does not have .md extension. Proceeding anyway..."
    fi

    # Run validation
    validate_submission
    exit $?
}

# Run main function
main "$@"
