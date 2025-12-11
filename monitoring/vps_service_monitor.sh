#!/bin/bash
set -euo pipefail

# VPS Service Monitor - Monitor services and auto-restart on failure
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-service-monitor.sh [--config=/path/to/config] [--dry-run] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/service-monitor.log"
readonly STATE_FILE="/tmp/service-monitor-state"

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
CONFIG_FILE=""
DRY_RUN=false
ALERT_EMAIL=""
RESTART_COUNT=0
FAILED_SERVICES=()

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --config=*) CONFIG_FILE="${arg#*=}" ;;
            --dry-run) DRY_RUN=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
    log_info "=== Service Monitor Started (v${SCRIPT_VERSION}) ==="
}

load_config() {
    if [[ -n "$CONFIG_FILE" && -f "$CONFIG_FILE" ]]; then
        log_info "Loading config from: $CONFIG_FILE"
        source "$CONFIG_FILE"
    else
        create_default_config
    fi
}

create_default_config() {
    local config_file="/etc/vps-tools/service-monitor.conf"
    
    if [[ -f "$config_file" ]]; then
        source "$config_file"
        return
    fi
    
    log_info "Creating default config: $config_file"
    mkdir -p /etc/vps-tools
    
    cat > "$config_file" << 'EOF'
# VPS Service Monitor Configuration

# Service monitoring: "service_name:check_type:max_restarts:restart_delay"
# check_type: systemd, docker, process
# max_restarts: max restarts per hour (0 = no limit)
# restart_delay: seconds between check intervals

SERVICES=(
    "ssh:systemd:0:60"
    "ufw:systemd:0:60"
    "docker:systemd:3:120"
    "unattended-upgrades:systemd:0:300"
)

# Docker containers to monitor (name:max_restarts:restart_delay)
DOCKER_CONTAINERS=(
    "coolify:3:120"
    "dokploy:3:120"
    "n8n:3:120"
    "nextcloud:3:120"
)

# Alert settings
ALERT_ON_RESTART=true
ALERT_ON_REPEATED_FAILURE=true
RESTART_THRESHOLD=3  # Alert after N restarts in 1 hour

# Logging
KEEP_LOGS_DAYS=30
EOF
    
    source "$config_file"
}

check_service_systemd() {
    local service=$1
    
    if ! systemctl is-active --quiet "$service" 2>/dev/null; then
        return 1
    fi
    return 0
}

check_docker_container() {
    local container=$1
    
    if ! docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${container}$"; then
        return 1
    fi
    return 0
}

check_process() {
    local process=$1
    
    if ! pgrep -f "$process" > /dev/null; then
        return 1
    fi
    return 0
}

restart_service() {
    local service=$1
    
    if $DRY_RUN; then
        log_warning "[DRY-RUN] Would restart: $service"
        return 0
    fi
    
    if systemctl restart "$service" 2>/dev/null; then
        log_success "Restarted systemd service: $service"
        return 0
    else
        log_error "Failed to restart: $service"
        return 1
    fi
}

restart_docker_container() {
    local container=$1
    
    if $DRY_RUN; then
        log_warning "[DRY-RUN] Would restart container: $container"
        return 0
    fi
    
    if docker restart "$container" 2>/dev/null; then
        log_success "Restarted Docker container: $container"
        return 0
    else
        log_error "Failed to restart container: $container"
        return 1
    fi
}

get_restart_count() {
    local service=$1
    local current_hour=$(date +%Y%m%d%H)
    local state_key="${service}:${current_hour}"
    
    if [[ -f "$STATE_FILE" ]]; then
        grep "^${state_key}:" "$STATE_FILE" 2>/dev/null | cut -d: -f3 || echo 0
    else
        echo 0
    fi
}

increment_restart_count() {
    local service=$1
    local current_hour=$(date +%Y%m%d%H)
    local state_key="${service}:${current_hour}"
    local count=$(get_restart_count "$service")
    count=$((count + 1))
    
    mkdir -p "$(dirname "$STATE_FILE")"
    grep -v "^${service}:" "$STATE_FILE" 2>/dev/null > "$STATE_FILE.tmp" || true
    echo "${state_key}:${count}" >> "$STATE_FILE.tmp"
    mv "$STATE_FILE.tmp" "$STATE_FILE"
}

