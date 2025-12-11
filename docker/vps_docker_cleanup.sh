#!/bin/bash
set -euo pipefail

# Docker Cleanup - Remove unused images, volumes, and networks
# Usage: bash vps-docker-cleanup.sh [--dry-run] [--aggressive] [--prune-all]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/docker-cleanup.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DRY_RUN=false
AGGRESSIVE=false
PRUNE_ALL=false
SPACE_FREED=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --dry-run) DRY_RUN=true ;;
            --aggressive) AGGRESSIVE=true ;;
            --prune-all) PRUNE_ALL=true ;;
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

cleanup_dangling_images() {
    log_info "=== Dangling Images ==="
    
    local dangling=$(docker images --filter "dangling=true" --format "{{.ID}}")
    
    if [[ -z "$dangling" ]]; then
        log_success "No dangling images found"
        return
    fi
    
    while IFS= read -r image_id; do
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove: $image_id"
        else
            docker rmi "$image_id" 2>/dev/null && log_success "Removed: $image_id" || log_error "Failed to remove: $image_id"
        fi
    done <<< "$dangling"
    echo
}

cleanup_unused_images() {
    log_info "=== Unused Images ==="
    
    # Get all images not used by any container
    local unused=$(docker images --format "{{.ID}}:{{.Repository}}:{{.Tag}}" | while IFS=: read -r id repo tag; do
        if ! docker ps -a --format "{{.Image}}" | grep -q "$id\|$repo:$tag"; then
            echo "$id"
        fi
    done)
    
    if [[ -z "$unused" ]]; then
        log_success "No unused images found"
        return
    fi
    
    while IFS= read -r image_id; do
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove: $image_id"
        else
            docker rmi "$image_id" 2>/dev/null && log_success "Removed: $image_id" || log_warning "Could not remove: $image_id (in use)"
        fi
    done <<< "$unused"
    echo
}

cleanup_unused_volumes() {
    log_info "=== Unused Volumes ==="
    
    local unused=$(docker volume ls --format "{{.Name}}" | while read -r vol; do
        if ! docker ps -a --format "{{.Mounts}}" | grep -q "$vol"; then
            echo "$vol"
        fi
    done)
    
    if [[ -z "$unused" ]]; then
        log_success "No unused volumes found"
        return
    fi
    
    while IFS= read -r volume; do
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove: $volume"
        else
            docker volume rm "$volume" 2>/dev/null && log_success "Removed: $volume" || log_error "Failed to remove: $volume"
        fi
    done <<< "$unused"
    echo
}

cleanup_stopped_containers() {
    log_info "=== Stopped Containers ==="
    
    if [[ "$AGGRESSIVE" != true ]]; then
        log_info "Skipped (use --aggressive to clean stopped containers)"
        return
    fi
    
    local stopped=$(docker ps -a --filter "status=exited" --format "{{.Names}}")
    
    if [[ -z "$stopped" ]]; then
        log_success "No stopped containers found"
        return
    fi
    
    while IFS= read -r container; do
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove: $container"
        else
            docker rm "$container" 2>/dev/null && log_success "Removed: $container" || log_error "Failed to remove: $container"
        fi
    done <<< "$stopped"
    echo
}

cleanup_unused_networks() {
    log_info "=== Unused Networks ==="
    
    local unused=$(docker network ls --format "{{.Name}}" | while read -r net; do
        if [[ "$net" == "bridge" || "$net" == "host" || "$net" == "none" ]]; then
            continue
        fi
        if [[ $(docker network inspect "$net" --format='{{len .Containers}}' 2>/dev/null || echo 0) -eq 0 ]]; then
            echo "$net"
        fi
    done)
    
    if [[ -z "$unused" ]]; then
        log_success "No unused networks found"
        return
    fi
    
    while IFS= read -r network; do
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove: $network"
        else
            docker network rm "$network" 2>/dev/null && log_success "Removed: $network" || log_error "Failed to remove: $network"
        fi
    done <<< "$unused"
    echo
}

prune_build_cache() {
    log_info "=== Build Cache ==="
    
    if [[ "$DRY_RUN" == true ]]; then
        docker builder du 2>/dev/null && log_warning "[DRY-RUN] Would clean build cache" || true
        return
    fi
    
    docker builder prune --force 2>/dev/null && log_success "Build cache cleaned" || log_info "No build cache to clean"
    echo
}

system_prune() {
    log_info "=== Docker System Prune ==="
    
    if [[ "$PRUNE_ALL" != true ]]; then
        log_info "Skipped (use --prune-all for full system prune)"
        return
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "[DRY-RUN] Would run: docker system prune -a"
        return
    fi
    
    log_warning "Running full system prune..."
    docker system prune -a --force 2>/dev/null && log_success "System prune completed" || log_error "System prune failed"
    echo
}

show_disk_usage() {
    log_info "=== Docker Disk Usage ==="
    
    if command -v docker &>/dev/null; then
        docker system df | sed 's/^/  /'
    fi
    echo
}

cleanup_old_logs() {
    log_info "=== Old Container Logs ==="
    
    if [[ ! -d /var/lib/docker/containers ]]; then
        log_info "Docker data directory not found"
        return
    fi
    
    local find_days=30
    
    if [[ "$AGGRESSIVE" == true ]]; then
        find_days=7
    fi
    
    local old_logs=$(find /var/lib/docker/containers -name "*-json.log" -mtime +$find_days 2>/dev/null | wc -l)
    
    if [[ $old_logs -gt 0 ]]; then
        log_warning "Found $old_logs logs older than $find_days days"
        
        if [[ "$DRY_RUN" != true && "$AGGRESSIVE" == true ]]; then
            find /var/lib/docker/containers -name "*-json.log" -mtime +$find_days -delete 2>/dev/null
            log_success "Deleted $old_logs old logs"
        fi
    else
        log_success "No old logs found"
    fi
    echo
}

show_summary() {
    echo
    log_success "=== Cleanup Complete ==="
    echo "Timestamp: $TIMESTAMP"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY-RUN mode (no changes made)"
    fi
    
    echo
    echo "Usage recommendations:"
    echo "  - Regular cleanup: bash vps-docker-cleanup.sh"
    echo "  - Remove stopped containers: bash vps-docker-cleanup.sh --aggressive"
    echo "  - Full cleanup: bash vps-docker-cleanup.sh --aggressive --prune-all"
    echo "  - Test first: bash vps-docker-cleanup.sh --dry-run --aggressive"
    echo
}

main() {
    parse_args "$@"
    check_docker
    setup_logging
    
    log_info "Docker Cleanup Manager v${SCRIPT_VERSION}"
    [[ "$DRY_RUN" == true ]] && log_warning "DRY-RUN mode enabled"
    [[ "$AGGRESSIVE" == true ]] && log_warning "AGGRESSIVE mode enabled"
    echo
    
    show_disk_usage
    cleanup_dangling_images
    cleanup_unused_images
    cleanup_unused_volumes
    cleanup_unused_networks
    cleanup_stopped_containers
    cleanup_old_logs
    prune_build_cache
    system_prune
    
    show_summary
}

main "$@"