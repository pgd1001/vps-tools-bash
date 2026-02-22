#!/bin/bash
# VPS Tools - Input Validation Library
# Provides sanitization and validation functions for user inputs
# Source this in scripts: source "${TOOLS_DIR:-/opt/vps-tools}/lib/validate.sh"

set -euo pipefail

# Validate port number (1-65535)
validate_port() {
    local port="$1"
    if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -ge 1 ]] && [[ "$port" -le 65535 ]]; then
        return 0
    fi
    return 1
}

# Validate IPv4 address
validate_ip() {
    local ip="$1"
    local regex='^([0-9]{1,3}\.){3}[0-9]{1,3}$'
    if [[ "$ip" =~ $regex ]]; then
        # Check each octet is 0-255
        local IFS='.'
        read -ra octets <<< "$ip"
        for octet in "${octets[@]}"; do
            if [[ "$octet" -gt 255 ]]; then
                return 1
            fi
        done
        return 0
    fi
    return 1
}

# Validate DNS-safe hostname
validate_hostname() {
    local hostname="$1"
    local regex='^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$'
    if [[ "$hostname" =~ $regex ]] && [[ ${#hostname} -le 253 ]]; then
        return 0
    fi
    return 1
}

# Validate basic email format
validate_email() {
    local email="$1"
    local regex='^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$'
    if [[ "$email" =~ $regex ]]; then
        return 0
    fi
    return 1
}

# Validate path has no traversal (no ../)
validate_path() {
    local path="$1"
    if [[ "$path" == *".."* ]]; then
        return 1
    fi
    if [[ "$path" =~ [^a-zA-Z0-9_.\/\-] ]]; then
        return 1
    fi
    return 0
}

# Validate command name (alphanumeric, hyphens, underscores only)
validate_command_name() {
    local cmd="$1"
    if [[ "$cmd" =~ ^[a-zA-Z0-9_\-]+$ ]]; then
        return 0
    fi
    return 1
}

# Validate UFW rule action
validate_ufw_action() {
    local action="$1"
    case "$action" in
        allow|deny|reject|limit|delete) return 0 ;;
        *) return 1 ;;
    esac
}

# Validate database name (alphanumeric and underscores only)
validate_db_name() {
    local name="$1"
    if [[ "$name" =~ ^[a-zA-Z0-9_]+$ ]]; then
        return 0
    fi
    return 1
}

# Sanitize string for use in sed patterns
sanitize_for_sed() {
    local input="$1"
    # Escape sed special characters: \ / & . * [ ] ^ $
    printf '%s' "$input" | sed 's/[\\\/&.*\[\]^$]/\\&/g'
}

# Sanitize string for JSON value embedding
sanitize_for_json() {
    local input="$1"
    # Escape backslash, double quote, newline, tab, carriage return
    printf '%s' "$input" | sed -e 's/\\/\\\\/g' \
                                -e 's/"/\\"/g' \
                                -e 's/\t/\\t/g' \
                                -e ':a;N;$!ba;s/\n/\\n/g'
}

# Validate integer
validate_integer() {
    local value="$1"
    if [[ "$value" =~ ^[0-9]+$ ]]; then
        return 0
    fi
    return 1
}
