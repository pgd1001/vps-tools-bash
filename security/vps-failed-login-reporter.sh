#!/bin/bash
set -euo pipefail

# VPS Failed Login Reporter - Monitor SSH authentication failures
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-failed-login-reporter.sh [--days=7] [--threshold=10] [--block-ips] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/failed-login-report.log"
readonly AUTH_LOG="/var/log/auth.log"
readonly BLOCKED_IPS_FILE="/etc/vps-tools/blocked-ips.txt"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

# Config
DAYS=7
THRESHOLD=10
BLOCK_IPS=false
ALERT_EMAIL=""
TOTAL_FAILURES=0

parse_args() {
    for arg in "$@"; do
        case $arg in
            --days=*) DAYS="${arg#*=}" ;;
            --threshold=*) THRESHOLD="${arg#*=}" ;;
            --block-ips) BLOCK_IPS=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "Run with sudo for blocking functionality"
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

analyze_failed_passwords() {
    log_info "=== Failed Password Attempts (last $DAYS days) ==="
    
    if [[ ! -f "$AUTH_LOG" ]]; then
        log_error "Auth log not found: $AUTH_LOG"
        return
    fi
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    local temp_file=$(mktemp)
    
    grep "Failed password" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | sort | uniq -c | sort -rn > "$temp_file"
    
    if [[ ! -s "$temp_file" ]]; then
        log_success "No failed password attempts"
        rm -f "$temp_file"
        return
    fi
    
    while read -r count ip; do
        ((TOTAL_FAILURES+=count))
        
        if [[ $count -ge $THRESHOLD ]]; then
            log_error "IP: $ip - $count attempts (exceeds threshold of $THRESHOLD)"
        elif [[ $count -ge $((THRESHOLD/2)) ]]; then
            log_warning "IP: $ip - $count attempts"
        else
            log_info "IP: $ip - $count attempts"
        fi
    done < "$temp_file"
    
    rm -f "$temp_file"
    echo
}

analyze_invalid_users() {
    log_info "=== Invalid User Attempts (last $DAYS days) ==="
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    local temp_file=$(mktemp)
    
    grep "Invalid user" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | sort | uniq -c | sort -rn > "$temp_file"
    
    if [[ ! -s "$temp_file" ]]; then
        log_success "No invalid user attempts"
        rm -f "$temp_file"
        return
    fi
    
    while read -r count ip; do
        ((TOTAL_FAILURES+=count))
        
        if [[ $count -ge $THRESHOLD ]]; then
            log_error "IP: $ip - $count invalid user attempts"
        else
            log_warning "IP: $ip - $count invalid user attempts"
        fi
    done < "$temp_file"
    
    rm -f "$temp_file"
    echo
}

analyze_authentication_methods() {
    log_info "=== Authentication Methods (last $DAYS days) ==="
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    
    local publickey=$(grep "Accepted publickey" "$AUTH_LOG" | grep "$filter_date" | wc -l)
    local password=$(grep "Accepted password" "$AUTH_LOG" | grep "$filter_date" | wc -l)
    
    [[ $publickey -gt 0 ]] && log_success "PublicKey logins: $publickey"
    [[ $password -gt 0 ]] && log_warning "Password logins: $password (consider disabling)"
    
    echo
}

analyze_attack_patterns() {
    log_info "=== Attack Patterns (last $DAYS days) ==="
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    
    # Brute force pattern: 5+ failures from same IP in 1 hour
    local bruteforce=$(grep "Failed password" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | uniq -c | awk '$1 >= 5 {print $2}' | sort -u | wc -l)
    
    if [[ $bruteforce -gt 0 ]]; then
        log_error "Potential brute force attacks detected from $bruteforce IPs"
        grep "Failed password" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | uniq -c | awk '$1 >= 5 {print $2}' | sort -u | sed 's/^/    /'
    else
        log_success "No brute force patterns detected"
    fi
    
    # Rate limiting check
    local rate_limited=$(grep "Too many authentication failures" "$AUTH_LOG" | grep "$filter_date" | wc -l)
    [[ $rate_limited -gt 0 ]] && log_success "Rate limiting active: $rate_limited events"
    
    echo
}

