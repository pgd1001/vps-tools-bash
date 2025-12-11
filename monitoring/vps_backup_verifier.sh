#!/bin/bash
set -euo pipefail

# VPS Backup Verifier - Monitor backup status and verify integrity
# Integrates with vps-build.sh provisioned systems
# Usage: bash vps-backup-verifier.sh [--config=/path/to/config] [--verify-all] [--alert-email=email]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/backup-verifier.log"

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
CONFIG_FILE=""
VERIFY_ALL=false
ALERT_EMAIL=""
BACKUP_ISSUES=0
BACKUPS_OK=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; ((BACKUP_ISSUES++)); }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; ((BACKUP_ISSUES++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --config=*) CONFIG_FILE="${arg#*=}" ;;
            --verify-all) VERIFY_ALL=true ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "Run with sudo for full backup access"
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
    log_info "=== Backup Verifier Started (v${SCRIPT_VERSION}) ==="
}

load_config() {
    if [[ -n "$CONFIG_FILE" && -f "$CONFIG_FILE" ]]; then
        source "$CONFIG_FILE"
    else
        create_default_config
    fi
}

create_default_config() {
    local config_file="/etc/vps-tools/backup-verifier.conf"
    
    if [[ -f "$config_file" ]]; then
        source "$config_file"
        return
    fi
    
    log_info "Creating default config: $config_file"
    mkdir -p /etc/vps-tools
    
    cat > "$config_file" << 'EOF'
# VPS Backup Verifier Configuration

# Backup locations to monitor
# Format: "path:type:retention_hours:max_age_hours"
# type: archive, docker, database, custom
# retention_hours: how long to keep backups
# max_age_hours: alert if backup older than this

BACKUP_LOCATIONS=(
    "/opt/backups:archive:720:168"
    "/var/lib/docker/volumes:docker:720:168"
)

# Docker volumes to backup (volume:backup_path)
DOCKER_VOLUMES=(
    "portainer_data:/opt/backups/portainer"
    "coolify:/opt/backups/coolify"
    "dokploy:/opt/backups/dokploy"
    "n8n_data:/opt/backups/n8n"
    "nextcloud_data:/opt/backups/nextcloud"
)

# Databases to backup (type:name:backup_path)
# type: postgresql, mysql
DATABASES=(
    "postgresql:postgres:/opt/backups/postgres"
    "mysql:mysql:/opt/backups/mysql"
)

# Backup retention policy
RETENTION_DAYS=30
RETENTION_WEEKLY=12
RETENTION_MONTHLY=6

# Verification settings
VERIFY_CHECKSUMS=true
TEST_RESTORE=false  # Can be intensive
CHECKSUM_ALGORITHM="sha256"

# Alerts
ALERT_ON_OLD_BACKUPS=true
ALERT_ON_MISSING_BACKUPS=true
ALERT_ON_VERIFICATION_FAILURE=true
EOF
    
    source "$config_file"
}

check_backup_location() {
    local path=$1
    local type=$2
    local retention=$3
    local max_age=$4
    
    if [[ ! -e "$path" ]]; then
        log_error "Backup location not found: $path"
        return 1
    fi
    
    local backup_count=0
    local total_size=0
    local newest_backup=""
    local newest_time=0
    
    case "$type" in
        archive)
            while IFS= read -r -d '' file; do
                ((backup_count++))
                local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
                ((total_size+=size))
                
                local mtime=$(stat -f%m "$file" 2>/dev/null || stat -c%Y "$file" 2>/dev/null)
                if [[ $mtime -gt $newest_time ]]; then
                    newest_time=$mtime
                    newest_backup=$(basename "$file")
                fi
            done < <(find "$path" -maxdepth 1 -type f \( -name "*.tar.gz" -o -name "*.tar" -o -name "*.zip" \) -print0)
            ;;
        docker)
            if command -v docker &> /dev/null; then
                while IFS= read -r volume; do
                    local vol_path=$(docker volume inspect "$volume" --format '{{.Mountpoint}}' 2>/dev/null)
                    if [[ -d "$vol_path" ]]; then
                        ((backup_count++))
                        local size=$(du -sb "$vol_path" 2>/dev/null | awk '{print $1}')
                        ((total_size+=size))
                    fi
                done < <(docker volume ls --format "{{.Name}}")
            fi
            ;;
    esac
    
    if [[ $backup_count -eq 0 ]]; then
        log_warning "$type backups: none found at $path"
        return 1
    fi
    
    local current_time=$(date +%s)
    local backup_age=$(( (current_time - newest_time) / 3600 ))
    
    if [[ $backup_age -gt $max_age ]]; then
        log_error "$type backup too old: ${backup_age}h (limit: ${max_age}h) - newest: $newest_backup"
        return 1
    elif [[ $backup_age -gt $((max_age / 2)) ]]; then
        log_warning "$type backup aging: ${backup_age}h - newest: $newest_backup"
    else
        log_success "$type backups: $backup_count found, newest: ${backup_age}h old, size: $(numfmt --to=iec $total_size 2>/dev/null || echo $total_size bytes)"
        ((BACKUPS_OK++))
    fi
}

