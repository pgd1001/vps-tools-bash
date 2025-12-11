# VPS Tools - Complete VPS Management Toolkit

A comprehensive suite of bash scripts for provisioning, monitoring, securing, and maintaining Ubuntu 24.04 VPS deployments. Designed to integrate seamlessly with Docker-based applications and follow industry best practices.

## Features

### Provisioning
- **vps-build.sh** - Complete VPS setup with security hardening, user management, SSH configuration, firewall rules, and optional application installation (Docker, Coolify, Dokploy, n8n, Mailu, Nextcloud, Zentyal)

### Monitoring & Health
- System health dashboard (disk, memory, CPU, swap usage)
- Service status monitoring with auto-restart capability
- Real-time log analysis for errors and anomalies
- Backup verification and integrity checking
- Continuous health monitoring with 5-minute intervals

### Security
- SSH key audit and rotation with weak key detection
- Failed login analysis with brute force detection and IP blocking
- SSL/TLS certificate expiration monitoring across all platforms
- Open port auditor with expected/unexpected port tracking
- GitHub SSH key integration

### Docker Management
- Real-time container health dashboard
- Automatic log rotation with size limits
- Intelligent cleanup of unused images, volumes, networks
- Container and volume backup/restore automation

### Maintenance
- Automated cleanup of logs, temp files, cache
- System and Docker package updates with safety checks
- Database backups (PostgreSQL, MySQL, MongoDB)
- Safe system upgrades with pre-flight checks and rollback capability

### Reporting & Orchestration
- Unified system reports combining all monitoring data
- Scheduled automation via cron jobs
- Email alerting for critical issues
- Centralized logging and log rotation

## Quick Start

### Installation (System-Wide)

```bash
# Clone repository
git clone https://github.com/YOUR_USERNAME/vps-tools.git
cd vps-tools

# Run installer (as root or with sudo)
sudo bash install.sh

# Verify installation
vps-tools help
```

The installer:
- Copies all scripts to `/opt/vps-tools`
- Creates `/etc/vps-tools` config directory
- Creates `/var/log/vps-tools` log directory
- Installs `vps-tools` command for system-wide access
- Creates convenient shortcuts

### Using VPS Tools

**Interactive Menu:**
```bash
vps-tools
```

**Direct Commands:**
```bash
vps-tools health                    # Quick health check
vps-tools build                     # Initial VPS setup
vps-tools docker-health             # Docker status
vps-tools report                    # Full system report
vps-tools help                      # Command reference
```

**With Arguments:**
```bash
sudo vps-tools health --check=docker
sudo vps-tools ssl --warn-days=60
sudo vps-tools report --email=admin@example.com
```

### Initial VPS Setup

```bash
# Interactive provisioning
vps-tools build

# Or direct command
sudo bash /opt/vps-tools/vps-build.sh
```

Follow prompts for hostname, timezone, SSH, user, firewall, and application selection.

### Enable Automated Monitoring

```bash
vps-tools

# Then select option 20: "Install Cron Jobs"
```

Or manually:
```bash
sudo cp /opt/vps-tools/vps-tools-cron.conf /etc/cron.d/vps-tools
sudo nano /etc/cron.d/vps-tools  # Edit email
```

### Generate First Report

```bash
vps-tools report --email=admin@example.com
```

### Uninstall

```bash
sudo bash /opt/vps-tools/install.sh --uninstall
```

Removes scripts and cron jobs but keeps configs and logs.

## Directory Structure