should_restart() {
    local service=$1
    local max_restarts=$2
    
    if [[ $max_restarts -eq 0 ]]; then
        return 0
    fi
    
    local count=$(get_restart_count "$service")
    
    if [[ $count -ge $max_restarts ]]; then
        log_error "Max restarts ($max_restarts) reached for $service in this hour"
        FAILED_SERVICES+=("$service")
        return 1
    fi
    
    return 0
}

monitor_services() {
    log_info "=== Monitoring Systemd Services ==="
    
    for service_config in "${SERVICES[@]}"; do
        IFS=: read -r service check_type max_restarts restart_delay <<< "$service_config"
        
        if [[ "$check_type" != "systemd" ]]; then
            continue
        fi
        
        if ! check_service_systemd "$service"; then
            log_error "$service is not running"
            
            if should_restart "$service" "$max_restarts"; then
                restart_service "$service"
                increment_restart_count "$service"
                ((RESTART_COUNT++))
            fi
        else
            log_success "$service running"
        fi
    done
    echo
}

monitor_docker_containers() {
    if ! command -v docker &> /dev/null; then
        return
    fi
    
    if ! systemctl is-active --quiet docker 2>/dev/null; then
        return
    fi
    
    log_info "=== Monitoring Docker Containers ==="
    
    for container_config in "${DOCKER_CONTAINERS[@]:-}"; do
        IFS=: read -r container max_restarts restart_delay <<< "$container_config"
        
        if ! docker ps -a --format "{{.Names}}" 2>/dev/null | grep -q "^${container}$"; then
            log_warning "Container $container not found"
            continue
        fi
        
        if ! check_docker_container "$container"; then
            log_error "$container container is not running"
            
            if should_restart "$container" "$max_restarts"; then
                restart_docker_container "$container"
                increment_restart_count "$container"
                ((RESTART_COUNT++))
            fi
        else
            log_success "$container running"
        fi
    done
    echo
}

check_container_health() {
    if ! command -v docker &> /dev/null; then
        return
    fi
    
    log_info "=== Container Health Status ==="
    
    docker ps --format "table {{.Names}}\t{{.Status}}" 2>/dev/null | tail -n +2 | while read -r name status; do
        if [[ "$status" == *"unhealthy"* ]]; then
            log_error "$name: unhealthy"
        elif [[ "$status" == *"Up"* ]]; then
            log_success "$name: healthy"
        else
            log_warning "$name: $status"
        fi
    done
    echo
}

cleanup_logs() {
    log_info "Cleaning up logs older than ${KEEP_LOGS_DAYS:-30} days"
    
    find "$LOG_DIR" -name "*.log" -type f -mtime +${KEEP_LOGS_DAYS:-30} -delete
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $RESTART_COUNT -eq 0 && ${#FAILED_SERVICES[@]} -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Service Alert] $(hostname) - $RESTART_COUNT restarts, ${#FAILED_SERVICES[@]} failures"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nServices Restarted: $RESTART_COUNT\nFailed Services: ${#FAILED_SERVICES[@]}\n\n"
    
    if [[ ${#FAILED_SERVICES[@]} -gt 0 ]]; then
        body+="Failed Services:\n"
        for svc in "${FAILED_SERVICES[@]}"; do
            body+="  - $svc\n"
        done
    fi
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Service Monitor Check Complete ==="
    echo "Timestamp: $TIMESTAMP"
    echo "Services Restarted: $RESTART_COUNT"
    echo "Failed Services: ${#FAILED_SERVICES[@]}"
    
    if [[ ${#FAILED_SERVICES[@]} -gt 0 ]]; then
        echo "Services Requiring Attention:"
        for svc in "${FAILED_SERVICES[@]}"; do
            echo "  - $svc"
        done
    fi
    echo
}

main() {
    parse_args "$@"
    check_root
    setup_logging
    load_config
    
    log_info "VPS Service Monitor v${SCRIPT_VERSION}"
    
    monitor_services
    monitor_docker_containers
    check_container_health
    cleanup_logs
    send_alert
    show_summary
}

main "$@"