verify_backup_integrity() {
    local path=$1
    
    if [[ ! -d "$path" ]]; then
        return
    fi
    
    log_info "Verifying checksums in: $path"
    
    local checksum_file="$path/.backup-checksums"
    
    if [[ ! -f "$checksum_file" ]]; then
        log_warning "No checksum file found: $checksum_file"
        return
    fi
    
    if ! "$CHECKSUM_ALGORITHM"sum -c "$checksum_file" --quiet > /dev/null 2>&1; then
        log_error "Checksum verification failed in $path"
        return 1
    fi
    
    log_success "Checksums verified for $path"
}

check_docker_volumes() {
    if ! command -v docker &> /dev/null; then
        return
    fi
    
    log_info "=== Docker Volume Backups ==="
    
    for volume_config in "${DOCKER_VOLUMES[@]:-}"; do
        IFS=: read -r volume backup_path <<< "$volume_config"
        
        if ! docker volume inspect "$volume" &>/dev/null; then
            log_warning "Docker volume not found: $volume"
            continue
        fi
        
        if [[ -d "$backup_path" ]]; then
            local backup_age=$(( ($(date +%s) - $(stat -f%m "$backup_path" 2>/dev/null || stat -c%Y "$backup_path")) / 3600 ))
            
            if [[ $backup_age -lt 168 ]]; then
                log_success "$volume: backup current (${backup_age}h old)"
            else
                log_warning "$volume: backup stale (${backup_age}h old)"
            fi
        else
            log_error "$volume: no backup found at $backup_path"
        fi
    done
    echo
}

check_databases() {
    log_info "=== Database Backups ==="
    
    for db_config in "${DATABASES[@]:-}"; do
        IFS=: read -r db_type db_name backup_path <<< "$db_config"
        
        case "$db_type" in
            postgresql)
                if command -v psql &>/dev/null; then
                    if psql -U postgres -d "$db_name" -c "SELECT 1" &>/dev/null 2>&1; then
                        log_success "PostgreSQL: $db_name accessible"
                        
                        if [[ -f "$backup_path/latest.dump" ]]; then
                            local backup_age=$(( ($(date +%s) - $(stat -f%m "$backup_path/latest.dump" 2>/dev/null || stat -c%Y "$backup_path/latest.dump")) / 3600 ))
                            [[ $backup_age -lt 24 ]] && log_success "PostgreSQL: $db_name backup current" || log_warning "PostgreSQL: $db_name backup stale"
                        else
                            log_warning "PostgreSQL: no backup found for $db_name"
                        fi
                    else
                        log_warning "PostgreSQL: $db_name not accessible"
                    fi
                fi
                ;;
            mysql)
                if command -v mysql &>/dev/null; then
                    if mysql -u root -e "SELECT 1" &>/dev/null 2>&1; then
                        log_success "MySQL: $db_name accessible"
                        
                        if [[ -f "$backup_path/latest.sql" ]]; then
                            local backup_age=$(( ($(date +%s) - $(stat -f%m "$backup_path/latest.sql" 2>/dev/null || stat -c%Y "$backup_path/latest.sql")) / 3600 ))
                            [[ $backup_age -lt 24 ]] && log_success "MySQL: $db_name backup current" || log_warning "MySQL: $db_name backup stale"
                        else
                            log_warning "MySQL: no backup found for $db_name"
                        fi
                    else
                        log_warning "MySQL: not accessible"
                    fi
                fi
                ;;
        esac
    done
    echo
}

generate_backup_report() {
    log_info "=== Backup Status Report ==="
    echo "Generated: $TIMESTAMP"
    echo "Hostname: $(hostname)"
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    if [[ $BACKUP_ISSUES -eq 0 ]]; then
        return
    fi
    
    local subject="[VPS Backup Alert] $(hostname) - $BACKUP_ISSUES issues found"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nBackup Issues: $BACKUP_ISSUES\nBackups OK: $BACKUPS_OK\n\nRun 'sudo bash vps-backup-verifier.sh' for full report."
    
    if command -v mail &> /dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Backup Verification Complete ==="
    echo "Backups OK: $BACKUPS_OK"
    echo "Issues Found: $BACKUP_ISSUES"
    echo
}

main() {
    parse_args "$@"
    check_root
    setup_logging
    load_config
    
    log_info "VPS Backup Verifier v${SCRIPT_VERSION}"
    echo
    
    generate_backup_report
    
    for location_config in "${BACKUP_LOCATIONS[@]}"; do
        IFS=: read -r path type retention max_age <<< "$location_config"
        check_backup_location "$path" "$type" "$retention" "$max_age"
    done
    echo
    
    if $VERIFY_ALL; then
        for location_config in "${BACKUP_LOCATIONS[@]}"; do
            IFS=: read -r path type retention max_age <<< "$location_config"
            [[ "$type" == "archive" ]] && verify_backup_integrity "$path"
        done
        echo
    fi
    
    check_docker_volumes
    check_databases
    
    send_alert
    show_summary
}

main "$@"