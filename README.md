# vps-tools

VPS provisioning and configuration scripts for Ubuntu 24.04. Automates system hardening, security configuration, and application deployment.

## Features

- **Fresh Installation** - Complete VPS setup with security hardening
- **Reconfiguration** - Update system settings without full reprovisioning
- **Troubleshooting Mode** - Review system status and make targeted adjustments
- **Security-First** - SSH hardening, firewall, kernel parameters, unattended updates
- **GitHub SSH Integration** - Auto-populate SSH keys from GitHub user profiles
- **Application Support** - Docker, Coolify, Dokploy, n8n, Mailu, Nextcloud, Zentyal
- **Re-runnable** - Safe to run multiple times on existing systems

## Quick Start

```bash
git clone https://github.com/YOUR_USERNAME/vps-tools.git
cd vps-tools
chmod +x vps-build.sh
sudo bash vps-build.sh
```

## Script Modes

| Mode | Purpose |
|------|---------|
| 1. Fresh Install | Full VPS provisioning from initial setup |
| 2. Reconfigure | Update specific system components |
| 3. Troubleshoot | Review system status and make adjustments |

## Configuration Highlights

- Hostname and timezone setup
- SSH hardening (prohibit-password root login, key-based auth, MaxAuthTries/MaxSessions)
- Non-root sudo user creation
- GitHub SSH key integration
- UFW firewall with app-specific rules
- Swap space configuration
- Unattended security updates
- Kernel hardening parameters

## Supported Applications

- Docker + Portainer
- Coolify
- Dokploy
- n8n
- Mailu (mail server)
- Nextcloud
- Zentyal

## SSH Security

By default, the script:
- Disables password authentication
- Enables key-based auth only (Coolify compatible with `prohibit-password`)
- Limits connection attempts
- Disables root password login
- Disables X11 forwarding

GitHub SSH keys can be auto-populated during setup—just provide your GitHub username.

## Firewall

UFW (Uncomplicated Firewall) enabled by default with:
- Deny all incoming traffic
- Allow outgoing traffic
- SSH port whitelisted
- Application-specific ports auto-configured
- Reconfigurable via script or manual commands

## Reconfiguration Examples

Update GitHub SSH keys for existing user:
```bash
sudo bash vps-build.sh
# Select mode 2 → Option 1 → SSH keys from GitHub
```

Adjust firewall rules:
```bash
sudo bash vps-build.sh
# Select mode 3 → Option 3 → Manage firewall rules
```

Reconfigure SSH (port, root login, auth):
```bash
sudo bash vps-build.sh
# Select mode 2 → Option 2 → SSH configuration
```

## Requirements

- Ubuntu 24.04 LTS
- Root or sudo access
- Internet connectivity

## Documentation

See [USAGE.md](./USAGE.md) for comprehensive instructions and troubleshooting.

## Notes

- Fresh install does NOT wipe existing data/services
- SSH backups automatically created before modifications
- All configuration changes can be reverted via backups
- Safe to re-run on existing systems

## License

MIT

## Support

For issues or questions, check the troubleshooting section in USAGE.md.