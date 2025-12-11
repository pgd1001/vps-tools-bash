#!/bin/bash
set -euo pipefail

# Database Backup Manager - Backup PostgreSQL, MySQL, MongoDB
# Usage: bash vps-database-backup.sh [--type=all|postgresql|mysql|mongodb] [--backup-dir=/opt/backups] [--retention=30]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly LOG_FILE="$LOG_DIR/database-backup.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

DB_TYPE="all"
BACKUP_DIR="/opt/backups"
RETENTION_DAYS=30
ALERT_EMAIL=""
BACKUP_COUNT=0
BACKUP_FAILURES=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*" | tee -a "$LOG_FILE"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*" | tee -a "$LOG_FILE"; ((BACKUP_FAILURES++)); }
log_error() { echo -e "${RED}[✗]${NC} $*" | tee -a "$LOG_FILE"; ((BACKUP_FAILURES++)); }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --type=*) DB_TYPE="${arg#*=}" ;;
            --backup-dir=*) BACKUP_DIR="${arg#*=}" ;;
            --retention=*) RETENTION_DAYS="${arg#*=}" ;;
            --alert-email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

setup_logging() {
    mkdir -p "$LOG_DIR" "$BACKUP_DIR"
    touch "$LOG_FILE"
}

backup_postgresql() {
    if ! command -v pg_dump &>/dev/null; then
        log_warning "PostgreSQL not installed"
        return
    fi
    
    if ! pg_isready -h localhost &>/dev/null; then
        log_error "PostgreSQL not accessible"
        return
    fi
    
    log_info "=== PostgreSQL Backup ==="
    
    local pg_backup_dir="$BACKUP_DIR/postgresql"
    mkdir -p "$pg_backup_dir"
    
    # Get list of databases
    local databases=$(psql -U postgres -t -c "SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres');" 2>/dev/null || echo "")
    
    if [[ -z "$databases" ]]; then
        log_warning "No PostgreSQL databases found"
        return
    fi
    
    while IFS= read -r db; do
        [[ -z "$db" ]] && continue
        
        local backup_file="$pg_backup_dir/${db}.$(date +%Y%m%d-%H%M%S).dump"
        
        log_info "Backing up: $db"
        
        if pg_dump -U postgres "$db" | gzip > "$backup_file"; then
            log_success "PostgreSQL backup: $db"
            ((BACKUP_COUNT++))
        else
            log_error "PostgreSQL backup failed: $db"
        fi
    done <<< "$databases"
    
    # Backup globals (roles, tablespaces)
    log_info "Backing up PostgreSQL globals"
    pg_dumpall -U postgres --globals-only | gzip > "$pg_backup_dir/globals.$(date +%Y%m%d-%H%M%S).dump.gz" 2>/dev/null && log_success "Globals backed up" || log_warning "Globals backup incomplete"
    echo
}

backup_mysql() {
    if ! command -v mysqldump &>/dev/null; then
        log_warning "MySQL not installed"
        return
    fi
    
    if ! mysqladmin ping -u root 2>/dev/null | grep -q "mysqld is alive"; then
        log_error "MySQL not accessible"
        return
    fi
    
    log_info "=== MySQL Backup ==="
    
    local mysql_backup_dir="$BACKUP_DIR/mysql"
    mkdir -p "$mysql_backup_dir"
    
    # All databases at once
    local backup_file="$mysql_backup_dir/all-databases.$(date +%Y%m%d-%H%M%S).sql"
    
    log_info "Backing up all MySQL databases"
    
    if mysqldump -u root --all-databases 2>/dev/null | gzip > "$backup_file.gz"; then
        log_success "MySQL full backup completed"
        ((BACKUP_COUNT++))
    else
        log_error "MySQL backup failed"
    fi
    
    # Individual database backups
    local databases=$(mysql -u root -e "SHOW DATABASES;" 2>/dev/null | grep -v "Database\|information_schema\|mysql\|performance_schema\|sys" || true)
    
    while IFS= read -r db; do
        [[ -z "$db" ]] && continue
        
        local db_backup="$mysql_backup_dir/${db}.$(date +%Y%m%d-%H%M%S).sql.gz"
        
        if mysqldump -u root "$db" 2>/dev/null | gzip > "$db_backup"; then
            log_success "MySQL backup: $db"
            ((BACKUP_COUNT++))
        fi
    done <<< "$databases"
    echo
}

