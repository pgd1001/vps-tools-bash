# VPS Provisioning Script - Usage Guide

## Overview

This script automates Ubuntu 24.04 VPS provisioning with best-practice security hardening, application installation, and system reconfiguration capabilities.

## Installation

```bash
git clone https://github.com/YOUR_USERNAME/vps-tools.git
cd vps-tools
chmod +x vps-build.sh
```

## Usage

Run as root or with sudo:

```bash
sudo bash vps-build.sh
```

## Script Modes

### Mode 1: Fresh Install

Performs full VPS provisioning from initial setup through application installation.

**Configuration prompts:**
- Hostname (auto-generates default from IP)
- Timezone (e.g., Europe/Dublin, UTC)
- SSH port (default: 22)
- Block root SSH login (default: yes - uses `prohibit-password` for Coolify compatibility)
- Create non-root sudo user (default: yes)
- SSH keys from GitHub (optional - enter GitHub username)
- Disable password authentication (default: yes)
- Enable UFW firewall (default: yes)
- Create swap space (default: yes, size configurable)
- Unattended security updates (default: yes)
- Application selection (see below)

**What it does:**
- System update (apt upgrade, dist-upgrade)
- Configures hostname, timezone, SSH
- Creates non-root user with optional GitHub SSH keys
- Hardens system (sysctl parameters, kernel tuning)
- Configures firewall with app-specific rules
- Installs selected application
- Applies security best practices

**Important:** Fresh install does NOT wipe existing data/services. It applies configuration on top of current system state.

### Mode 2: Reconfigure

Update specific system components without full provisioning.

**Options:**
- **SSH keys from GitHub** - Add/update keys for existing user
- **SSH configuration** - Adjust port, root login, password auth
- **Firewall rules** - Add custom rules, delete rules, reset, enable/disable
- **Hostname** - Change system hostname
- **Timezone** - Change system timezone

### Mode 3: Troubleshooting/Status

View system configuration summary and adjust settings interactively.

**Displays:**
- Hostname
- Timezone
- SSH configuration (port, root login, password auth)
- System users
- UFW firewall status and rules
- Swap configuration
- Running services

**Allows adjustments** to any of the above settings from the status view.

## Application Options

Install one of:

1. **Docker + Portainer** - Container runtime with web UI (port 9000)
2. **Coolify** - Self-hosted PaaS (ports 80, 443, 8000)
3. **Dokploy** - Container deployment platform (port 3000)
4. **n8n** - Workflow automation (port 5678)
5. **Mailu** - Mail server (SMTP, IMAP, POP3 + web ports)
6. **Nextcloud** - File sync and sharing (port 80)
7. **Zentyal** - Network infrastructure management (port 443)
8. **None** - Base system only

## SSH Configuration

### Root Login
- **prohibit-password** - Allows key-based auth, blocks password (default, Coolify compatible)
- **yes** - Allows all authentication methods
- **no** - Blocks all root access

### Authentication
- **Key-based (recommended)** - GitHub SSH keys auto-populated during setup
- **Password-based** - Manual password entry if keys not available
- **Disabled** - Both disabled, script will disable after user creation

## Firewall Rules

Default rules applied:
- SSH port (configurable)
- Application-specific ports (auto-configured)
- Deny all incoming, allow all outgoing

UFW can be:
- Managed via reconfigure mode (option 3)
- Reset to defaults
- Enabled/disabled
- Updated with custom rules

## Security Features

- Disables root password login (allows key-based auth)
- Disables SSH password authentication
- Limits SSH connection attempts (MaxAuthTries: 3, MaxSessions: 5)
- X11 forwarding disabled
- UFW firewall enabled with sensible defaults
- Kernel hardening (SYN cookies, ICMP redirect rejection)
- Unattended security updates enabled
- Automatic package cleanup

## SSH Key Setup

### Using GitHub Keys (Recommended)

During user creation, opt to add GitHub SSH keys:
- Script prompts for GitHub username
- Automatically fetches public keys from `https://github.com/{username}.keys`
- Sets proper permissions (700 on .ssh, 600 on authorized_keys)

### Manual Key Setup

If not using GitHub:
1. Generate key locally: `ssh-keygen -t ed25519`
2. Add public key to `~/.ssh/authorized_keys` on server
3. Set permissions: `chmod 600 ~/.ssh/authorized_keys`

## Troubleshooting

### Lost SSH Access

If locked out:
1. VPS provider console access
2. Check SSH config: `cat /etc/ssh/sshd_config`
3. Check backups: `ls -la /etc/ssh/sshd_config.backup*`
4. Run mode 3 to review/adjust SSH settings

### Service Status

Check running services:
```bash
sudo systemctl status [service-name]
sudo docker ps  # For containerized apps
```

### Firewall Issues

Review rules:
```bash
sudo ufw status numbered
```

Add rule:
```bash
sudo ufw allow 8080/tcp
```

Delete rule:
```bash
sudo ufw delete allow 8080/tcp
```

## Re-running the Script

Safe to run multiple times. Use appropriate mode:
- **Fresh install** to apply full config again
- **Reconfigure** to adjust specific settings
- **Troubleshoot** to review and fix issues

## File Locations

- SSH config: `/etc/ssh/sshd_config`
- SSH backups: `/etc/ssh/sshd_config.backup.*`
- Firewall config: `/etc/ufw/`
- System hardening: `/etc/sysctl.conf`
- Docker apps: `/opt/[app-name]/`