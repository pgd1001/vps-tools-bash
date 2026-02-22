#!/bin/bash
set -euo pipefail

# Docker Backup & Restore - Automate container and volume backups
# Usage: bash vps-docker-backup.sh --mode=backup [--containers=all] [--backup-dir=/opt/backups]
#        bash vps-docker-backup.sh --mode=restore --backup-file=/path/to/backup.tar.gz

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/docker-backup.log"

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
source "${TOOLS_DIR}/lib/output.sh"

MODE=""
CONTAINERS="all"
VOLUMES="all"
BACKUP_DIR="/opt/backups"
BACKUP_FILE=""
RETENTION_DAYS=30
ALERT_EMAIL=""

parse_args() {
    for arg in "$@"; do
        case $arg in
            --mode=*) MODE="${arg#*=}" ;;
            --containers=*) CONTAINERS="${arg#*=}" ;;
            --volumes=*) VOLUMES="${arg#*=}" ;;
            --backup-dir=*) BACKUP_DIR="${arg#*=}" ;;
            --backup-file=*) BACKUP_FILE="${arg#*=}" ;;
            --retention-days=*) RETENTION_DAYS="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker not installed"
        exit 1
    fi
}

setup_logging() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

backup_container_config() {
    local container=$1
    local backup_dir=$2
    
    log_info "Backing up container config: $container"
    
    docker inspect "$container" > "$backup_dir/${container}-inspect.json"
    docker logs "$container" > "$backup_dir/${container}-logs.txt" 2>&1 || true
    docker diff "$container" > "$backup_dir/${container}-diff.txt" 2>&1 || true
    
    log_success "Config backed up: $container"
}

backup_container_filesystem() {
    local container=$1
    local backup_dir=$2
    
    log_info "Backing up filesystem: $container"
    
    docker export "$container" | gzip > "$backup_dir/${container}-filesystem.tar.gz"
    
    log_success "Filesystem backed up: $container"
}

backup_volumes() {
    local container=$1
    local backup_dir=$2
    
    log_info "Backing up volumes for: $container"
    
    docker inspect "$container" --format='{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{"\n"}}{{end}}{{end}}' | sort -u | while read -r volume; do
        [[ -z "$volume" ]] && continue
        
        log_info "  Backing up volume: $volume"
        
        local vol_path=$(docker volume inspect "$volume" --format='{{.Mountpoint}}')
        
        if [[ -d "$vol_path" ]]; then
            tar -czf "$backup_dir/${container}-${volume}.tar.gz" -C "$vol_path" . 2>/dev/null || log_warning "  Could not backup volume: $volume"
        fi
    done
}

backup_container() {
    local container=$1
    local backup_dir=$2
    
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${container}$"; then
        log_error "Container not found: $container"
        return 1
    fi
    
    log_info "=== Backing up container: $container ==="
    
    mkdir -p "$backup_dir/$container"
    
    backup_container_config "$container" "$backup_dir/$container"
    backup_container_filesystem "$container" "$backup_dir/$container"
    backup_volumes "$container" "$backup_dir/$container"
    
    log_success "Container backup complete: $container"
    echo
}

backup_all_containers() {
    local backup_dir=$1
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local full_backup_dir="$backup_dir/full-backup-$timestamp"
    
    mkdir -p "$full_backup_dir"
    
    log_info "=== Full System Backup ==="
    
    docker ps -a --format "{{.Names}}" | while read -r container; do
        backup_container "$container" "$full_backup_dir"
    done
    
    # Create manifest
    docker ps -a --format "{{.Names}}\t{{.Image}}\t{{.Status}}" > "$full_backup_dir/manifest.txt"
    docker volume ls --format "{{.Name}}" > "$full_backup_dir/volumes.txt"
    docker network ls --format "{{.Name}}" > "$full_backup_dir/networks.txt"
    
    # Compress full backup
    log_info "Compressing backup..."
    tar -czf "$full_backup_dir.tar.gz" -C "$backup_dir" "full-backup-$timestamp" && {
        rm -rf "$full_backup_dir"
        log_success "Full backup compressed: $full_backup_dir.tar.gz"
    }
    
    echo "$full_backup_dir.tar.gz"
}

