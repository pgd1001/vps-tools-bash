# VPS Tools - API Reference

Complete reference for all VPS Tools scripts and their command-line options.

---

## Table of Contents

- [Core Scripts](#core-scripts)
- [Monitoring Scripts](#monitoring-scripts)
- [Security Scripts](#security-scripts)
- [Docker Scripts](#docker-scripts)
- [Maintenance Scripts](#maintenance-scripts)
- [Orchestration Scripts](#orchestration-scripts)
- [Configuration Files](#configuration-files)

---

## Core Scripts

### install.sh

System-wide installation and uninstallation.

```
Usage: sudo bash install.sh [--uninstall]

Options:
  (none)       Install VPS Tools to /opt/vps-tools
  --uninstall  Remove VPS Tools (keeps configs and logs)

Installs:
  /opt/vps-tools/        All scripts
  /etc/vps-tools/        Configuration directory
  /var/log/vps-tools/    Log directory
  /usr/local/bin/vps-tools   Command dispatcher
```

### vps-build.sh

VPS provisioning and configuration.

```
Usage: sudo bash vps-build.sh

Modes:
  1) Fresh install     Complete provisioning
  2) Reconfigure       Modify specific settings
  3) Troubleshoot      Status review and fixes

Interactive prompts for:
  - Hostname and timezone
  - SSH port and security
  - User creation
  - GitHub SSH keys
  - UFW firewall
  - Swap configuration
  - Application installation
```

---

## Monitoring Scripts

### vps-health-monitor.sh

```
Usage: sudo bash vps-health-monitor.sh [OPTIONS]

Options:
  --check=RESOURCE      Check specific resource only
                        Values: disk, memory, cpu, services, docker,
                               ssh, firewall, logins, ssl, updates, swap
  --alert-email=EMAIL   Send alerts to email address

Exit Codes:
  0   All checks passed
  1   Warnings or critical issues found
```

### vps-service-monitor.sh

```
Usage: sudo bash vps-service-monitor.sh [OPTIONS]

Options:
  --config=PATH        Path to custom config file
  --dry-run           Show what would happen without making changes
  --alert-email=EMAIL  Send alerts on restart/failure

Config: /etc/vps-tools/service-monitor.conf
```

### vps-log-analyzer.sh

```
Usage: sudo bash vps-log-analyzer.sh [OPTIONS]

Options:
  --days=N            Analyze last N days (default: 7)
  --type=TYPE         Log type to analyze
                      Values: auth, service, system, all (default)
  --alert-email=EMAIL Send alerts if issues found
```

### vps-backup-verifier.sh

```
Usage: sudo bash vps-backup-verifier.sh [OPTIONS]

Options:
  --config=PATH       Path to custom config file
  --verify-all        Verify backup checksums
  --alert-email=EMAIL Send alerts on issues

Config: /etc/vps-tools/backup-verifier.conf
```

---

## Security Scripts

### vps-ssh-audit.sh

```
Usage: sudo bash vps-ssh-audit.sh [OPTIONS]

Options:
  --audit               Audit all SSH keys (default)
  --rotate-user=USER    Generate new keys for user
  --alert-email=EMAIL   Send alerts on security issues
```

### vps-failed-login-reporter.sh

```
Usage: sudo bash vps-failed-login-reporter.sh [OPTIONS]

Options:
  --days=N             Analyze last N days (default: 7)
  --threshold=N        Alert threshold for attempts (default: 10)
  --block-ips          Automatically block suspicious IPs
  --alert-email=EMAIL  Send alerts on suspicious activity

Blocked IPs: /etc/vps-tools/blocked-ips.txt
```

### vps-ssl-checker.sh

```
Usage: sudo bash vps-ssl-checker.sh [OPTIONS]

Options:
  --warn-days=N       Days before expiry to warn (default: 30)
  --alert-email=EMAIL Send alerts on expiring certificates

Checks:
  - /etc/letsencrypt/live/
  - /etc/docker/certs.d/
  - /etc/nginx/certs/
  - /etc/apache2/certs/
```

### vps-open-ports-auditor.sh

```
Usage: sudo bash vps-open-ports-auditor.sh [OPTIONS]

Options:
  --expected-ports=LIST  Comma-separated expected ports (e.g., 22,80,443)
  --scan-external        Perform external port scan (requires nmap)
  --alert-email=EMAIL    Send alerts on unexpected ports
```

---

## Docker Scripts

### vps-docker-health.sh

```
Usage: sudo bash vps-docker-health.sh [OPTIONS]

Options:
  --interval=N          Refresh interval in seconds (default: 60)
  --restart-unhealthy   Automatically restart unhealthy containers
  --alert-email=EMAIL   Send alerts on issues

Note: Runs continuously until Ctrl+C
```

### vps-docker-cleanup.sh

```
Usage: sudo bash vps-docker-cleanup.sh [OPTIONS]

Options:
  --dry-run      Show what would be removed without removing
  --aggressive   Also remove stopped containers
  --prune-all    Full system prune (removes all unused)

Cleanup order:
  1. Dangling images
  2. Unused images
  3. Unused volumes
  4. Unused networks
  5. Stopped containers (--aggressive)
  6. Build cache
  7. System prune (--prune-all)
```

### vps-docker-log-rotation.sh

```
Usage: sudo bash vps-docker-log-rotation.sh [OPTIONS]

Options:
  --max-size=SIZE   Maximum log size (default: 100m)
  --max-file=N      Maximum log files to keep (default: 5)
  --apply           Apply configuration changes
  --alert-email=EMAIL Send alerts on issues

Config: /etc/docker/daemon.json
```

### vps-docker-backup-restore.sh

```
Usage: sudo bash vps-docker-backup-restore.sh --mode=MODE [OPTIONS]

Modes:
  backup    Create backup
  restore   Restore from backup
  status    Show backup inventory

Options:
  --containers=LIST      Container names or 'all' (default: all)
  --volumes=LIST         Volume names or 'all'
  --backup-dir=PATH      Backup directory (default: /opt/backups)
  --backup-file=PATH     Backup file for restore
  --retention-days=N     Days to keep backups (default: 30)
  --alert-email=EMAIL    Send notification on completion
```

---

## Maintenance Scripts

### vps-automated-cleanup.sh

```
Usage: sudo bash vps-automated-cleanup.sh [OPTIONS]

Options:
  --dry-run      Show what would be removed
  --aggressive   Enable all cleanup operations

Cleanup targets:
  - Rotated logs (>30 days)
  - Journal logs (>2 weeks)
  - Temp files (>7 days)
  - APT cache
  - Broken symlinks
  - Core dumps (--aggressive)
  - Old backups (--aggressive)
```

### vps-package-updater.sh

```
Usage: sudo bash vps-package-updater.sh [OPTIONS]

Options:
  --check           Check for available updates (default)
  --update-system   Upgrade system packages
  --update-docker   Update Docker and pull images
  --alert-email=EMAIL Send alerts on security updates
```

### vps-database-backup.sh

```
Usage: sudo bash vps-database-backup.sh [OPTIONS]

Options:
  --type=TYPE        Database type
                     Values: all (default), postgresql, mysql, mongodb
  --backup-dir=PATH  Backup directory (default: /opt/backups)
  --retention=N      Days to keep backups (default: 30)
  --alert-email=EMAIL Send notification on completion

Backup locations:
  /opt/backups/postgresql/
  /opt/backups/mysql/
  /opt/backups/mongodb/
```

### vps-system-upgrade.sh

```
Usage: sudo bash vps-system-upgrade.sh [OPTIONS]

Options:
  --dry-run            Show what would happen
  --backup             Create backup before upgrade
  --skip-docker-pull   Skip Docker image updates

Process:
  1. Pre-flight checks (disk space, services)
  2. Backup (optional)
  3. Stop services
  4. Upgrade packages
  5. Restart services
  6. Verify health
```

---

## Orchestration Scripts

### vps-orchestration.sh

```
Usage: sudo bash vps-orchestration.sh --mode=MODE [OPTIONS]

Modes:
  report    Run monitoring + security audits (default)
  monitor   Run monitoring tasks only
  maintain  Run maintenance tasks only
  full      Run all tasks + cleanup

Options:
  --email=EMAIL  Send report via email

Report: /tmp/vps-system-report-TIMESTAMP.txt
```

---

## Configuration Files

### /etc/vps-tools/service-monitor.conf

```bash
# Service monitoring format: "name:type:max_restarts:delay"
SERVICES=(
    "ssh:systemd:0:60"
    "docker:systemd:3:120"
)

# Container monitoring format: "name:max_restarts:delay"
DOCKER_CONTAINERS=(
    "coolify:3:120"
)
```

### /etc/vps-tools/backup-verifier.conf

```bash
# Backup locations format: "path:type:retention:max_age"
BACKUP_LOCATIONS=(
    "/opt/backups:archive:720:168"
)

# Docker volumes format: "volume:backup_path"
DOCKER_VOLUMES=(
    "portainer_data:/opt/backups/portainer"
)

# Databases format: "type:name:backup_path"
DATABASES=(
    "postgresql:postgres:/opt/backups/postgres"
)
```

### /etc/docker/daemon.json

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "5"
  }
}
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success / No issues |
| 1 | Warnings or errors found |
| 2 | Configuration error |
| 126 | Permission denied |
| 127 | Command not found |

## Log Files

All scripts log to `/var/log/vps-tools/`:

| Log File | Script |
|----------|--------|
| `health-cron.log` | vps-health-monitor.sh |
| `service-monitor.log` | vps-service-monitor.sh |
| `backup-verifier.log` | vps-backup-verifier.sh |
| `ssh-audit.log` | vps-ssh-audit.sh |
| `failed-login-report.log` | vps-failed-login-reporter.sh |
| `ssl-checker.log` | vps-ssl-checker.sh |
| `ports-audit.log` | vps-open-ports-auditor.sh |
| `docker-health.log` | vps-docker-health.sh |
| `docker-cleanup.log` | vps-docker-cleanup.sh |
| `docker-logs.log` | vps-docker-log-rotation.sh |
| `docker-backup.log` | vps-docker-backup-restore.sh |
| `cleanup.log` | vps-automated-cleanup.sh |
| `package-updates.log` | vps-package-updater.sh |
| `database-backup.log` | vps-database-backup.sh |
| `system-upgrade.log` | vps-system-upgrade.sh |
