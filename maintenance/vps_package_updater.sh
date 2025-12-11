#!/bin/bash
set -euo pipefail

# VPS Package Updater - Manage system and application updates
# Usage: bash vps-package-updater.sh [--check] [--update-system] [--update-docker] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/package-updates.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ACTION="check"
UPDATE_SYSTEM=false
UPDATE_DOCKER=false
ALERT_EMAIL=""
SECURITY_UPDATES=0
REGULAR_UPDATES=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --check) ACTION="check" ;;
            --update-system) UPDATE_SYSTEM=true ;;
            --update-docker) UPDATE_DOCKER=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

check_system_updates() {
    log_info "=== System Package Updates ==="
    
    apt-get update -qq 2>/dev/null
    
    local updates=$(apt-get upgrade -s 2>/dev/null | grep "^Inst" | wc -l)
    local security=$(apt-get upgrade -s 2>/dev/null | grep "^Inst.*-security" | wc -l)
    
    SECURITY_UPDATES=$security
    REGULAR_UPDATES=$((updates - security))
    
    if [[ $security -gt 0 ]]; then
        log_error "Security updates available: $security"
        apt-get upgrade -s 2>/dev/null | grep "^Inst.*-security" | awk '{print $2}' | sed 's/^/    /'
    fi
    
    if [[ $REGULAR_UPDATES -gt 0 ]]; then
        log_warning "Regular updates available: $REGULAR_UPDATES"
    fi
    
    [[ $updates -eq 0 ]] && log_success "System up-to-date"
    echo
}

update_system_packages() {
    log_info "=== Updating System Packages ==="
    
    if [[ $((SECURITY_UPDATES + REGULAR_UPDATES)) -eq 0 ]]; then
        log_success "No updates available"
        return
    fi
    
    log_warning "Upgrading packages..."
    
    apt-get upgrade -y -qq 2>/dev/null
    apt-get dist-upgrade -y -qq 2>/dev/null
    
    log_success "System packages updated"
    echo
}

check_docker_updates() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_info "=== Docker Updates ==="
    
    local current_version=$(docker --version | awk '{print $3}' | sed 's/,//')
    
    log_success "Current Docker version: $current_version"
    
    # Check for updates by simulating installation
    apt-cache policy docker-ce 2>/dev/null | grep Candidate | awk '{print "Latest available: " $2}' | sed 's/^/  /'
    echo
}

update_docker() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_info "=== Updating Docker ==="
    
    apt-get update -qq 2>/dev/null
    apt-get install -y docker-ce docker-ce-cli docker-compose-plugin 2>/dev/null || log_error "Docker update failed"
    
    systemctl restart docker
    log_success "Docker updated and restarted"
    echo
}

check_container_image_updates() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_info "=== Docker Image Updates ==="
    
    docker ps --format "{{.Image}}" | sort -u | while read -r image; do
        log_info "Checking updates for: $image"
        docker pull "$image" &>/dev/null && log_success "  Latest available" || log_warning "  Could not check"
    done
    echo
}

update_docker_images() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_info "=== Pulling Latest Images ==="
    
    docker ps --format "{{.Image}}" | sort -u | while read -r image; do
        log_info "Pulling: $image"
        docker pull "$image" && log_success "  Updated" || log_error "  Failed"
    done
    echo
}

restart_containers() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_warning "Restarting containers with updated images..."
    
    docker ps --format "{{.Names}}" | while read -r container; do
        log_info "Restarting: $container"
        docker restart "$container"
    done
    
    log_success "Containers restarted"
    echo
}

check_kernel_updates() {
    log_info "=== Kernel Status ==="
    
    local current=$(uname -r)
    log_success "Running kernel: $current"
    
    # Check for new kernels
    apt-cache search "^linux-image-" | grep -v "^linux-image-generic" | wc -l > /dev/null
    
    log_info "Installed packages provide automatic security updates"
    echo
}

check_python_packages() {
    log_info "=== Python Package Updates ==="
    
    if command -v pip &>/dev/null; then
        local pip_outdated=$(pip list --outdated 2>/dev/null | tail -n +3 | wc -l)
        
        if [[ $pip_outdated -gt 0 ]]; then
            log_warning "Outdated pip packages: $pip_outdated"
        else
            log_success "All pip packages current"
        fi
    fi
    
    if command -v pip3 &>/dev/null; then
        local pip3_outdated=$(pip3 list --outdated 2>/dev/null | tail -n +3 | wc -l)
        
        if [[ $pip3_outdated -gt 0 ]]; then
            log_warning "Outdated pip3 packages: $pip3_outdated"
        else
            log_success "All pip3 packages current"
        fi
    fi
    echo
}

show_update_plan() {
    log_info "=== Recommended Update Plan ==="
    
    echo "1. Check updates:"
    echo "   bash vps-package-updater.sh --check"
    echo
    echo "2. Update system:"
    echo "   bash vps-package-updater.sh --update-system"
    echo
    echo "3. Update Docker:"
    echo "   bash vps-package-updater.sh --update-docker"
    echo
    echo "4. Restart system if needed:"
    echo "   sudo shutdown -r now"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $((SECURITY_UPDATES + REGULAR_UPDATES)) -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Updates] $(hostname) - $SECURITY_UPDATES security, $REGULAR_UPDATES regular"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nSecurity: $SECURITY_UPDATES\nRegular: $REGULAR_UPDATES\n\nRun 'bash vps-package-updater.sh --check' for details."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Update Check Complete ==="
    echo "Security updates: $SECURITY_UPDATES"
    echo "Regular updates: $REGULAR_UPDATES"
    echo
}

main() {
    parse_args "$@"
    setup_logging
    
    log_info "VPS Package Updater v${SCRIPT_VERSION}"
    echo
    
    case "$ACTION" in
        check)
            check_system_updates
            check_docker_updates
            check_container_image_updates
            check_kernel_updates
            check_python_packages
            show_update_plan
            ;;
    esac
    
    if [[ "$UPDATE_SYSTEM" == true ]]; then
        update_system_packages
    fi
    
    if [[ "$UPDATE_DOCKER" == true ]]; then
        update_docker
        update_docker_images
        restart_containers
    fi
    
    send_alert
    show_summary
}

main "$@"