# VPS Tools - Complete VPS Management Toolkit

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Ubuntu 24.04](https://img.shields.io/badge/Ubuntu-24.04-orange.svg)](https://ubuntu.com/)
[![Bash](https://img.shields.io/badge/Bash-5.0+-green.svg)](https://www.gnu.org/software/bash/)

A comprehensive suite of bash scripts for provisioning, monitoring, securing, and maintaining Ubuntu 24.04 VPS deployments. Designed to integrate seamlessly with Docker-based applications.

## 📚 Documentation

| Document | Description |
|----------|-------------|
| **[Quick Start](docs/QUICK_START.md)** | Get running in 5 minutes |
| **[Usage Guide](USAGE.md)** | Detailed script documentation |
| **[API Reference](docs/API_REFERENCE.md)** | Complete CLI reference |
| **[Security Guide](docs/SECURITY.md)** | Security best practices |
| **[Contributing](CONTRIBUTING.md)** | How to contribute |
| **[Changelog](CHANGELOG.md)** | Version history |

## ✨ Features

| Category | Capabilities |
|----------|-------------|
| **Provisioning** | Complete VPS setup, SSH hardening, firewall, user management |
| **Monitoring** | Health checks, service monitoring, log analysis, backup verification |
| **Security** | SSH key audit, failed login detection, SSL monitoring, port scanning |
| **Docker** | Container health, log rotation, cleanup, backup/restore |
| **Maintenance** | System cleanup, package updates, database backups, upgrades |
| **Orchestration** | Unified reporting, cron automation, email alerts |

## 🚀 Quick Start

```bash
# Clone and install
git clone https://github.com/YOUR_USERNAME/vps-tools-bash.git
cd vps-tools-bash
sudo bash install.sh

# Verify
vps-tools help

# Initial VPS setup
vps-tools build

# Health check
vps-tools health
```

**→ See [Quick Start Guide](docs/QUICK_START.md) for detailed setup instructions.**

## 📁 Directory Structure

```
vps-tools-bash/
├── vps-build.sh                    # Main provisioning script
├── install.sh                      # Installer with plugin support
├── plugins.conf.default            # Default plugin registry
├── vps-tools-cron.conf             # Cron configuration
│
├── custom/                         # User custom scripts (gitignored)
│   └── README.md
│
├── monitoring/
│   ├── vps-health-monitor.sh       # System health
│   ├── vps-service-monitor.sh      # Service status
│   ├── vps-log-analyzer.sh         # Log analysis
│   └── vps-backup-verifier.sh      # Backup verification
│
├── security/
│   ├── vps-ssh-audit.sh            # SSH key audit
│   ├── vps-failed-login-reporter.sh # Login analysis
│   ├── vps-ssl-checker.sh          # SSL monitoring
│   └── vps-open-ports-auditor.sh   # Port scanning
│
├── docker/
│   ├── vps-docker-health.sh        # Container health
│   ├── vps-docker-cleanup.sh       # Resource cleanup
│   ├── vps-docker-log-rotation.sh  # Log rotation
│   └── vps-docker-backup-restore.sh # Backup automation
│
├── maintenance/
│   ├── vps-automated-cleanup.sh    # System cleanup
│   ├── vps-package-updater.sh      # Package updates
│   ├── vps-database-backup.sh      # Database backups
│   └── vps-system-upgrade.sh       # Safe upgrades
│
├── orchestration/
│   └── vps-orchestration.sh        # Master control
│
└── docs/
    ├── QUICK_START.md              # Getting started
    ├── API_REFERENCE.md            # CLI reference
    └── SECURITY.md                 # Security guide
```

## 🔧 Common Commands

```bash
# Interactive menu
vps-tools

# Quick commands
vps-tools health          # System health check
vps-tools build           # VPS provisioning
vps-tools report          # Full system report
vps-tools docker-health   # Container status
vps-tools ssl             # SSL certificate check
vps-tools logins          # Failed login analysis
```

**→ See [Usage Guide](USAGE.md) for all commands and options.**

## 🔌 Plugin System

Add, disable, or replace scripts through the plugin registry:

```bash
# List all enabled plugins
vps-tools plugin list

# Add custom script
vps-tools plugin add my-backup custom/my-backup.sh "My backup script"

# Disable a script
vps-tools plugin disable ports

# Enable a script
vps-tools plugin enable ports
```

**Custom Scripts:**
- Place in `/opt/vps-tools/custom/`
- Register in `/etc/vps-tools/plugins.conf`
- Survives updates (gitignored)

## ⏰ Automation

Install cron jobs for automated monitoring:

```bash
sudo cp vps-tools-cron.conf /etc/cron.d/vps-tools
sudo nano /etc/cron.d/vps-tools  # Update MAILTO
```

| Schedule | Task |
|----------|------|
| Every 5 min | Health monitoring |
| Daily 2 AM | Log analysis, security |
| Daily 1 AM | Database backups |
| Weekly | SSH audit, SSL check, ports audit |

## 🔒 Security

- SSH key-based authentication only
- UFW firewall with minimal rules
- Failed login detection and IP blocking
- SSL certificate monitoring
- System hardening (sysctl)

**→ See [Security Guide](docs/SECURITY.md) for best practices.**

## 🐛 Troubleshooting

```bash
# Permission denied
sudo vps-tools health

# Command not found
sudo bash install.sh

# View logs
sudo tail -f /var/log/vps-tools/*.log

# Debug mode
sudo bash -x /opt/vps-tools/script.sh
```

## 📄 License

MIT License - See [LICENSE](LICENSE) for details.

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.