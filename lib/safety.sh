#!/bin/bash
# VPS Tools - Operational Safety Library
# Provides root enforcement, locking, traps, and temp file management
# Source this in scripts: source "${TOOLS_DIR:-/opt/vps-tools}/lib/safety.sh"

set -euo pipefail

# Track temp files and lock file descriptors for cleanup
declare -a _VPS_TEMP_FILES=()
declare -a _VPS_CLEANUP_COMMANDS=()
_VPS_LOCK_FD=""
_VPS_LOCK_FILE=""

# Require root or exit
require_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "[ERROR] This script must be run as root" >&2
        exit 1
    fi
}

# Soft root check (warn only, for read-only scripts)
check_root_soft() {
    if [[ $EUID -ne 0 ]]; then
        echo "[WARNING] Run with sudo for full diagnostics" >&2
    fi
}

# Check that a command exists or exit with message
require_command() {
    local cmd="$1"
    local msg="${2:-Required command not found: $cmd}"
    if ! command -v "$cmd" &>/dev/null; then
        echo "[ERROR] $msg" >&2
        exit 1
    fi
}

# Acquire flock-based lock. Non-blocking: logs warning and exits 0 if locked.
# Usage: acquire_lock "/var/run/vps-tools-scriptname.lock"
acquire_lock() {
    local lock_file="$1"
    _VPS_LOCK_FILE="$lock_file"

    # Create lock directory if needed
    mkdir -p "$(dirname "$lock_file")"

    # Open lock file on fd 200
    exec 200>"$lock_file"
    _VPS_LOCK_FD=200

    if ! flock -n 200; then
        echo "[WARNING] Another instance is already running (lock: $lock_file)" >&2
        exit 0
    fi

    # Write PID for debugging
    echo $$ >&200
}

# Release lock (called automatically by trap)
release_lock() {
    if [[ -n "$_VPS_LOCK_FD" ]]; then
        flock -u "$_VPS_LOCK_FD" 2>/dev/null || true
        exec 200>&- 2>/dev/null || true
        _VPS_LOCK_FD=""
    fi
    if [[ -n "$_VPS_LOCK_FILE" ]]; then
        rm -f "$_VPS_LOCK_FILE" 2>/dev/null || true
        _VPS_LOCK_FILE=""
    fi
}

# Register cleanup handler for EXIT/INT/TERM
# Call this once at the start of your script after sourcing this library.
setup_trap() {
    trap '_vps_cleanup' EXIT INT TERM
}

# Internal cleanup function
_vps_cleanup() {
    local exit_code=$?

    # Remove tracked temp files
    for tmp in "${_VPS_TEMP_FILES[@]:-}"; do
        [[ -n "$tmp" ]] && rm -f "$tmp" 2>/dev/null || true
    done

    # Run registered cleanup commands
    for cmd in "${_VPS_CLEANUP_COMMANDS[@]:-}"; do
        [[ -n "$cmd" ]] && eval "$cmd" 2>/dev/null || true
    done

    # Release lock
    release_lock

    # Re-exit with original code (unless we trapped a signal)
    exit "$exit_code"
}

# Create a temp file that is automatically cleaned up on exit
# Usage: local my_tmp=$(create_temp)
create_temp() {
    local prefix="${1:-vps-tools}"
    local tmp
    tmp=$(mktemp "/tmp/${prefix}.XXXXXX")
    _VPS_TEMP_FILES+=("$tmp")
    echo "$tmp"
}

# Register an arbitrary command to run on cleanup
# Usage: register_cleanup "systemctl start docker"
register_cleanup() {
    _VPS_CLEANUP_COMMANDS+=("$1")
}

# Backup a file before modifying it
# Creates file.bak.TIMESTAMP
backup_file() {
    local file="$1"
    if [[ -f "$file" ]]; then
        local backup="${file}.bak.$(date +%s)"
        cp "$file" "$backup"
        echo "$backup"
    fi
}
