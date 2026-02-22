#!/bin/bash
# VPS Tools - Notification Library
# Provides webhook and email notification functions
# Source this in scripts: source "${TOOLS_DIR:-/opt/vps-tools}/lib/notify.sh"

set -euo pipefail

# Load config
NOTIFY_CONFIG_FILE="${NOTIFY_CONFIG_FILE:-/etc/vps-tools/config.conf}"

# Load notification settings from config
load_notify_config() {
    if [[ -f "$NOTIFY_CONFIG_FILE" ]]; then
        source "$NOTIFY_CONFIG_FILE"
    fi
}

# Escape a string for safe JSON embedding
_notify_json_escape() {
    local input="$1"
    printf '%s' "$input" | sed -e 's/\\/\\\\/g' \
                                -e 's/"/\\"/g' \
                                -e 's/\t/\\t/g' \
                                -e ':a;N;$!ba;s/\n/\\n/g' \
                                -e 's/\r/\\r/g'
}

# Send webhook notification
send_webhook() {
    local message="$1"
    local level="${2:-info}"  # info, warning, critical
    local title="${3:-VPS Tools Alert}"

    load_notify_config

    [[ -z "${WEBHOOK_URL:-}" ]] && return 0

    local webhook_type="${WEBHOOK_TYPE:-generic}"
    local payload=""
    local color=""

    # Sanitize inputs for JSON
    local safe_title safe_message
    safe_title=$(_notify_json_escape "$title")
    safe_message=$(_notify_json_escape "$message")
    local safe_level
    safe_level=$(_notify_json_escape "$level")

    # Set color based on level
    case "$level" in
        critical) color="#dc3545" ;;  # red
        warning)  color="#ffc107" ;;  # yellow
        info)     color="#17a2b8" ;;  # blue
        success)  color="#28a745" ;;  # green
        *)        color="#6c757d" ;;  # gray
    esac

    # Format payload based on webhook type
    case "$webhook_type" in
        slack)
            payload="{\"attachments\":[{\"color\":\"$color\",\"title\":\"${safe_title}\",\"text\":\"${safe_message}\",\"footer\":\"VPS Tools\",\"ts\":$(date +%s)}]}"
            ;;
        discord)
            payload="{\"embeds\":[{\"title\":\"${safe_title}\",\"description\":\"${safe_message}\",\"color\":$(printf "%d" "0x${color:1}"),\"footer\":{\"text\":\"VPS Tools\"},\"timestamp\":\"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"}]}"
            ;;
        teams)
            payload="{\"@type\":\"MessageCard\",\"themeColor\":\"${color:1}\",\"title\":\"${safe_title}\",\"text\":\"${safe_message}\"}"
            ;;
        generic|*)
            payload="{\"level\":\"${safe_level}\",\"title\":\"${safe_title}\",\"message\":\"${safe_message}\",\"timestamp\":\"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\",\"source\":\"vps-tools\"}"
            ;;
    esac

    # Send webhook with timeout
    curl -s --max-time 30 -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "$payload" >/dev/null 2>&1 || true

    return 0
}

# Send email notification (wrapper for existing mail functionality)
send_email() {
    local subject="$1"
    local message="$2"
    local email="${3:-${ALERT_EMAIL:-}}"

    [[ -z "$email" ]] && return 0

    if command -v mail &>/dev/null; then
        echo "$message" | mail -s "$subject" "$email"
        return $?
    else
        return 1
    fi
}

# Send notification via all configured channels
notify() {
    local message="$1"
    local level="${2:-info}"
    local title="${3:-VPS Tools Alert}"

    load_notify_config

    # Send webhook if configured
    if [[ -n "${WEBHOOK_URL:-}" ]]; then
        send_webhook "$message" "$level" "$title"
    fi

    # Send email if configured
    if [[ -n "${ALERT_EMAIL:-}" ]]; then
        send_email "[$level] $title" "$message"
    fi
}

# Convenience functions
notify_info()     { notify "$1" "info" "${2:-Info}"; }
notify_success()  { notify "$1" "success" "${2:-Success}"; }
notify_warning()  { notify "$1" "warning" "${2:-Warning}"; }
notify_critical() { notify "$1" "critical" "${2:-Critical Alert}"; }
