#!/bin/bash
# VPS Tools - Common Library
# Shared functions and utilities for all scripts
# Source this in scripts: source "${TOOLS_DIR:-/opt/vps-tools}/lib/common.sh"

set -euo pipefail

# Version
readonly VPS_TOOLS_VERSION="2.0.0"

# Directories
readonly VPS_TOOLS_DIR="${VPS_TOOLS_DIR:-/opt/vps-tools}"
readonly VPS_CONFIG_DIR="${VPS_CONFIG_DIR:-/etc/vps-tools}"
readonly VPS_LOG_DIR="${VPS_LOG_DIR:-/var/log/vps-tools}"

# Configuration
readonly VPS_CONFIG_FILE="$VPS_CONFIG_DIR/config.conf"
readonly VPS_PLUGINS_FILE="$VPS_CONFIG_DIR/plugins.conf"

# Load configuration safely (validates file ownership and permissions)
load_config() {
    if [[ -f "$VPS_CONFIG_FILE" ]]; then
        # Only source config owned by root with no world-write
        if [[ -O "$VPS_CONFIG_FILE" ]] || [[ $EUID -ne 0 ]]; then
            source "$VPS_CONFIG_FILE"
        else
            local owner
            owner=$(stat -c '%U' "$VPS_CONFIG_FILE" 2>/dev/null || echo "unknown")
            if [[ "$owner" == "root" ]]; then
                source "$VPS_CONFIG_FILE"
            else
                echo "[WARNING] Config file not owned by root ($owner), skipping" >&2
            fi
        fi
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" &>/dev/null
}

# Get system info
get_hostname() {
    hostname
}

get_os_version() {
    if [[ -f /etc/os-release ]]; then
        # shellcheck disable=SC1091
        source /etc/os-release
        echo "$PRETTY_NAME"
    else
        uname -a
    fi
}

get_uptime() {
    uptime -p 2>/dev/null || uptime
}

# Disk usage percentage for a path
get_disk_usage() {
    local path="${1:-/}"
    df "$path" 2>/dev/null | awk 'NR==2 {print int($5)}'
}

# Memory usage percentage
get_memory_usage() {
    free -b 2>/dev/null | awk '/Mem:/ {printf "%.0f", $3/$2*100}'
}

# CPU load (1 minute average as percentage of cores)
get_cpu_load() {
    local cores
    cores=$(nproc 2>/dev/null || echo 1)
    local load
    load=$(awk '{print $1}' /proc/loadavg 2>/dev/null || echo "0")
    echo "$load $cores" | awk '{printf "%.0f", ($1/$2)*100}'
}

# Check if service is running
is_service_running() {
    local service="$1"
    systemctl is-active --quiet "$service" 2>/dev/null
}

# Check if Docker is available
is_docker_available() {
    command_exists docker && docker info &>/dev/null
}

# Log to file
log_to_file() {
    local message="$1"
    local log_file="${2:-$VPS_LOG_DIR/vps-tools.log}"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    mkdir -p "$(dirname "$log_file")"
    echo "[$timestamp] $message" >> "$log_file"
}

# Timestamp for filenames
get_timestamp() {
    date '+%Y%m%d_%H%M%S'
}

# ISO timestamp
get_iso_timestamp() {
    date -u '+%Y-%m-%dT%H:%M:%SZ'
}
