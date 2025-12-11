# VPS Tools

A modern, comprehensive VPS management suite written in Go that provides both CLI and TUI interfaces for server inventory management, health monitoring, security auditing, Docker management, and system maintenance.

## 🚀 Features

### Core Functionality
- **📋 Server Inventory Management**: Add, edit, delete, and organize servers with tags
- **🔍 Real-time Health Monitoring**: Monitor CPU, memory, disk, and network metrics
- **⚡ Command Execution**: Run single or batch commands across multiple servers
- **🛡️ Security Auditing**: SSH key analysis, port scanning, and vulnerability detection
- **🐳 Docker Management**: Container lifecycle management and monitoring
- **🔧 System Maintenance**: Automated cleanup, updates, and backup operations

### User Interfaces
- **🖥️ Interactive TUI**: Rich terminal interface with real-time updates
- **⌨️ CLI Commands**: Full command-line interface for scripting and automation
- **📊 Visual Monitoring**: Live dashboards and health indicators

## 📦 Installation

### From Source
```bash
git clone https://github.com/pgd1001/vps-tools.git
cd vps-tools
make build
```

### Binary Release
Download the latest release from the [releases page](https://github.com/pgd1001/vps-tools/releases) and add to your PATH.

## 🎯 Quick Start

### 1. Initialize Configuration
```bash
vps-tools config init
```

### 2. Add Your First Server
```bash
vps-tools inventory add --name "web-server" --host "192.168.1.100" --user "admin" --tags "web,production"
```

### 3. Launch TUI
```bash
vps-tools tui
```

### 4. Check Server Health
```bash
vps-tools health check --all
```

## 📖 Usage

### CLI Commands

#### Server Management
```bash
# Add a server
vps-tools inventory add --name "db-server" --host "192.168.1.101" --user "admin" --key "~/.ssh/id_rsa"

# List all servers
vps-tools inventory list

# Edit server details
vps-tools inventory edit --name "web-server" --tags "web,production,updated"

# Remove a server
vps-tools inventory remove --name "web-server"

# Test SSH connectivity
vps-tools inventory test --name "web-server"
```

#### Health Monitoring
```bash
# Check all servers
vps-tools health check --all

# Check specific server
vps-tools health check --name "web-server"

# Continuous monitoring
vps-tools health monitor --interval 30s

# Generate health report
vps-tools health report --output health-report.json
```

#### Command Execution
```bash
# Run single command
vps-tools run command --name "web-server" --cmd "uptime"

# Run batch commands
vps-tools run batch --servers "web-server,db-server" --commands "uptime,df -h"

# Execute script
vps-tools run script --name "web-server" --script "./deploy.sh"

# Schedule job
vps-tools run schedule --name "web-server" --cmd "backup.sh" --schedule "0 2 * * *"
```

#### Security Auditing
```bash
# Full security audit
vps-tools security audit --name "web-server"

# SSH key analysis
vps-tools security ssh-keys --name "web-server"

# Port scanning
vps-tools security ports --name "web-server" --range "1-1000"

# Generate security report
vps-tools security report --name "web-server" --output security-report.json
```

#### Docker Management
```bash
# List containers
vps-tools docker list --name "web-server"

# Start container
vps-tools docker start --name "web-server" --container "nginx"

# Stop container
vps-tools docker stop --name "web-server" --container "nginx"

# Remove container
vps-tools docker remove --name "web-server" --container "nginx"

# Container health check
vps-tools docker health --name "web-server" --container "nginx"

# Backup container
vps-tools docker backup --name "web-server" --container "nginx" --output "nginx-backup.tar"
```

#### System Maintenance
```bash
# System cleanup
vps-tools maintenance cleanup --name "web-server" --level "standard"

# Update packages
vps-tools maintenance update --name "web-server" --packages "nginx,postgresql"

# Create backup
vps-tools maintenance backup --name "web-server" --path "/var/www" --output "backup.tar.gz"

# Restore from backup
vps-tools maintenance restore --name "web-server" --backup "backup.tar.gz" --path "/var/www"
```

### TUI Interface

Launch the interactive terminal user interface:
```bash
vps-tools tui
```

#### TUI Navigation
- **Tab/Shift+Tab**: Navigate between sections
- **↑/↓**: Navigate within lists
- **Enter**: Select item/confirm action
- **Esc**: Go back/cancel
- **q**: Quit application
- **1-6**: Quick switch between views
- **c**: Create new item
- **e**: Edit selected item
- **d**: Delete selected item
- **r**: Refresh data
- **f**: Filter/search
- **s**: Sort items
- **h**: Show help

#### TUI Views
1. **Servers**: Server inventory management
2. **Health**: Real-time health monitoring
3. **Jobs**: Command execution and job management
4. **Notifications**: System notifications and alerts
5. **Settings**: Application configuration
6. **Help**: Interactive help and shortcuts

## ⚙️ Configuration

### Configuration File
Configuration is stored in `~/.config/vps-tools/config.yaml`:

```yaml
# Database
database:
  path: "~/.local/share/vps-tools/data.db"

# Logging
logging:
  level: "info"
  format: "json"
  file: "~/.local/share/vps-tools/logs/vps-tools.log"

# SSH Settings
ssh:
  timeout: "30s"
  max_retries: 3
  key_paths:
    - "~/.ssh/id_rsa"
    - "~/.ssh/id_ed25519"

# Health Monitoring
health:
  default_interval: "60s"
  thresholds:
    cpu_warning: 70
    cpu_critical: 90
    memory_warning: 80
    memory_critical: 95
    disk_warning: 80
    disk_critical: 95

# Security
security:
  default_port_range: "1-1000"
  ssh_key_scan: true
  vulnerability_check: true

# Docker
docker:
  socket_path: "/var/run/docker.sock"
  default_registry: "docker.io"
  cleanup_interval: "24h"

# Maintenance
maintenance:
  backup_path: "~/.local/share/vps-tools/backups"
  log_retention: "30d"
  auto_cleanup: true
```

### Environment Variables
```bash
# Override configuration file location
export VPS_TOOLS_CONFIG="/path/to/config.yaml"

# Set log level
export VPS_TOOLS_LOG_LEVEL="debug"

# Override database path
export VPS_TOOLS_DB_PATH="/path/to/database.db"
```

## 🔧 Development

### Prerequisites
- Go 1.21 or later
- Make
- Git

### Build Commands
```bash
# Build binary
make build

# Run tests
make test

# Run specific test
go test ./internal/server -run TestServer

# Format code
make fmt

# Run linter
make lint

# Run TUI in development
make tui

# Run CLI in development
make run
```

### Project Structure
```
vps-tools/
├── cmd/                 # CLI commands
│   ├── inventory/      # Server inventory commands
│   ├── health/         # Health monitoring commands
│   ├── run/            # Command execution commands
│   ├── security/       # Security auditing commands
│   ├── docker/         # Docker management commands
│   ├── maintenance/    # System maintenance commands
│   └── root.go         # Root command setup
├── internal/            # Internal packages
│   ├── app/           # Application configuration
│   ├── server/        # Server domain models
│   ├── config/        # Configuration management
│   ├── store/         # Database storage layer
│   ├── logger/        # Structured logging
│   └── ssh/           # SSH client infrastructure
├── tui/                # Terminal user interface
│   ├── main.go        # TUI application entry point
│   ├── models/        # TUI data models
│   ├── components/    # TUI UI components
│   └── styles/        # TUI styling system
├── pkg/               # Public packages
├── scripts/           # Utility scripts
├── docs/              # Documentation
├── Makefile           # Build configuration
├── go.mod             # Go module definition
└── README.md          # This file
```

### Contributing
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style
- Use `gofmt` for code formatting
- Follow Go conventions and best practices
- Write comprehensive tests for new features
- Update documentation for API changes

## 🧪 Testing

### Unit Tests
```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/server
```

### Integration Tests
```bash
# Run integration tests
go test -tags=integration ./tests/integration/...

# Run end-to-end tests
go test -tags=e2e ./tests/e2e/...
```

## 📚 Documentation

- [User Guide](docs/user-guide.md) - Comprehensive usage documentation
- [API Reference](docs/api-reference.md) - CLI and TUI API documentation
- [Architecture](docs/architecture.md) - System architecture and design
- [Development Guide](docs/development.md) - Development setup and guidelines

## 🤝 Support

- **Issues**: [GitHub Issues](https://github.com/pgd1001/vps-tools/issues)
- **Discussions**: [GitHub Discussions](https://github.com/pgd1001/vps-tools/discussions)
- **Wiki**: [GitHub Wiki](https://github.com/pgd1001/vps-tools/wiki)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Cobra](https://github.com/spf13/cobra) for the CLI framework
- [BoltDB](https://github.com/etcd-io/bbolt) for embedded database
- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) for SSH operations

## 🔄 Migration from Bash Scripts

If you're migrating from the original bash scripts, see the [Migration Guide](docs/migration.md) for a complete mapping of old commands to new ones.

---

**VPS Tools** - Modern VPS management for the terminal age. 🚀