backup_volumes_only() {
    local backup_dir=$1
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local vol_backup_dir="$backup_dir/volumes-backup-$timestamp"
    
    mkdir -p "$vol_backup_dir"
    
    log_info "=== Backing up all volumes ==="
    
    docker volume ls --format "{{.Name}}" | while read -r volume; do
        [[ -z "$volume" ]] && continue
        
        log_info "Backing up volume: $volume"
        
        local vol_path=$(docker volume inspect "$volume" --format='{{.Mountpoint}}' 2>/dev/null)
        
        if [[ -d "$vol_path" ]]; then
            tar -czf "$vol_backup_dir/$volume.tar.gz" -C "$vol_path" . 2>/dev/null && log_success "  $volume backed up" || log_error "  Failed to backup $volume"
        fi
    done
    
    tar -czf "$vol_backup_dir.tar.gz" -C "$backup_dir" "volumes-backup-$timestamp" && rm -rf "$vol_backup_dir"
    
    log_success "Volumes backup complete"
    echo "$vol_backup_dir.tar.gz"
}

restore_from_backup() {
    local backup_file=$1
    
    if [[ ! -f "$backup_file" ]]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    log_info "=== Restoring from backup ==="
    log_warning "This will overwrite existing containers/volumes"
    read -p "Continue with restore? (yes/no): " confirm
    
    if [[ "$confirm" != "yes" ]]; then
        log_warning "Restore cancelled"
        return
    fi
    
    local restore_dir=$(mktemp -d)
    
    log_info "Extracting backup..."
    tar -xzf "$backup_file" -C "$restore_dir"
    
    # Restore volumes
    if [[ -f "$restore_dir/manifest.txt" ]]; then
        log_info "Restoring volumes..."
        find "$restore_dir" -name "*.tar.gz" | while read -r vol_backup; do
            local vol_name=$(basename "$vol_backup" .tar.gz | sed 's/-filesystem//')
            
            if docker volume inspect "$vol_name" &>/dev/null; then
                log_info "Restoring volume: $vol_name"
                local vol_path=$(docker volume inspect "$vol_name" --format='{{.Mountpoint}}')
                tar -xzf "$vol_backup" -C "$vol_path"
            fi
        done
    fi
    
    rm -rf "$restore_dir"
    log_success "Restore complete"
}

cleanup_old_backups() {
    log_info "=== Cleaning old backups (older than $RETENTION_DAYS days) ==="
    
    find "$BACKUP_DIR" -name "*backup*.tar.gz" -mtime +$RETENTION_DAYS -type f | while read -r backup; do
        log_info "Removing: $(basename "$backup")"
        rm -f "$backup"
    done
    
    log_success "Cleanup complete"
}

show_backup_status() {
    log_info "=== Backup Status ==="
    
    local backup_count=$(find "$BACKUP_DIR" -name "*backup*.tar.gz" 2>/dev/null | wc -l)
    local total_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | awk '{print $1}')
    
    log_success "Backups found: $backup_count"
    log_success "Total size: $total_size"
    
    echo
    echo "Recent backups:"
    find "$BACKUP_DIR" -name "*backup*.tar.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -5 | cut -d' ' -f2- | sed 's/^/  /'
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    local subject="[Docker Backup] $(hostname) - $MODE completed"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nMode: $MODE\nBackup Dir: $BACKUP_DIR\n\nCheck logs for details."
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

main() {
    parse_args "$@"
    check_docker
    setup_logging
    
    log_info "Docker Backup & Restore v${SCRIPT_VERSION}"
    echo
    
    if [[ -z "$MODE" ]]; then
        log_error "Mode required: --mode=backup|restore|status"
        exit 1
    fi
    
    mkdir -p "$BACKUP_DIR"
    
    case "$MODE" in
        backup)
            if [[ "$CONTAINERS" == "all" ]]; then
                backup_all_containers "$BACKUP_DIR"
            else
                backup_container "$CONTAINERS" "$BACKUP_DIR"
            fi
            cleanup_old_backups
            ;;
        restore)
            if [[ -z "$BACKUP_FILE" ]]; then
                log_error "Backup file required: --backup-file=/path/to/backup.tar.gz"
                exit 1
            fi
            restore_from_backup "$BACKUP_FILE"
            ;;
        status)
            show_backup_status
            ;;
        *)
            log_error "Invalid mode: $MODE"
            exit 1
            ;;
    esac
    
    send_alert
    log_success "Complete"
}

main "$@"