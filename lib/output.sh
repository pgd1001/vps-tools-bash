#!/bin/bash
# VPS Tools - Output Library
# Provides JSON and text output formatting functions
# All scripts source this for log_* functions instead of defining their own.
# Source this in scripts: source "${TOOLS_DIR:-/opt/vps-tools}/lib/output.sh"

set -euo pipefail

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
declare -A JSON_DATA 2>/dev/null || true
JSON_CHECKS=""

# Escape a string for safe JSON embedding
_json_escape() {
    local input="$1"
    # Escape backslash, double quote, and control characters
    printf '%s' "$input" | sed -e 's/\\/\\\\/g' \
                                -e 's/"/\\"/g' \
                                -e 's/\t/\\t/g' \
                                -e ':a;N;$!ba;s/\n/\\n/g' \
                                -e 's/\r/\\r/g'
}

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

# Add check result to JSON (with proper escaping)
json_add_check() {
    local name="$1"
    local status="$2"
    local value="$3"
    local message="${4:-}"

    local safe_name safe_status safe_message
    safe_name=$(_json_escape "$name")
    safe_status=$(_json_escape "$status")
    safe_message=$(_json_escape "$message")

    [[ -n "$JSON_CHECKS" ]] && JSON_CHECKS+=","
    JSON_CHECKS+="\"${safe_name}\":{\"status\":\"${safe_status}\",\"value\":${value}"
    [[ -n "$safe_message" ]] && JSON_CHECKS+=",\"message\":\"${safe_message}\""
    JSON_CHECKS+="}"
}

# Output final JSON
output_json() {
    local status="${1:-ok}"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo "{"
    echo "  \"timestamp\": \"$timestamp\","
    echo "  \"status\": \"$(_json_escape "$status")\","

    # Add custom fields
    for key in "${!JSON_DATA[@]}"; do
        echo "  \"$(_json_escape "$key")\": ${JSON_DATA[$key]},"
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

# Log to file AND console (for scripts that need both)
log_info_file() {
    local log_file="${VPS_LOG_DIR:-/var/log/vps-tools}/${SCRIPT_LOG_NAME:-vps-tools}.log"
    mkdir -p "$(dirname "$log_file")"
    [[ "$OUTPUT_FORMAT" != "json" ]] && echo -e "${BLUE}[INFO]${NC} $*"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $*" >> "$log_file"
}

log_success_file() {
    local log_file="${VPS_LOG_DIR:-/var/log/vps-tools}/${SCRIPT_LOG_NAME:-vps-tools}.log"
    mkdir -p "$(dirname "$log_file")"
    [[ "$OUTPUT_FORMAT" != "json" ]] && echo -e "${GREEN}[✓]${NC} $*"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [OK] $*" >> "$log_file"
}

log_warning_file() {
    local log_file="${VPS_LOG_DIR:-/var/log/vps-tools}/${SCRIPT_LOG_NAME:-vps-tools}.log"
    mkdir -p "$(dirname "$log_file")"
    [[ "$OUTPUT_FORMAT" != "json" ]] && echo -e "${YELLOW}[⚠]${NC} $*"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $*" >> "$log_file"
}

log_error_file() {
    local log_file="${VPS_LOG_DIR:-/var/log/vps-tools}/${SCRIPT_LOG_NAME:-vps-tools}.log"
    mkdir -p "$(dirname "$log_file")"
    [[ "$OUTPUT_FORMAT" != "json" ]] && echo -e "${RED}[✗]${NC} $*"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $*" >> "$log_file"
}
