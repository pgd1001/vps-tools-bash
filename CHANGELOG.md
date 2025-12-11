# Changelog

All notable changes to VPS Tools will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2024-12-11

### Changed
- **BREAKING**: Renamed all script files from underscores to hyphens
  - `vps_health_monitor.sh` → `vps-health-monitor.sh`
  - `vps_service_monitor.sh` → `vps-service-monitor.sh`
  - `vps_log_analyzer.sh` → `vps-log-analyzer.sh`
  - `vps_backup_verifier.sh` → `vps-backup-verifier.sh`
  - `vps_ssh_audit.sh` → `vps-ssh-audit.sh`
  - `vps_failed_login_reporter.sh` → `vps-failed-login-reporter.sh`
  - `vps_ssl_checker.sh` → `vps-ssl-checker.sh`
  - `vps_open_ports_auditor.sh` → `vps-open-ports-auditor.sh`
  - `vps_docker_health.sh` → `vps-docker-health.sh`
  - `vps_docker_cleanup.sh` → `vps-docker-cleanup.sh`
  - `vps_docker_log_rotation.sh` → `vps-docker-log-rotation.sh`
  - `vps_docker_backup_restore.sh` → `vps-docker-backup-restore.sh`
  - `vps_automated_cleanup.sh` → `vps-automated-cleanup.sh`
  - `vps_database_backup.sh` → `vps-database-backup.sh`
  - `vps_package_updater.sh` → `vps-package-updater.sh`
  - `vps_system_upgrade.sh` → `vps-system-upgrade.sh`
  - `vps_orchestration.sh` → `vps-orchestration.sh`

### Fixed
- Cron config now references correct `vps-orchestration.sh` script
- System hardening no longer duplicates sysctl entries on re-run
- SSH config error now provides better recovery instructions
- Memory calculation now handles all memory sizes (not just GB)

### Added
- CONTRIBUTING.md with contribution guidelines
- CHANGELOG.md to track version history
- docs/ folder with API reference

## [1.0.0] - 2024-12-01

### Added
- Initial release of VPS Tools

#### Provisioning
- `vps-build.sh` - Complete VPS setup with security hardening

#### Monitoring
- `vps-health-monitor.sh` - System health monitoring
- `vps-service-monitor.sh` - Service status with auto-restart
- `vps-log-analyzer.sh` - Log analysis for errors
- `vps-backup-verifier.sh` - Backup status verification

#### Security
- `vps-ssh-audit.sh` - SSH key audit and rotation
- `vps-failed-login-reporter.sh` - Login failure analysis
- `vps-ssl-checker.sh` - Certificate expiration tracking
- `vps-open-ports-auditor.sh` - Port scanning and validation

#### Docker
- `vps-docker-health.sh` - Container health dashboard
- `vps-docker-cleanup.sh` - Unused resource cleanup
- `vps-docker-log-rotation.sh` - Log rotation management
- `vps-docker-backup-restore.sh` - Container backup automation

#### Maintenance
- `vps-automated-cleanup.sh` - System cleanup
- `vps-package-updater.sh` - Package update management
- `vps-database-backup.sh` - Database backups
- `vps-system-upgrade.sh` - Safe OS upgrades

#### Orchestration
- `vps-orchestration.sh` - Master control and reporting
- `vps-tools-cron.conf` - Cron job configuration
- `install.sh` - System-wide installation

---

## Upgrade Notes

### Upgrading from 1.0.0 to 1.1.0

If you have an existing installation, run the installer again to update:

```bash
cd vps-tools-bash
git pull
sudo bash install.sh
```

If you have custom cron jobs referencing old script names, update them:

```bash
# Old (will break)
/opt/vps-tools/monitoring/vps_health_monitor.sh

# New (correct)
/opt/vps-tools/monitoring/vps-health-monitor.sh
```
