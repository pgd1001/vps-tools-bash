#!/bin/bash
set -euo pipefail

# Docker Health Dashboard - Monitor container status and health
# Usage: bash vps-docker-health.sh [--interval=60] [--alert-email=email] [--restart-unhealthy]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/docker-health.log"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

INTERVAL=60
ALERT_EMAIL=""
RESTART_UNHEALTHY=false
UNHEALTHY_COUNT=0
STOPPED_COUNT=0

parse_args() {
    for arg in "$@"; do
        case $arg in
            --interval=*) INTERVAL="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
            --restart-unhealthy) RESTART_UNHEALTHY=true ;;
        esac
    done
}

check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker not installed"
        exit 1
    fi
    
    if ! docker ps &>/dev/null; then
        log_error "Cannot access Docker daemon"
        exit 1
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

dashboard_header() {
    clear
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║           Docker Container Health Dashboard                   ║"
    echo "║  Updated: $(date '+%Y-%m-%d %H:%M:%S')                               ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo
}

show_container_stats() {
    log_info "=== Container Status ==="
    
    local total=$(docker ps -a --format "{{.Names}}" | wc -l)
    local running=$(docker ps --format "{{.Names}}" | wc -l)
    local stopped=$((total - running))
    
    log_success "Total: $total | Running: $running | Stopped: $stopped"
    echo
    
    docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Size}}" | (
        read -r header
        echo "$header"
        echo "─────────────────────────────────────────────────────────────"
        
        while read -r name status size; do
            if [[ "$status" == "Up"* ]]; then
                echo -e "${GREEN}$name${NC}\t$status\t$size"
            elif [[ "$status" == *"Exited"* ]]; then
                echo -e "${YELLOW}$name${NC}\t$status\t$size"
                ((STOPPED_COUNT++))
            elif [[ "$status" == *"unhealthy"* ]]; then
                echo -e "${RED}$name${NC}\t$status\t$size"
                ((UNHEALTHY_COUNT++))
            else
                echo -e "${BLUE}$name${NC}\t$status\t$size"
            fi
        done
    )
    echo
}

show_resource_usage() {
    log_info "=== Resource Usage ==="
    
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" 2>/dev/null | (
        read -r header
        echo "$header"
        echo "─────────────────────────────────────────────────────────────"
        
        while read -r container cpu mem net; do
            [[ -z "$container" ]] && continue
            
            local cpu_val=${cpu%\%}
            cpu_val=${cpu_val// /}
            
            if (( $(echo "$cpu_val > 75" | bc -l) )); then
                echo -e "${RED}$container${NC}\t$cpu\t$mem\t$net"
            elif (( $(echo "$cpu_val > 50" | bc -l) )); then
                echo -e "${YELLOW}$container${NC}\t$cpu\t$mem\t$net"
            else
                echo -e "${GREEN}$container${NC}\t$cpu\t$mem\t$net"
            fi
        done
    ) || log_warning "Could not retrieve container stats"
    echo
}

show_container_health() {
    log_info "=== Container Health Status ==="
    
    docker ps --format "{{.Names}}" | while read -r container; do
        local health=$(docker inspect "$container" --format='{{.State.Health.Status}}' 2>/dev/null || echo "none")
        
        case "$health" in
            healthy)
                log_success "$container: healthy"
                ;;
            unhealthy)
                log_error "$container: UNHEALTHY"
                if [[ "$RESTART_UNHEALTHY" == true ]]; then
                    log_warning "Restarting $container..."
                    docker restart "$container"
                fi
                ((UNHEALTHY_COUNT++))
                ;;
            starting)
                log_warning "$container: starting"
                ;;
            *)
                log_info "$container: no health check"
                ;;
        esac
    done
    echo
}

show_container_logs() {
    log_info "=== Recent Errors in Logs ==="
    
    docker ps --format "{{.Names}}" | while read -r container; do
        local errors=$(docker logs --since 1h "$container" 2>/dev/null | grep -i "error\|exception\|fatal" | wc -l)
        
        if [[ $errors -gt 0 ]]; then
            log_warning "$container: $errors errors in last hour"
            docker logs --since 1h "$container" 2>/dev/null | grep -i "error\|exception\|fatal" | head -2 | sed 's/^/    /'
        fi
    done
    echo
}

show_volume_status() {
    log_info "=== Volume Status ==="
    
    local volume_count=$(docker volume ls --quiet | wc -l)
    log_success "Total volumes: $volume_count"
    
    # Unused volumes
    local used_volumes=$(docker ps -a --format "{{.Mounts}}" | tr ',' '\n' | grep -v '^$' | sort -u | wc -l)
    local unused=$((volume_count - used_volumes))
    
    [[ $unused -gt 0 ]] && log_warning "Unused volumes: $unused"
    echo
}

show_image_status() {
    log_info "=== Image Status ==="
    
    local image_count=$(docker images --format "{{.Repository}}" | wc -l)
    log_success "Total images: $image_count"
    
    # Dangling images
    local dangling=$(docker images --filter "dangling=true" --format "{{.ID}}" | wc -l)
    [[ $dangling -gt 0 ]] && log_warning "Dangling images: $dangling"
    
    # Unused images
    local used_images=$(docker ps -a --format "{{.Image}}" | sort -u | wc -l)
    local unused=$((image_count - used_images))
    [[ $unused -gt 0 ]] && log_warning "Unused images: $unused"
    echo
}

show_network_status() {
    log_info "=== Network Status ==="
    
    docker network ls --format "table {{.Name}}\t{{.Driver}}\t{{.Containers}}" | (
        read -r header
        echo "$header"
        echo "─────────────────────────────────────────────"
        while read -r name driver containers; do
            [[ -z "$name" ]] && continue
            echo "$name\t$driver\t$containers"
        done
    )
    echo
}

generate_report() {
    log_info "=== Docker Health Report ==="
    echo "Generated: $TIMESTAMP"
    echo "Hostname: $(hostname)"
    echo "Docker Version: $(docker --version)"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $UNHEALTHY_COUNT -eq 0 && $STOPPED_COUNT -eq 0 ]]; then
        return
    fi
    
    local subject="[Docker Alert] $(hostname) - $UNHEALTHY_COUNT unhealthy, $STOPPED_COUNT stopped"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nUnhealthy: $UNHEALTHY_COUNT\nStopped: $STOPPED_COUNT\n\nRun dashboard for details."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

continuous_monitoring() {
    while true; do
        dashboard_header
        generate_report
        show_container_stats
        show_resource_usage
        show_container_health
        show_container_logs
        show_volume_status
        show_image_status
        show_network_status
        
        echo "Refreshing in $INTERVAL seconds... (Ctrl+C to exit)"
        sleep "$INTERVAL"
    done
}

main() {
    parse_args "$@"
    check_docker
    setup_logging
    
    continuous_monitoring
}

main "$@"