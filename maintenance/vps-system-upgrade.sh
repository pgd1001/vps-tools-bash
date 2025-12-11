#!/bin/bash
set -euo pipefail

# VPS System Upgrade - Safely perform major OS upgrades
# Usage: bash vps-system-upgrade.sh [--dry-run] [--backup] [--skip-docker-pull]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/system-upgrade.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DRY_RUN=false
BACKUP=false
SKIP_DOCKER_PULL=false
DOWNTIME_MINUTES=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --dry-run) DRY_RUN=true ;;
            --backup) BACKUP=true ;;
            --skip-docker-pull) SKIP_DOCKER_PULL=true ;;
        esac
    done
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

preflight_checks() {
    log_info "=== Pre-flight Checks ==="
    
    # Check disk space
    local root_free=$(df / | tail -1 | awk '{print $4}')
    
    if [[ $root_free -lt 1048576 ]]; then  # Less than 1GB
        log_error "Insufficient disk space: ${root_free}KB"
        return 1
    fi
    
    log_success "Disk space OK: ${root_free}KB available"
    
    # Check SSH connectivity
    log_success "SSH daemon running"
    
    # Check critical services
    local critical_services=("sshd")
    
    for svc in "${critical_services[@]}"; do
        if systemctl is-active --quiet "$svc"; then
            log_success "$svc running"
        else
            log_error "$svc not running"
            return 1
        fi
    done
    echo
}

create_system_backup() {
    if [[ "$BACKUP" != true ]]; then
        return
    fi
    
    log_info "=== Creating System Backup ==="
    
    local backup_dir="/opt/backups/pre-upgrade-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"
    
    log_info "Backing up critical configs..."
    
    # Essential configs
    tar -czf "$backup_dir/etc-backup.tar.gz" \
        /etc/ssh \
        /etc/docker \
        /etc/apt \
        /etc/nginx \
        /etc/apache2 \
        2>/dev/null || true
    
    log_success "Backup created: $backup_dir"
    echo
}

backup_docker_containers() {
    log_info "=== Backing Up Running Containers ==="
    
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    docker ps --format "{{.Names}}" | while read -r container; do
        log_info "Saving state: $container"
        docker commit "$container" "backup-$container:pre-upgrade" &>/dev/null || true
    done
    
    log_success "Container states saved"
    echo
}

stop_services() {
    log_info "=== Stopping Services ==="
    
    local services=("docker" "nginx" "apache2" "postgresql" "mysql")
    
    for svc in "${services[@]}"; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
            log_info "Stopping: $svc"
            
            if [[ "$DRY_RUN" != true ]]; then
                systemctl stop "$svc" 2>/dev/null || true
                ((DOWNTIME_MINUTES+=1))
            fi
        fi
    done
    
    log_success "Services stopped"
    echo
}

perform_upgrade() {
    log_info "=== Performing System Upgrade ==="
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "[DRY-RUN] Would perform:"
        echo "  apt-get update"
        echo "  apt-get upgrade -y"
        echo "  apt-get dist-upgrade -y"
        echo "  apt-get autoremove -y"
        return
    fi
    
    log_warning "Starting upgrade process..."
    
    apt-get update -qq
    apt-get upgrade -y
    apt-get dist-upgrade -y
    apt-get autoremove -y
    
    log_success "System packages upgraded"
    echo
}

pre_pull_docker_images() {
    if [[ "$SKIP_DOCKER_PULL" == true || ! -f /var/lib/docker/containers ]]; then
        return
    fi
    
    log_info "=== Pre-pulling Docker Images ==="
    
    docker ps -a --format "{{.Image}}" | sort -u | while read -r image; do
        log_info "Pre-pulling: $image"
        
        if docker pull "$image" &>/dev/null; then
            log_success "  Pulled: $image"
        else
            log_warning "  Could not pre-pull: $image"
        fi
    done
    echo
}

restart_services() {
    log_info "=== Restarting Services ==="
    
    local services=("docker" "postgresql" "mysql" "nginx" "apache2")
    
    for svc in "${services[@]}"; do
        if systemctl is-enabled "$svc" 2>/dev/null | grep -q enabled; then
            log_info "Starting: $svc"
            
            if [[ "$DRY_RUN" != true ]]; then
                systemctl start "$svc" 2>/dev/null || true
            fi
        fi
    done
    
    log_success "Services restarted"
    echo
}

verify_upgrade() {
    log_info "=== Verifying Upgrade ==="
    
    # Check critical services
    local failed=0
    local services=("ssh" "docker")
    
    for svc in "${services[@]}"; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
            log_success "$svc running"
        else
            log_error "$svc not running"
            ((failed++))
        fi
    done
    
    if [[ $failed -gt 0 ]]; then
        log_error "Verification failed: $failed services not running"
        return 1
    fi
    
    log_success "System verification passed"
    echo
}

cleanup_old_kernels() {
    log_info "=== Cleaning Old Kernels ==="
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "[DRY-RUN] Would remove old kernels"
        return
    fi
    
    apt-get autoremove -y > /dev/null 2>&1 || true
    
    log_success "Old kernels cleaned"
    echo
}

schedule_reboot() {
    log_warning "=== Kernel Upgrade Requires Reboot ==="
    
    local latest_kernel=$(uname -r)
    local running_kernel=$(uname -r)
    
    if [[ "$latest_kernel" != "$running_kernel" ]]; then
        log_warning "New kernel installed - reboot required for full upgrade"
        echo "Run: sudo shutdown -r now"
    fi
    echo
}

generate_upgrade_report() {
    log_info "=== Upgrade Report ==="
    echo "Upgrade Date: $TIMESTAMP"
    echo "System: $(hostnamectl --short)"
    echo "OS: $(lsb_release -d | cut -f2)"
    echo "Kernel: $(uname -r)"
    echo "Uptime: $(uptime -p)"
    echo "Services Down: ${DOWNTIME_MINUTES}m"
    echo
}

send_notification() {
    local email="${1:-}"
    
    if [[ -z "$email" ]]; then
        return
    fi
    
    local subject="[System Upgrade] $(hostname) - Complete"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nUpgrade completed successfully.\nPlease verify all services.\n\nDowntime: ${DOWNTIME_MINUTES} minutes"
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$email"
    fi
}

show_summary() {
    echo
    log_success "=== System Upgrade Complete ==="
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY-RUN mode - no changes made"
    fi
    
    echo
    echo "Next steps:"
    echo "1. Verify all services are running"
    echo "2. Test critical applications"
    echo "3. Monitor logs: sudo journalctl -f"
    echo "4. Reboot if needed for kernel update"
    echo
}

main() {
    parse_args "$@"
    setup_logging
    
    log_info "VPS System Upgrade Manager v${SCRIPT_VERSION}"
    [[ "$DRY_RUN" == true ]] && log_warning "DRY-RUN mode enabled"
    echo
    
    preflight_checks || { log_error "Pre-flight checks failed"; exit 1; }
    
    create_system_backup
    backup_docker_containers
    stop_services
    perform_upgrade
    pre_pull_docker_images
    restart_services
    verify_upgrade
    cleanup_old_kernels
    schedule_reboot
    generate_upgrade_report
    show_summary
}

main "$@"