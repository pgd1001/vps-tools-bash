# API Reference

This document provides a comprehensive reference for all VPS Tools commands, options, and programmatic interfaces.

## Table of Contents

1. [CLI Commands](#cli-commands)
2. [TUI Interface](#tui-interface)
3. [Go Library API](#go-library-api)
4. [Configuration API](#configuration-api)
5. [Plugin API](#plugin-api)

## CLI Commands

### Root Command

```bash
vps-tools [flags]
```

#### Global Flags
- `--config, -c`: Path to configuration file (default: `~/.config/vps-tools/config.yaml`)
- `--help, -h`: Show help information
- `--version, -v`: Show version information
- `--debug`: Enable debug mode
- `--quiet`: Suppress non-error output
- `--output, -o`: Output format (json, yaml, table, csv)
- `--no-color`: Disable colored output

### Configuration Commands

#### config init
Initialize configuration with defaults.

```bash
vps-tools config init [flags]
```

**Flags:**
- `--force, -f`: Overwrite existing configuration
- `--template`: Use specific template (default, minimal, production)

**Examples:**
```bash
# Initialize with defaults
vps-tools config init

# Force overwrite
vps-tools config init --force

# Use production template
vps-tools config init --template production
```

#### config show
Display current configuration.

```bash
vps-tools config show [flags]
```

**Flags:**
- `--section, -s`: Show specific section only
- `--format, -f`: Output format (yaml, json, toml)

**Examples:**
```bash
# Show all configuration
vps-tools config show

# Show SSH configuration only
vps-tools config show --section ssh

# Show as JSON
vps-tools config show --format json
```

#### config validate
Validate configuration file.

```bash
vps-tools config validate [flags]
```

**Flags:**
- `--strict`: Enable strict validation
- `--show-warnings`: Show validation warnings

#### config set
Set configuration values.

```bash
vps-tools config set <key> <value> [flags]
```

**Examples:**
```bash
# Set log level
vps-tools config set logging.level debug

# Set SSH timeout
vps-tools config set ssh.timeout 60s

# Set health threshold
vps-tools config set health.thresholds.cpu.warning 80
```

### Inventory Commands

#### inventory add
Add a new server to inventory.

```bash
vps-tools inventory add [flags]
```

**Required Flags:**
- `--name, -n`: Server name (unique identifier)
- `--host, -h`: IP address or hostname
- `--user, -u`: SSH username

**Optional Flags:**
- `--port, -p`: SSH port (default: 22)
- `--key, -k`: Path to SSH private key
- `--password`: SSH password (not recommended)
- `--tags, -t`: Comma-separated tags
- `--description, -d`: Server description
- `--group`: Server group

**Examples:**
```bash
# Basic server addition
vps-tools inventory add --name "web1" --host "192.168.1.100" --user "admin"

# Advanced server configuration
vps-tools inventory add \
  --name "db1" \
  --host "192.168.1.101" \
  --user "admin" \
  --port 2222 \
  --key "~/.ssh/id_rsa" \
  --tags "database,production" \
  --description "Primary database server"
```

#### inventory list
List servers in inventory.

```bash
vps-tools inventory list [flags]
```

**Flags:**
- `--tags, -t`: Filter by tags
- `--search, -s`: Search by name or description
- `--sort`: Sort by field (name, host, tags, created)
- `--format, -f`: Output format (table, json, yaml, csv)
- `--limit, -l`: Limit number of results
- `--group`: Filter by group

**Examples:**
```bash
# List all servers
vps-tools inventory list

# Filter by tags
vps-tools inventory list --tags "production,web"

# Search servers
vps-tools inventory list --search "database"

# Sort by host
vps-tools inventory list --sort host

# JSON output
vps-tools inventory list --format json
```

#### inventory edit
Edit server configuration.

```bash
vps-tools inventory edit <name> [flags]
```

**Flags:**
- `--host`: Update host
- `--user`: Update username
- `--port`: Update port
- `--key`: Update SSH key path
- `--password`: Update password
- `--tags`: Update tags
- `--description`: Update description
- `--group`: Update group

**Examples:**
```bash
# Update tags
vps-tools inventory edit --name "web1" --tags "web,production,updated"

# Update SSH user
vps-tools inventory edit --name "web1" --user "newuser"

# Update description
vps-tools inventory edit --name "web1" --description "Updated web server"
```

#### inventory remove
Remove server from inventory.

```bash
vps-tools inventory remove <name> [flags]
```

**Flags:**
- `--force, -f`: Skip confirmation prompt
- `--backup`: Backup server data before removal

**Examples:**
```bash
# Remove server
vps-tools inventory remove --name "old-server"

# Force remove without confirmation
vps-tools inventory remove --name "old-server" --force
```

#### inventory test
Test SSH connectivity to servers.

```bash
vps-tools inventory test [flags]
```

**Flags:**
- `--name, -n`: Test specific server
- `--tags, -t`: Test servers by tags
- `--all, -a`: Test all servers
- `--timeout`: Connection timeout (default: 30s)
- `--verbose, -v`: Verbose output

**Examples:**
```bash
# Test specific server
vps-tools inventory test --name "web1"

# Test all servers
vps-tools inventory test --all

# Test production servers
vps-tools inventory test --tags "production"
```

### Health Commands

#### health check
Run health checks on servers.

```bash
vps-tools health check [flags]
```

**Flags:**
- `--name, -n`: Check specific server
- `--tags, -t`: Check servers by tags
- `--all, -a`: Check all servers
- `--timeout`: Check timeout (default: 30s)
- `--interval`: Check interval for continuous monitoring
- `--cpu-threshold`: CPU warning threshold (default: 70)
- `--memory-threshold`: Memory warning threshold (default: 80)
- `--disk-threshold`: Disk warning threshold (default: 80)
- `--output, -o`: Save results to file
- `--format, -f`: Output format (table, json, yaml)

**Examples:**
```bash
# Check all servers
vps-tools health check --all

# Check specific server
vps-tools health check --name "web1"

# Custom thresholds
vps-tools health check --all --cpu-threshold 80 --memory-threshold 85

# Save results
vps-tools health check --all --output health-results.json
```

#### health monitor
Start continuous health monitoring.

```bash
vps-tools health monitor [flags]
```

**Flags:**
- `--name, -n`: Monitor specific server
- `--tags, -t`: Monitor servers by tags
- `--all, -a`: Monitor all servers
- `--interval`: Check interval (default: 60s)
- `--duration`: Monitoring duration
- `--cpu-threshold`: CPU warning threshold
- `--memory-threshold`: Memory warning threshold
- `--disk-threshold`: Disk warning threshold
- `--alert`: Enable alerts
- `--output, -o`: Save monitoring data

**Examples:**
```bash
# Monitor all servers
vps-tools health monitor --all

# Monitor with 30-second interval
vps-tools health monitor --all --interval 30s

# Monitor for 1 hour
vps-tools health monitor --all --duration 1h

# Monitor with alerts
vps-tools health monitor --all --alert
```

#### health report
Generate health reports.

```bash
vps-tools health report [flags]
```

**Flags:**
- `--name, -n`: Report for specific server
- `--tags, -t`: Report for servers by tags
- `--all, -a`: Report for all servers
- `--from`: Start date (YYYY-MM-DD)
- `--to`: End date (YYYY-MM-DD)
- `--output, -o`: Output file path
- `--format, -f`: Report format (json, html, csv, pdf)

**Examples:**
```bash
# Generate report for all servers
vps-tools health report --all --output health-report.html

# Generate date range report
vps-tools health report --all --from 2024-01-01 --to 2024-01-31

# Generate CSV report
vps-tools health report --all --format csv --output health-report.csv
```

#### health analyze
Analyze health trends.

```bash
vps-tools health analyze [flags]
```

**Flags:**
- `--name, -n`: Analyze specific server
- `--tags, -t`: Analyze servers by tags
- `--all, -a`: Analyze all servers
- `--days, -d`: Number of days to analyze (default: 7)
- `--metrics`: Metrics to analyze (cpu,memory,disk,load)
- `--output, -o`: Save analysis results

**Examples:**
```bash
# Analyze last 7 days
vps-tools health analyze --all --days 7

# Analyze specific metrics
vps-tools health analyze --all --metrics cpu,memory

# Save analysis
vps-tools health analyze --all --output analysis.json
```

### Run Commands

#### run command
Execute single command on server.

```bash
vps-tools run command [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--cmd, -c`: Command to execute

**Optional Flags:**
- `--timeout`: Command timeout (default: 30s)
- `--capture-output`: Capture command output
- `--save-result`: Save execution result
- `--working-dir, -w`: Working directory
- `--env, -e`: Environment variables
- `--user, -u`: Execute as different user

**Examples:**
```bash
# Execute simple command
vps-tools run command --name "web1" --cmd "uptime"

# Execute with timeout
vps-tools run command --name "web1" --cmd "sleep 60" --timeout 120s

# Execute with environment variables
vps-tools run command --name "web1" --cmd "echo $PATH" --env "PATH=/custom/bin:$PATH"
```

#### run batch
Execute commands on multiple servers.

```bash
vps-tools run batch [flags]
```

**Required Flags:**
- `--servers, -s`: Target servers (comma-separated)
- `--commands, -c`: Commands to execute (comma-separated)

**Optional Flags:**
- `--tags, -t`: Target servers by tags
- `--parallel, -p`: Maximum parallel executions (default: 5)
- `--timeout`: Command timeout
- `--continue-on-error`: Continue on command failure
- `--output, -o`: Save batch results

**Examples:**
```bash
# Execute on multiple servers
vps-tools run batch --servers "web1,web2" --commands "uptime,df -h"

# Execute by tags
vps-tools run batch --tags "production" --commands "systemctl status nginx"

# Parallel execution
vps-tools run batch --servers "web1,web2,db1" --commands "uptime" --parallel 3
```

#### run script
Execute script on server.

```bash
vps-tools run script [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--script, -s`: Script path

**Optional Flags:**
- `--remote`: Execute remote script
- `--args, -a`: Script arguments
- `--timeout`: Script timeout
- `--upload`: Upload script before execution
- `--remove-after`: Remove script after execution

**Examples:**
```bash
# Execute local script
vps-tools run script --name "web1" --script "./deploy.sh"

# Execute with arguments
vps-tools run script --name "web1" --script "./deploy.sh" --args "--env=production,--backup"

# Execute remote script
vps-tools run script --name "web1" --remote "/opt/scripts/backup.sh"
```

#### run schedule
Schedule recurring command execution.

```bash
vps-tools run schedule [flags]
```

**Subcommands:**
- `add`: Add new scheduled job
- `list`: List scheduled jobs
- `remove`: Remove scheduled job
- `update`: Update scheduled job

**Add Flags:**
- `--name, -n`: Job name (required)
- `--server, -s`: Target server
- `--cmd, -c`: Command to execute (required)
- `--schedule`: Cron schedule (required)
- `--description`: Job description
- `--enabled`: Enable/disable job (default: true)

**Examples:**
```bash
# Add scheduled job
vps-tools run schedule add \
  --name "daily-backup" \
  --server "web1" \
  --cmd "/opt/scripts/backup.sh" \
  --schedule "0 2 * * *" \
  --description "Daily backup job"

# List scheduled jobs
vps-tools run schedule list

# Remove scheduled job
vps-tools run schedule remove --name "daily-backup"
```

#### run history
View command execution history.

```bash
vps-tools run history [flags]
```

**Flags:**
- `--server, -s`: Filter by server
- `--command, -c`: Filter by command
- `--from`: Start date
- `--to`: End date
- `--status`: Filter by status (success, failure, timeout)
- `--limit, -l`: Limit number of results
- `--format, -f`: Output format

**Examples:**
```bash
# View all history
vps-tools run history

# Filter by server
vps-tools run history --server "web1"

# Filter by date range
vps-tools run history --from 2024-01-01 --to 2024-01-31

# Show only failures
vps-tools run history --status failure
```

### Security Commands

#### security audit
Run comprehensive security audit.

```bash
vps-tools security audit [flags]
```

**Flags:**
- `--name, -n`: Audit specific server
- `--tags, -t`: Audit servers by tags
- `--all, -a`: Audit all servers
- `--config, -c`: Custom audit configuration
- `--output, -o`: Save audit results
- `--format, -f`: Output format (json, html, pdf)
- `--severity`: Minimum severity level (low, medium, high, critical)

**Examples:**
```bash
# Audit all servers
vps-tools security audit --all

# Audit with custom config
vps-tools security audit --name "web1" --config audit-config.yaml

# Generate HTML report
vps-tools security audit --all --format html --output security-report.html
```

#### security ssh-keys
Analyze SSH keys.

```bash
vps-tools security ssh-keys [flags]
```

**Flags:**
- `--name, -n`: Analyze specific server
- `--tags, -t`: Analyze servers by tags
- `--all, -a`: Analyze all servers
- `--check-weak-keys`: Check for weak key algorithms
- `--check-expired`: Check for expired keys
- `--check-unauthorized`: Check for unauthorized keys
- `--scan-home`: Scan user home directories
- `--scan-system`: Scan system-wide SSH directories
- `--remove-weak`: Remove weak keys
- `--recommendations`: Show key recommendations

**Examples:**
```bash
# Analyze SSH keys
vps-tools security ssh-keys --name "web1"

# Comprehensive analysis
vps-tools security ssh-keys --all --check-weak-keys --check-expired --scan-home

# Get recommendations
vps-tools security ssh-keys --name "web1" --recommendations
```

#### security ports
Scan ports on servers.

```bash
vps-tools security ports [flags]
```

**Flags:**
- `--name, -n`: Scan specific server
- `--tags, -t`: Scan servers by tags
- `--all, -a`: Scan all servers
- `--range, -r`: Port range (default: 1-1000)
- `--scan-type`: Scan type (tcp, udp, both)
- `--timeout`: Port timeout (default: 5s)
- `--max-concurrent`: Maximum concurrent scans
- `--detect-services`: Detect running services
- `--banner-grab`: Grab service banners
- `--output, -o`: Save scan results

**Examples:**
```bash
# Basic port scan
vps-tools security ports --name "web1" --range "1-1000"

# Comprehensive scan
vps-tools security ports --name "web1" --range "1-65535" --detect-services --banner-grab

# Fast scan
vps-tools security ports --all --range "1-1000" --max-concurrent 100
```

#### security vuln
Vulnerability assessment.

```bash
vps-tools security vuln [flags]
```

**Flags:**
- `--name, -n`: Assess specific server
- `--tags, -t`: Assess servers by tags
- `--all, -a`: Assess all servers
- `--check-packages`: Check package vulnerabilities
- `--check-services`: Check service vulnerabilities
- `--check-permissions`: Check permission issues
- `--check-configs`: Check configuration vulnerabilities
- `--severity`: Minimum severity level
- `--output, -o`: Save assessment results

**Examples:**
```bash
# Basic vulnerability assessment
vps-tools security vuln --name "web1"

# Comprehensive assessment
vps-tools security vuln --all --check-packages --check-services --check-permissions

# High severity only
vps-tools security vuln --all --severity high
```

#### security harden
Apply security hardening.

```bash
vps-tools security harden [flags]
```

**Flags:**
- `--name, -n`: Harden specific server
- `--tags, -t`: Harden servers by tags
- `--all, -a`: Harden all servers
- `--level`: Hardening level (basic, standard, strict)
- `--config, -c`: Custom hardening configuration
- `--dry-run`: Show changes without applying
- `--backup`: Backup before changes
- `--confirm`: Confirm before applying changes

**Examples:**
```bash
# Standard hardening
vps-tools security harden --name "web1" --level standard

# Dry run
vps-tools security harden --all --level standard --dry-run

# Custom configuration
vps-tools security harden --name "web1" --config hardening.yaml
```

### Docker Commands

#### docker list
List Docker containers.

```bash
vps-tools docker list [flags]
```

**Flags:**
- `--name, -n`: Target server name
- `--running, -r`: List only running containers
- `--all, -a`: List all containers (default)
- `--detailed, -d`: Show detailed information
- `--format, -f`: Output format (table, json)

**Examples:**
```bash
# List all containers
vps-tools docker list --name "web1"

# List only running containers
vps-tools docker list --name "web1" --running

# Detailed information
vps-tools docker list --name "web1" --detailed
```

#### docker start
Start Docker container.

```bash
vps-tools docker start [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--attach`: Attach to container after start
- `--timeout`: Start timeout

**Examples:**
```bash
# Start container
vps-tools docker start --name "web1" --container "nginx"

# Start and attach
vps-tools docker start --name "web1" --container "nginx" --attach
```

#### docker stop
Stop Docker container.

```bash
vps-tools docker stop [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--timeout`: Stop timeout (default: 10s)
- `--force, -f`: Force stop

**Examples:**
```bash
# Stop container
vps-tools docker stop --name "web1" --container "nginx"

# Force stop
vps-tools docker stop --name "web1" --container "nginx" --force
```

#### docker restart
Restart Docker container.

```bash
vps-tools docker restart [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--timeout`: Restart timeout
- `--attach`: Attach after restart

**Examples:**
```bash
# Restart container
vps-tools docker restart --name "web1" --container "nginx"
```

#### docker remove
Remove Docker container.

```bash
vps-tools docker remove [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--force, -f`: Force remove
- `--volumes, -v`: Remove associated volumes

**Examples:**
```bash
# Remove container
vps-tools docker remove --name "web1" --container "nginx"

# Force remove with volumes
vps-tools docker remove --name "web1" --container "nginx" --force --volumes
```

#### docker exec
Execute command in container.

```bash
vps-tools docker exec [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID
- `--cmd, -e`: Command to execute

**Optional Flags:**
- `--interactive, -i`: Interactive mode
- `--tty, -t`: Allocate TTY
- `--user, -u`: Execute as user
- `--workdir, -w`: Working directory

**Examples:**
```bash
# Execute command
vps-tools docker exec --name "web1" --container "nginx" --cmd "nginx -t"

# Interactive shell
vps-tools docker exec --name "web1" --container "nginx" --cmd "/bin/bash" --interactive --tty
```

#### docker logs
View container logs.

```bash
vps-tools docker logs [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--follow, -f`: Follow log output
- `--tail, -t`: Number of lines to show
- `--since`: Show logs since timestamp
- `--until`: Show logs until timestamp
- `--timestamps, -T`: Show timestamps

**Examples:**
```bash
# Show logs
vps-tools docker logs --name "web1" --container "nginx"

# Follow logs
vps-tools docker logs --name "web1" --container "nginx" --follow

# Last 100 lines
vps-tools docker logs --name "web1" --container "nginx" --tail 100
```

#### docker stats
View container statistics.

```bash
vps-tools docker stats [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--stream, -s`: Stream stats continuously
- `--interval`: Stats interval (default: 1s)
- `--format, -f`: Output format

**Examples:**
```bash
# Show stats
vps-tools docker stats --name "web1" --container "nginx"

# Stream stats
vps-tools docker stats --name "web1" --container "nginx" --stream
```

#### docker health
Check container health.

```bash
vps-tools docker health [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID

**Optional Flags:**
- `--all, -a`: Check all containers
- `--detailed, -d`: Detailed health information
- `--check-cmd`: Custom health check command
- `--interval`: Health check interval
- `--timeout`: Health check timeout

**Examples:**
```bash
# Check container health
vps-tools docker health --name "web1" --container "nginx"

# Check all containers
vps-tools docker health --name "web1" --all

# Custom health check
vps-tools docker health --name "web1" --container "nginx" --check-cmd "curl -f http://localhost/health"
```

#### docker backup
Backup container.

```bash
vps-tools docker backup [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--container, -c`: Container name or ID
- `--output, -o`: Backup output file

**Optional Flags:**
- `--include-volumes`: Include associated volumes
- `--compress`: Compress backup
- `--format`: Backup format (tar, tar.gz)

**Examples:**
```bash
# Backup container
vps-tools docker backup --name "web1" --container "nginx" --output nginx-backup.tar

# Backup with volumes
vps-tools docker backup --name "web1" --container "nginx" --output nginx-backup.tar --include-volumes

# Compressed backup
vps-tools docker backup --name "web1" --container "nginx" --output nginx-backup.tar.gz --compress
```

#### docker restore
Restore container from backup.

```bash
vps-tools docker restore [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--backup, -b`: Backup file path

**Optional Flags:**
- `--name, -n`: New container name
- `--volumes`: Restore volumes
- `--force, -f`: Force restore

**Examples:**
```bash
# Restore container
vps-tools docker restore --name "web1" --backup nginx-backup.tar

# Restore with new name
vps-tools docker restore --name "web1" --backup nginx-backup.tar --name nginx-restored
```

### Maintenance Commands

#### maintenance cleanup
System cleanup operations.

```bash
vps-tools maintenance cleanup [flags]
```

**Flags:**
- `--name, -n`: Target server name
- `--tags, -t`: Target servers by tags
- `--all, -a`: Target all servers
- `--level`: Cleanup level (minimal, standard, aggressive)
- `--paths`: Custom paths to clean
- `--exclude`: Patterns to exclude
- `--max-age`: Maximum file age
- `--dry-run`: Show what would be cleaned
- `--confirm`: Confirm before cleanup

**Examples:**
```bash
# Standard cleanup
vps-tools maintenance cleanup --name "web1" --level standard

# Aggressive cleanup
vps-tools maintenance cleanup --all --level aggressive

# Custom paths
vps-tools maintenance cleanup --name "web1" --paths "/tmp,/var/tmp" --max-age 30d

# Dry run
vps-tools maintenance cleanup --name "web1" --dry-run
```

#### maintenance update
Update system packages.

```bash
vps-tools maintenance update [flags]
```

**Flags:**
- `--name, -n`: Target server name
- `--tags, -t`: Target servers by tags
- `--all, -a`: Target all servers
- `--packages, -p`: Specific packages to update
- `--security-only`: Update security packages only
- `--exclude`: Packages to exclude
- `--autoremove`: Remove unused packages
- `--dry-run`: Show what would be updated

**Examples:**
```bash
# Update all packages
vps-tools maintenance update --name "web1"

# Update specific packages
vps-tools maintenance update --name "web1" --packages "nginx,postgresql"

# Security updates only
vps-tools maintenance update --all --security-only

# Dry run
vps-tools maintenance update --name "web1" --dry-run
```

#### maintenance backup
Backup system data.

```bash
vps-tools maintenance backup [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--output, -o`: Backup output file

**Optional Flags:**
- `--path`: Path to backup (default: /)
- `--database`: Database type (postgresql, mysql)
- `--database-name`: Database name
- `--compress`: Compress backup
- `--exclude`: Patterns to exclude
- `--schedule`: Schedule recurring backup

**Examples:**
```bash
# Backup directory
vps-tools maintenance backup --name "web1" --path "/var/www" --output web-backup.tar.gz

# Database backup
vps-tools maintenance backup --name "db1" --database postgresql --database-name app_db --output db-backup.sql

# Compressed backup
vps-tools maintenance backup --name "web1" --path "/var/www" --output web-backup.tar.gz --compress

# Schedule backup
vps-tools maintenance backup --name "web1" --path "/var/www" --output "web-backup-$(date +%Y%m%d).tar.gz" --schedule "0 2 * * *"
```

#### maintenance restore
Restore from backup.

```bash
vps-tools maintenance restore [flags]
```

**Required Flags:**
- `--name, -n`: Target server name
- `--backup, -b`: Backup file path

**Optional Flags:**
- `--path`: Restore path (default: /)
- `--database`: Database type
- `--database-name`: Database name
- `--force, -f`: Force restore
- `--verify`: Verify backup before restore

**Examples:**
```bash
# Restore files
vps-tools maintenance restore --name "web1" --backup web-backup.tar.gz --path "/var/www"

# Restore database
vps-tools maintenance restore --name "db1" --backup db-backup.sql --database postgresql --database-name app_db

# Verify before restore
vps-tools maintenance restore --name "web1" --backup web-backup.tar.gz --verify
```

#### maintenance optimize
System optimization.

```bash
vps-tools maintenance optimize [flags]
```

**Flags:**
- `--name, -n`: Target server name
- `--tags, -t`: Target servers by tags
- `--all, -a`: Target all servers
- `--level`: Optimization level (basic, standard, aggressive)
- `--config`: Custom optimization configuration
- `--dry-run`: Show what would be optimized
- `--reboot`: Reboot if required

**Examples:**
```bash
# Standard optimization
vps-tools maintenance optimize --name "web1" --level standard

# Aggressive optimization
vps-tools maintenance optimize --all --level aggressive

# Custom configuration
vps-tools maintenance optimize --name "web1" --config optimization.yaml

# Dry run
vps-tools maintenance optimize --name "web1" --dry-run
```

## TUI Interface

### Launch TUI

```bash
vps-tools tui [flags]
```

**Flags:**
- `--view, -v`: Initial view (servers, health, jobs, notifications, settings, help)
- `--theme, -t`: TUI theme (default, dark, light)
- `--refresh`: Refresh interval (default: 5s)
- `--debug`: Enable debug mode

### TUI Views

#### Servers View
- **Navigation**: ↑↓ to navigate, Enter to select
- **Actions**: 
  - `c`: Create new server
  - `e`: Edit selected server
  - `d`: Delete selected server
  - `t`: Test SSH connection
  - `h`: Run health check
  - `r`: Refresh list

#### Health View
- **Navigation**: ↑↓ to navigate, Tab to switch sections
- **Actions**:
  - `r`: Run health check
  - `m`: Start/stop monitoring
  - `c`: Configure thresholds
  - `e`: Export health data
  - `a`: Analyze trends

#### Jobs View
- **Navigation**: ↑↓ to navigate, Tab to switch sections
- **Actions**:
  - `c`: Create new job
  - `Enter`: View job details
  - `s`: Stop running job
  - `r`: Rerun job
  - `l`: View job logs

#### Notifications View
- **Navigation**: ↑↓ to navigate
- **Actions**:
  - `Enter`: View notification details
  - `a`: Acknowledge notification
  - `d`: Dismiss notification
  - `c`: Clear all notifications

#### Settings View
- **Navigation**: ↑↓ to navigate, Tab to switch sections
- **Actions**:
  - `e`: Edit setting
  - `r`: Reset to default
  - `s`: Save configuration
  - `l`: Reload configuration

#### Help View
- **Navigation**: ↑↓ to navigate, Tab to switch sections
- **Actions**:
  - `Enter`: Expand/collapse topics
  - `f`: Search help content
  - `Esc`: Return to previous view

### Global TUI Shortcuts
- `Tab`/`Shift+Tab`: Navigate between sections
- `Enter`: Select item/confirm action
- `Esc`: Go back/cancel action
- `q`: Quit application
- `Ctrl+C`: Force quit
- `r`: Refresh current view
- `f`: Filter/search items
- `s`: Sort items
- `h`: Show context help

## Go Library API

### Core Types

#### Server
```go
type Server struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Host        string    `json:"host"`
    Port        int       `json:"port"`
    User        string    `json:"user"`
    KeyPath     string    `json:"key_path,omitempty"`
    Password    string    `json:"password,omitempty"`
    Tags        []string  `json:"tags"`
    Description string    `json:"description,omitempty"`
    Group       string    `json:"group,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### HealthResult
```go
type HealthResult struct {
    ServerID    string                 `json:"server_id"`
    Timestamp   time.Time              `json:"timestamp"`
    Status      HealthStatus           `json:"status"`
    Metrics     map[string]interface{} `json:"metrics"`
    Checks      map[string]CheckResult `json:"checks"`
    Warnings    []string               `json:"warnings,omitempty"`
    Errors      []string               `json:"errors,omitempty"`
}

type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusWarning   HealthStatus = "warning"
    HealthStatusCritical  HealthStatus = "critical"
    HealthStatusUnknown   HealthStatus = "unknown"
)
```

#### Job
```go
type Job struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Type        JobType           `json:"type"`
    ServerID    string            `json:"server_id"`
    Command     string            `json:"command"`
    Status      JobStatus         `json:"status"`
    CreatedAt   time.Time         `json:"created_at"`
    StartedAt   *time.Time        `json:"started_at,omitempty"`
    CompletedAt *time.Time        `json:"completed_at,omitempty"`
    Duration    time.Duration     `json:"duration"`
    ExitCode    *int              `json:"exit_code,omitempty"`
    Output      string            `json:"output,omitempty"`
    Error       string            `json:"error,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

type JobType string

const (
    JobTypeCommand JobType = "command"
    JobTypeScript  JobType = "script"
    JobTypeBatch   JobType = "batch"
)

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
    JobStatusCancelled JobStatus = "cancelled"
)
```

### Application Interface

#### App
```go
type App interface {
    // Configuration
    Config() *config.Config
    
    // Storage
    Store() Store
    
    // Services
    SSHClient() ssh.Client
    HealthChecker() health.Checker
    JobRunner() job.Runner
    SecurityAuditor() security.Auditor
    DockerManager() docker.Manager
    MaintenanceManager() maintenance.Manager
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

#### Store Interface
```go
type Store interface {
    // Servers
    CreateServer(server *Server) error
    GetServer(id string) (*Server, error)
    UpdateServer(server *Server) error
    DeleteServer(id string) error
    ListServers(filter ServerFilter) ([]*Server, error)
    
    // Health Results
    SaveHealthResult(result *HealthResult) error
    GetHealthResults(filter HealthFilter) ([]*HealthResult, error)
    DeleteHealthResults(filter HealthFilter) error
    
    // Jobs
    CreateJob(job *Job) error
    GetJob(id string) (*Job, error)
    UpdateJob(job *Job) error
    ListJobs(filter JobFilter) ([]*Job, error)
    DeleteJob(id string) error
    
    // Close database connection
    Close() error
}
```

### Service Interfaces

#### SSH Client
```go
type Client interface {
    Connect(ctx context.Context, server *Server) (Session, error)
    TestConnection(ctx context.Context, server *Server) error
    Close() error
}

type Session interface {
    Execute(ctx context.Context, cmd string, opts ExecuteOptions) (*Result, error)
    Upload(ctx context.Context, localPath, remotePath string) error
    Download(ctx context.Context, remotePath, localPath string) error
    Close() error
}

type ExecuteOptions struct {
    Timeout      time.Duration
    WorkingDir   string
    Environment  map[string]string
    User         string
    CaptureOutput bool
}

type Result struct {
    ExitCode int
    Output   string
    Error    string
    Duration time.Duration
}
```

#### Health Checker
```go
type Checker interface {
    Check(ctx context.Context, server *Server, opts CheckOptions) (*HealthResult, error)
    Monitor(ctx context.Context, servers []*Server, opts MonitorOptions) (<-chan *HealthResult, error)
    Analyze(ctx context.Context, filter HealthFilter) (*Analysis, error)
}

type CheckOptions struct {
    Timeout         time.Duration
    CustomChecks    []CustomCheck
    Thresholds      map[string]float64
    IncludeMetrics  bool
}

type MonitorOptions struct {
    Interval        time.Duration
    Duration        time.Duration
    Thresholds      map[string]float64
    AlertCallback   func(*HealthResult)
}
```

#### Job Runner
```go
type Runner interface {
    Execute(ctx context.Context, job *Job) error
    ExecuteBatch(ctx context.Context, jobs []*Job, opts BatchOptions) ([]*Job, error)
    Schedule(ctx context.Context, job *Job, schedule string) error
    Cancel(ctx context.Context, jobID string) error
    GetStatus(ctx context.Context, jobID string) (*JobStatus, error)
}

type BatchOptions struct {
    MaxConcurrent   int
    ContinueOnError bool
    Timeout         time.Duration
}
```

### Usage Examples

#### Basic Usage
```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/pgd1001/vps-tools/internal/app"
    "github.com/pgd1001/vps-tools/internal/config"
    "github.com/pgd1001/vps-tools/internal/server"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create application
    application, err := app.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer application.Stop(context.Background())
    
    // Start application
    if err := application.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
    
    // List servers
    servers, err := application.Store().ListServers(server.Filter{})
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d servers\n", len(servers))
    
    // Run health check
    for _, srv := range servers {
        result, err := application.HealthChecker().Check(context.Background(), srv, health.CheckOptions{})
        if err != nil {
            fmt.Printf("Health check failed for %s: %v\n", srv.Name, err)
            continue
        }
        fmt.Printf("Server %s health: %s\n", srv.Name, result.Status)
    }
}
```

#### Custom Health Check
```go
// Custom health check implementation
type CustomHealthCheck struct {
    name string
    cmd  string
}

func (c *CustomHealthCheck) Name() string {
    return c.name
}

func (c *CustomHealthCheck) Execute(ctx context.Context, session ssh.Session) (*health.CheckResult, error) {
    result, err := session.Execute(ctx, c.cmd, ssh.ExecuteOptions{
        Timeout: 30 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    
    return &health.CheckResult{
        Name:    c.name,
        Status:  health.CheckStatus(result.ExitCode == 0),
        Output:  result.Output,
        Error:   result.Error,
        Metrics: map[string]interface{}{
            "exit_code": result.ExitCode,
            "duration":  result.Duration,
        },
    }, nil
}

// Register custom check
customCheck := &CustomHealthCheck{
    name: "nginx_status",
    cmd:  "curl -f http://localhost/nginx_status",
}

checker := application.HealthChecker()
checker.RegisterCustomCheck(customCheck)
```

#### Job Execution
```go
// Create and execute job
job := &server.Job{
    Name:     "deploy_app",
    Type:     server.JobTypeScript,
    ServerID: srv.ID,
    Command:  "/opt/scripts/deploy.sh",
    Metadata: map[string]string{
        "environment": "production",
        "backup":      "true",
    },
}

err := application.JobRunner().Execute(context.Background(), job)
if err != nil {
    log.Printf("Job execution failed: %v", err)
}

// Batch execution
jobs := []*server.Job{
    {
        Name:     "update_packages",
        Type:     server.JobTypeCommand,
        ServerID: srv1.ID,
        Command:  "apt update && apt upgrade -y",
    },
    {
        Name:     "update_packages",
        Type:     server.JobTypeCommand,
        ServerID: srv2.ID,
        Command:  "apt update && apt upgrade -y",
    },
}

results, err := application.JobRunner().ExecuteBatch(context.Background(), jobs, job.BatchOptions{
    MaxConcurrent:   5,
    ContinueOnError: true,
    Timeout:         10 * time.Minute,
})
if err != nil {
    log.Printf("Batch execution failed: %v", err)
}

for _, result := range results {
    fmt.Printf("Job %s on server %s: %s\n", result.Name, result.ServerID, result.Status)
}
```

## Configuration API

### Configuration Structure

```go
type Config struct {
    App         AppConfig         `yaml:"app"`
    Database    DatabaseConfig    `yaml:"database"`
    Logging     LoggingConfig     `yaml:"logging"`
    SSH         SSHConfig         `yaml:"ssh"`
    Health      HealthConfig      `yaml:"health"`
    Security    SecurityConfig    `yaml:"security"`
    Docker      DockerConfig      `yaml:"docker"`
    Maintenance MaintenanceConfig `yaml:"maintenance"`
    TUI         TUIConfig         `yaml:"tui"`
    Notifications NotificationsConfig `yaml:"notifications"`
    Plugins     PluginsConfig     `yaml:"plugins"`
}
```

### Configuration Loading

```go
// Load configuration from file
cfg, err := config.LoadFromFile("/path/to/config.yaml")
if err != nil {
    return err
}

// Load configuration with overrides
cfg, err := config.LoadWithOverrides(config.LoadOptions{
    FilePath: "/path/to/config.yaml",
    EnvPrefix: "VPS_TOOLS_",
    Overrides: map[string]interface{}{
        "logging.level": "debug",
        "ssh.timeout": "60s",
    },
})
if err != nil {
    return err
}

// Validate configuration
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

### Configuration Management

```go
// Get configuration value
value := cfg.Get("ssh.timeout")
if value != nil {
    timeout := value.(time.Duration)
    fmt.Printf("SSH timeout: %v\n", timeout)
}

// Set configuration value
cfg.Set("logging.level", "debug")

// Save configuration
if err := cfg.Save("/path/to/config.yaml"); err != nil {
    return err
}

// Watch for configuration changes
watcher, err := cfg.Watch()
if err != nil {
    return err
}

go func() {
    for event := range watcher.Events() {
        fmt.Printf("Configuration changed: %s = %v\n", event.Key, event.Value)
    }
}()
```

## Plugin API

### Plugin Interface

```go
type Plugin interface {
    // Plugin metadata
    Name() string
    Version() string
    Description() string
    Author() string
    
    // Plugin lifecycle
    Initialize(ctx context.Context, cfg *config.Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Cleanup() error
    
    // Plugin functionality
    Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error)
    Validate(input *PluginInput) error
}

type PluginInput struct {
    Action   string                 `json:"action"`
    Server   *Server                `json:"server,omitempty"`
    Data     map[string]interface{} `json:"data"`
    Options  map[string]interface{} `json:"options"`
}

type PluginOutput struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data"`
    Error   string                 `json:"error,omitempty"`
    Metrics map[string]interface{} `json:"metrics,omitempty"`
}
```

### Plugin Registration

```go
// Plugin registration function
func init() {
    plugin.Register(&CustomPlugin{})
}

// Custom plugin implementation
type CustomPlugin struct {
    cfg *config.Config
}

func (p *CustomPlugin) Name() string {
    return "custom"
}

func (p *CustomPlugin) Version() string {
    return "1.0.0"
}

func (p *CustomPlugin) Description() string {
    return "Custom plugin for specific functionality"
}

func (p *CustomPlugin) Author() string {
    return "Your Name"
}

func (p *CustomPlugin) Initialize(ctx context.Context, cfg *config.Config) error {
    p.cfg = cfg
    return nil
}

func (p *CustomPlugin) Start(ctx context.Context) error {
    // Plugin startup logic
    return nil
}

func (p *CustomPlugin) Stop(ctx context.Context) error {
    // Plugin shutdown logic
    return nil
}

func (p *CustomPlugin) Cleanup() error {
    // Plugin cleanup logic
    return nil
}

func (p *CustomPlugin) Execute(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
    switch input.Action {
    case "custom_action":
        return p.executeCustomAction(ctx, input)
    default:
        return nil, fmt.Errorf("unknown action: %s", input.Action)
    }
}

func (p *CustomPlugin) Validate(input *PluginInput) error {
    if input.Action == "" {
        return fmt.Errorf("action is required")
    }
    return nil
}

func (p *CustomPlugin) executeCustomAction(ctx context.Context, input *PluginInput) (*PluginOutput, error) {
    // Custom action implementation
    return &PluginOutput{
        Success: true,
        Data: map[string]interface{}{
            "result": "custom action completed",
        },
    }, nil
}
```

### Plugin Manager

```go
// Plugin manager interface
type Manager interface {
    LoadPlugins(directory string) error
    RegisterPlugin(plugin Plugin) error
    GetPlugin(name string) (Plugin, error)
    ListPlugins() []Plugin
    ExecutePlugin(ctx context.Context, name string, input *PluginInput) (*PluginOutput, error)
    Shutdown() error
}

// Use plugin manager
manager := plugin.NewManager()

// Load plugins from directory
err := manager.LoadPlugins("/path/to/plugins")
if err != nil {
    log.Fatal(err)
}

// Execute plugin
output, err := manager.ExecutePlugin(context.Background(), "custom", &plugin.PluginInput{
    Action: "custom_action",
    Data: map[string]interface{}{
        "param1": "value1",
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Plugin output: %+v\n", output)
```

This comprehensive API reference covers all aspects of VPS Tools, from CLI commands to programmatic interfaces and plugin development.