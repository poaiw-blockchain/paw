#!/bin/bash
# PAW Blockchain Release Script
# Reproducible builds with checksums and optional GPG signatures
#
# Usage:
#   ./scripts/release.sh                    # Build snapshot release
#   ./scripts/release.sh --tag v1.0.0       # Build tagged release
#   ./scripts/release.sh --sign             # Build with GPG signing
#   ./scripts/release.sh --docker           # Build with Docker images
#   ./scripts/release.sh --dry-run          # Preview without building
#   ./scripts/release.sh --clean            # Clean build artifacts

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/dist"

# Default values
SNAPSHOT=true
SIGN=false
DOCKER=false
DRY_RUN=false
CLEAN=false
TAG=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --tag)
            TAG="$2"
            SNAPSHOT=false
            shift 2
            ;;
        --sign)
            SIGN=true
            shift
            ;;
        --docker)
            DOCKER=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            echo "PAW Release Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --tag VERSION    Create a tagged release (e.g., v1.0.0)"
            echo "  --sign           Sign artifacts with GPG"
            echo "  --docker         Build Docker images"
            echo "  --dry-run        Preview without building"
            echo "  --clean          Clean build artifacts"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Build snapshot release"
            echo "  $0 --tag v1.0.0       # Build tagged release"
            echo "  $0 --sign --docker    # Build with signing and Docker"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Clean function