backup_mongodb() {
    if ! command -v mongodump &>/dev/null; then
        log_warning "MongoDB not installed"
        return
    fi
    
    if ! mongo --eval "db.adminCommand('ping')" &>/dev/null; then
        log_error "MongoDB not accessible"
        return
    fi
    
    log_info "=== MongoDB Backup ==="
    
    local mongo_backup_dir="$BACKUP_DIR/mongodb"
    local backup_date=$(date +%Y%m%d-%H%M%S)
    local dump_dir="$mongo_backup_dir/dump-$backup_date"
    
    mkdir -p "$dump_dir"
    
    log_info "Dumping MongoDB"
    
    if mongodump --out="$dump_dir" 2>/dev/null; then
        log_success "MongoDB dump completed"
        
        # Compress
        tar -czf "$mongo_backup_dir/mongodb-$backup_date.tar.gz" -C "$mongo_backup_dir" "dump-$backup_date" 2>/dev/null && {
            rm -rf "$dump_dir"
            log_success "MongoDB backup compressed"
            ((BACKUP_COUNT++))
        } || log_warning "MongoDB compression failed"
    else
        log_error "MongoDB dump failed"
    fi
    echo
}

verify_backups() {
    log_info "=== Verifying Backups ==="
    
    find "$BACKUP_DIR" -type f \( -name "*.dump.gz" -o -name "*.sql.gz" -o -name "*.tar.gz" \) | while read -r backup; do
        if gunzip -t "$backup" &>/dev/null; then
            log_success "Verified: $(basename "$backup")"
        else
            log_error "Corrupted: $(basename "$backup")"
        fi
    done
    echo
}

cleanup_old_backups() {
    log_info "=== Cleaning Old Backups (older than $RETENTION_DAYS days) ==="
    
    local removed=0
    
    find "$BACKUP_DIR" -type f \( -name "*.dump*" -o -name "*.sql*" -o -name "*.tar.gz" \) -mtime +$RETENTION_DAYS | while read -r backup; do
        log_info "Removing: $(basename "$backup")"
        rm -f "$backup"
        ((removed++))
    done
    
    [[ $removed -eq 0 ]] && log_success "No old backups found" || log_success "Removed $removed old backups"
    echo
}

show_backup_status() {
    log_info "=== Backup Status ==="
    
    echo "Backup Location: $BACKUP_DIR"
    echo "Total Size: $(du -sh "$BACKUP_DIR" 2>/dev/null | awk '{print $1}')"
    echo
    
    echo "Recent Backups:"
    find "$BACKUP_DIR" -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -10 | cut -d' ' -f2- | while read -r file; do
        local size=$(du -h "$file" | awk '{print $1}')
        local name=$(basename "$file")
        echo "  $name ($size)"
    done
    echo
}

send_alert() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    local subject="[Database Backup] $(hostname) - $BACKUP_COUNT successful, $BACKUP_FAILURES failed"
    local body="Timestamp: $TIMESTAMP\nHostname: $(hostname)\n\nBackups Completed: $BACKUP_COUNT\nFailures: $BACKUP_FAILURES\n\nLocation: $BACKUP_DIR"
    
    if command -v mail &>/dev/null; then
        echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    fi
}

show_summary() {
    echo
    log_success "=== Database Backup Complete ==="
    echo "Type: $DB_TYPE"
    echo "Backups Created: $BACKUP_COUNT"
    echo "Failures: $BACKUP_FAILURES"
    echo
}

main() {
    parse_args "$@"
    setup_logging
    
    log_info "Database Backup Manager v${SCRIPT_VERSION}"
    echo
    
    case "$DB_TYPE" in
        all)
            backup_postgresql
            backup_mysql
            backup_mongodb
            ;;
        postgresql)
            backup_postgresql
            ;;
        mysql)
            backup_mysql
            ;;
        mongodb)
            backup_mongodb
            ;;
        *)
            log_error "Unknown type: $DB_TYPE"
            exit 1
            ;;
    esac
    
    verify_backups
    cleanup_old_backups
    show_backup_status
    send_alert
    show_summary
}

main "$@"