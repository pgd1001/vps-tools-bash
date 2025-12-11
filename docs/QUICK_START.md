# VPS Tools - Quick Start Guide

Get VPS Tools up and running in 5 minutes.

> **Related Docs:** [Usage Guide](../USAGE.md) | [API Reference](API_REFERENCE.md) | [Security Guide](SECURITY.md)

---

## Prerequisites

- Fresh Ubuntu 24.04 VPS
- Root or sudo access
- SSH connection

## Step 1: Install

```bash
# Connect to your server
ssh root@your-server-ip

# Clone repository
git clone https://github.com/YOUR_USERNAME/vps-tools-bash.git
cd vps-tools-bash

# Install
sudo bash install.sh

# Verify
vps-tools help
```

## Step 2: Initial VPS Setup

```bash
vps-tools build
```

Follow the prompts:
- Hostname and timezone
- SSH port and security
- Create non-root user
- Enable firewall
- Install Docker or applications

## Step 3: Check System Health

```bash
vps-tools health
```

## Step 4: Enable Automation

```bash
# Install cron jobs
sudo cp vps-tools-cron.conf /etc/cron.d/vps-tools

# Configure email alerts
sudo nano /etc/cron.d/vps-tools
# Change: MAILTO=admin@example.com
```

## Step 5: Generate First Report

```bash
vps-tools report
```

---

## Quick Reference

| Command | Description |
|---------|-------------|
| `vps-tools` | Interactive menu |
| `vps-tools health` | System health check |
| `vps-tools build` | VPS provisioning |
| `vps-tools report` | Full system report |
| `vps-tools docker-health` | Container status |
| `vps-tools help` | Command reference |

---

## Next Steps

1. **[Usage Guide](../USAGE.md)** - Detailed documentation for all scripts
2. **[Security Guide](SECURITY.md)** - Security best practices
3. **[API Reference](API_REFERENCE.md)** - Complete CLI options

---

## Getting Help

```bash
# View documentation
vps-tools help

# Check logs
sudo tail -f /var/log/vps-tools/*.log

# Debug mode
sudo bash -x /opt/vps-tools/script.sh
```
