#!/bin/bash
# VPS Tools - Notification Library
# Provides webhook and email notification functions
# Source this in scripts: source "$TOOLS_DIR/lib/notify.sh"

# Load config
NOTIFY_CONFIG_FILE="${NOTIFY_CONFIG_FILE:-/etc/vps-tools/config.conf}"

# Load notification settings from config
load_notify_config() {
    if [[ -f "$NOTIFY_CONFIG_FILE" ]]; then
        source "$NOTIFY_CONFIG_FILE"
    fi
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
            payload=$(cat << EOF
{
    "attachments": [{
        "color": "$color",
        "title": "$title",
        "text": "$message",
        "footer": "VPS Tools",
        "ts": $(date +%s)
    }]
}
EOF
)
            ;;
        
        discord)
            payload=$(cat << EOF
{
    "embeds": [{
        "title": "$title",
        "description": "$message",
        "color": $(printf "%d" "0x${color:1}"),
        "footer": {"text": "VPS Tools"},
        "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    }]
}
EOF
)
            ;;
        
        teams)
            payload=$(cat << EOF
{
    "@type": "MessageCard",
    "themeColor": "${color:1}",
    "title": "$title",
    "text": "$message"
}
EOF
)
            ;;
        
        generic|*)
            payload=$(cat << EOF
{
    "level": "$level",
    "title": "$title",
    "message": "$message",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "source": "vps-tools"
}
EOF
)
            ;;
    esac
    
    # Send webhook
    curl -s -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "$payload" > /dev/null 2>&1
    
    return $?
}

# Send email notification (wrapper for existing mail functionality)
send_email() {
    local subject="$1"
    local message="$2"
    local email="${3:-${ALERT_EMAIL:-}}"
    
    [[ -z "$email" ]] && return 0
    
    if command -v mail &> /dev/null; then
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
