#!/bin/bash
set -euo pipefail

# VPS Tools Installation Script
# Makes all scripts available system-wide
# Usage: bash install.sh [--uninstall]

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly INSTALL_DIR="/opt/vps-tools"
readonly BIN_DIR="/usr/local/bin"
readonly CONFIG_DIR="/etc/vps-tools"
readonly LOG_DIR="/var/log/vps-tools"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*"; }
log_error() { echo -e "${RED}[✗]${NC} $*"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run with sudo"
        exit 1
    fi
}

install() {
    log_info "VPS Tools Installation"
    echo
    
    check_root
    
    # Create directories
    log_info "Creating directories..."
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    # Copy all files
    log_info "Copying files to $INSTALL_DIR..."
    cp -r "$SCRIPT_DIR"/* "$INSTALL_DIR/"
    chmod -R 755 "$INSTALL_DIR"
    chmod -R 755 "$INSTALL_DIR"/*.sh
    find "$INSTALL_DIR" -name "*.sh" -exec chmod 755 {} \;
    
    # Create main dispatcher script
    log_info "Creating dispatcher script..."
    cat > "$BIN_DIR/vps-tools" << 'DISPATCHER'
#!/bin/bash
set -euo pipefail

readonly TOOLS_DIR="/opt/vps-tools"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_menu() {
    clear
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║              VPS Tools Management Suite                        ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo
    echo "PROVISIONING"
    echo "  1) Initial VPS Setup (vps-build)"
    echo
    echo "MONITORING & HEALTH"
    echo "  2) System Health Monitor"
    echo "  3) Service Monitor & Auto-Restart"
    echo "  4) Log Analyzer"
    echo "  5) Backup Verifier"
    echo
    echo "SECURITY"
    echo "  6) SSH Key Audit & Rotation"
    echo "  7) Failed Login Reporter"
    echo "  8) SSL Certificate Checker"
    echo "  9) Open Ports Auditor"
    echo
    echo "DOCKER MANAGEMENT"
    echo " 10) Docker Health Dashboard"
    echo " 11) Docker Log Rotation"
    echo " 12) Docker Cleanup"
    echo " 13) Docker Backup/Restore"
    echo
    echo "MAINTENANCE"
    echo " 14) Automated Cleanup"
    echo " 15) Package Updater"
    echo " 16) Database Backup"
    echo " 17) System Upgrade"
    echo
    echo "ORCHESTRATION & REPORTING"
    echo " 18) System Report & Orchestration"
    echo " 19) View Documentation"
    echo " 20) Install Cron Jobs"
    echo
    echo "  0) Exit"
    echo
}

run_script() {
    local script_path="$1"
    shift
    
    if [[ ! -f "$script_path" ]]; then
        echo -e "${RED}[ERROR]${NC} Script not found: $script_path"
        return 1
    fi
    
    sudo bash "$script_path" "$@"
}

main() {
    while true; do
        show_menu
        
        read -p "Select option: " choice
        
        case "$choice" in
            1) run_script "$TOOLS_DIR/vps-build.sh" ;;
            2) run_script "$TOOLS_DIR/monitoring/vps-health-monitor.sh" ;;
            3) run_script "$TOOLS_DIR/monitoring/vps-service-monitor.sh" ;;
            4) run_script "$TOOLS_DIR/monitoring/vps-log-analyzer.sh" ;;
            5) run_script "$TOOLS_DIR/monitoring/vps-backup-verifier.sh" ;;
            6) run_script "$TOOLS_DIR/security/vps-ssh-audit.sh" --audit ;;
            7) run_script "$TOOLS_DIR/security/vps-failed-login-reporter.sh" ;;
            8) run_script "$TOOLS_DIR/security/vps-ssl-checker.sh" ;;
            9) run_script "$TOOLS_DIR/security/vps-open-ports-auditor.sh" ;;
            10) run_script "$TOOLS_DIR/docker/vps-docker-health.sh" ;;
            11) run_script "$TOOLS_DIR/docker/vps-docker-log-rotation.sh" ;;
            12) run_script "$TOOLS_DIR/docker/vps-docker-cleanup.sh" ;;
            13) run_script "$TOOLS_DIR/docker/vps-docker-backup-restore.sh" --mode=status ;;
            14) run_script "$TOOLS_DIR/maintenance/vps-automated-cleanup.sh" ;;
            15) run_script "$TOOLS_DIR/maintenance/vps-package-updater.sh" --check ;;
            16) run_script "$TOOLS_DIR/maintenance/vps-database-backup.sh" --type=all ;;
            17) run_script "$TOOLS_DIR/maintenance/vps-system-upgrade.sh" --dry-run ;;
            18) run_script "$TOOLS_DIR/orchestration/vps-orchestration.sh" --mode=report ;;
            19) less "$TOOLS_DIR/USAGE.md" ;;
            20) 
                if [[ -f "$TOOLS_DIR/vps-tools-cron.conf" ]]; then
                    sudo cp "$TOOLS_DIR/vps-tools-cron.conf" /etc/cron.d/vps-tools
                    sudo chmod 644 /etc/cron.d/vps-tools
                    echo -e "${GREEN}[✓]${NC} Cron jobs installed"
                    read -p "Press enter to continue..."
                else
                    echo -e "${RED}[ERROR]${NC} Cron config not found"
                    read -p "Press enter to continue..."
                fi
                ;;
            0) exit 0 ;;
            *) echo -e "${RED}Invalid option${NC}"; read -p "Press enter to continue..." ;;
        esac
    done
}

# If arguments provided, run directly
if [[ $# -gt 0 ]]; then
    case "$1" in
        build) shift; sudo bash "$TOOLS_DIR/vps-build.sh" "$@" ;;
        health) shift; sudo bash "$TOOLS_DIR/monitoring/vps-health-monitor.sh" "$@" ;;
        services) shift; sudo bash "$TOOLS_DIR/monitoring/vps-service-monitor.sh" "$@" ;;
        logs) shift; sudo bash "$TOOLS_DIR/monitoring/vps-log-analyzer.sh" "$@" ;;
        backups) shift; sudo bash "$TOOLS_DIR/monitoring/vps-backup-verifier.sh" "$@" ;;
        ssh-audit) shift; sudo bash "$TOOLS_DIR/security/vps-ssh-audit.sh" "$@" ;;
        logins) shift; sudo bash "$TOOLS_DIR/security/vps-failed-login-reporter.sh" "$@" ;;
        ssl) shift; sudo bash "$TOOLS_DIR/security/vps-ssl-checker.sh" "$@" ;;
        ports) shift; sudo bash "$TOOLS_DIR/security/vps-open-ports-auditor.sh" "$@" ;;
        docker-health) shift; sudo bash "$TOOLS_DIR/docker/vps-docker-health.sh" "$@" ;;
        docker-logs) shift; sudo bash "$TOOLS_DIR/docker/vps-docker-log-rotation.sh" "$@" ;;
        docker-clean) shift; sudo bash "$TOOLS_DIR/docker/vps-docker-cleanup.sh" "$@" ;;
        docker-backup) shift; sudo bash "$TOOLS_DIR/docker/vps-docker-backup-restore.sh" "$@" ;;
        cleanup) shift; sudo bash "$TOOLS_DIR/maintenance/vps-automated-cleanup.sh" "$@" ;;
        updates) shift; sudo bash "$TOOLS_DIR/maintenance/vps-package-updater.sh" "$@" ;;
        db-backup) shift; sudo bash "$TOOLS_DIR/maintenance/vps-database-backup.sh" "$@" ;;
        upgrade) shift; sudo bash "$TOOLS_DIR/maintenance/vps-system-upgrade.sh" "$@" ;;
        report) shift; sudo bash "$TOOLS_DIR/orchestration/vps-orchestration.sh" "$@" ;;
        help|--help|-h)
            echo "VPS Tools - Direct command usage:"
            echo "  vps-tools build              - Initial setup"
            echo "  vps-tools health             - Health check"
            echo "  vps-tools services           - Service monitoring"
            echo "  vps-tools logs               - Log analysis"
            echo "  vps-tools backups            - Backup verification"
            echo "  vps-tools ssh-audit          - SSH audit"
            echo "  vps-tools logins             - Login analysis"
            echo "  vps-tools ssl                - SSL certificate check"
            echo "  vps-tools ports              - Port scan"
            echo "  vps-tools docker-health      - Docker health"
            echo "  vps-tools docker-logs        - Docker log rotation"
            echo "  vps-tools docker-clean       - Docker cleanup"
            echo "  vps-tools docker-backup      - Docker backup/restore"
            echo "  vps-tools cleanup            - System cleanup"
            echo "  vps-tools updates            - Package updates"
            echo "  vps-tools db-backup          - Database backup"
            echo "  vps-tools upgrade            - System upgrade"
            echo "  vps-tools report             - Full report"
            ;;
        *)
            echo "Unknown command: $1"
            echo "Run 'vps-tools help' for usage"
            exit 1
            ;;
    esac
else
    main
fi
DISPATCHER
    
    chmod 755 "$BIN_DIR/vps-tools"
    log_success "Dispatcher created: vps-tools"
    
    # Create convenience symlinks
    log_info "Creating convenience shortcuts..."
    ln -sf "$TOOLS_DIR/vps-build.sh" "$BIN_DIR/vps-build" 2>/dev/null || true
    ln -sf "$BIN_DIR/vps-tools" "$BIN_DIR/vps-health" 2>/dev/null || true
    
    # Set proper permissions
    chown -R root:root "$INSTALL_DIR"
    chmod 755 "$LOG_DIR"
    
    echo
    log_success "Installation Complete!"
    echo
    echo "Available commands:"
    echo "  • vps-tools              - Interactive menu"
    echo "  • vps-tools health       - Quick health check"
    echo "  • vps-tools build        - Initial setup"
    echo "  • vps-tools report       - Full system report"
    echo "  • vps-build              - Direct access to vps-build.sh"
    echo
    echo "Documentation:"
    echo "  • vps-tools help         - Command reference"
    echo "  • cat $INSTALL_DIR/README.md"
    echo "  • less $INSTALL_DIR/USAGE.md"
    echo
    echo "Logs:"
    echo "  • $LOG_DIR/"
    echo
}

uninstall() {
    log_warning "Uninstalling VPS Tools..."
    
    check_root
    
    read -p "Remove all VPS Tools? This cannot be undone. (yes/no): " confirm
    
    if [[ "$confirm" != "yes" ]]; then
        log_warning "Cancelled"
        return
    fi
    
    # Remove cron jobs
    [[ -f /etc/cron.d/vps-tools ]] && rm -f /etc/cron.d/vps-tools
    
    # Remove symlinks
    rm -f "$BIN_DIR/vps-tools"
    rm -f "$BIN_DIR/vps-build"
    rm -f "$BIN_DIR/vps-health"
    
    # Remove installation
    rm -rf "$INSTALL_DIR"
    
    log_success "Uninstalled"
    echo "Config and logs remain in:"
    echo "  • $CONFIG_DIR/"
    echo "  • $LOG_DIR/"
    echo
}

main_installer() {
    if [[ "${1:-}" == "--uninstall" ]]; then
        uninstall
    else
        install
    fi
}

main_installer "$@"