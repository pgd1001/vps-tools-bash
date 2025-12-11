#!/bin/bash

# Deployment script for VPS Tools
# This script handles deployment to various environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="vps-tools"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
DEPLOY_ENV="${DEPLOY_ENV:-staging}"
REMOTE_USER="${REMOTE_USER:-deploy}"
REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_PATH="${REMOTE_PATH:-/opt/${APP_NAME}}"
SERVICE_NAME="${SERVICE_NAME:-${APP_NAME}}"

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

# Function to check dependencies
check_dependencies() {
    print_status "Checking dependencies..."
    
    local deps=("ssh" "scp" "rsync")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            print_error "Missing dependency: $dep"
            exit 1
        fi
    done
    
    print_success "All dependencies found"
}

# Function to validate environment
validate_environment() {
    print_status "Validating deployment environment..."
    
    if [ -z "$REMOTE_HOST" ]; then
        print_error "REMOTE_HOST is not set"
        exit 1
    fi
    
    if [ -z "$REMOTE_USER" ]; then
        print_error "REMOTE_USER is not set"
        exit 1
    fi
    
    # Validate deployment environment
    case "$DEPLOY_ENV" in
        staging|production|development)
            print_success "Valid deployment environment: $DEPLOY_ENV"
            ;;
        *)
            print_error "Invalid deployment environment: $DEPLOY_ENV"
            print_error "Valid environments: staging, production, development"
            exit 1
            ;;
    esac
}

# Function to build application
build_application() {
    print_status "Building application..."
    
    # Build for target platform
    ./scripts/build.sh --version "$VERSION"
    
    if [ ! -f "release/${APP_NAME}-linux-amd64.tar.gz" ]; then
        print_error "Build failed - binary not found"
        exit 1
    fi
    
    print_success "Application built successfully"
}

# Function to test connection
test_connection() {
    print_status "Testing SSH connection to $REMOTE_USER@$REMOTE_HOST..."
    
    if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$REMOTE_USER@$REMOTE_HOST" "echo 'Connection successful'"; then
        print_error "SSH connection failed"
        exit 1
    fi
    
    print_success "SSH connection successful"
}

# Function to create backup
create_backup() {
    print_status "Creating backup of current deployment..."
    
    # Check if application exists
    if ssh "$REMOTE_USER@$REMOTE_HOST" "[ -d '$REMOTE_PATH' ]"; then
        local backup_name="${APP_NAME}-backup-$(date +%Y%m%d-%H%M%S)"
        ssh "$REMOTE_USER@$REMOTE_HOST" "sudo cp -r '$REMOTE_PATH' '/opt/backups/$backup_name'"
        print_success "Backup created: $backup_name"
    else
        print_warning "No existing deployment to backup"
    fi
}

# Function to deploy files
deploy_files() {
    print_status "Deploying files to remote server..."
    
    # Create remote directory if it doesn't exist
    ssh "$REMOTE_USER@$REMOTE_HOST" "sudo mkdir -p '$REMOTE_PATH'"
    
    # Extract and upload binary
    local temp_dir="/tmp/${APP_NAME}-deploy-$$"
    ssh "$REMOTE_USER@$REMOTE_HOST" "mkdir -p '$temp_dir'"
    
    # Upload binary
    scp "release/${APP_NAME}-linux-amd64.tar.gz" "$REMOTE_USER@$REMOTE_HOST:$temp_dir/"
    
    # Extract on remote server
    ssh "$REMOTE_USER@$REMOTE_HOST" "
        cd '$temp_dir'
        tar -xzf ${APP_NAME}-linux-amd64.tar.gz
        sudo mv ${APP_NAME}-linux-amd64 '$REMOTE_PATH/${APP_NAME}'
        sudo chmod +x '$REMOTE_PATH/${APP_NAME}'
        rm -rf '$temp_dir'
    "
    
    print_success "Files deployed successfully"
}

# Function to deploy configuration
deploy_configuration() {
    print_status "Deploying configuration files..."
    
    # Create configuration directory
    ssh "$REMOTE_USER@$REMOTE_HOST" "sudo mkdir -p '$REMOTE_PATH/config'"
    
    # Upload configuration files if they exist
    if [ -d "config/$DEPLOY_ENV" ]; then
        rsync -avz "config/$DEPLOY_ENV/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH/config/"
        print_success "Configuration files deployed"
    else
        print_warning "No configuration files found for $DEPLOY_ENV"
    fi
}

# Function to install systemd service
install_service() {
    print_status "Installing systemd service..."
    
    # Create service file
    cat > "/tmp/${SERVICE_NAME}.service" << EOF
[Unit]
Description=VPS Tools
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$REMOTE_PATH
ExecStart=$REMOTE_PATH/${APP_NAME} daemon
Restart=always
RestartSec=5
Environment=VPS_TOOLS_CONFIG=$REMOTE_PATH/config/config.yaml

[Install]
WantedBy=multi-user.target
EOF
    
    # Upload and install service
    scp "/tmp/${SERVICE_NAME}.service" "$REMOTE_USER@$REMOTE_HOST:/tmp/"
    ssh "$REMOTE_USER@$REMOTE_HOST" "
        sudo mv /tmp/${SERVICE_NAME}.service /etc/systemd/system/
        sudo systemctl daemon-reload
        sudo systemctl enable ${SERVICE_NAME}
    "
    
    rm -f "/tmp/${SERVICE_NAME}.service"
    print_success "Systemd service installed"
}

