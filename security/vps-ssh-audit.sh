#!/bin/bash
set -euo pipefail

# VPS SSH Key Audit & Rotation - Monitor and manage SSH keys
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-ssh-audit.sh [--audit] [--rotate-user=username] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/ssh-audit.log"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"
source "${TOOLS_DIR}/lib/validate.sh"

# Config
ACTION="audit"
ROTATE_USER=""
ALERT_EMAIL=""
SECURITY_ISSUES=0

# Override: add counter side effects
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; ((SECURITY_ISSUES++)); }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; ((SECURITY_ISSUES++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --audit) ACTION="audit" ;;
            --rotate-user=*) ACTION="rotate"; ROTATE_USER="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

audit_ssh_keys() {
    log_info "=== SSH Key Audit ==="
    echo
    
    for user_home in /home/*/; do
        local user=$(basename "$user_home")
        local ssh_dir="$user_home/.ssh"
        
        if [[ ! -d "$ssh_dir" ]]; then
            continue
        fi
        
        log_info "User: $user"
        
        if [[ ! -f "$ssh_dir/authorized_keys" ]]; then
            log_warning "  No authorized_keys file"
            continue
        fi
        
        local key_count=$(wc -l < "$ssh_dir/authorized_keys")
        log_success "  Authorized keys: $key_count"
        
        while IFS= read -r line; do
            [[ -z "$line" || "$line" =~ ^# ]] && continue
            
            # Extract key type and fingerprint
            local key_type=$(echo "$line" | awk '{print $1}')
            local key_data=$(echo "$line" | awk '{print $2}')
            local comment=$(echo "$line" | awk '{$1=$2=""; print $0}' | xargs)
            
            if [[ -z "$key_data" ]]; then
                continue
            fi
            
            local fingerprint=$(echo "$key_data" | base64 -d 2>/dev/null | sha256sum | awk '{print $1}' | cut -c1-16)
            
            case "$key_type" in
                ssh-rsa)
                    log_warning "    Weak key type (RSA): ${comment:-no comment} [$fingerprint]"
                    ;;
                ssh-ed25519)
                    log_success "    Strong key (Ed25519): ${comment:-no comment} [$fingerprint]"
                    ;;
                ecdsa-sha2-nistp256)
                    log_success "    Strong key (ECDSA): ${comment:-no comment} [$fingerprint]"
                    ;;
                *)
                    log_info "    Key type $key_type: ${comment:-no comment}"
                    ;;
            esac
        done < "$ssh_dir/authorized_keys"
        
        # Check SSH key pairs
        if [[ -f "$ssh_dir/id_ed25519" ]]; then
            local key_age=$(( ($(date +%s) - $(stat -c%Y "$ssh_dir/id_ed25519" 2>/dev/null || stat -f%m "$ssh_dir/id_ed25519")) / 86400 ))
            if [[ $key_age -gt 365 ]]; then
                log_warning "  Private key id_ed25519 is ${key_age} days old"
            fi
        fi
        
        if [[ -f "$ssh_dir/id_rsa" ]]; then
            log_warning "  RSA private key found (consider Ed25519)"
        fi
        
        # Check key permissions
        local auth_keys_perms=$(stat -c%a "$ssh_dir/authorized_keys" 2>/dev/null || stat -f%OLp "$ssh_dir/authorized_keys" | tail -c 4)
        if [[ "$auth_keys_perms" != "600" ]]; then
            log_error "  Bad permissions on authorized_keys: $auth_keys_perms (should be 600)"
        fi
        
        echo
    done
}

audit_root_ssh() {
    log_info "=== Root SSH Configuration ==="
    
    if [[ -f /root/.ssh/authorized_keys ]]; then
        local root_keys=$(wc -l < /root/.ssh/authorized_keys)
        log_warning "  Root has $root_keys authorized keys (should be minimal)"
    else
        log_success "  Root has no authorized_keys"
    fi
    
    if [[ -f /root/.ssh/id_rsa ]]; then
        log_warning "  Root RSA private key found"
    fi
    
    echo
}

rotate_user_keys() {
    local user=$1

    if ! id "$user" &>/dev/null; then
        log_error "User $user does not exist"
        return 1
    fi

    local user_home
    user_home=$(getent passwd "$user" | cut -d: -f6)
    local ssh_dir="$user_home/.ssh"
    local backup_dir="$ssh_dir/backups"
    
    log_info "=== SSH Key Rotation for $user ==="
    
    mkdir -p "$backup_dir"
    
    # Backup existing authorized_keys
    if [[ -f "$ssh_dir/authorized_keys" ]]; then
        cp "$ssh_dir/authorized_keys" "$backup_dir/authorized_keys.backup.$(date +%s)"
        log_success "Backed up authorized_keys"
    fi
    
    # Generate new Ed25519 key
    log_info "Generating new Ed25519 key for $user..."
    sudo -u "$user" ssh-keygen -t ed25519 -C "$user@$(hostname)-$(date +%Y%m%d)" -f "$ssh_dir/id_ed25519" -N "" -q
    log_success "Generated new Ed25519 private key"
    
    # Extract public key
    local pubkey=$(cat "$ssh_dir/id_ed25519.pub")
    
    # Append to authorized_keys if not already present
    if ! grep -q "$(echo "$pubkey" | awk '{print $2}')" "$ssh_dir/authorized_keys" 2>/dev/null; then
        echo "$pubkey" >> "$ssh_dir/authorized_keys"
        log_success "Added new public key to authorized_keys"
    fi
    
    chmod 600 "$ssh_dir/authorized_keys"
    chown "$user:$user" "$ssh_dir/id_ed25519" "$ssh_dir/id_ed25519.pub"
    
    log_warning "Old keys remain in authorized_keys for 7 days (remove manually to complete rotation)"
    log_info "New public key:"
    echo "$pubkey"
    echo
}

audit_github_keys() {
    log_info "=== GitHub SSH Key Audit ==="
    
    for user_home in /home/*/; do
        local user=$(basename "$user_home")
        local github_file="$user_home/.github_user"
        
        if [[ ! -f "$github_file" ]]; then
            continue
        fi
        
        local github_user=$(cat "$github_file")
        log_info "User $user configured with GitHub: $github_user"
        
        if ! curl -fsSL "https://github.com/${github_user}.keys" &>/dev/null; then
            log_error "  Cannot fetch keys from GitHub (user may not exist)"
            continue
        fi
        
        local github_keys=$(curl -fsSL "https://github.com/${github_user}.keys" | wc -l)
        log_success "  GitHub has $github_keys public keys"
    done
    echo
}