analyze_successful_logins() {
    log_info "=== Successful Logins (last $DAYS days) ==="
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    local temp_file=$(mktemp)
    
    grep "Accepted" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | sort | uniq -c | sort -rn | head -10 > "$temp_file"
    
    if [[ ! -s "$temp_file" ]]; then
        log_warning "No successful logins recorded"
        rm -f "$temp_file"
        return
    fi
    
    log_success "Top source IPs:"
    while read -r count ip; do
        echo "    $ip: $count logins"
    done < "$temp_file"
    
    rm -f "$temp_file"
    echo
}

get_suspicious_ips() {
    log_info "=== Suspicious IP Analysis ==="
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    local temp_file=$(mktemp)
    local suspicious=()
    
    grep -E "Failed password|Invalid user" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | sort | uniq -c | sort -rn | while read -r count ip; do
        if [[ $count -ge $THRESHOLD ]]; then
            suspicious+=("$ip")
            echo "$ip"
        fi
    done > "$temp_file"
    
    if [[ -s "$temp_file" ]]; then
        while IFS= read -r ip; do
            if [[ "$BLOCK_IPS" == true ]]; then
                block_ip "$ip"
            fi
        done < "$temp_file"
    else
        log_success "No IPs exceed threshold"
    fi
    
    rm -f "$temp_file"
    echo
}

block_ip() {
    local ip=$1
    
    if [[ -z "$ip" || "$ip" == "?" ]]; then
        return
    fi
    
    mkdir -p /etc/vps-tools
    
    if grep -q "^$ip$" "$BLOCKED_IPS_FILE" 2>/dev/null; then
        return
    fi
    
    echo "$ip" >> "$BLOCKED_IPS_FILE"
    log_warning "Blocked IP: $ip"
    
    if command -v ufw &>/dev/null; then
        ufw deny from "$ip" &>/dev/null || true
    fi
}

analyze_geographic_distribution() {
    log_info "=== Geographic Analysis (requires geoiplookup) ==="
    
    if ! command -v geoiplookup &>/dev/null; then
        log_info "geoiplookup not installed (install geoip-bin for full analysis)"
        return
    fi
    
    local filter_date=$(date -d "$DAYS days ago" '+%b %d' 2>/dev/null || date -d "$DAYS days ago" '+%b %e')
    local temp_file=$(mktemp)
    
    grep "Failed password" "$AUTH_LOG" | grep "$filter_date" | awk '{print $11}' | sort -u | while read -r ip; do
        geoiplookup "$ip" 2>/dev/null || echo "Unknown: $ip"
    done | sort | uniq -c | sort -rn | head -10 > "$temp_file"
    
    if [[ -s "$temp_file" ]]; then
        cat "$temp_file" | sed 's/^/    /'
    fi
    
    rm -f "$temp_file"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $TOTAL_FAILURES -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Login Alert] $(hostname) - $TOTAL_FAILURES failures in $DAYS days"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nTotal Failures: $TOTAL_FAILURES\nThreshold: $THRESHOLD\n\nRun 'sudo bash vps-failed-login-reporter.sh' for full report."
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Failed Login Report Complete ==="
    echo "Period: Last $DAYS days"
    echo "Total Failures: $TOTAL_FAILURES"
    echo "Threshold: $THRESHOLD"
    echo
}

main() {
    parse_args "$@"
    check_root
    setup_logging
    
    log_info "VPS Failed Login Reporter v${SCRIPT_VERSION}"
    echo
    
    analyze_failed_passwords
    analyze_invalid_users
    analyze_authentication_methods
    analyze_attack_patterns
    analyze_successful_logins
    get_suspicious_ips
    analyze_geographic_distribution
    
    send_alert
    show_summary
}

main "$@"