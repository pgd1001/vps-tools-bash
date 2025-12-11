#!/bin/bash
set -euo pipefail

# VPS Health Monitor - System monitoring and alerting
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-health-monitor.sh [--alert-email=email@example.com] [--check=service]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
ALERT_EMAIL=""
CHECK_SPECIFIC=""
EXIT_CODE=0
CRITICAL_COUNT=0
WARNING_COUNT=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*"; EXIT_CODE=1; ((WARNING_COUNT++)); }
log_critical() { echo -e "${RED}[✗]${NC} $*"; EXIT_CODE=1; ((CRITICAL_COUNT++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
            --check=*) CHECK_SPECIFIC="${arg#*=}" ;;
            *) log_info "Unknown arg: $arg" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "Run with sudo for full diagnostics"
    fi
}

check_disk_usage() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "disk" ]] && return
    
    log_info "=== Disk Usage ==="
    local threshold=80
    
    while IFS= read -r line; do
        local usage=$(echo "$line" | awk '{print $5}' | sed 's/%//')
        local mount=$(echo "$line" | awk '{print $6}')
        
        if [[ $usage -ge $threshold ]]; then
            log_critical "Disk usage: $usage% on $mount"
        elif [[ $usage -ge $((threshold-10)) ]]; then
            log_warning "Disk usage: $usage% on $mount (approaching limit)"
        else
            log_success "Disk usage: $usage% on $mount"
        fi
    done < <(df -h | tail -n +2)
    echo
}

check_memory_usage() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "memory" ]] && return
    
    log_info "=== Memory Usage ==="
    local mem_info=$(free -h | grep "Mem:")
    local total=$(echo "$mem_info" | awk '{print $2}')
    local used=$(echo "$mem_info" | awk '{print $3}')
    local available=$(echo "$mem_info" | awk '{print $7}')
    
    # Calculate usage percentage using bytes for accuracy
    local mem_used_bytes=$(free -b | grep Mem | awk '{print $3}')
    local mem_total_bytes=$(free -b | grep Mem | awk '{print $2}')
    local usage_percent=$((mem_used_bytes * 100 / mem_total_bytes))
    
    if [[ $usage_percent -ge 85 ]]; then
        log_critical "Memory: $used/$total used ($usage_percent%)"
    elif [[ $usage_percent -ge 75 ]]; then
        log_warning "Memory: $used/$total used ($usage_percent%)"
    else
        log_success "Memory: $used/$total used ($usage_percent%)"
    fi
    echo
}

check_cpu_load() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "cpu" ]] && return
    
    log_info "=== CPU Load ==="
    local cpu_count=$(nproc)
    local load=$(uptime | awk -F'load average:' '{print $2}')
    local load_1=$(echo "$load" | awk '{print $1}' | sed 's/,//')
    local load_threshold=$(echo "scale=2; $cpu_count * 0.8" | bc)
    
    if (( $(echo "$load_1 > $cpu_count" | bc -l) )); then
        log_critical "Load average (1m): $load_1 (CPUs: $cpu_count)"
    elif (( $(echo "$load_1 > $load_threshold" | bc -l) )); then
        log_warning "Load average (1m): $load_1 (CPUs: $cpu_count, threshold: $load_threshold)"
    else
        log_success "Load average (1m): $load_1 (CPUs: $cpu_count)"
    fi
    echo
}

check_system_services() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "services" ]] && return
    
    log_info "=== System Services ==="
    
    local services=("ssh" "ufw" "docker" "unattended-upgrades")
    
    for svc in "${services[@]}"; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
            log_success "$svc: running"
        elif systemctl is-enabled --quiet "$svc" 2>/dev/null; then
            log_warning "$svc: enabled but not running"
        else
            log_info "$svc: not installed/disabled"
        fi
    done
    echo
}

check_docker_containers() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "docker" ]] && return
    
    if ! command -v docker &> /dev/null; then
        return
    fi
    
    log_info "=== Docker Containers ==="
    
    local total=$(docker ps -a --format "{{.Names}}" | wc -l)
    local running=$(docker ps --format "{{.Names}}" | wc -l)
    
    if [[ $total -eq 0 ]]; then
        log_info "No containers found"
    else
        log_success "Containers: $running/$total running"
        
        docker ps -a --format "table {{.Names}}\t{{.Status}}" | tail -n +2 | while read -r name status; do
            if [[ $status == "Up"* ]]; then
                log_success "  $name: ${status:0:15}"
            else
                log_warning "  $name: $status"
            fi
        done
    fi
    echo
}

check_ssh_config() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "ssh" ]] && return
    
    log_info "=== SSH Configuration ==="
    
    local ssh_port=$(grep "^Port" /etc/ssh/sshd_config | awk '{print $2}' || echo "22")
    local root_login=$(grep "^PermitRootLogin" /etc/ssh/sshd_config | awk '{print $2}')
    local pwd_auth=$(grep "^PasswordAuthentication" /etc/ssh/sshd_config | awk '{print $2}')
    
    log_success "SSH Port: $ssh_port"
    
    if [[ "$root_login" == "prohibit-password" ]]; then
        log_success "Root login: prohibit-password (Coolify compatible)"
    elif [[ "$root_login" == "no" ]]; then
        log_success "Root login: disabled"
    else
        log_warning "Root login: $root_login"
    fi
    
    [[ "$pwd_auth" == "no" ]] && log_success "Password auth: disabled" || log_warning "Password auth: enabled"
    echo
}

