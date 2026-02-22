#!/bin/bash
set -euo pipefail

# VPS Log Analyzer - Parse logs for failures, errors, and warnings
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-log-analyzer.sh [--days=7] [--type=auth|service|system|all] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

# Config
DAYS=7
LOG_TYPE="all"
ALERT_EMAIL=""
CRITICAL_ISSUES=0
WARNING_ISSUES=0

# Override: add counter side effects
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*"; ((WARNING_ISSUES++)); }
log_critical() { echo -e "${RED}[✗]${NC} $*"; ((CRITICAL_ISSUES++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --days=*) DAYS="${arg#*=}" ;;
            --type=*) LOG_TYPE="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "Run with sudo for full log access"
    fi
}

get_log_filter_date() {
    date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e'
}

analyze_auth_logs() {
    [[ "$LOG_TYPE" != "auth" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== Authentication Logs (last $DAYS days) ==="
    
    local filter_date=$(get_log_filter_date)
    local log_file="/var/log/auth.log"
    
    if [[ ! -f "$log_file" ]]; then
        log_info "No auth.log found"
        return
    fi
    
    # Failed passwords
    local failed_pwd=$(grep "Failed password" "$log_file" | grep "$filter_date" 2>/dev/null | wc -l)
    if [[ $failed_pwd -gt 20 ]]; then
        log_critical "Failed password attempts: $failed_pwd"
    elif [[ $failed_pwd -gt 5 ]]; then
        log_warning "Failed password attempts: $failed_pwd"
    else
        log_success "Failed password attempts: $failed_pwd"
    fi
    
    # Invalid users
    local invalid=$(grep "Invalid user" "$log_file" | grep "$filter_date" 2>/dev/null | wc -l)
    if [[ $invalid -gt 10 ]]; then
        log_critical "Invalid user attempts: $invalid"
        grep "Invalid user" "$log_file" | grep "$filter_date" 2>/dev/null | awk '{print $8}' | sort | uniq -c | sort -rn | head -3 | sed 's/^/    /'
    elif [[ $invalid -gt 0 ]]; then
        log_warning "Invalid user attempts: $invalid"
    fi
    
    # SSH connection accepted
    local accepted=$(grep "Accepted" "$log_file" | grep "$filter_date" 2>/dev/null | wc -l)
    log_success "Successful SSH logins: $accepted"
    
    # Disconnections
    local disconnected=$(grep "Disconnected from" "$log_file" | grep "$filter_date" 2>/dev/null | wc -l)
    if [[ $disconnected -gt 0 ]]; then
        log_warning "Disconnected sessions: $disconnected"
    fi
    
    # sudo usage
    local sudo_count=$(grep "COMMAND=" "$log_file" | grep "$filter_date" 2>/dev/null | wc -l)
    [[ $sudo_count -gt 0 ]] && log_info "Sudo commands executed: $sudo_count"
    
    echo
}

analyze_service_logs() {
    [[ "$LOG_TYPE" != "service" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== Service Logs (last $DAYS days) ==="
    
    local services=("ssh" "docker" "ufw" "unattended-upgrades")
    
    for service in "${services[@]}"; do
        local errors=$(journalctl -u "$service" --since "$DAYS days ago" 2>/dev/null | grep -i "error\|critical\|failed" | wc -l || echo 0)
        local warnings=$(journalctl -u "$service" --since "$DAYS days ago" 2>/dev/null | grep -i "warn\|alert" | wc -l || echo 0)
        
        if [[ $errors -gt 5 ]]; then
            log_critical "$service: $errors errors"
        elif [[ $errors -gt 0 ]]; then
            log_warning "$service: $errors errors"
        else
            log_success "$service: no errors"
        fi
        
        [[ $warnings -gt 0 ]] && log_warning "$service: $warnings warnings"
    done
    
    echo
}

analyze_docker_logs() {
    [[ "$LOG_TYPE" != "service" && "$LOG_TYPE" != "all" ]] && return
    
    if ! command -v docker &> /dev/null; then
        return
    fi
    
    log_info "=== Docker Container Logs (last $DAYS days) ==="
    
    docker ps -a --format "{{.Names}}" 2>/dev/null | while read -r container; do
        local errors=$(docker logs --since "${DAYS}d" "$container" 2>/dev/null | grep -i "error\|exception\|failed" | wc -l || echo 0)
        
        if [[ $errors -gt 0 ]]; then
            log_warning "$container: $errors error lines in logs"
        fi
    done
    
    echo
}

analyze_system_logs() {
    [[ "$LOG_TYPE" != "system" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== System Logs (last $DAYS days) ==="
    
    # Kernel errors
    local kernel_errors=$(journalctl --since "$DAYS days ago" -p err 2>/dev/null | wc -l)
    if [[ $kernel_errors -gt 10 ]]; then
        log_critical "Kernel errors: $kernel_errors"
    elif [[ $kernel_errors -gt 0 ]]; then
        log_warning "Kernel errors: $kernel_errors"
    else
        log_success "Kernel errors: none"
    fi
    
    # Critical messages
    local critical=$(journalctl --since "$DAYS days ago" -p crit 2>/dev/null | wc -l)
    [[ $critical -gt 0 ]] && log_critical "Critical messages: $critical"
    
    # Out of memory
    local oom=$(journalctl --since "$DAYS days ago" 2>/dev/null | grep -i "out of memory\|oom" | wc -l)
    [[ $oom -gt 0 ]] && log_critical "Out of memory events: $oom"
    
    # Disk errors
    local disk_errors=$(journalctl --since "$DAYS days ago" 2>/dev/null | grep -i "disk\|i/o\|sector" | wc -l)
    [[ $disk_errors -gt 0 ]] && log_warning "Disk I/O errors: $disk_errors"
    
    # Segmentation faults
    local segfaults=$(journalctl --since "$DAYS days ago" 2>/dev/null | grep -i "segfault\|segmentation fault" | wc -l)
    [[ $segfaults -gt 0 ]] && log_warning "Segmentation faults: $segfaults"
    
    echo
}

analyze_package_logs() {
    [[ "$LOG_TYPE" != "system" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== Package Manager Logs (last $DAYS days) ==="
    
    local apt_log="/var/log/apt/term.log"
    
    if [[ -f "$apt_log" ]]; then
        local failed=$(grep -i "WARNING\|ERROR" "$apt_log" | tail -100 | wc -l)
        
        if [[ $failed -gt 0 ]]; then
            log_warning "APT errors/warnings: $failed"
        else
            log_success "No APT errors"
        fi
    else
        log_info "No APT logs found"
    fi
    
    echo
}

analyze_cron_logs() {
    [[ "$LOG_TYPE" != "system" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== Cron Logs (last $DAYS days) ==="
    
    local cron_errors=$(journalctl -u cron --since "$DAYS days ago" 2>/dev/null | grep -i "error\|failed" | wc -l || echo 0)
    
    if [[ $cron_errors -gt 0 ]]; then
        log_warning "Cron errors: $cron_errors"
    else
        log_success "Cron: no errors"
    fi
    
    echo
}

analyze_failed_units() {
    [[ "$LOG_TYPE" != "service" && "$LOG_TYPE" != "all" ]] && return
    
    log_info "=== Failed Systemd Units ==="
    
    local failed=$(systemctl list-units --failed --no-legend 2>/dev/null | wc -l)
    
    if [[ $failed -gt 0 ]]; then
        log_critical "Failed units: $failed"
        systemctl list-units --failed --no-legend 2>/dev/null | awk '{print $1}' | sed 's/^/    /'
    else
        log_success "No failed units"
    fi
    
    echo
}

generate_report() {
    log_info "=== Log Analysis Report ==="
    echo "Period: Last $DAYS days"
    echo "Analysis Date: $TIMESTAMP"
    echo "Hostname: $(hostname)"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $CRITICAL_ISSUES -eq 0 && $WARNING_ISSUES -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Log Alert] $(hostname) - $CRITICAL_ISSUES critical, $WARNING_ISSUES warnings"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nCritical Issues: $CRITICAL_ISSUES\nWarnings: $WARNING_ISSUES\n\nRun 'sudo bash vps-log-analyzer.sh' for full report."
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    echo "============================================"
    log_success "Log Analysis Complete"
    echo "============================================"
    echo "Critical Issues: $CRITICAL_ISSUES"
    echo "Warnings: $WARNING_ISSUES"
    echo "============================================"
    echo
}

main() {
    parse_args "$@"
    check_root
    
    log_info "VPS Log Analyzer v${SCRIPT_VERSION}"
    echo
    
    generate_report
    analyze_auth_logs
    analyze_service_logs
    analyze_docker_logs
    analyze_system_logs
    analyze_package_logs
    analyze_cron_logs
    analyze_failed_units
    
    send_alert
    show_summary
}

main "$@"