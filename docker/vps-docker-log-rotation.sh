#!/bin/bash
set -euo pipefail

# Docker Log Rotation - Manage container log sizes and retention
# Usage: bash vps-docker-log-rotation.sh [--max-size=100m] [--max-file=5] [--apply] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/docker-logs.log"
readonly DOCKER_DAEMON_CONFIG="/etc/docker/daemon.json"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

MAX_SIZE="100m"
MAX_FILE="5"
APPLY_CONFIG=false
ALERT_EMAIL=""
CONTAINERS_CHECKED=0
CONFIG_ISSUES=0

# Override: add counter side effects
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; ((CONFIG_ISSUES++)); }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; ((CONFIG_ISSUES++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --max-size=*) MAX_SIZE="${arg#*=}" ;;
            --max-file=*) MAX_FILE="${arg#*=}" ;;
            --apply) APPLY_CONFIG=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker not installed"
        exit 1
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

check_log_sizes() {
    log_info "=== Container Log Sizes ==="
    
    docker ps -a --format "{{.Names}}" | while read -r container; do
        local log_path=$(docker inspect "$container" --format='{{.LogPath}}' 2>/dev/null)
        
        if [[ ! -f "$log_path" ]]; then
            continue
        fi
        
        local log_size=$(du -h "$log_path" | awk '{print $1}')
        local log_bytes=$(du -b "$log_path" | awk '{print $1}')
        ((CONTAINERS_CHECKED++))
        
        # Convert MAX_SIZE to bytes for comparison
        local max_bytes=$(echo "$MAX_SIZE" | sed 's/m/*1048576/g; s/g/*1073741824/g; s/k/*1024/g' | bc)
        
        if [[ $log_bytes -gt $max_bytes ]]; then
            log_error "$container: $log_size (exceeds $MAX_SIZE limit)"
        elif [[ $log_bytes -gt $((max_bytes/2)) ]]; then
            log_warning "$container: $log_size"
        else
            log_success "$container: $log_size"
        fi
    done
    echo
}

check_logging_driver() {
    log_info "=== Logging Driver Configuration ==="
    
    docker ps -a --format "{{.Names}}" | while read -r container; do
        local driver=$(docker inspect "$container" --format='{{.HostConfig.LogConfig.Type}}' 2>/dev/null)
        
        if [[ "$driver" == "json-file" ]]; then
            log_success "$container: json-file"
        elif [[ "$driver" == "local" ]]; then
            log_success "$container: local (with rotation support)"
        else
            log_info "$container: $driver"
        fi
    done
    echo
}

check_daemon_config() {
    log_info "=== Docker Daemon Log Configuration ==="
    
    if [[ ! -f "$DOCKER_DAEMON_CONFIG" ]]; then
        log_warning "No daemon.json found. Global log rotation not configured."
        return
    fi
    
    if grep -q '"log-driver"' "$DOCKER_DAEMON_CONFIG"; then
        local driver=$(grep -o '"log-driver": "[^"]*"' "$DOCKER_DAEMON_CONFIG" | cut -d'"' -f4)
        log_success "Global driver: $driver"
    else
        log_warning "No global log driver configured"
    fi
    
    if grep -q '"max-size"' "$DOCKER_DAEMON_CONFIG"; then
        local max=$(grep -o '"max-size": "[^"]*"' "$DOCKER_DAEMON_CONFIG" | cut -d'"' -f4)
        log_success "Global max-size: $max"
    else
        log_warning "No global max-size configured"
    fi
    
    if grep -q '"max-file"' "$DOCKER_DAEMON_CONFIG"; then
        local max_file=$(grep -o '"max-file": "[^"]*"' "$DOCKER_DAEMON_CONFIG" | cut -d'"' -f4)
        log_success "Global max-file: $max_file"
    else
        log_warning "No global max-file configured"
    fi
    echo
}

