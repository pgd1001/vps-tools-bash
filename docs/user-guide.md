# User Guide

This comprehensive guide covers all aspects of using VPS Tools, from basic setup to advanced features.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Server Management](#server-management)
3. [Health Monitoring](#health-monitoring)
4. [Command Execution](#command-execution)
5. [Security Auditing](#security-auditing)
6. [Docker Management](#docker-management)
7. [System Maintenance](#system-maintenance)
8. [TUI Interface](#tui-interface)
9. [Configuration](#configuration)
10. [Advanced Usage](#advanced-usage)

## Getting Started

### Initial Setup

1. **Install VPS Tools** (see README.md for installation instructions)

2. **Initialize Configuration**
   ```bash
   vps-tools config init
   ```
   This creates the default configuration file and database.

3. **Verify Installation**
   ```bash
   vps-tools --version
   vps-tools config show
   ```

### First Server Addition

Add your first server to the inventory:

```bash
vps-tools inventory add \
  --name "production-web" \
  --host "192.168.1.100" \
  --user "admin" \
  --key "~/.ssh/id_rsa" \
  --tags "web,production"
```

Test the connection:
```bash
vps-tools inventory test --name "production-web"
```

## Server Management

### Adding Servers

#### Basic Server Addition
```bash
vps-tools inventory add --name "server-name" --host "IP_ADDRESS" --user "USERNAME"
```

#### Advanced Server Configuration
```bash
vps-tools inventory add \
  --name "database-server" \
  --host "192.168.1.101" \
  --user "admin" \
  --port 2222 \
  --key "~/.ssh/id_rsa" \
  --password "SSH_PASSWORD" \
  --tags "database,production,critical" \
  --description "Primary PostgreSQL database server"
```

#### Server Addition Options
- `--name`: Unique server identifier (required)
- `--host`: IP address or hostname (required)
- `--user`: SSH username (required)
- `--port`: SSH port (default: 22)
- `--key`: Path to SSH private key
- `--password`: SSH password (not recommended for production)
- `--tags`: Comma-separated tags for organization
- `--description`: Server description

### Managing Servers

#### List All Servers
```bash
# Basic list
vps-tools inventory list

# Filter by tags
vps-tools inventory list --tags "web,production"

# Search by name or description
vps-tools inventory list --search "database"

# Sort by different fields
vps-tools inventory list --sort "name"
vps-tools inventory list --sort "host"
vps-tools inventory list --sort "tags"
```

#### Edit Server Information
```bash
# Update tags
vps-tools inventory edit --name "web-server" --tags "web,production,updated"

# Change SSH user
vps-tools inventory edit --name "web-server" --user "newuser"

# Update description
vps-tools inventory edit --name "web-server" --description "Updated description"
```

#### Remove Servers
```bash
# Remove single server
vps-tools inventory remove --name "old-server"

# Remove multiple servers
vps-tools inventory remove --names "server1,server2,server3"
```

#### Test Connectivity
```bash
# Test single server
vps-tools inventory test --name "web-server"

# Test all servers
vps-tools inventory test --all

# Test servers by tag
vps-tools inventory test --tags "production"
```

## Health Monitoring

### Basic Health Checks

#### Check All Servers
```bash
vps-tools health check --all
```

#### Check Specific Server
```bash
vps-tools health check --name "web-server"
```

#### Check Servers by Tag
```bash
vps-tools health check --tags "production"
```

### Continuous Monitoring

#### Real-time Monitoring
```bash
# Monitor all servers with 30-second intervals
vps-tools health monitor --interval 30s

# Monitor specific servers
vps-tools health monitor --names "web-server,db-server" --interval 60s

# Monitor with custom thresholds
vps-tools health monitor --cpu-threshold 80 --memory-threshold 85
```

#### Health Check Options
- `--interval`: Check interval (default: 60s)
- `--cpu-threshold`: CPU warning threshold (default: 70%)
- `--memory-threshold`: Memory warning threshold (default: 80%)
- `--disk-threshold`: Disk warning threshold (default: 80%)
- `--output`: Save results to file

### Health Reports

#### Generate Reports
```bash
# Generate JSON report
vps-tools health report --output health-report.json

# Generate CSV report
vps-tools health report --output health-report.csv --format csv

# Generate HTML report
vps-tools health report --output health-report.html --format html
```

#### Report Analysis
```bash
# Analyze trends over time
vps-tools health analyze --days 7 --output trends.json

# Compare servers
vps-tools health compare --servers "web-server,db-server" --output comparison.json
```

### Custom Health Checks

#### Define Custom Checks
Create a custom health check configuration:

```yaml
# ~/.config/vps-tools/health-checks.yaml
checks:
  - name: "disk_space"
    command: "df -h /"
    pattern: "(\d+)%"
    threshold: 80
    
  - name: "nginx_status"
    command: "curl -s http://localhost/nginx_status"
    pattern: "Active connections: (\d+)"
    threshold: 1000
    
  - name: "database_connections"
    command: "psql -U postgres -c 'SELECT count(*) FROM pg_stat_activity;'"
    pattern: "(\d+)"
    threshold: 50
```

Run custom checks:
```bash
vps-tools health custom --config health-checks.yaml
```

## Command Execution

### Single Command Execution

#### Basic Command Execution
```bash
vps-tools run command --name "web-server" --cmd "uptime"
```

#### Command with Options
```bash
vps-tools run command \
  --name "web-server" \
  --cmd "ls -la /var/log" \
  --timeout 30s \
  --capture-output \
  --save-result
```

### Batch Command Execution

#### Run Commands on Multiple Servers
```bash
vps-tools run batch \
  --servers "web-server,db-server" \
  --commands "uptime,df -h,free -m"
```

#### Batch Execution with Tags
```bash
vps-tools run batch \
  --tags "production" \
  --commands "systemctl status nginx,ps aux | grep nginx"
```

#### Parallel Execution
```bash
# Run on up to 5 servers concurrently
vps-tools run batch \
  --tags "production" \
  --commands "apt update && apt upgrade -y" \
  --parallel 5
```

### Script Execution

#### Execute Local Script
```bash
vps-tools run script --name "web-server" --script "./deploy.sh"
```

#### Execute Remote Script
```bash
vps-tools run script --name "web-server" --remote "/opt/scripts/backup.sh"
```

#### Script with Arguments
```bash
vps-tools run script \
  --name "web-server" \
  --script "./deploy.sh" \
  --args "--env=production,--backup=true"
```

### Job Scheduling

#### Schedule Recurring Jobs
```bash
# Daily backup at 2 AM
vps-tools run schedule \
  --name "web-server" \
  --cmd "/opt/scripts/backup.sh" \
  --schedule "0 2 * * *" \
  --name "daily-backup"

# Every 15 minutes health check
vps-tools run schedule \
  --name "web-server" \
  --cmd "/opt/scripts/health-check.sh" \
  --schedule "*/15 * * * *" \
  --name "health-monitor"
```

#### Manage Scheduled Jobs
```bash
# List scheduled jobs
vps-tools run schedule --list

# Remove scheduled job
vps-tools run schedule --remove "daily-backup"

# Update scheduled job
vps-tools run schedule --update "daily-backup" --schedule "0 3 * * *"
```

### Command History

#### View Command History
```bash
# View all history
vps-tools run history

# Filter by server
vps-tools run history --server "web-server"

# Filter by date range
vps-tools run history --from "2024-01-01" --to "2024-01-31"

# Search commands
vps-tools run history --search "nginx"
```

#### Re-run Previous Commands
```bash
# Re-run by ID
vps-tools run rerun --id 123

# Re-run last command
vps-tools run rerun --last

# Re-run on different server
vps-tools run rerun --id 123 --server "db-server"
```

## Security Auditing

### Comprehensive Security Audits

#### Full Security Audit
```bash
vps-tools security audit --name "web-server"
```

#### Audit Multiple Servers
```bash
vps-tools security audit --tags "production"
```

#### Custom Audit Configuration
```bash
vps-tools security audit \
  --name "web-server" \
  --config "security-config.yaml" \
  --output "security-report.json"
```

### SSH Key Analysis

#### Analyze SSH Keys
```bash
vps-tools security ssh-keys --name "web-server"
```

#### SSH Key Options
- `--check-weak-keys`: Check for weak key algorithms
- `--check-expired`: Check for expired keys
- `--check-unauthorized`: Check for unauthorized keys
- `--scan-home`: Scan user home directories
- `--scan-system`: Scan system-wide SSH directories

#### SSH Key Management
```bash
# List all SSH keys
vps-tools security ssh-keys --name "web-server" --list

# Remove weak keys
vps-tools security ssh-keys --name "web-server" --remove-weak

# Generate key recommendations
vps-tools security ssh-keys --name "web-server" --recommendations
```

### Port Scanning

#### Basic Port Scan
```bash
vps-tools security ports --name "web-server" --range "1-1000"
```

#### Advanced Port Scanning
```bash
vps-tools security ports \
  --name "web-server" \
  --range "1-65535" \
  --scan-type "tcp" \
  --timeout 5s \
  --max-concurrent 100
```

#### Service Detection
```bash
vps-tools security ports \
  --name "web-server" \
  --range "1-1000" \
  --detect-services \
  --banner-grab
```

### Vulnerability Assessment

#### Basic Vulnerability Scan
```bash
vps-tools security vuln --name "web-server"
```

#### Comprehensive Assessment
```bash
vps-tools security vuln \
  --name "web-server" \
  --check-packages \
  --check-services \
  --check-permissions \
  --check-configs
```

#### Security Reports
```bash
# Generate detailed security report
vps-tools security report \
  --name "web-server" \
  --output "security-report.json" \
  --format "detailed"

# Generate summary report
vps-tools security report \
  --tags "production" \
  --output "security-summary.json" \
  --format "summary"
```

### Security Hardening

#### Apply Security Recommendations
```bash
vps-tools security harden --name "web-server" --level "standard"
```

#### Hardening Levels
- `basic`: Essential security measures
- `standard`: Comprehensive security hardening
- `strict`: Maximum security (may break some functionality)

#### Custom Hardening
```bash
vps-tools security harden \
  --name "web-server" \
  --config "hardening-rules.yaml" \
  --dry-run
```

## Docker Management

### Container Management

#### List Containers
```bash
# List all containers
vps-tools docker list --name "web-server"

# List only running containers
vps-tools docker list --name "web-server" --running

# List with detailed information
vps-tools docker list --name "web-server" --detailed
```

#### Container Lifecycle
```bash
# Start container
vps-tools docker start --name "web-server" --container "nginx"

# Stop container
vps-tools docker stop --name "web-server" --container "nginx"

# Restart container
vps-tools docker restart --name "web-server" --container "nginx"

# Remove container
vps-tools docker remove --name "web-server" --container "nginx" --force
```

#### Container Operations
```bash
# Execute command in container
vps-tools docker exec \
  --name "web-server" \
  --container "nginx" \
  --cmd "nginx -t"

# View container logs
vps-tools docker logs --name "web-server" --container "nginx" --tail 100

# View container stats
vps-tools docker stats --name "web-server" --container "nginx"
```

### Image Management

#### List Images
```bash
vps-tools docker images --name "web-server"
```

#### Image Operations
```bash
# Pull image
vps-tools docker pull --name "web-server" --image "nginx:latest"

# Remove image
vps-tools docker rmi --name "web-server" --image "nginx:old"

# Prune unused images
vps-tools docker prune --name "web-server" --images
```

### Container Health Monitoring

#### Health Checks
```bash
# Check container health
vps-tools docker health --name "web-server" --container "nginx"

# Health status for all containers
vps-tools docker health --name "web-server" --all
```

#### Custom Health Checks
```bash
vps-tools docker health \
  --name "web-server" \
  --container "nginx" \
  --check-cmd "curl -f http://localhost/health" \
  --interval 30s \
  --timeout 10s
```

### Container Backup and Restore

#### Backup Containers
```bash
# Backup single container
vps-tools docker backup \
  --name "web-server" \
  --container "nginx" \
  --output "nginx-backup.tar"

# Backup with data
vps-tools docker backup \
  --name "web-server" \
  --container "nginx" \
  --output "nginx-backup.tar" \
  --include-volumes
```

#### Restore Containers
```bash
vps-tools docker restore \
  --name "web-server" \
  --backup "nginx-backup.tar" \
  --name "nginx-restored"
```

### Container Networking

#### Network Management
```bash
# List networks
vps-tools docker network --name "web-server" --list

# Connect container to network
vps-tools docker network \
  --name "web-server" \
  --container "nginx" \
  --network "web-network" \
  --action "connect"

# Disconnect from network
vps-tools docker network \
  --name "web-server" \
  --container "nginx" \
  --network "web-network" \
  --action "disconnect"
```

## System Maintenance

### System Cleanup

#### Basic Cleanup
```bash
vps-tools maintenance cleanup --name "web-server" --level "standard"
```

#### Cleanup Levels
- `minimal`: Remove temporary files and logs
- `standard`: Additional package cache and old logs
- `aggressive`: Remove old kernels, unused packages, and large files

#### Custom Cleanup
```bash
vps-tools maintenance cleanup \
  --name "web-server" \
  --paths "/tmp,/var/tmp,/var/log" \
  --exclude "*.log" \
  --max-age 30d
```

### Package Management

#### System Updates
```bash
# Update all packages
vps-tools maintenance update --name "web-server"

# Update specific packages
vps-tools maintenance update \
  --name "web-server" \
  --packages "nginx,postgresql,redis"

# Security updates only
vps-tools maintenance update --name "web-server" --security-only
```

#### Package Operations
```bash
# Install packages
vps-tools maintenance install \
  --name "web-server" \
  --packages "htop,tree,ncdu"

# Remove packages
vps-tools maintenance remove \
  --name "web-server" \
  --packages "old-package,unused-package"

# List upgradable packages
vps-tools maintenance list-upgradable --name "web-server"
```

### Backup Operations

#### File System Backup
```bash
# Backup directory
vps-tools maintenance backup \
  --name "web-server" \
  --path "/var/www" \
  --output "web-backup.tar.gz"

# Backup with compression
vps-tools maintenance backup \
  --name "web-server" \
  --path "/var/www" \
  --output "web-backup.tar.gz" \
  --compress "gzip" \
  --exclude "*.tmp,*.log"
```

#### Database Backup
```bash
# PostgreSQL backup
vps-tools maintenance backup \
  --name "db-server" \
  --database "postgresql" \
  --database-name "app_db" \
  --output "db-backup.sql"

# MySQL backup
vps-tools maintenance backup \
  --name "db-server" \
  --database "mysql" \
  --database-name "app_db" \
  --output "db-backup.sql"
```

#### Automated Backups
```bash
# Schedule daily backup
vps-tools maintenance backup \
  --name "web-server" \
  --path "/var/www" \
  --output "web-backup-$(date +%Y%m%d).tar.gz" \
  --schedule "0 2 * * *"
```

### Restore Operations

#### File System Restore
```bash
vps-tools maintenance restore \
  --name "web-server" \
  --backup "web-backup.tar.gz" \
  --path "/var/www"
```

#### Database Restore
```bash
vps-tools maintenance restore \
  --name "db-server" \
  --backup "db-backup.sql" \
  --database "postgresql" \
  --database-name "app_db"
```

### System Optimization

#### Performance Tuning
```bash
vps-tools maintenance optimize --name "web-server" --level "standard"
```

#### Optimization Levels
- `basic`: Basic system tuning
- `standard`: Comprehensive optimization
- `aggressive`: Maximum performance tuning

#### Custom Optimization
```bash
vps-tools maintenance optimize \
  --name "web-server" \
  --config "optimization.yaml" \
  --dry-run
```

## TUI Interface

### Launching the TUI

```bash
# Start TUI
vps-tools tui

# Start with specific view
vps-tools tui --view servers
vps-tools tui --view health
vps-tools tui --view jobs
```

### Navigation Basics

#### Global Shortcuts
- `Tab` / `Shift+Tab`: Navigate between sections
- `↑` / `↓`: Navigate within lists
- `Enter`: Select item/confirm action
- `Esc`: Go back/cancel action
- `q`: Quit application
- `Ctrl+C`: Force quit

#### View Shortcuts
- `1`: Servers view
- `2`: Health view
- `3`: Jobs view
- `4`: Notifications view
- `5`: Settings view
- `6`: Help view

#### Action Shortcuts
- `c`: Create new item
- `e`: Edit selected item
- `d`: Delete selected item
- `r`: Refresh current view
- `f`: Filter/search items
- `s`: Sort items
- `h`: Show context help

### Servers View

#### Server List
- Shows all configured servers
- Color-coded status indicators
- Sort by name, host, tags, or status
- Filter by tags or search terms

#### Server Actions
- `Enter`: View server details
- `e`: Edit server configuration
- `d`: Delete server
- `t`: Test SSH connection
- `h`: Run health check
- `c`: Execute command

#### Server Details
- Comprehensive server information
- Real-time status updates
- Quick action buttons
- Historical data

### Health View

#### Health Dashboard
- Real-time health metrics
- Visual status indicators
- Threshold warnings
- Historical trends

#### Health Actions
- `r`: Run health check
- `m`: Start/stop monitoring
- `c`: Configure thresholds
- `e`: Export health data
- `a`: Analyze trends

#### Health Details
- CPU, memory, disk usage
- Network statistics
- Service status
- Custom check results

### Jobs View

#### Job Management
- Active and completed jobs
- Real-time progress updates
- Job status indicators
- Execution history

#### Job Actions
- `Enter`: View job details
- `c`: Create new job
- `s`: Stop running job
- `r`: Rerun job
- `l`: View job logs

#### Job Creation
- Single command execution
- Batch command execution
- Script execution
- Scheduled jobs

### Notifications View

#### Notification Center
- System notifications
- Alert messages
- Warning indicators
- Information messages

#### Notification Actions
- `Enter`: View notification details
- `a`: Acknowledge notification
- `d`: Dismiss notification
- `c`: Clear all notifications

### Settings View

#### Configuration Management
- Application settings
- SSH configuration
- Health thresholds
- Security settings

#### Setting Actions
- `e`: Edit setting
- `r`: Reset to default
- `s`: Save configuration
- `l`: Reload configuration

### Help View

#### Interactive Help
- Keyboard shortcuts
- Command reference
- Feature descriptions
- Tips and tricks

#### Help Navigation
- `Tab`: Navigate help sections
- `Enter`: Expand/collapse topics
- `f`: Search help content
- `Esc`: Return to previous view

## Configuration

### Configuration File Structure

The main configuration file is located at `~/.config/vps-tools/config.yaml`:

```yaml
# Application settings
app:
  name: "vps-tools"
  version: "1.0.0"
  debug: false

# Database configuration
database:
  path: "~/.local/share/vps-tools/data.db"
  backup_enabled: true
  backup_interval: "24h"

# Logging configuration
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  file: "~/.local/share/vps-tools/logs/vps-tools.log"
  max_size: "100MB"
  max_backups: 10
  max_age: "30d"

# SSH configuration
ssh:
  timeout: "30s"
  max_retries: 3
  keep_alive: "10s"
  key_paths:
    - "~/.ssh/id_rsa"
    - "~/.ssh/id_ed25519"
    - "~/.ssh/id_ecdsa"
  known_hosts: "~/.ssh/known_hosts"
  strict_host_key: true

# Health monitoring configuration
health:
  default_interval: "60s"
  timeout: "30s"
  thresholds:
    cpu:
      warning: 70
      critical: 90
    memory:
      warning: 80
      critical: 95
    disk:
      warning: 80
      critical: 95
    load:
      warning: 2.0
      critical: 4.0
  custom_checks: "~/.config/vps-tools/health-checks.yaml"

# Security configuration
security:
  default_port_range: "1-1000"
  scan_timeout: "5s"
  max_concurrent_scans: 100
  ssh_key_scan: true
  vulnerability_check: true
  hardening_rules: "~/.config/vps-tools/hardening-rules.yaml"

# Docker configuration
docker:
  socket_path: "/var/run/docker.sock"
  timeout: "30s"
  default_registry: "docker.io"
  cleanup_interval: "24h"
  health_check_interval: "30s"

# Maintenance configuration
maintenance:
  backup_path: "~/.local/share/vps-tools/backups"
  log_retention: "30d"
  auto_cleanup: true
  cleanup_schedule: "0 2 * * 0"  # Weekly
  optimization_rules: "~/.config/vps-tools/optimization.yaml"

# TUI configuration
tui:
  theme: "default"  # default, dark, light
  refresh_interval: "5s"
  max_log_lines: 1000
  confirm_destructive: true
  show_notifications: true

# Notifications configuration
notifications:
  enabled: true
  methods:
    - "tui"
    - "log"
  email:
    enabled: false
    smtp_server: ""
    smtp_port: 587
    username: ""
    password: ""
    from: ""
    to: []
```

### Environment Variables

Override configuration with environment variables:

```bash
# Configuration file location
export VPS_TOOLS_CONFIG="/path/to/config.yaml"

# Database path
export VPS_TOOLS_DB_PATH="/path/to/database.db"

# Log level
export VPS_TOOLS_LOG_LEVEL="debug"

# Log file
export VPS_TOOLS_LOG_FILE="/path/to/vps-tools.log"

# SSH timeout
export VPS_TOOLS_SSH_TIMEOUT="60s"

# Health check interval
export VPS_TOOLS_HEALTH_INTERVAL="30s"

# TUI theme
export VPS_TOOLS_TUI_THEME="dark"
```

### Configuration Validation

Validate your configuration:

```bash
# Validate configuration file
vps-tools config validate

# Show current configuration
vps-tools config show

# Show specific section
vps-tools config show --section ssh
vps-tools config show --section health
```

### Configuration Migration

Migrate configuration between versions:

```bash
# Check for configuration updates
vps-tools config check-updates

# Migrate configuration
vps-tools config migrate

# Backup current configuration
vps-tools config backup --output config-backup.yaml
```

## Advanced Usage

### Custom Plugins

Create custom plugins for extended functionality:

```go
// plugins/custom.go
package plugins

import (
    "github.com/pgd1001/vps-tools/internal/plugin"
)

type CustomPlugin struct{}

func (p *CustomPlugin) Name() string {
    return "custom"
}

func (p *CustomPlugin) Execute(ctx plugin.Context) error {
    // Custom plugin logic
    return nil
}

func (p *CustomPlugin) Description() string {
    return "Custom plugin for specific functionality"
}
```

Register plugins in configuration:

```yaml
plugins:
  enabled: true
  directory: "~/.config/vps-tools/plugins"
  load:
    - "custom"
    - "monitoring"
    - "backup"
```

### API Integration

Use VPS Tools as a library:

```go
package main

import (
    "context"
    "fmt"
"github.com/pgd1001/vps-tools/internal/app"
"github.com/pgd1001/vps-tools/internal/server"
)

func main() {
    // Initialize application
    cfg, err := app.LoadConfig()
    if err != nil {
        panic(err)
    }

    app, err := app.New(cfg)
    if err != nil {
        panic(err)
    }

    // List servers
    servers, err := app.Store().ListServers()
    if err != nil {
        panic(err)
    }

    // Run health check
    for _, srv := range servers {
        health, err := app.HealthChecker().Check(context.Background(), srv)
        if err != nil {
            fmt.Printf("Health check failed for %s: %v\n", srv.Name, err)
            continue
        }
        fmt.Printf("Server %s health: %+v\n", srv.Name, health)
    }
}
```

### Automation Scripts

Create automation scripts using VPS Tools:

```bash
#!/bin/bash
# automated-health-check.sh

# Check all production servers
vps-tools health check --tags "production" --output "health-$(date +%Y%m%d).json"

# Send alerts if any issues
if [ $? -ne 0 ]; then
    echo "Health check failed for production servers" | mail -s "VPS Health Alert" admin@example.com
fi

# Generate daily report
vps-tools health report --tags "production" --output "daily-health-$(date +%Y%m%d).html"
```

### Integration with CI/CD

Integrate VPS Tools in CI/CD pipelines:

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup VPS Tools
        run: |
          wget https://github.com/pgd1001/vps-tools/releases/latest/download/vps-tools-linux-amd64
          chmod +x vps-tools-linux-amd64
          sudo mv vps-tools-linux-amd64 /usr/local/bin/vps-tools
      
      - name: Pre-deploy health check
        run: vps-tools health check --tags "production"
      
      - name: Deploy application
        run: vps-tools run batch --tags "production" --commands "./deploy.sh"
      
      - name: Post-deploy verification
        run: vps-tools health check --tags "production"
```

### Performance Optimization

Optimize VPS Tools performance:

```yaml
# config.yaml
app:
  max_concurrent_operations: 10
  cache_enabled: true
  cache_ttl: "5m"

ssh:
  connection_pool_size: 5
  keep_alive_interval: "10s"

health:
  parallel_checks: true
  max_parallel_checks: 20
```

### Troubleshooting

Common issues and solutions:

#### SSH Connection Issues
```bash
# Test SSH connection manually
ssh -v user@hostname

# Check SSH configuration
vps-tools config show --section ssh

# Update SSH keys
vps-tools inventory edit --name "server" --key "~/.ssh/new_key"
```

#### Health Check Failures
```bash
# Run health check with debug output
VPS_TOOLS_LOG_LEVEL=debug vps-tools health check --name "server"

# Check custom health check configuration
vps-tools config validate --section health
```

#### Performance Issues
```bash
# Monitor resource usage
vps-tools health monitor --name "server" --interval 10s

# Check application logs
tail -f ~/.local/share/vps-tools/logs/vps-tools.log
```

This comprehensive user guide covers all aspects of VPS Tools, from basic usage to advanced automation and integration scenarios.