#!/bin/bash
set -euo pipefail

# VPS Tools Orchestration - Run all monitoring and maintenance tasks
# Usage: bash vps-orchestration.sh [--mode=report|monitor|maintain|full] [--email=admin@example.com]

readonly SCRIPT_VERSION="1.0"
readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
readonly LOG_DIR="/var/log/vps-tools"
readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

MODE="report"
ALERT_EMAIL=""
REPORT_FILE="/tmp/vps-system-report-$TIMESTAMP.txt"

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $*"; }

parse_args() {
    for arg in "$@"; do
        case $arg in
            --mode=*) MODE="${arg#*=}" ;;
            --email=*) ALERT_EMAIL="${arg#*=}" ;;
        esac
    done
}

header() {
    echo "╔════════════════════════════════════════════════════════════════╗" | tee -a "$REPORT_FILE"
    echo "║         VPS Tools - System Report & Orchestration              ║" | tee -a "$REPORT_FILE"
    echo "║  Generated: $TIMESTAMP                                           ║" | tee -a "$REPORT_FILE"
    echo "╚════════════════════════════════════════════════════════════════╝" | tee -a "$REPORT_FILE"
    echo | tee -a "$REPORT_FILE"
}

run_monitoring() {
    log_info "=== Running Monitoring Tasks ==="
    echo "=== Monitoring Tasks ===" >> "$REPORT_FILE"
    
    bash "$TOOLS_DIR/monitoring/vps-health-monitor.sh" 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/monitoring/vps-service-monitor.sh" 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/security/vps-ssl-checker.sh" 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/security/vps-open-ports-auditor.sh" 2>&1 | tee -a "$REPORT_FILE"
}

run_maintenance() {
    log_info "=== Running Maintenance Tasks ==="
    echo "=== Maintenance Tasks ===" >> "$REPORT_FILE"
    
    bash "$TOOLS_DIR/maintenance/vps-package-updater.sh" --check 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/monitoring/vps-backup-verifier.sh" 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/docker/vps-docker-health.sh" 2>&1 | tee -a "$REPORT_FILE"
}

run_security_audit() {
    log_info "=== Running Security Audits ==="
    echo "=== Security Audits ===" >> "$REPORT_FILE"
    
    bash "$TOOLS_DIR/security/vps-ssh-audit.sh" --audit 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/security/vps-failed-login-reporter.sh" --days=7 2>&1 | tee -a "$REPORT_FILE"
    bash "$TOOLS_DIR/monitoring/vps-log-analyzer.sh" --days=7 2>&1 | tee -a "$REPORT_FILE"
}

run_full_suite() {
    log_info "=== Running Full Suite ==="
    
    run_monitoring
    run_maintenance
    run_security_audit
    
    log_info "Running Docker cleanup..."
    bash "$TOOLS_DIR/docker/vps-docker-cleanup.sh" --aggressive 2>&1 | tee -a "$REPORT_FILE"
    
    log_info "Running system cleanup..."
    bash "$TOOLS_DIR/maintenance/vps-automated-cleanup.sh" --aggressive 2>&1 | tee -a "$REPORT_FILE"
}

generate_summary() {
    echo | tee -a "$REPORT_FILE"
    echo "╔════════════════════════════════════════════════════════════════╗" | tee -a "$REPORT_FILE"
    echo "║                      System Summary                            ║" | tee -a "$REPORT_FILE"
    echo "╚════════════════════════════════════════════════════════════════╝" | tee -a "$REPORT_FILE"
    echo | tee -a "$REPORT_FILE"
    
    echo "Hostname: $(hostname)" | tee -a "$REPORT_FILE"
    echo "Uptime: $(uptime -p)" | tee -a "$REPORT_FILE"
    echo "Load: $(cat /proc/loadavg | awk '{print $1, $2, $3}')" | tee -a "$REPORT_FILE"
    echo "Memory: $(free -h | grep Mem | awk '{print $3 "/" $2}')" | tee -a "$REPORT_FILE"
    echo "Disk: $(df -h / | tail -1 | awk '{print $4 " free (" $5 " used)"}')" | tee -a "$REPORT_FILE"
    echo | tee -a "$REPORT_FILE"
}

send_report() {
    if [[ -z "$ALERT_EMAIL" ]]; then
        return
    fi
    
    log_info "Sending report to $ALERT_EMAIL"
    
    if command -v mail &>/dev/null; then
        mail -s "[VPS Report] $(hostname) - $TIMESTAMP" "$ALERT_EMAIL" < "$REPORT_FILE"
    fi
}

cleanup_reports() {
    log_info "Archiving old reports..."
    find "$LOG_DIR" -name "*report*.txt" -mtime +30 -delete 2>/dev/null || true
}

show_modes() {
    echo "Available modes:"
    echo "  --mode=report      Run monitoring and generate report (default)"
    echo "  --mode=monitor     Run all monitoring tasks"
    echo "  --mode=maintain    Run maintenance tasks"
    echo "  --mode=full        Run complete suite (monitoring + maintenance + cleanup)"
}

main() {
    parse_args "$@"
    
    header
    
    case "$MODE" in
        report)
            run_monitoring
            run_security_audit
            ;;
        monitor)
            run_monitoring
            ;;
        maintain)
            run_maintenance
            ;;
        full)
            run_full_suite
            ;;
        *)
            log_warning "Unknown mode: $MODE"
            show_modes
            exit 1
            ;;
    esac
    
    generate_summary
    send_report
    cleanup_reports
    
    log_success "Report saved: $REPORT_FILE"
    echo
}

main "$@"