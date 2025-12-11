#!/bin/bash

# Build script for VPS Tools
# This script builds the application for multiple platforms and creates release artifacts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="vps-tools"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
BUILD_TIME="$(date -u '+%Y-%m-%d_%H:%M:%S')"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
LDFLAGS="-ldflags=-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Platforms to build
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Output directory
DIST_DIR="dist"
RELEASE_DIR="release"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    rm -rf "${DIST_DIR}"
}

# Set trap for cleanup
trap cleanup EXIT

# Create directories
mkdir -p "${DIST_DIR}"
mkdir -p "${RELEASE_DIR}"

echo -e "${BLUE}Building ${APP_NAME} version ${VERSION}${NC}"
echo -e "${BLUE}Git commit: ${GIT_COMMIT}${NC}"
echo -e "${BLUE}Build time: ${BUILD_TIME}${NC}"
echo ""

# Function to build for a specific platform
build_platform() {
    local platform=$1
    local goos=${platform%/*}
    local goarch=${platform#*/}
    local output_name="${APP_NAME}-${goos}-${goarch}"
    
    if [ "${goos}" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo -e "${GREEN}Building for ${platform}...${NC}"
    
    # Set environment variables
    export GOOS=${goos}
    export GOARCH=${goarch}
    export CGO_ENABLED=0
    
    # Build
    go build ${LDFLAGS} -o "${DIST_DIR}/${output_name}" ./cmd/vps-tools
    
    # Create archive
    cd "${DIST_DIR}"
    if [ "${goos}" = "windows" ]; then
        zip -q "${RELEASE_DIR}/${output_name}.zip" "${output_name}"
    else
        tar -czf "${RELEASE_DIR}/${output_name}.tar.gz" "${output_name}"
    fi
    cd ..
    
    echo -e "${GREEN}✓ Built ${platform}${NC}"
}

# Function to build Docker image
build_docker() {
    echo -e "${GREEN}Building Docker image...${NC}"
    
    # Build for multiple platforms
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --tag "${APP_NAME}:${VERSION}" \
        --tag "${APP_NAME}:latest" \
        --push \
        . || {
        echo -e "${YELLOW}Docker buildx push failed, building local image only${NC}"
        docker build -t "${APP_NAME}:${VERSION}" -t "${APP_NAME}:latest" .
    }
    
    echo -e "${GREEN}✓ Docker image built${NC}"
}

# Function to generate checksums
generate_checksums() {
    echo -e "${GREEN}Generating checksums...${NC}"
    
    cd "${RELEASE_DIR}"
    sha256sum * > checksums.txt
    cd ..
    
    echo -e "${GREEN}✓ Checksums generated${NC}"
}

# Function to run tests
run_tests() {
    echo -e "${GREEN}Running tests...${NC}"
    
    # Unit tests
    go test -v -race -coverprofile=coverage.out ./...
    
    # Integration tests (if not in CI)
    if [ -z "${CI}" ]; then
        echo -e "${YELLOW}Running integration tests...${NC}"
        ./scripts/run-integration-tests.sh
    fi
    
    echo -e "${GREEN}✓ Tests completed${NC}"
}

# Function to run linting
run_lint() {
    echo -e "${GREEN}Running linters...${NC}"
    
    # go fmt
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        echo -e "${RED}Error: The following files are not formatted:${NC}"
        gofmt -s -l .
        exit 1
    fi
    
    # go vet
    go vet ./...
    
    # golangci-lint (if available)
    if command -v golangci-lint >/dev/null 2>&1; then
        golangci-lint run
    else
        echo -e "${YELLOW}golangci-lint not found, skipping${NC}"
    fi
    
    echo -e "${GREEN}✓ Linting completed${NC}"
}

# Function to create release notes
create_release_notes() {
    echo -e "${GREEN}Creating release notes...${NC}"
    
    cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# ${APP_NAME} ${VERSION}

## Build Information
- **Version**: ${VERSION}
- **Git Commit**: ${GIT_COMMIT}
- **Build Time**: ${BUILD_TIME}
- **Go Version**: $(go version)

## Installation

### Binary Installation
1. Download the appropriate binary for your platform:
   - Linux (AMD64): ${APP_NAME}-linux-amd64.tar.gz
   - Linux (ARM64): ${APP_NAME}-linux-arm64.tar.gz
   - macOS (Intel): ${APP_NAME}-darwin-amd64.tar.gz
   - macOS (Apple Silicon): ${APP_NAME}-darwin-arm64.tar.gz
   - Windows (AMD64): ${APP_NAME}-windows-amd64.zip

2. Extract the archive:
   \`\`\`bash
   # Linux/macOS
   tar -xzf ${APP_NAME}-<platform>.tar.gz
   
   # Windows
   unzip ${APP_NAME}-windows-amd64.zip
   \`\`\`

3. Move the binary to your PATH:
   \`\`\`bash
   sudo mv ${APP_NAME} /usr/local/bin/
   \`\`\`

### Docker Installation
\`\`\`bash
docker pull ${APP_NAME}:${VERSION}
docker run -it --rm ${APP_NAME}:${VERSION} --help
\`\`\`

### From Source
\`\`\`bash
git clone https://github.com/pgd1001/${APP_NAME}.git
cd ${APP_NAME}
make build
\`\`\`

## Verification
Verify the downloaded binaries using the provided checksums:
\`\`\`bash
sha256sum -c checksums.txt
\`\`\`

## Changelog
$(git log --oneline --pretty="- %s" $(git describe --tags --abbrev=0 2>/dev/null || echo "")..HEAD 2>/dev/null || echo "- Initial release")

## Support
- **Documentation**: https://github.com/pgd1001/${APP_NAME}/docs
- **Issues**: https://github.com/pgd1001/${APP_NAME}/issues
- **Discussions**: https://github.com/pgd1001/${APP_NAME}/discussions
EOF

    echo -e "${GREEN}✓ Release notes created${NC}"
}

# Main build process
main() {
    echo -e "${BLUE}Starting build process...${NC}"
    
    # Check dependencies
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    
    # Clean previous builds
    rm -rf "${DIST_DIR}" "${RELEASE_DIR}"
    mkdir -p "${DIST_DIR}" "${RELEASE_DIR}"
    
    # Run tests (unless disabled)
    if [ "${SKIP_TESTS}" != "true" ]; then
        run_tests
    fi
    
    # Run linting (unless disabled)
    if [ "${SKIP_LINT}" != "true" ]; then
        run_lint
    fi
    
    # Build for all platforms
    for platform in "${PLATFORMS[@]}"; do
        build_platform "${platform}"
    done
    
    # Generate checksums
    generate_checksums
    
    # Create release notes
    create_release_notes
    
    # Build Docker image (if requested)
    if [ "${BUILD_DOCKER}" = "true" ]; then
        build_docker
    fi
    
    echo ""
    echo -e "${GREEN}✅ Build completed successfully!${NC}"
    echo -e "${BLUE}Release artifacts created in: ${RELEASE_DIR}${NC}"
    echo ""
    echo -e "${BLUE}Built files:${NC}"
    ls -la "${RELEASE_DIR}"
    
    # Show sizes
    echo ""
    echo -e "${BLUE}File sizes:${NC}"
    cd "${RELEASE_DIR}"
    for file in *; do
        if [ -f "$file" ]; then
            size=$(du -h "$file" | cut -f1)
            echo -e "${GREEN}  ${file}: ${size}${NC}"
        fi
    done
    cd ..
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --skip-lint)
            SKIP_LINT=true
            shift
            ;;
        --build-docker)
            BUILD_DOCKER=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --version VERSION    Set version (default: git describe)"
            echo "  --skip-tests        Skip running tests"
            echo "  --skip-lint         Skip running linters"
            echo "  --build-docker      Build Docker image"
            echo "  --help              Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main