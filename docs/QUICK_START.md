# VPS Tools - Quick Start Guide

Get up and running with VPS Tools in 5 minutes.

## Prerequisites

- Fresh Ubuntu 24.04 VPS
- Root or sudo access
- SSH connection to server

## Installation

### 1. Connect to Your Server

```bash
ssh root@your-server-ip
```

### 2. Clone and Install

```bash
git clone https://github.com/YOUR_USERNAME/vps-tools-bash.git
cd vps-tools-bash
sudo bash install.sh
```

### 3. Verify Installation

```bash
vps-tools help
```

You should see a list of available commands.

## First Steps

### Run Initial VPS Setup

```bash
vps-tools build
```

Follow the interactive prompts to:
- Set hostname and timezone
- Configure SSH security
- Create a non-root user
- Enable firewall
- Install Docker or other applications

### Check System Health

```bash
vps-tools health
```

Displays disk, memory, CPU, and service status.

### Enable Automated Monitoring

```bash
vps-tools
# Select option 20: "Install Cron Jobs"
```

Or manually:
```bash
sudo cp /opt/vps-tools/vps-tools-cron.conf /etc/cron.d/vps-tools
sudo nano /etc/cron.d/vps-tools  # Update email address
```

## Most Common Commands

| Command | Description |
|---------|-------------|
| `vps-tools` | Interactive menu |
| `vps-tools health` | Quick health check |
| `vps-tools report` | Full system report |
| `vps-tools docker-health` | Docker container status |
| `vps-tools ssl` | SSL certificate check |
| `vps-tools logins` | Failed login analysis |

## Next Steps

1. **Read the full documentation**: [USAGE.md](../USAGE.md)
2. **Configure email alerts**: Update `MAILTO` in cron config
3. **Set up backups**: Run `vps-tools db-backup` for databases
4. **Review security**: Run `vps-tools ssh-audit`

## Getting Help

```bash
# Command reference
vps-tools help

# View documentation
less /opt/vps-tools/USAGE.md

# Check logs
sudo tail -f /var/log/vps-tools/*.log

# Run with debugging
sudo bash -x /opt/vps-tools/script.sh
```

## Troubleshooting

### Permission Denied

```bash
# Always run with sudo
sudo vps-tools health
```

### Command Not Found

```bash
# Re-run installer
cd /path/to/vps-tools-bash
sudo bash install.sh
```

### Docker Issues

```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```
