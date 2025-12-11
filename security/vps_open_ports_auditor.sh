#!/bin/bash
set -euo pipefail

# VPS Open Ports Auditor - Identify and audit open ports/services
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-open-ports-auditor.sh [--expected-ports=22,80,443] [--scan-external] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/ports-audit.log"

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
EXPECTED_PORTS=""
SCAN_EXTERNAL=false
ALERT_EMAIL=""
UNEXPECTED_PORTS=0
UNUSED_PORTS=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --expected-ports=*) EXPECTED_PORTS="${arg#*=}" ;;
            --scan-external) SCAN_EXTERNAL=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "Run with sudo for full port analysis"
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

get_listening_ports() {
    if command -v ss &>/dev/null; then
        ss -tlnup 2>/dev/null | awk 'NR>1 {print $4}' | sed 's/.*://'
    elif command -v netstat &>/dev/null; then
        netstat -tlnup 2>/dev/null | grep LISTEN | awk '{print $4}' | sed 's/.*://'
    else
        log_warning "Neither ss nor netstat found"
        return
    fi
}

get_port_service() {
    local port=$1
    local service=""
    
    # Check systemd services
    if systemctl list-units --all --plain 2>/dev/null | grep -q running; then
        service=$(lsof -i :"$port" 2>/dev/null | awk 'NR>1 {print $1}' | head -1)
    fi
    
    # Fall back to /etc/services
    if [[ -z "$service" ]]; then
        service=$(grep "^[^#].*[[:space:]]$port/" /etc/services 2>/dev/null | awk '{print $1}' | head -1)
    fi
    
    echo "${service:-unknown}"
}