check_weak_keys() {
    log_info "=== Checking for Weak Keys ==="
    
    local weak_count=0
    
    for user_home in /home/*/; do
        local user=$(basename "$user_home")
        local ssh_dir="$user_home/.ssh"
        
        if [[ ! -f "$ssh_dir/authorized_keys" ]]; then
            continue
        fi
        
        while IFS= read -r line; do
            [[ -z "$line" || "$line" =~ ^# ]] && continue
            
            local key_type=$(echo "$line" | awk '{print $1}')
            
            if [[ "$key_type" == "ssh-dss" ]]; then
                log_error "$user: DSS key found (CRITICAL - remove immediately)"
                ((weak_count++))
            elif [[ "$key_type" == "ssh-rsa" ]]; then
                log_warning "$user: RSA key found (migrate to Ed25519)"
                ((weak_count++))
            fi
        done < "$ssh_dir/authorized_keys"
    done
    
    [[ $weak_count -eq 0 ]] && log_success "No weak keys found"
    echo
}

check_duplicate_keys() {
    log_info "=== Checking for Duplicate Keys ==="
    
    local temp_keyfile=$(mktemp)
    local duplicate_count=0
    
    for user_home in /home/*/; do
        local ssh_dir="$user_home/.ssh"
        [[ -f "$ssh_dir/authorized_keys" ]] && cat "$ssh_dir/authorized_keys" >> "$temp_keyfile"
    done
    
    if [[ -f /root/.ssh/authorized_keys ]]; then
        cat /root/.ssh/authorized_keys >> "$temp_keyfile"
    fi
    
    while IFS= read -r line; do
        [[ -z "$line" || "$line" =~ ^# ]] && continue
        local key=$(echo "$line" | awk '{print $2}')
        local count=$(grep "$key" "$temp_keyfile" | wc -l)
        
        if [[ $count -gt 1 ]]; then
            log_warning "Duplicate key found ($count instances)"
            ((duplicate_count++))
        fi
    done < "$temp_keyfile" | sort -u
    
    [[ $duplicate_count -eq 0 ]] && log_success "No duplicate keys found"
    rm -f "$temp_keyfile"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $SECURITY_ISSUES -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS SSH Security Alert] $(hostname) - $SECURITY_ISSUES issues"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nSecurity Issues: $SECURITY_ISSUES\n\nRun 'sudo bash vps-ssh-audit.sh --audit' for full report."
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== SSH Audit Complete ==="
    echo "Timestamp: $TIMESTAMP"
    echo "Security Issues: $SECURITY_ISSUES"
    echo
}

main() {
    parse_args "$@"
    check_root
    setup_logging
    
    log_info "VPS SSH Key Audit v${SCRIPT_VERSION}"
    echo
    
    case "$ACTION" in
        audit)
            audit_ssh_keys
            audit_root_ssh
            audit_github_keys
            check_weak_keys
            check_duplicate_keys
            ;;
        rotate)
            rotate_user_keys "$ROTATE_USER"
            ;;
    esac
    
    send_alert
    show_summary
}

main "$@"