```
vps-tools/
├── README.md                          # This file
├── USAGE.md                           # Detailed usage documentation
├── vps-build.sh                       # Main provisioning script
├── vps-tools-cron.conf               # Cron job configuration
│
├── monitoring/                        # Health & status monitoring
│   ├── vps-health-monitor.sh         # System health (disk, memory, CPU)
│   ├── vps-service-monitor.sh        # Service status with auto-restart
│   ├── vps-log-analyzer.sh           # Log analysis for errors
│   └── vps-backup-verifier.sh        # Backup status & integrity
│
├── security/                          # Security auditing & hardening
│   ├── vps-ssh-audit.sh              # SSH key audit & rotation
│   ├── vps-failed-login-reporter.sh  # Login failure analysis
│   ├── vps-ssl-checker.sh            # Certificate expiration tracking
│   └── vps-open-ports-auditor.sh     # Port scanning & validation
│
├── docker/                            # Docker/container management
│   ├── vps-docker-health.sh          # Container health dashboard
│   ├── vps-docker-log-rotation.sh    # Log size & rotation management
│   ├── vps-docker-cleanup.sh         # Unused image/volume cleanup
│   └── vps-docker-backup-restore.sh  # Container backup automation
│
├── maintenance/                       # System maintenance
│   ├── vps-automated-cleanup.sh      # Cache, logs, temp files
│   ├── vps-package-updater.sh        # Package update management
│   ├── vps-database-backup.sh        # Database backups
│   └── vps-system-upgrade.sh         # Safe OS upgrades
│
└── orchestration/                     # Unified management
    └── vps-orchestration.sh          # Master control & reporting
```

## Usage Examples

### Health Monitoring

```bash
# Quick health check
sudo bash monitoring/vps-health-monitor.sh

# Check specific resource (disk, memory, cpu, services, docker, ssh, firewall, logins, ssl, updates, swap)
sudo bash monitoring/vps-health-monitor.sh --check=docker

# Enable email alerts
sudo bash monitoring/vps-health-monitor.sh --alert-email=admin@example.com
```

### Security Auditing

```bash
# Audit SSH keys for weak algorithms
sudo bash security/vps-ssh-audit.sh --audit

# Rotate SSH keys for user
sudo bash security/vps-ssh-audit.sh --rotate-user=ubuntu

# Analyze failed login attempts
sudo bash security/vps-failed-login-reporter.sh --days=14 --threshold=20

# Block suspicious IPs automatically
sudo bash security/vps-failed-login-reporter.sh --block-ips --alert-email=admin@example.com

# Check certificate expiration
sudo bash security/vps-ssl-checker.sh --warn-days=60
```

### Docker Management

```bash
# Real-time container health dashboard (updates every 60s)
sudo bash docker/vps-docker-health.sh --interval=60

# Auto-restart unhealthy containers
sudo bash docker/vps-docker-health.sh --restart-unhealthy

# Configure log rotation
sudo bash docker/vps-docker-log-rotation.sh --max-size=100m --max-file=5 --apply

# Clean unused resources (dry-run first)
sudo bash docker/vps-docker-cleanup.sh --dry-run
sudo bash docker/vps-docker-cleanup.sh --aggressive  # Remove stopped containers

# Backup all containers
sudo bash docker/vps-docker-backup-restore.sh --mode=backup --containers=all --backup-dir=/opt/backups

# Restore from backup
sudo bash docker/vps-docker-backup-restore.sh --mode=restore --backup-file=/opt/backups/full-backup.tar.gz
```

### Maintenance

```bash
# Check available updates
sudo bash maintenance/vps-package-updater.sh --check

# Update system packages
sudo bash maintenance/vps-package-updater.sh --update-system

# Update Docker and restart containers
sudo bash maintenance/vps-package-updater.sh --update-docker

# Backup all databases
sudo bash maintenance/vps-database-backup.sh --type=all --retention=30

# Backup specific database
sudo bash maintenance/vps-database-backup.sh --type=postgresql

# Clean system files and cache
sudo bash maintenance/vps-automated-cleanup.sh --dry-run
sudo bash maintenance/vps-automated-cleanup.sh --aggressive

# Perform safe system upgrade
sudo bash maintenance/vps-system-upgrade.sh --dry-run  # Test first
sudo bash maintenance/vps-system-upgrade.sh --backup --skip-docker-pull
```