analyze_listening_ports() {
    log_info "=== Listening Ports (IPv4/IPv6) ==="
    
    local listening_ports=()
    local temp_file=$(mktemp)
    
    get_listening_ports > "$temp_file"
    
    if [[ ! -s "$temp_file" ]]; then
        log_warning "No listening ports detected"
        rm -f "$temp_file"
        return
    fi
    
    while IFS= read -r port; do
        [[ -z "$port" ]] && continue
        listening_ports+=("$port")
    done < "$temp_file"
    
    if [[ ${#listening_ports[@]} -eq 0 ]]; then
        log_warning "No listening ports detected"
        rm -f "$temp_file"
        return
    fi
    
    # Sort and unique
    printf '%s\n' "${listening_ports[@]}" | sort -u | while read -r port; do
        local service=$(get_port_service "$port")
        
        if [[ -n "$EXPECTED_PORTS" && "$EXPECTED_PORTS" == *"$port"* ]]; then
            log_success "Port $port: $service (expected)"
        elif [[ -z "$EXPECTED_PORTS" ]]; then
            log_info "Port $port: $service"
        else
            log_error "Port $port: $service (UNEXPECTED)"
            ((UNEXPECTED_PORTS++))
        fi
    done
    
    rm -f "$temp_file"
    echo
}

check_port_processes() {
    log_info "=== Process-to-Port Mapping ==="
    
    if ! command -v lsof &>/dev/null; then
        log_warning "lsof not installed, skipping process mapping"
        return
    fi
    
    lsof -i -P -n 2>/dev/null | grep LISTEN | awk '{print $1, $9}' | sort -u | while read -r process port; do
        [[ -z "$process" || -z "$port" ]] && continue
        log_info "  $process -> $port"
    done
    echo
}

check_udp_ports() {
    log_info "=== UDP Listening Ports ==="
    
    local udp_count=0
    
    if command -v ss &>/dev/null; then
        udp_count=$(ss -ulnp 2>/dev/null | tail -n +2 | wc -l)
        if [[ $udp_count -gt 0 ]]; then
            ss -ulnp 2>/dev/null | tail -n +2 | while read -r proto recvq sendq localaddr foreignaddr state pid; do
                log_info "  $localaddr - $(awk -F/ '{print $1}' <<< "$pid")"
            done
        else
            log_success "No UDP ports listening"
        fi
    fi
    echo
}

check_firewall_rules() {
    log_info "=== UFW Firewall Rules ==="
    
    if ! command -v ufw &>/dev/null; then
        log_info "UFW not installed"
        return
    fi
    
    if ! ufw status | grep -q "Status: active"; then
        log_warning "UFW is disabled"
        return
    fi
    
    log_success "UFW enabled with rules:"
    ufw status numbered | tail -n +3 | sed 's/^/  /'
    echo
}

scan_external_ports() {
    if [[ "$SCAN_EXTERNAL" != true ]]; then
        return
    fi
    
    if ! command -v nmap &>/dev/null; then
        log_warning "nmap not installed, cannot scan external ports"
        return
    fi
    
    log_info "=== External Port Scan (localhost) ==="
    
    local hostname=$(hostname)
    local ip=$(hostname -I | awk '{print $1}')
    
    log_info "Scanning $ip ($hostname)..."
    nmap -sV --script smb-enum-shares localhost 2>/dev/null | grep -E "^[0-9]|open|Service" | head -20 | sed 's/^/  /'
    echo
}

check_common_services() {
    log_info "=== Common Service Ports ==="
    
    local services=(
        "22:SSH"
        "25:SMTP"
        "53:DNS"
        "80:HTTP"
        "110:POP3"
        "143:IMAP"
        "443:HTTPS"
        "465:SMTPS"
        "587:SMTP-TLS"
        "993:IMAPS"
        "995:POP3S"
        "3000:Web-App"
        "5432:PostgreSQL"
        "3306:MySQL"
        "6379:Redis"
        "27017:MongoDB"
        "9000:Portainer"
        "5678:n8n"
        "8080:HTTP-Alt"
        "8443:HTTPS-Alt"
    )
    
    local listening=$(mktemp)
    get_listening_ports | sort -u > "$listening"
    
    for service in "${services[@]}"; do
        IFS=: read -r port name <<< "$service"
        
        if grep -q "^$port$" "$listening"; then
            log_success "$port ($name): listening"
        else
            log_info "$port ($name): not listening"
        fi
    done
    
    rm -f "$listening"
    echo
}

check_port_security() {
    log_info "=== Port Security Analysis ==="
    
    # Check for ports bound to 0.0.0.0 (all interfaces)
    if command -v ss &>/dev/null; then
        local exposed=$(ss -tlnp 2>/dev/null | grep "0.0.0.0:" | wc -l)
        [[ $exposed -gt 0 ]] && log_warning "Ports exposed to all interfaces: $exposed"
    fi
    
    # Check for high numbered ports (potential backdoors)
    local high_ports=$(get_listening_ports | awk '$1 > 10000' | wc -l)
    [[ $high_ports -gt 0 ]] && log_warning "Listening on high-numbered ports: $high_ports"
    
    echo
}

check_disabled_ports() {
    log_info "=== Expected Ports Not Listening ==="
    
    if [[ -z "$EXPECTED_PORTS" ]]; then
        log_info "No expected ports specified"
        return
    fi
    
    local listening=$(mktemp)
    get_listening_ports | sort -u > "$listening"
    
    IFS=',' read -ra ports <<< "$EXPECTED_PORTS"
    
    for port in "${ports[@]}"; do
        if ! grep -q "^$port$" "$listening"; then
            log_warning "Expected port $port is not listening"
            ((UNUSED_PORTS++))
        fi
    done
    
    rm -f "$listening"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $UNEXPECTED_PORTS -eq 0 && $UNUSED_PORTS -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Port Alert] $(hostname) - $UNEXPECTED_PORTS unexpected, $UNUSED_PORTS expected missing"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nUnexpected Ports: $UNEXPECTED_PORTS\nExpected Missing: $UNUSED_PORTS\n\nRun 'sudo bash vps-open-ports-auditor.sh' for full report."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Port Audit Complete ==="
    echo "Timestamp: $TIMESTAMP"
    echo "Hostname: $(hostname)"
    echo "Unexpected Ports: $UNEXPECTED_PORTS"
    echo "Expected Missing: $UNUSED_PORTS"
    echo
}

main() {
    parse_args "$@"
    check_root
    setup_logging
    
    log_info "VPS Open Ports Auditor v${SCRIPT_VERSION}"
    echo
    
    [[ -n "$EXPECTED_PORTS" ]] && log_info "Expected ports: $EXPECTED_PORTS" && echo
    
    analyze_listening_ports
    check_port_processes
    check_udp_ports
    check_firewall_rules
    check_common_services
    check_port_security
    [[ -n "$EXPECTED_PORTS" ]] && check_disabled_ports
    scan_external_ports
    
    send_alert
    show_summary
}

main "$@"