generate_daemon_config() {
    log_info "=== Generating Daemon Configuration ==="
    
    if [[ ! -f "$DOCKER_DAEMON_CONFIG" ]]; then
        log_info "Creating $DOCKER_DAEMON_CONFIG"
        mkdir -p "$(dirname "$DOCKER_DAEMON_CONFIG")"
        echo "{}" > "$DOCKER_DAEMON_CONFIG"
    fi
    
    # Backup original
    cp "$DOCKER_DAEMON_CONFIG" "$DOCKER_DAEMON_CONFIG.backup.$(date +%s)"
    log_success "Backed up daemon.json"
    
    # Update config with log rotation settings
    local tmp_config=$(mktemp)
    
    python3 - "$DOCKER_DAEMON_CONFIG" "$MAX_SIZE" "$MAX_FILE" "$tmp_config" << 'PYTHON'
import json
import sys

config_file = sys.argv[1]
max_size = sys.argv[2]
max_file = sys.argv[3]
tmp_file = sys.argv[4]

try:
    with open(config_file, 'r') as f:
        config = json.load(f)
except:
    config = {}

# Ensure log-driver is set
if 'log-driver' not in config:
    config['log-driver'] = 'json-file'

# Ensure log-opts exists
if 'log-opts' not in config:
    config['log-opts'] = {}

# Set max-size and max-file
config['log-opts']['max-size'] = max_size
config['log-opts']['max-file'] = max_file
config['log-opts']['labels'] = 'true'

with open(tmp_file, 'w') as f:
    json.dump(config, f, indent=2)
PYTHON
    
    if [[ $? -eq 0 ]]; then
        cp "$tmp_config" "$DOCKER_DAEMON_CONFIG"
        log_success "Updated daemon.json with log rotation settings"
        log_info "max-size: $MAX_SIZE, max-file: $MAX_FILE"
    else
        log_error "Failed to update daemon.json"
        cp "$DOCKER_DAEMON_CONFIG.backup.$(date +%s | head -c10)" "$DOCKER_DAEMON_CONFIG"
    fi
    
    rm -f "$tmp_config"
}

apply_container_config() {
    log_info "=== Applying Container Log Rotation ==="
    
    docker ps -a --format "{{.Names}}" | while read -r container; do
        local config_file="/var/lib/docker/containers/$container/config.v2.json"
        
        if [[ ! -f "$config_file" ]]; then
            continue
        fi
        
        log_info "Updating $container..."
        
        # Note: Requires container restart to take full effect
        docker inspect "$container" --format='{{json .HostConfig.LogConfig}}' 2>/dev/null | grep -q "max-size" || {
            log_warning "$container: may need restart for log rotation"
        }
    done
    echo
}

restart_docker() {
    log_warning "Docker daemon restart required to apply global settings"
    
    if [[ "$APPLY_CONFIG" == true ]]; then
        read -p "Restart Docker daemon now? (y/n): " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            systemctl restart docker
            log_success "Docker daemon restarted"
            sleep 5
        fi
    fi
}

manual_log_cleanup() {
    log_info "=== Manual Log Cleanup (Truncate Only) ==="
    
    if [[ "$APPLY_CONFIG" != true ]]; then
        log_info "Use --apply flag to perform cleanup"
        return
    fi
    
    docker ps -a --format "{{.Names}}" | while read -r container; do
        local log_path=$(docker inspect "$container" --format='{{.LogPath}}' 2>/dev/null)
        
        if [[ -f "$log_path" ]]; then
            local original_size=$(du -h "$log_path" | awk '{print $1}')
            > "$log_path"  # Truncate file
            log_success "$container: log truncated ($original_size freed)"
        fi
    done
    echo
}

show_recommendations() {
    log_info "=== Configuration Recommendations ==="
    
    echo "1. Set global log rotation in $DOCKER_DAEMON_CONFIG:"
    echo "   {\"log-driver\": \"json-file\", \"log-opts\": {\"max-size\": \"$MAX_SIZE\", \"max-file\": \"$MAX_FILE\"}}"
    echo
    echo "2. Or configure per-container in docker-compose.yml:"
    echo "   logging:"
    echo "     driver: json-file"
    echo "     options:"
    echo "       max-size: $MAX_SIZE"
    echo "       max-file: '$MAX_FILE'"
    echo
    echo "3. Use 'local' driver for better performance:"
    echo "   {\"log-driver\": \"local\", \"log-opts\": {\"max-size\": \"$MAX_SIZE\"}}"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $CONFIG_ISSUES -eq 0 ]]; then
        return
    fi
    
    local subject="[Docker Log Alert] $(hostname) - $CONFIG_ISSUES issues"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nLog Configuration Issues: $CONFIG_ISSUES\n\nRun 'sudo bash vps-docker-log-rotation.sh' for details."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Log Rotation Check Complete ==="
    echo "Containers Checked: $CONTAINERS_CHECKED"
    echo "Configuration Issues: $CONFIG_ISSUES"
    echo "Settings: max-size=$MAX_SIZE, max-file=$MAX_FILE"
    echo
}

main() {
    parse_args "$@"
    check_docker
    setup_logging
    
    log_info "Docker Log Rotation Manager v${SCRIPT_VERSION}"
    echo
    
    check_log_sizes
    check_logging_driver
    check_daemon_config
    show_recommendations
    
    if [[ "$APPLY_CONFIG" == true ]]; then
        generate_daemon_config
        apply_container_config
        restart_docker
        manual_log_cleanup
    fi
    
    send_alert
    show_summary
}

main "$@"