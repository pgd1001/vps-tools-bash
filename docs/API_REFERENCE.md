# VPS Tools - API Reference

Complete CLI reference for all VPS Tools scripts.

> **Related Docs:** [Quick Start](QUICK_START.md) | [Usage Guide](../USAGE.md) | [Security Guide](SECURITY.md)

---

## Command Dispatcher

The `vps-tools` command provides unified access to all scripts:

```bash
vps-tools [command] [options]
```

### Available Commands

| Command | Script | Description |
|---------|--------|-------------|
| `build` | vps-build.sh | VPS provisioning |
| `health` | vps-health-monitor.sh | System health |
| `services` | vps-service-monitor.sh | Service status |
| `logs` | vps-log-analyzer.sh | Log analysis |
| `backups` | vps-backup-verifier.sh | Backup status |
| `ssh-audit` | vps-ssh-audit.sh | SSH key audit |
| `logins` | vps-failed-login-reporter.sh | Login analysis |
| `ssl` | vps-ssl-checker.sh | SSL monitoring |
| `ports` | vps-open-ports-auditor.sh | Port scanning |
| `docker-health` | vps-docker-health.sh | Container health |
| `docker-logs` | vps-docker-log-rotation.sh | Log rotation |
| `docker-clean` | vps-docker-cleanup.sh | Docker cleanup |
| `docker-backup` | vps-docker-backup-restore.sh | Container backup |
| `cleanup` | vps-automated-cleanup.sh | System cleanup |
| `updates` | vps-package-updater.sh | Package updates |
| `db-backup` | vps-database-backup.sh | Database backup |
| `upgrade` | vps-system-upgrade.sh | System upgrade |
| `report` | vps-orchestration.sh | Full report |

---

## Script Reference

### Monitoring Scripts

#### vps-health-monitor.sh

```
Options:
  --check=RESOURCE      Check specific resource
                        Values: disk, memory, cpu, services, docker,
                               ssh, firewall, logins, ssl, updates, swap
  --alert-email=EMAIL   Send alerts to email

Exit Codes:
  0   All checks passed
  1   Warnings or critical issues
```

#### vps-service-monitor.sh

```
Options:
  --config=PATH        Custom config file
  --dry-run            No changes, show only
  --alert-email=EMAIL  Send alerts

Config: /etc/vps-tools/service-monitor.conf
```

#### vps-log-analyzer.sh

```
Options:
  --days=N             Days to analyze (default: 7)
  --type=TYPE          auth, service, system, all
  --alert-email=EMAIL  Send alerts
```

#### vps-backup-verifier.sh

```
Options:
  --config=PATH        Custom config file
  --verify-all         Verify checksums
  --alert-email=EMAIL  Send alerts

Config: /etc/vps-tools/backup-verifier.conf
```

---

### Security Scripts

#### vps-ssh-audit.sh

```
Options:
  --audit              Audit all SSH keys (default)
  --rotate-user=USER   Generate new keys
  --alert-email=EMAIL  Send alerts
```

#### vps-failed-login-reporter.sh

```
Options:
  --days=N             Days to analyze (default: 7)
  --threshold=N        Alert threshold (default: 10)
  --block-ips          Block suspicious IPs
  --alert-email=EMAIL  Send alerts
```

#### vps-ssl-checker.sh

```
Options:
  --warn-days=N        Warning threshold (default: 30)
  --alert-email=EMAIL  Send alerts
```

#### vps-open-ports-auditor.sh

```
Options:
  --expected-ports=LIST  Comma-separated expected ports
  --scan-external        External port scan (requires nmap)
  --alert-email=EMAIL    Send alerts
```

---

### Docker Scripts

#### vps-docker-health.sh

```
Options:
  --interval=N          Refresh interval (default: 60s)
  --restart-unhealthy   Auto-restart unhealthy containers
  --alert-email=EMAIL   Send alerts
```

#### vps-docker-cleanup.sh

```
Options:
  --dry-run       Show what would be removed
  --aggressive    Also remove stopped containers
  --prune-all     Full system prune
```

#### vps-docker-log-rotation.sh

```
Options:
  --max-size=SIZE   Max log size (default: 100m)
  --max-file=N      Max files (default: 5)
  --apply           Apply changes
  --alert-email=EMAIL  Send alerts
```

#### vps-docker-backup-restore.sh

```
Options:
  --mode=MODE           backup, restore, status (required)
  --containers=LIST     Container names or 'all'
  --volumes=LIST        Volume names or 'all'
  --backup-dir=PATH     Backup directory
  --backup-file=PATH    Backup file for restore
  --retention-days=N    Days to keep (default: 30)
  --alert-email=EMAIL   Send notification
```

---

### Maintenance Scripts

#### vps-automated-cleanup.sh

```
Options:
  --dry-run       Show what would be removed
  --aggressive    Enable all cleanup
```

#### vps-package-updater.sh

```
Options:
  --check           Check updates (default)
  --update-system   Upgrade packages
  --update-docker   Update Docker
  --alert-email=EMAIL  Send alerts
```

#### vps-database-backup.sh

```
Options:
  --type=TYPE        all, postgresql, mysql, mongodb
  --backup-dir=PATH  Backup directory
  --retention=N      Days to keep (default: 30)
  --alert-email=EMAIL  Send notification
```

#### vps-system-upgrade.sh

```
Options:
  --dry-run            Show what would happen
  --backup             Create backup first
  --skip-docker-pull   Skip image updates
```

---

### Orchestration

#### vps-orchestration.sh

```
Options:
  --mode=MODE      report, monitor, maintain, full
  --email=EMAIL    Send report via email
```

---

## Configuration Files

| File | Purpose |
|------|---------|
| `/etc/vps-tools/service-monitor.conf` | Services to monitor |
| `/etc/vps-tools/backup-verifier.conf` | Backup locations |
| `/etc/docker/daemon.json` | Docker log settings |

---

## Log Files

All logs: `/var/log/vps-tools/`

| Log | Script |
|-----|--------|
| `health-cron.log` | vps-health-monitor.sh |
| `docker-health.log` | vps-docker-health.sh |
| `cleanup.log` | vps-automated-cleanup.sh |
| `database-backup.log` | vps-database-backup.sh |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Warnings/errors |
| 2 | Config error |
| 126 | Permission denied |
| 127 | Command not found |