check_firewall() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "firewall" ]] && return
    
    log_info "=== Firewall (UFW) ==="
    
    if ufw status | grep -q "Status: active"; then
        log_success "UFW: enabled"
        log_info "Active rules:"
        ufw status numbered | tail -n +3 | head -5 | sed 's/^/  /'
        [[ $(ufw status numbered | tail -n +3 | wc -l) -gt 5 ]] && echo "  ... (more rules)"
    else
        log_warning "UFW: disabled"
    fi
    echo
}

check_failed_logins() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "logins" ]] && return
    
    log_info "=== Failed SSH Logins (last 24h) ==="
    
    local failed=$(grep "Failed password" /var/log/auth.log 2>/dev/null | grep "$(date '+%b %d' -d '1 day ago')" | wc -l)
    
    if [[ $failed -gt 10 ]]; then
        log_critical "Failed logins: $failed attempts"
        grep "Failed password" /var/log/auth.log 2>/dev/null | grep "$(date '+%b %d' -d '1 day ago')" | awk '{print $11}' | sort | uniq -c | sort -rn | head -3 | sed 's/^/  /'
    elif [[ $failed -gt 0 ]]; then
        log_warning "Failed logins: $failed attempts"
    else
        log_success "Failed logins: none"
    fi
    echo
}

check_ssl_certificates() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "ssl" ]] && return
    
    log_info "=== SSL Certificates ==="
    
    if [[ ! -d /etc/letsencrypt/live ]]; then
        log_info "No Let's Encrypt certificates found"
        return
    fi
    
    for cert_dir in /etc/letsencrypt/live/*/; do
        local domain=$(basename "$cert_dir")
        local expiry=$(openssl x509 -noout -dates -in "$cert_dir/cert.pem" 2>/dev/null | grep notAfter | cut -d= -f2)
        local expiry_epoch=$(date -d "$expiry" +%s 2>/dev/null || echo 0)
        local now_epoch=$(date +%s)
        local days_left=$(( (expiry_epoch - now_epoch) / 86400 ))
        
        if [[ $days_left -lt 0 ]]; then
            log_critical "$domain: EXPIRED"
        elif [[ $days_left -lt 30 ]]; then
            log_warning "$domain: expires in $days_left days"
        else
            log_success "$domain: expires in $days_left days"
        fi
    done
    echo
}

check_system_updates() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "updates" ]] && return
    
    log_info "=== System Updates ==="
    
    apt-get update -qq 2>/dev/null
    local updates=$(apt-get upgrade -s 2>/dev/null | grep -c "^Inst" || echo 0)
    local security=$(apt-get upgrade -s 2>/dev/null | grep -c "^Inst.*-security" || echo 0)
    
    if [[ $security -gt 0 ]]; then
        log_critical "Security updates available: $security"
    elif [[ $updates -gt 0 ]]; then
        log_warning "Updates available: $updates packages"
    else
        log_success "System up-to-date"
    fi
    echo
}

check_swap() {
    [[ -n "$CHECK_SPECIFIC" && "$CHECK_SPECIFIC" != "swap" ]] && return
    
    log_info "=== Swap ==="
    
    local swap_info=$(free -h | grep Swap)
    local swap_total=$(echo "$swap_info" | awk '{print $2}')
    local swap_used=$(echo "$swap_info" | awk '{print $3}')
    
    if [[ "$swap_total" != "0B" ]]; then
        log_success "Swap: $swap_used/$swap_total"
    else
        log_warning "No swap configured"
    fi
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $CRITICAL_COUNT -eq 0 && $WARNING_COUNT -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Health Alert] $(hostname) - $CRITICAL_COUNT critical, $WARNING_COUNT warnings"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nCritical Issues: $CRITICAL_COUNT\nWarnings: $WARNING_COUNT\n\nRun 'sudo bash vps-health-monitor.sh' for details."
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    elif command -v sendmail &> /dev/null; then
        echo -e "To: $ALERT_EMAIL\nSubject: $subject\n\n$body" | sendmail "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    echo "============================================"
    log_success "Health Check Complete"
    echo "============================================"
    echo "Timestamp: $TIMESTAMP"
    echo "Hostname: $(hostname)"
    echo "Uptime: $(uptime -p)"
    echo
    echo "Summary:"
    echo "  Critical Issues: $CRITICAL_COUNT"
    echo "  Warnings: $WARNING_COUNT"
    echo "============================================"
    echo
}

main() {
    parse_args "$@"
    check_root
    
    log_info "VPS Health Monitor v${SCRIPT_VERSION}"
    echo
    
    check_disk_usage
    check_memory_usage
    check_cpu_load
    check_swap
    check_system_services
    check_docker_containers
    check_ssh_config
    check_firewall
    check_failed_logins
    check_ssl_certificates
    check_system_updates
    
    send_alert
    show_summary
    
    exit $EXIT_CODE
}

main "$@"