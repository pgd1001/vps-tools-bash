#!/bin/bash
# VPS Tools - Output Library
# Provides JSON and text output formatting functions
# Source this in scripts: source "$TOOLS_DIR/lib/output.sh"

# Output format (text or json)
OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# JSON data accumulator
declare -A JSON_DATA
JSON_CHECKS=""

# Initialize output
init_output() {
    OUTPUT_FORMAT="${1:-text}"
    JSON_DATA=()
    JSON_CHECKS=""
}

# Parse --output argument
parse_output_arg() {
    for arg in "$@"; do
        case $arg in
            --output=*) OUTPUT_FORMAT="${arg#*=}" ;;
        esac
    done
}

# Set JSON field
json_set() {
    local key="$1"
    local value="$2"
    JSON_DATA[$key]="$value"
}

# Add check result to JSON
json_add_check() {
    local name="$1"
    local status="$2"
    local value="$3"
    local message="${4:-}"
    
    [[ -n "$JSON_CHECKS" ]] && JSON_CHECKS+=","
    JSON_CHECKS+="\"$name\":{\"status\":\"$status\",\"value\":$value"
    [[ -n "$message" ]] && JSON_CHECKS+=",\"message\":\"$message\""
    JSON_CHECKS+="}"
}

# Output final JSON
output_json() {
    local status="${1:-ok}"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    echo "{"
    echo "  \"timestamp\": \"$timestamp\","
    echo "  \"status\": \"$status\","
    
    # Add custom fields
    for key in "${!JSON_DATA[@]}"; do
        echo "  \"$key\": ${JSON_DATA[$key]},"
    done
    
    # Add checks
    if [[ -n "$JSON_CHECKS" ]]; then
        echo "  \"checks\": {$JSON_CHECKS}"
    else
        echo "  \"checks\": {}"
    fi
    
    echo "}"
}

# Text output functions (only output in text mode)
log_info() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${GREEN}[✓]${NC} $*"
}

log_warning() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${YELLOW}[⚠]${NC} $*"
}

log_error() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${RED}[✗]${NC} $*"
}

log_critical() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${RED}[✗]${NC} $*"
}

log_header() {
    [[ "$OUTPUT_FORMAT" == "json" ]] && return
    echo -e "${CYAN}=== $* ===${NC}"
}

# Dual output - records for JSON, prints for text
check_result() {
    local name="$1"
    local status="$2"      # ok, warning, critical
    local value="$3"       # numeric value
    local message="$4"     # human message
    
    # Record for JSON
    json_add_check "$name" "$status" "$value" "$message"
    
    # Print for text
    if [[ "$OUTPUT_FORMAT" == "text" ]]; then
        case "$status" in
            ok)       echo -e "${GREEN}[✓]${NC} $message" ;;
            warning)  echo -e "${YELLOW}[⚠]${NC} $message" ;;
            critical) echo -e "${RED}[✗]${NC} $message" ;;
        esac
    fi
}

# Finalize and output
finalize_output() {
    local overall_status="${1:-ok}"
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        output_json "$overall_status"
    fi
}