# Function to run health check
run_health_check() {
    print_status "Running deployment health check..."
    
    # Wait for service to start
    sleep 5
    
    # Check if service is running
    if ssh "$REMOTE_USER@$REMOTE_HOST" "sudo systemctl is-active --quiet $SERVICE_NAME"; then
        print_success "Service is running"
    else
        print_error "Service failed to start"
        ssh "$REMOTE_USER@$REMOTE_HOST" "sudo systemctl status $SERVICE_NAME"
        exit 1
    fi
    
    # Check if application responds
    if ssh "$REMOTE_USER@$REMOTE_HOST" "$REMOTE_PATH/${APP_NAME} --version >/dev/null 2>&1"; then
        print_success "Application is responding"
    else
        print_warning "Application not responding (may be starting up)"
    fi
}

# Function to rollback deployment
rollback_deployment() {
    print_status "Rolling back deployment..."
    
    # Find latest backup
    local latest_backup=$(ssh "$REMOTE_USER@$REMOTE_HOST" "ls -1t /opt/backups/ | grep '${APP_NAME}-backup' | head -1" 2>/dev/null || echo "")
    
    if [ -n "$latest_backup" ]; then
        print_status "Rolling back to: $latest_backup"
        
        ssh "$REMOTE_USER@$REMOTE_HOST" "
            sudo systemctl stop $SERVICE_NAME || true
            sudo rm -rf '$REMOTE_PATH'
            sudo mv '/opt/backups/$latest_backup' '$REMOTE_PATH'
            sudo systemctl start $SERVICE_NAME
        "
        
        print_success "Rollback completed"
    else
        print_error "No backup found for rollback"
        exit 1
    fi
}

# Function to cleanup old backups
cleanup_backups() {
    print_status "Cleaning up old backups..."
    
    # Keep last 5 backups
    ssh "$REMOTE_USER@$REMOTE_HOST" "
        cd /opt/backups
        ls -1t ${APP_NAME}-backup-* | tail -n +6 | xargs -r rm -rf
    "
    
    print_success "Old backups cleaned up"
}

# Function to show deployment info
show_deployment_info() {
    print_status "Deployment Information:"
    echo "  Application: $APP_NAME"
    echo "  Version: $VERSION"
    echo "  Environment: $DEPLOY_ENV"
    echo "  Remote Host: $REMOTE_USER@$REMOTE_HOST"
    echo "  Remote Path: $REMOTE_PATH"
    echo "  Service: $SERVICE_NAME"
    echo ""
}

# Function to deploy to staging
deploy_staging() {
    print_status "Deploying to staging environment..."
    
    DEPLOY_ENV="staging"
    REMOTE_HOST="${STAGING_HOST:-staging.example.com}"
    REMOTE_USER="${STAGING_USER:-deploy}"
    REMOTE_PATH="${STAGING_PATH:-/opt/${APP_NAME}-staging}"
    SERVICE_NAME="${APP_NAME}-staging"
    
    perform_deployment
}

# Function to deploy to production
deploy_production() {
    print_status "Deploying to production environment..."
    
    # Confirm production deployment
    read -p "Are you sure you want to deploy to production? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Production deployment cancelled"
        exit 0
    fi
    
    DEPLOY_ENV="production"
    REMOTE_HOST="${PRODUCTION_HOST:-production.example.com}"
    REMOTE_USER="${PRODUCTION_USER:-deploy}"
    REMOTE_PATH="${PRODUCTION_PATH:-/opt/${APP_NAME}}"
    SERVICE_NAME="${APP_NAME}"
    
    perform_deployment
}

# Function to perform deployment
perform_deployment() {
    show_deployment_info
    
    check_dependencies
    validate_environment
    test_connection
    build_application
    create_backup
    deploy_files
    deploy_configuration
    install_service
    run_health_check
    cleanup_backups
    
    print_success "Deployment to $DEPLOY_ENV completed successfully!"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  staging       Deploy to staging environment"
    echo "  production   Deploy to production environment"
    echo "  rollback     Rollback to previous deployment"
    echo "  health-check  Run health check on deployed application"
    echo ""
    echo "Options:"
    echo "  --version VERSION    Set version (default: git describe)"
    echo "  --env ENVIRONMENT  Set deployment environment (staging|production|development)"
    echo "  --host HOST        Set remote host"
    echo "  --user USER        Set remote user"
    echo "  --path PATH        Set remote path"
    echo "  --help             Show this help"
    echo ""
    echo "Environment Variables:"
    echo "  VERSION            Application version"
    echo "  DEPLOY_ENV         Deployment environment"
    echo "  REMOTE_HOST        Remote host"
    echo "  REMOTE_USER        Remote user"
    echo "  REMOTE_PATH        Remote path"
    echo "  SERVICE_NAME       Systemd service name"
    echo "  STAGING_HOST       Staging host"
    echo "  STAGING_USER       Staging user"
    echo "  STAGING_PATH       Staging path"
    echo "  PRODUCTION_HOST    Production host"
    echo "  PRODUCTION_USER    Production user"
    echo "  PRODUCTION_PATH    Production path"
}

# Main function
main() {
    case "${1:-}" in
        staging)
            deploy_staging
            ;;
        production)
            deploy_production
            ;;
        rollback)
            rollback_deployment
            ;;
        health-check)
            run_health_check
            ;;
        --help|help)
            show_usage
            exit 0
            ;;
        "")
            print_error "No command specified"
            show_usage
            exit 1
            ;;
        *)
            print_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --env)
            DEPLOY_ENV="$2"
            shift 2
            ;;
        --host)
            REMOTE_HOST="$2"
            shift 2
            ;;
        --user)
            REMOTE_USER="$2"
            shift 2
            ;;
        --path)
            REMOTE_PATH="$2"
            shift 2
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            # Pass to main function
            break
            ;;
    esac
done

# Run main function
main "$@"