### Reporting & Orchestration

```bash
# Generate system report
sudo bash orchestration/vps-orchestration.sh --mode=report

# Run all monitoring tasks
sudo bash orchestration/vps-orchestration.sh --mode=monitor

# Run full maintenance and cleanup
sudo bash orchestration/vps-orchestration.sh --mode=full --email=admin@example.com
```

## Automation with Cron

### Install Cron Jobs

```bash
# Install provided cron schedule
sudo cp vps-tools-cron.conf /etc/cron.d/vps-tools

# Edit to customize
sudo nano /etc/cron.d/vps-tools

# Set email for alerts
sudo sed -i 's/admin@example.com/your-email@domain.com/g' /etc/cron.d/vps-tools
```

### Cron Schedule Overview

- **Every 5 minutes**: Health monitoring & service checks
- **Daily 2 AM**: Log analysis & security reporting
- **Weekly Sunday 3 AM**: SSH, SSL, and port audits
- **Daily 1 AM**: Database backups with 30-day retention
- **Daily 4-5 AM**: Docker and system cleanup
- **Daily 6 AM**: Package update checks
- **Weekly Monday 11 PM**: Container backups
- **Weekly Sunday 8 AM**: Full system report

### Monitor Cron Execution

```bash
# View cron logs
sudo journalctl -u cron --follow

# Check last execution times
sudo ls -la /var/log/vps-tools/

# Verify cron syntax
sudo crontab -l
```

## Configuration Files

### Global Configuration

Create `/etc/vps-tools/` for centralized configs:

```bash
sudo mkdir -p /etc/vps-tools

# Copy and edit configurations as needed
sudo nano /etc/vps-tools/service-monitor.conf
sudo nano /etc/vps-tools/backup-verifier.conf
```

### Docker Daemon Configuration

Edit `/etc/docker/daemon.json` for global log settings:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "5",
    "labels": "true"
  }
}
```

## Integration with vps-build.sh

All management scripts automatically detect and integrate with vps-build.sh provisioned systems:

- SSH configuration (port, root login, authentication method)
- Firewall rules (UFW) and application-specific ports
- Docker installation and container detection
- User accounts and permissions
- System hardening (sysctl parameters)

## Troubleshooting

### Scripts Not Executable

```bash
chmod +x /opt/vps-tools/**/*.sh
```

### Permission Denied

All scripts require root/sudo:
```bash
sudo bash script-name.sh
```

### Docker Socket Access Issues

Add current user to docker group:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Cron Jobs Not Running

Check cron is enabled:
```bash
sudo systemctl status cron
sudo systemctl enable cron
```

### Email Alerts Not Sending

Install and configure mail:
```bash
sudo apt-get install mailutils
echo "Test" | mail -s "Test" your-email@example.com
```

## Best Practices

### Security
- Run all monitoring scripts via cron as root
- Use HTTPS for remote access to dashboards
- Regularly audit SSH keys and failed logins
- Keep SSL certificates monitored
- Enable firewall and review open ports regularly

### Maintenance
- Schedule backups during off-peak hours
- Test restore procedures monthly
- Run major upgrades on non-critical systems first
- Always use --dry-run before making system changes
- Review logs before and after maintenance

### Monitoring
- Set appropriate alert thresholds for your environment
- Review health reports at least weekly
- Monitor disk usage proactively
- Keep service restart limits to prevent loops
- Archive old logs to manage disk space

## Support & Contributing

For issues, suggestions, or contributions:
1. Check USAGE.md for detailed documentation
2. Review script logs: `sudo tail -f /var/log/vps-tools/*.log`
3. Run troubleshooting checks: `bash monitoring/vps-health-monitor.sh`
4. Test with --dry-run flags before applying changes

## License

MIT

## Author

Created for managing vps-build.sh provisioned Ubuntu 24.04 systems