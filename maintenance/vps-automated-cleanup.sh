#!/bin/bash
set -euo pipefail

# VPS Automated Cleanup - Remove old logs, temp files, cache
# Usage: bash vps-automated-cleanup.sh [--dry-run] [--aggressive]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/cleanup.log"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

DRY_RUN=false
AGGRESSIVE=false
SPACE_FREED=0

parse_args() {
    for arg in "$@"; do
        case $arg in
            --dry-run) DRY_RUN=true ;;
            --aggressive) AGGRESSIVE=true ;;
        esac
    done
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

cleanup_logs() {
    log_info "=== Cleaning System Logs ==="
    
    # Clean rotated logs older than 30 days
    local old_logs=$(find /var/log -name "*.gz" -mtime +30 -type f 2>/dev/null | wc -l)
    
    if [[ $old_logs -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $old_logs compressed logs"
        else
            find /var/log -name "*.gz" -mtime +30 -type f -delete 2>/dev/null
            log_success "Removed $old_logs compressed logs"
        fi
    fi
    
    # Clean journal logs (keep 2 weeks)
    if command -v journalctl &>/dev/null; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would clean journal logs older than 14 days"
        else
            journalctl --vacuum=time=14d &>/dev/null || true
            log_success "Journal logs cleaned"
        fi
    fi
    echo
}

cleanup_temp_files() {
    log_info "=== Cleaning Temporary Files ==="
    
    # Clean /tmp older than 7 days
    local tmp_files=$(find /tmp -type f -mtime +7 2>/dev/null | wc -l)
    
    if [[ $tmp_files -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $tmp_files files from /tmp"
        else
            find /tmp -type f -mtime +7 -delete 2>/dev/null
            log_success "Removed $tmp_files files from /tmp"
        fi
    fi
    
    # Clean /var/tmp
    local var_tmp=$(find /var/tmp -type f -mtime +7 2>/dev/null | wc -l)
    
    if [[ $var_tmp -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $var_tmp files from /var/tmp"
        else
            find /var/tmp -type f -mtime +7 -delete 2>/dev/null
            log_success "Removed $var_tmp files from /var/tmp"
        fi
    fi
    echo
}

cleanup_package_cache() {
    log_info "=== Cleaning Package Cache ==="
    
    if command -v apt &>/dev/null; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would clean APT cache"
        else
            apt-get clean
            apt-get autoclean
            apt-get autoremove -y > /dev/null 2>&1 || true
            log_success "APT cache cleaned"
        fi
    fi
    echo
}

cleanup_application_cache() {
    log_info "=== Cleaning Application Cache ==="
    
    # Nginx cache
    if [[ -d /var/cache/nginx ]]; then
        local nginx_cache=$(find /var/cache/nginx -type f 2>/dev/null | wc -l)
        if [[ $nginx_cache -gt 0 ]]; then
            if [[ "$DRY_RUN" != true ]]; then
                rm -rf /var/cache/nginx/*
                log_success "Nginx cache cleaned: $nginx_cache files"
            fi
        fi
    fi
    
    # PHP cache
    if [[ -d /var/lib/php/sessions ]]; then
        local php_sessions=$(find /var/lib/php/sessions -type f -mtime +7 2>/dev/null | wc -l)
        if [[ $php_sessions -gt 0 ]]; then
            if [[ "$DRY_RUN" != true ]]; then
                find /var/lib/php/sessions -type f -mtime +7 -delete 2>/dev/null
                log_success "PHP sessions cleaned: $php_sessions files"
            fi
        fi
    fi
    echo
}

cleanup_broken_symlinks() {
    log_info "=== Removing Broken Symlinks ==="
    
    local broken=$(find /home /opt /var/www -type l ! -exec test -e {} \; 2>/dev/null | wc -l)
    
    if [[ $broken -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $broken broken symlinks"
        else
            find /home /opt /var/www -type l ! -exec test -e {} \; -delete 2>/dev/null
            log_success "Removed $broken broken symlinks"
        fi
    fi
    echo
}

cleanup_core_dumps() {
    if [[ "$AGGRESSIVE" != true ]]; then
        return
    fi
    
    log_info "=== Cleaning Core Dumps ==="
    
    local cores=$(find / -name "core.*" -type f 2>/dev/null | wc -l)
    
    if [[ $cores -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $cores core dumps"
        else
            find / -name "core.*" -type f -delete 2>/dev/null || true
            log_success "Removed $cores core dumps"
        fi
    fi
    echo
}

cleanup_old_files() {
    if [[ "$AGGRESSIVE" != true ]]; then
        return
    fi
    
    log_info "=== Cleaning Old Files ==="
    
    # Old backup files (older than 90 days)
    local old_backups=$(find /opt/backups -name "*.tar.gz" -mtime +90 2>/dev/null | wc -l)
    
    if [[ $old_backups -gt 0 ]]; then
        if [[ "$DRY_RUN" == true ]]; then
            log_warning "[DRY-RUN] Would remove $old_backups backups older than 90 days"
        else
            find /opt/backups -name "*.tar.gz" -mtime +90 -delete 2>/dev/null || true
            log_success "Removed $old_backups old backups"
        fi
    fi
    echo
}

optimize_disk() {
    if [[ "$AGGRESSIVE" != true ]]; then
        return
    fi
    
    log_info "=== Optimizing Disk ==="
    
    # TRIM SSDs if supported
    if command -v fstrim &>/dev/null; then
        if [[ "$DRY_RUN" != true ]]; then
            fstrim -v / 2>/dev/null || log_warning "TRIM not supported"
        fi
    fi
    echo
}

show_disk_space() {
    log_info "=== Disk Space After Cleanup ==="
    df -h / | tail -1 | awk '{print "Root: " $4 " free (" $5 " used)"}' | sed 's/^/  /'
    echo
}

show_summary() {
    echo
    log_success "=== Cleanup Complete ==="
    [[ "$DRY_RUN" == true ]] && log_warning "DRY-RUN mode (no changes made)"
    [[ "$AGGRESSIVE" == true ]] && log_warning "AGGRESSIVE mode"
    echo
}

main() {
    parse_args "$@"
    setup_logging
    
    log_info "VPS Automated Cleanup v${SCRIPT_VERSION}"
    [[ "$DRY_RUN" == true ]] && log_warning "DRY-RUN enabled"
    echo
    
    cleanup_logs
    cleanup_temp_files
    cleanup_package_cache
    cleanup_application_cache
    cleanup_broken_symlinks
    cleanup_core_dumps
    cleanup_old_files
    optimize_disk
    show_disk_space
    show_summary
}

main "$@"