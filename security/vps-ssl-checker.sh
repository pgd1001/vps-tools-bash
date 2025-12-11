#!/bin/bash
set -euo pipefail

# VPS SSL Certificate Checker - Monitor SSL/TLS certificate expiration
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-ssl-checker.sh [--warn-days=30] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/ssl-checker.log"

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
WARN_DAYS=30
ALERT_EMAIL=""
CERTS_OK=0
CERTS_WARNING=0
CERTS_CRITICAL=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; ((CERTS_WARNING++)); }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; ((CERTS_CRITICAL++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --warn-days=*) WARN_DAYS="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

check_letsencrypt() {
    log_info "=== Let's Encrypt Certificates ==="
    
    if [[ ! -d /etc/letsencrypt/live ]]; then
        log_info "No Let's Encrypt certificates found"
        return
    fi
    
    for cert_dir in /etc/letsencrypt/live/*/; do
        [[ ! -d "$cert_dir" ]] && continue
        
        local domain=$(basename "$cert_dir")
        local cert_file="$cert_dir/cert.pem"
        
        if [[ ! -f "$cert_file" ]]; then
            continue
        fi
        
        check_certificate "$cert_file" "$domain"
    done
    echo
}

check_docker_certs() {
    if ! command -v docker &>/dev/null; then
        return
    fi
    
    log_info "=== Docker Service Certificates ==="
    
    local docker_cert_dir="/etc/docker/certs.d"
    
    if [[ ! -d "$docker_cert_dir" ]]; then
        log_info "No Docker certificates found"
        return
    fi
    
    find "$docker_cert_dir" -name "*.crt" -type f | while read -r cert_file; do
        check_certificate "$cert_file" "Docker: $(basename $(dirname "$cert_file"))"
    done
    echo
}

check_nginx_certs() {
    if ! command -v nginx &>/dev/null; then
        return
    fi
    
    log_info "=== Nginx Certificates ==="
    
    local nginx_cert_dir="/etc/nginx/certs"
    
    if [[ ! -d "$nginx_cert_dir" ]]; then
        log_info "No Nginx certificates found"
        return
    fi
    
    find "$nginx_cert_dir" -name "*.crt" -o -name "*.cert" -o -name "*.pem" 2>/dev/null | while read -r cert_file; do
        [[ -z "$cert_file" ]] && continue
        check_certificate "$cert_file" "Nginx: $(basename "$cert_file")"
    done
    echo
}

check_apache_certs() {
    if ! command -v apache2 &>/dev/null && ! command -v httpd &>/dev/null; then
        return
    fi
    
    log_info "=== Apache Certificates ==="
    
    local apache_cert_dirs=("/etc/apache2/certs" "/etc/apache2/ssl" "/etc/httpd/certs" "/etc/pki/tls")
    
    for cert_dir in "${apache_cert_dirs[@]}"; do
        [[ ! -d "$cert_dir" ]] && continue
        
        find "$cert_dir" -name "*.crt" -o -name "*.cert" -o -name "*.pem" 2>/dev/null | while read -r cert_file; do
            [[ -z "$cert_file" ]] && continue
            check_certificate "$cert_file" "Apache: $(basename "$cert_file")"
        done
    done
    echo
}

check_certificate() {
    local cert_file=$1
    local label=${2:-$(basename "$cert_file")}
    
    if [[ ! -f "$cert_file" ]]; then
        return
    fi
    
    local expiry_date=$(openssl x509 -noout -enddate -in "$cert_file" 2>/dev/null | cut -d= -f2)
    local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || date -jf "%b %d %T %Z %Y" "$expiry_date" +%s 2>/dev/null || echo 0)
    local now_epoch=$(date +%s)
    local days_left=$(( (expiry_epoch - now_epoch) / 86400 ))
    
    # Certificate info
    local subject=$(openssl x509 -noout -subject -in "$cert_file" 2>/dev/null | sed 's/subject=//' || echo "Unknown")
    local issuer=$(openssl x509 -noout -issuer -in "$cert_file" 2>/dev/null | sed 's/issuer=//' | cut -d',' -f1 || echo "Unknown")
    
    if [[ $days_left -lt 0 ]]; then
        log_error "$label: EXPIRED $(date -d "$expiry_date" 2>/dev/null || echo "$expiry_date")"
    elif [[ $days_left -lt 7 ]]; then
        log_error "$label: EXPIRES IN ${days_left} DAYS - $expiry_date"
    elif [[ $days_left -lt $WARN_DAYS ]]; then
        log_warning "$label: expires in $days_left days ($expiry_date)"
    else
        log_success "$label: expires in $days_left days ($expiry_date)"
        ((CERTS_OK++))
    fi
}

check_self_signed() {
    log_info "=== Self-Signed Certificate Search ==="
    
    local self_signed_count=0
    
    for search_dir in /etc/ssl /etc/nginx /etc/apache2 /etc/docker /etc/pki; do
        [[ ! -d "$search_dir" ]] && continue
        
        find "$search_dir" \( -name "*.crt" -o -name "*.cert" -o -name "*.pem" \) -type f 2>/dev/null | while read -r cert_file; do
            if openssl x509 -noout -text -in "$cert_file" 2>/dev/null | grep -q "Subject:.*CN=" && \
               openssl x509 -noout -issuer -in "$cert_file" 2>/dev/null | grep -q "Issuer:.*CN="; then
                
                local subject=$(openssl x509 -noout -subject -in "$cert_file" 2>/dev/null | sed 's/subject=//')
                local issuer=$(openssl x509 -noout -issuer -in "$cert_file" 2>/dev/null | sed 's/issuer=//')
                
                if [[ "$subject" == "$issuer" ]]; then
                    log_warning "Self-signed: $(basename "$cert_file") - $subject"
                    ((self_signed_count++))
                fi
            fi
        done
    done
    
    [[ $self_signed_count -eq 0 ]] && log_success "No self-signed certificates found"
    echo
}

check_certificate_chain() {
    log_info "=== Certificate Chain Validation ==="
    
    if [[ ! -d /etc/letsencrypt/live ]]; then
        log_info "No Let's Encrypt certificates to validate"
        return
    fi
    
    for cert_dir in /etc/letsencrypt/live/*/; do
        [[ ! -d "$cert_dir" ]] && continue
        
        local domain=$(basename "$cert_dir")
        local cert_file="$cert_dir/cert.pem"
        local chain_file="$cert_dir/chain.pem"
        
        if [[ ! -f "$cert_file" || ! -f "$chain_file" ]]; then
            continue
        fi
        
        if openssl verify -CAfile "$chain_file" "$cert_file" &>/dev/null; then
            log_success "$domain: chain valid"
        else
            log_error "$domain: chain validation failed"
        fi
    done
    echo
}

check_certificate_pinning() {
    log_info "=== Certificate Fingerprints ==="
    
    if [[ ! -d /etc/letsencrypt/live ]]; then
        return
    fi
    
    for cert_dir in /etc/letsencrypt/live/*/; do
        [[ ! -d "$cert_dir" ]] && continue
        
        local domain=$(basename "$cert_dir")
        local cert_file="$cert_dir/cert.pem"
        
        if [[ ! -f "$cert_file" ]]; then
            continue
        fi
        
        local fingerprint=$(openssl x509 -noout -fingerprint -sha256 -in "$cert_file" 2>/dev/null | cut -d= -f2)
        log_info "$domain SHA256: $fingerprint"
    done
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $CERTS_CRITICAL -eq 0 && $CERTS_WARNING -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS SSL Alert] $(hostname) - $CERTS_CRITICAL critical, $CERTS_WARNING warnings"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nCritical: $CERTS_CRITICAL\nWarnings: $CERTS_WARNING\nOK: $CERTS_OK\n\nRun 'sudo bash vps-ssl-checker.sh' for full report."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== SSL Certificate Check Complete ==="
    echo "Expiry Warning Threshold: $WARN_DAYS days"
    echo "Certificates OK: $CERTS_OK"
    echo "Warnings: $CERTS_WARNING"
    echo "Critical: $CERTS_CRITICAL"
    echo
}

main() {
    parse_args "$@"
    setup_logging
    
    log_info "VPS SSL Certificate Checker v${SCRIPT_VERSION}"
    echo
    
    check_letsencrypt
    check_docker_certs
    check_nginx_certs
    check_apache_certs
    check_self_signed
    check_certificate_chain
    check_certificate_pinning
    
    send_alert
    show_summary
}

main "$@"