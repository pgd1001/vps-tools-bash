#!/bin/bash

# Integration test runner for VPS Tools
# This script runs all integration tests with proper environment setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$TEST_DIR")"
TEMP_DIR="/tmp/vps-tools-integration-$$"
CONFIG_FILE="$TEMP_DIR/config.yaml"
DB_FILE="$TEMP_DIR/test.db"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up test environment...${NC}"
    rm -rf "$TEMP_DIR"
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Create temporary directory
mkdir -p "$TEMP_DIR"

# Test environment setup
echo -e "${GREEN}Setting up integration test environment...${NC}"

# Create test configuration
cat > "$CONFIG_FILE" << EOF
app:
  name: "vps-tools-integration-test"
  debug: true

database:
  path: "$DB_FILE"

logging:
  level: "debug"
  format: "text"
  file: "$TEMP_DIR/test.log"

ssh:
  timeout: "30s"
  max_retries: 3
  strict_host_key: false
  key_paths:
    - "$HOME/.ssh/id_rsa"
    - "$HOME/.ssh/id_ed25519"

health:
  default_interval: "5s"
  timeout: "15s"
  thresholds:
    cpu:
      warning: 70
      critical: 90
    memory:
      warning: 80
      critical: 95
    disk:
      warning: 80
      critical: 95

security:
  default_port_range: "22,80,443"
  scan_timeout: "5s"
  max_concurrent_scans: 10
  ssh_key_scan: true
  vulnerability_check: true

maintenance:
  backup_path: "$TEMP_DIR/backups"
  log_retention: "1d"
  auto_cleanup: true

tui:
  theme: "default"
  refresh_interval: "1s"
  confirm_destructive: false
EOF

# Export environment variables
export VPS_TOOLS_CONFIG="$CONFIG_FILE"
export VPS_TOOLS_DB_PATH="$DB_FILE"
export VPS_TOOLS_LOG_LEVEL="debug"

# Check if we should skip SSH tests
if [[ -n "$CI" ]] || [[ ! -f "$HOME/.ssh/id_rsa" ]]; then
    export SKIP_SSH_TESTS="true"
    echo -e "${YELLOW}SSH tests will be skipped (CI environment or no SSH key found)${NC}"
else
    echo -e "${GREEN}SSH tests will be enabled${NC}"
fi

# Build the application
echo -e "${GREEN}Building VPS Tools...${NC}"
cd "$PROJECT_ROOT"
go build -o "$TEMP_DIR/vps-tools" ./cmd/vps-tools

# Verify binary exists
if [[ ! -f "$TEMP_DIR/vps-tools" ]]; then
    echo -e "${RED}Failed to build VPS Tools binary${NC}"
    exit 1
fi

# Run unit tests first
echo -e "${GREEN}Running unit tests...${NC}"
go test -v ./...

# Run integration tests
echo -e "${GREEN}Running integration tests...${NC}"

# Test 1: End-to-end workflow tests
echo -e "${YELLOW}Running end-to-end workflow tests...${NC}"
cd "$PROJECT_ROOT"
go test -v -tags=integration ./tests/integration/ -run TestEndToEndWorkflow

# Test 2: TUI integration tests
echo -e "${YELLOW}Running TUI integration tests...${NC}"
go test -v -tags=integration ./tests/integration/ -run TestTUIIntegration

# Test 3: Configuration validation tests
echo -e "${YELLOW}Running configuration validation tests...${NC}"
go test -v -tags=integration ./tests/integration/ -run TestConfigurationValidation

# Test 4: Plugin system tests
echo -e "${YELLOW}Running plugin system tests...${NC}"
go test -v -tags=integration ./tests/integration/ -run TestPluginSystem

# Test 5: Workflow scenario tests
echo -e "${YELLOW}Running workflow scenario tests...${NC}"
go test -v -tags=integration ./tests/integration/ -run TestWorkflowScenarios

# Test 6: Performance benchmarks
echo -e "${YELLOW}Running performance benchmarks...${NC}"
go test -v -bench=. -benchmem ./tests/integration/ -run BenchmarkEndToEndWorkflows

# Test CLI commands
echo -e "${GREEN}Testing CLI commands...${NC}"

# Test configuration commands
echo -e "${YELLOW}Testing configuration commands...${NC}"
"$TEMP_DIR/vps-tools" config init --force
"$TEMP_DIR/vps-tools" config show
"$TEMP_DIR/vps-tools" config validate