clean_artifacts() {
    echo -e "${BLUE}Cleaning build artifacts...${NC}"
    rm -rf "$BUILD_DIR"
    rm -f "$PROJECT_DIR"/*.tar.gz
    rm -f "$PROJECT_DIR"/*.zip
    rm -f "$PROJECT_DIR"/*_checksums.txt
    rm -f "$PROJECT_DIR"/*.sig
    echo -e "${GREEN}Clean complete${NC}"
}

# Clean and exit if requested
if [[ "$CLEAN" == true ]]; then
    clean_artifacts
    exit 0
fi

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"

    # Check Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "  Go: ${GREEN}$GO_VERSION${NC}"

    # Check goreleaser
    if ! command -v goreleaser &> /dev/null; then
        echo -e "${YELLOW}Warning: goreleaser not found, installing...${NC}"
        go install github.com/goreleaser/goreleaser@latest
    fi
    GORELEASER_VERSION=$(goreleaser --version 2>&1 | head -1)
    echo -e "  GoReleaser: ${GREEN}$GORELEASER_VERSION${NC}"

    # Check GPG if signing
    if [[ "$SIGN" == true ]]; then
        if ! command -v gpg &> /dev/null; then
            echo -e "${RED}Error: GPG is not installed (required for signing)${NC}"
            exit 1
        fi
        if [[ -z "${GPG_FINGERPRINT:-}" ]]; then
            echo -e "${YELLOW}Warning: GPG_FINGERPRINT not set${NC}"
            echo "Set it with: export GPG_FINGERPRINT=<your-fingerprint>"
            # Try to get default key
            DEFAULT_KEY=$(gpg --list-secret-keys --keyid-format LONG 2>/dev/null | grep -A1 "^sec" | tail -1 | awk '{print $1}' || true)
            if [[ -n "$DEFAULT_KEY" ]]; then
                echo -e "Using default key: ${BLUE}$DEFAULT_KEY${NC}"
                export GPG_FINGERPRINT="$DEFAULT_KEY"
            else
                echo -e "${RED}Error: No GPG key found${NC}"
                exit 1
            fi
        fi
        echo -e "  GPG Key: ${GREEN}$GPG_FINGERPRINT${NC}"
    fi

    # Check Docker if building images
    if [[ "$DOCKER" == true ]]; then
        if ! command -v docker &> /dev/null; then
            echo -e "${RED}Error: Docker is not installed${NC}"
            exit 1
        fi
        echo -e "  Docker: ${GREEN}$(docker --version | awk '{print $3}')${NC}"
    fi

    echo -e "${GREEN}Prerequisites OK${NC}"
}

# Get version info
get_version_info() {
    cd "$PROJECT_DIR"

    if [[ -n "$TAG" ]]; then
        VERSION="${TAG#v}"
    else
        VERSION=$(git describe --tags 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
    fi

    COMMIT=$(git log -1 --format='%H' 2>/dev/null || echo "unknown")
    SHORT_COMMIT=$(git log -1 --format='%h' 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo -e "${BLUE}Version Info:${NC}"
    echo -e "  Version: ${GREEN}$VERSION${NC}"
    echo -e "  Commit:  ${GREEN}$SHORT_COMMIT${NC}"
    echo -e "  Date:    ${GREEN}$BUILD_DATE${NC}"
}

# Build release
build_release() {
    cd "$PROJECT_DIR"

    echo -e "${BLUE}Building release...${NC}"

    # Prepare goreleaser arguments
    GORELEASER_ARGS=()

    if [[ "$SNAPSHOT" == true ]]; then
        GORELEASER_ARGS+=("--snapshot")
    fi

    if [[ "$SIGN" != true ]]; then
        GORELEASER_ARGS+=("--skip=sign")
    fi

    if [[ "$DOCKER" != true ]]; then
        GORELEASER_ARGS+=("--skip=docker")
    fi

    GORELEASER_ARGS+=("--clean")

    if [[ "$DRY_RUN" == true ]]; then
        echo -e "${YELLOW}Dry run mode - checking configuration...${NC}"
        goreleaser check
        echo -e "${GREEN}Configuration is valid${NC}"
        return
    fi

    # Run goreleaser
    echo -e "Running: goreleaser release ${GORELEASER_ARGS[*]}"
    goreleaser release "${GORELEASER_ARGS[@]}"

    echo -e "${GREEN}Build complete!${NC}"
}

# Generate checksums for manual builds
generate_checksums() {
    if [[ ! -d "$BUILD_DIR" ]]; then
        echo -e "${YELLOW}No dist directory found, skipping checksums${NC}"
        return
    fi

    cd "$BUILD_DIR"

    echo -e "${BLUE}Generating checksums...${NC}"

    # Generate SHA256 checksums
    CHECKSUM_FILE="paw_${VERSION}_checksums.txt"
    sha256sum *.tar.gz *.zip 2>/dev/null > "$CHECKSUM_FILE" || true

    if [[ -f "$CHECKSUM_FILE" ]] && [[ -s "$CHECKSUM_FILE" ]]; then
        echo -e "  Checksums: ${GREEN}$CHECKSUM_FILE${NC}"
        cat "$CHECKSUM_FILE"
    fi
}

# Sign checksums
sign_checksums() {
    if [[ "$SIGN" != true ]]; then
        return
    fi

    cd "$BUILD_DIR"

    CHECKSUM_FILE="paw_${VERSION}_checksums.txt"

    if [[ ! -f "$CHECKSUM_FILE" ]]; then
        echo -e "${YELLOW}No checksum file found, skipping signing${NC}"
        return
    fi

    echo -e "${BLUE}Signing checksums...${NC}"

    gpg --batch --local-user "$GPG_FINGERPRINT" --detach-sign --armor "$CHECKSUM_FILE"

    if [[ -f "${CHECKSUM_FILE}.asc" ]]; then
        echo -e "  Signature: ${GREEN}${CHECKSUM_FILE}.asc${NC}"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Release Build Complete${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "Version: ${BLUE}$VERSION${NC}"
    echo -e "Commit:  ${BLUE}$SHORT_COMMIT${NC}"
    echo ""

    if [[ -d "$BUILD_DIR" ]]; then
        echo -e "Artifacts in ${BLUE}$BUILD_DIR${NC}:"
        ls -lh "$BUILD_DIR"/*.tar.gz "$BUILD_DIR"/*.zip 2>/dev/null || true
        echo ""

        if [[ -f "$BUILD_DIR/paw_${VERSION}_checksums.txt" ]]; then
            echo -e "Checksums:"
            cat "$BUILD_DIR/paw_${VERSION}_checksums.txt"
        fi
    fi

    echo ""
    echo -e "${GREEN}Next steps:${NC}"
    if [[ "$SNAPSHOT" == true ]]; then
        echo "  1. Test the binaries locally"
        echo "  2. Create a git tag: git tag -a v$VERSION -m 'Release v$VERSION'"
        echo "  3. Push the tag: git push origin v$VERSION"
        echo "  4. Run release with --tag: ./scripts/release.sh --tag v$VERSION"
    else
        echo "  1. Verify checksums and signatures"
        echo "  2. Upload artifacts to GitHub release"
        echo "  3. Announce the release"
    fi
}

# Main
main() {
    echo ""
    echo -e "${BLUE}PAW Blockchain Release Builder${NC}"
    echo -e "${BLUE}===============================${NC}"
    echo ""

    check_prerequisites
    echo ""

    get_version_info
    echo ""

    build_release

    if [[ "$DRY_RUN" != true ]]; then
        generate_checksums
        sign_checksums
        print_summary
    fi
}

main
