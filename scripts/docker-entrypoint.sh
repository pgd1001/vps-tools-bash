#!/bin/sh

# Docker entrypoint script for VPS Tools
# This script handles initialization and configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to create default configuration
create_default_config() {
    if [ ! -f "/opt/vps-tools/config/config.yaml" ]; then
        print_status "Creating default configuration..."
        
        cat > /opt/vps-tools/config/config.yaml << 'EOF'
app:
  name: "vps-tools"
  debug: false

database:
  path: "/opt/vps-tools/data/vps-tools.db"

logging:
  level: "info"
  format: "json"
  file: "/opt/vps-tools/logs/vps-tools.log"
  max_size: "100MB"
  max_backups: 10
  max_age: "30d"

ssh:
  timeout: "30s"
  max_retries: 3
  keep_alive: "10s"
  key_paths:
    - "/opt/vps-tools/.ssh/id_rsa"
    - "/opt/vps-tools/.ssh/id_ed25519"
  strict_host_key: false

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

security:
  default_port_range: "1-1000"
  scan_timeout: "5s"
  max_concurrent_scans: 100
  ssh_key_scan: true
  vulnerability_check: true

docker:
  socket_path: "/var/run/docker.sock"
  timeout: "30s"
  default_registry: "docker.io"
  cleanup_interval: "24h"
  health_check_interval: "30s"

maintenance:
  backup_path: "/opt/vps-tools/backups"
  log_retention: "30d"
  auto_cleanup: true

tui:
  theme: "default"
  refresh_interval: "5s"
  max_log_lines: 1000
  confirm_destructive: true
  show_notifications: true

notifications:
  enabled: true
  methods:
    - "log"
  email:
    enabled: false

plugins:
  enabled: false
  directory: "/opt/vps-tools/plugins"
EOF

        print_success "Default configuration created"
    fi
}

# Function to create SSH directory
create_ssh_directory() {
    if [ ! -d "/opt/vps-tools/.ssh" ]; then
        print_status "Creating SSH directory..."
        mkdir -p /opt/vps-tools/.ssh
        chmod 700 /opt/vps-tools/.ssh
        print_success "SSH directory created"
    fi
}

# Function to setup permissions
setup_permissions() {
    print_status "Setting up permissions..."
    
    # Ensure proper ownership
    chown -R vps-tools:vps-tools /opt/vps-tools
    
    # Set proper permissions
    chmod 755 /opt/vps-tools
    chmod 755 /opt/vps-tools/config
    chmod 755 /opt/vps-tools/data
    chmod 755 /opt/vps-tools/logs
    chmod 755 /opt/vps-tools/backups
    chmod 755 /opt/vps-tools/plugins
    
    print_success "Permissions set up"
}

# Function to initialize database
initialize_database() {
    print_status "Initializing database..."
    
    # The application will create the database on first run
    # Just ensure the directory exists and has proper permissions
    mkdir -p /opt/vps-tools/data
    chmod 755 /opt/vps-tools/data
    
    print_success "Database directory ready"
}

# Function to check Docker socket
check_docker_socket() {
    if [ -S "/var/run/docker.sock" ]; then
        print_status "Docker socket found"
        
        # Check if we can access it
        if [ -r "/var/run/docker.sock" ]; then
            print_success "Docker socket is accessible"
        else
            print_warning "Docker socket is not accessible - Docker features will be limited"
        fi
    else
        print_warning "Docker socket not found - Docker features will be disabled"
    fi
}

# Function to show startup banner
show_banner() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}║                    VPS Tools v${VERSION:-dev}                        ║${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}║              Modern VPS Management Suite                        ║${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Function to handle signals
cleanup() {
    print_status "Received shutdown signal, cleaning up..."
    exit 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Main initialization
main() {
    show_banner
    
    # Only run initialization if not already done
    if [ ! -f "/opt/vps-tools/.initialized" ]; then
        print_status "Initializing VPS Tools..."
        
        create_default_config
        create_ssh_directory
        setup_permissions
        initialize_database
        check_docker_socket
        
        # Mark as initialized
        touch /opt/vps-tools/.initialized
        
        print_success "Initialization completed"
    else
        print_status "VPS Tools already initialized"
    fi
    
    # Show configuration info
    print_status "Configuration file: ${VPS_TOOLS_CONFIG:-/opt/vps-tools/config/config.yaml}"
    print_status "Data directory: ${VPS_TOOLS_DATA_PATH:-/opt/vps-tools/data}"
    print_status "Log directory: ${VPS_TOOLS_LOG_PATH:-/opt/vps-tools/logs}"
    
    echo ""
    
    # Execute the command
    if [ $# -eq 0 ]; then
        # Default to daemon mode
        exec /opt/vps-tools/vps-tools daemon
    else
        exec /opt/vps-tools/vps-tools "$@"
    fi
}

# Run main function
main "$@"