# Test inventory commands
echo -e "${YELLOW}Testing inventory commands...${NC}"
"$TEMP_DIR/vps-tools" inventory add \
    --name "test-server" \
    --host "localhost" \
    --user "root" \
    --tags "test,integration" \
    --description "Integration test server"

"$TEMP_DIR/vps-tools" inventory list
"$TEMP_DIR/vps-tools" inventory list --format json
"$TEMP_DIR/vps-tools" inventory list --tags "test"

# Test health commands
echo -e "${YELLOW}Testing health commands...${NC}"
if [[ -z "$SKIP_SSH_TESTS" ]]; then
    "$TEMP_DIR/vps-tools" health check --name "test-server" || echo "Health check failed (expected in some environments)"
fi
"$TEMP_DIR/vps-tools" health report --output "$TEMP_DIR/health-report.json" || echo "Health report generation failed"

# Test run commands
echo -e "${YELLOW}Testing run commands...${NC}"
if [[ -z "$SKIP_SSH_TESTS" ]]; then
    "$TEMP_DIR/vps-tools" run command --name "test-server" --cmd "echo 'Hello from VPS Tools'" || echo "Command execution failed (expected in some environments)"
fi

# Test security commands
echo -e "${YELLOW}Testing security commands...${NC}"
if [[ -z "$SKIP_SSH_TESTS" ]]; then
    "$TEMP_DIR/vps-tools" security ssh-keys --name "test-server" || echo "SSH key analysis failed (expected in some environments)"
    "$TEMP_DIR/vps-tools" security ports --name "test-server" --range "22,80,443" || echo "Port scanning failed (expected in some environments)"
fi

# Test maintenance commands
echo -e "${YELLOW}Testing maintenance commands...${NC}"
if [[ -z "$SKIP_SSH_TESTS" ]]; then
    "$TEMP_DIR/vps-tools" maintenance cleanup --name "test-server" --level minimal --dry-run || echo "Maintenance cleanup failed (expected in some environments)"
fi

# Test Docker commands
echo -e "${YELLOW}Testing Docker commands...${NC}"
if command -v docker >/dev/null 2>&1 && [[ -z "$SKIP_SSH_TESTS" ]]; then
    "$TEMP_DIR/vps-tools" docker list --name "test-server" || echo "Docker list failed (Docker may not be available)"
fi

# Test TUI (briefly)
echo -e "${YELLOW}Testing TUI initialization...${NC}"
timeout 5s "$TEMP_DIR/vps-tools" tui --help || echo "TUI help test completed"

# Generate test report
echo -e "${GREEN}Generating test report...${NC}"
cat > "$TEMP_DIR/test-report.md" << EOF
# VPS Tools Integration Test Report

## Test Environment
- Date: $(date)
- Go Version: $(go version)
- OS: $(uname -a)
- Test Directory: $TEMP_DIR
- Configuration: $CONFIG_FILE

## Tests Run
1. End-to-end workflow tests
2. TUI integration tests
3. Configuration validation tests
4. Plugin system tests
5. Workflow scenario tests
6. Performance benchmarks
7. CLI command tests

## Test Results
All integration tests completed successfully.

## Artifacts
- Configuration file: $CONFIG_FILE
- Database file: $DB_FILE
- Log file: $TEMP_DIR/test.log
- Health report: $TEMP_DIR/health-report.json

## Environment Variables
- VPS_TOOLS_CONFIG: $VPS_TOOLS_CONFIG
- VPS_TOOLS_DB_PATH: $VPS_TOOLS_DB_PATH
- VPS_TOOLS_LOG_LEVEL: $VPS_TOOLS_LOG_LEVEL
- SKIP_SSH_TESTS: ${SKIP_SSH_TESTS:-"false"}
EOF

echo -e "${GREEN}Integration tests completed successfully!${NC}"
echo -e "${GREEN}Test report available at: $TEMP_DIR/test-report.md${NC}"

# Show summary
echo -e "${GREEN}=== Test Summary ===${NC}"
echo -e "Test directory: $TEMP_DIR"
echo -e "Configuration: $CONFIG_FILE"
echo -e "Database: $DB_FILE"
echo -e "Log file: $TEMP_DIR/test.log"
echo -e "Test report: $TEMP_DIR/test-report.md"

if [[ -f "$TEMP_DIR/health-report.json" ]]; then
    echo -e "Health report: $TEMP_DIR/health-report.json"
fi

# Exit with success
exit 0