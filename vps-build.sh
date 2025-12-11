#!/bin/bash
set -euo pipefail

# Ubuntu 24.04 VPS Provisioning Script - Best Practice Configuration
# Run as root or with sudo: sudo bash provision_vps.sh

readonly SCRIPT_VERSION="1.3"

# Colours for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Application selection
SELECTED_APP=""
MODE="fresh"

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

select_mode() {
    echo
    log_info "=== Script Mode ==="
    echo
    echo "Choose mode:"
    echo "  1) Fresh install (full provisioning)"
    echo "  2) Reconfigure existing system"
    echo "  3) Troubleshooting/Status review"
    echo
    
    read -p "Enter selection (1-3, default: 1): " mode_choice
    mode_choice=${mode_choice:-1}
    
    case "$mode_choice" in
        1) MODE="fresh" ;;
        2) MODE="reconfigure"; show_reconfigure_options ;;
        3) MODE="troubleshoot"; show_troubleshoot_menu ;;
        *) log_error "Invalid selection"; select_mode ;;
    esac
}

show_reconfigure_options() {
    echo
    log_info "Reconfigurable components:"
    echo "  1) SSH keys from GitHub (add/update)"
    echo "  2) SSH configuration"
    echo "  3) Firewall rules (UFW)"
    echo "  4) Hostname"
    echo "  5) Timezone"
    echo "  0) Go back"
    echo
    
    read -p "Select component to reconfigure (0-5): " component_choice
    
    case "$component_choice" in
        1) reconfigure_github_keys; exit 0 ;;
        2) reconfigure_ssh ;;
        3) reconfigure_firewall_rules ;;
        4) reconfigure_hostname ;;
        5) reconfigure_timezone ;;
        0) select_mode ;;
        *) log_error "Invalid selection"; show_reconfigure_options ;;
    esac
}

reconfigure_github_keys() {
    read -p "Enter username to add SSH keys for: " fix_username
    
    if ! id "$fix_username" &>/dev/null; then
        log_error "User $fix_username does not exist"
        exit 1
    fi
    
    read -p "Enter GitHub username: " GITHUB_USER
    setup_github_ssh_keys "$fix_username" "$GITHUB_USER"
    log_success "GitHub SSH keys updated for $fix_username"
}

reconfigure_ssh() {
    read -p "Enter new SSH port (default: 22): " SSH_PORT
    SSH_PORT=${SSH_PORT:-22}
    read -p "Disable root SSH login? (yes/no, default: yes): " DISABLE_ROOT_LOGIN
    DISABLE_ROOT_LOGIN=${DISABLE_ROOT_LOGIN:-yes}
    read -p "Disable password authentication? (yes/no, default: yes): " DISABLE_PWD_AUTH
    DISABLE_PWD_AUTH=${DISABLE_PWD_AUTH:-yes}
    configure_ssh
    log_success "SSH reconfigured"
}

reconfigure_firewall() {
    read -p "Enable UFW firewall? (yes/no, default: yes): " ENABLE_FIREWALL
    ENABLE_FIREWALL=${ENABLE_FIREWALL:-yes}
    read -p "Select app for firewall rules (docker/coolify/dokploy/n8n/mailu/nextcloud/zentyal/none, default: none): " SELECTED_APP
    SELECTED_APP=${SELECTED_APP:-none}
    configure_firewall
    log_success "Firewall reconfigured"
}

reconfigure_firewall_rules() {
    echo
    log_info "Current UFW Status:"
    ufw status numbered || log_warning "UFW not enabled"
    echo
    
    echo "Firewall management options:"
    echo "  1) Add custom rule"
    echo "  2) Delete rule"
    echo "  3) Reset firewall to defaults"
    echo "  4) Enable/Disable firewall"
    echo "  0) Back"
    echo
    
    read -p "Select option (0-4): " fw_choice
    
    case "$fw_choice" in
        1)
            read -p "Enter rule (e.g., allow 8080/tcp): " rule
            ufw $rule
            log_success "Rule added"
            ;;
        2)
            read -p "Enter rule number to delete: " rule_num
            ufw --force delete $rule_num
            log_success "Rule deleted"
            ;;
        3)
            read -p "Reset firewall? This will remove all rules (yes/no): " confirm
            [[ "$confirm" == "yes" ]] && ufw --force reset
            ;;
        4)
            read -p "Enable or disable firewall? (enable/disable): " fw_action
            ufw --force $fw_action
            ;;
        0) reconfigure_ssh ;;
        *) log_error "Invalid selection"; reconfigure_firewall_rules ;;
    esac
}

show_troubleshoot_menu() {
    show_system_status
    echo
    echo "Options:"
    echo "  1) Update SSH keys from GitHub"
    echo "  2) Adjust SSH config"
    echo "  3) Manage firewall rules"
    echo "  4) Change hostname"
    echo "  5) Change timezone"
    echo "  0) Exit"
    echo
    
    read -p "Select option to adjust (0-5): " ts_choice
    
    case "$ts_choice" in
        1) reconfigure_github_keys ;;
        2) reconfigure_ssh ;;
        3) reconfigure_firewall_rules ;;
        4) reconfigure_hostname ;;
        5) reconfigure_timezone ;;
        0) exit 0 ;;
        *) log_error "Invalid selection"; show_troubleshoot_menu ;;
    esac
}

show_system_status() {
    echo
    log_success "=== System Status Summary ==="
    echo
    
    echo "Hostname:"
    echo "  $(hostname)"
    echo
    
    echo "Timezone:"
    timedatectl | grep "Time zone"
    echo
    
    echo "SSH Status:"
    SSH_PORT=$(grep "^Port" /etc/ssh/sshd_config | awk '{print $2}')
    echo "  Port: ${SSH_PORT:-22}"
    echo "  Root login: $(grep '^PermitRootLogin' /etc/ssh/sshd_config | awk '{print $2}')"
    echo "  Password auth: $(grep '^PasswordAuthentication' /etc/ssh/sshd_config | awk '{print $2}')"
    echo
    
    echo "Users:"
    getent passwd | grep "/home" | cut -d: -f1 | sed 's/^/  /'
    echo
    
    echo "Firewall (UFW):"
    if systemctl is-active --quiet ufw; then
        echo "  Status: Enabled"
        echo "  Rules:"
        ufw status numbered | tail -n +3 | sed 's/^/    /'
    else
        echo "  Status: Disabled"
    fi
    echo
    
    echo "Swap:"
    SWAP_INFO=$(free -h | grep Swap | awk '{print $2}')
    echo "  Total: $SWAP_INFO"
    echo
    
    echo "Services:"
    for svc in docker coolify n8n nextcloud zentyal; do
        if systemctl is-active --quiet $svc 2>/dev/null; then
            echo "  ✓ $svc running"
        fi
    done
    echo
}

reconfigure_hostname() {
    read -p "Enter new hostname: " HOSTNAME
    configure_hostname
    log_success "Hostname updated"
}

reconfigure_timezone() {
    read -p "Enter new timezone: " TIMEZONE
    configure_timezone
    log_success "Timezone updated"
}

collect_user_input() {
    log_info "=== VPS Provisioning Configuration ==="
    echo
    
    read -p "Enter hostname (default: vps-$(hostname -I | awk '{print $1}' | tr '.' '-')): " HOSTNAME
    HOSTNAME=${HOSTNAME:-vps-$(hostname -I | awk '{print $1}' | tr '.' '-')}
    
    read -p "Enter timezone (default: UTC, e.g., Europe/Dublin): " TIMEZONE
    TIMEZONE=${TIMEZONE:-UTC}
    
    read -p "Enter SSH port (default: 22): " SSH_PORT
    SSH_PORT=${SSH_PORT:-22}
    
    read -p "Disable root SSH login? (yes/no, default: yes): " DISABLE_ROOT_LOGIN
    DISABLE_ROOT_LOGIN=${DISABLE_ROOT_LOGIN:-yes}
    
    read -p "Create non-root sudo user? (yes/no, default: yes): " CREATE_USER
    CREATE_USER=${CREATE_USER:-yes}
    
    if [[ "$CREATE_USER" == "yes" ]]; then
        read -p "Enter username (default: ubuntu): " USERNAME
        USERNAME=${USERNAME:-ubuntu}
    fi
    
    read -p "Disable password authentication for SSH? (yes/no, default: yes): " DISABLE_PWD_AUTH
    DISABLE_PWD_AUTH=${DISABLE_PWD_AUTH:-yes}
    
    read -p "Enable UFW firewall? (yes/no, default: yes): " ENABLE_FIREWALL
    ENABLE_FIREWALL=${ENABLE_FIREWALL:-yes}
    
    read -p "Create swap space? (yes/no, default: yes): " CREATE_SWAP
    CREATE_SWAP=${CREATE_SWAP:-yes}
    
    if [[ "$CREATE_SWAP" == "yes" ]]; then
        read -p "Swap size in GB (default: 2): " SWAP_SIZE
        SWAP_SIZE=${SWAP_SIZE:-2}
    fi
    
    read -p "Enable unattended security updates? (yes/no, default: yes): " AUTO_UPDATES
    AUTO_UPDATES=${AUTO_UPDATES:-yes}
}

select_application() {
    echo
    log_info "=== Application Selection ==="
    echo "Select ONE application to install (or 'none' for base system only):"
    echo
    echo "  1) Docker + Portainer"
    echo "  2) Coolify"
    echo "  3) Dokploy"
    echo "  4) n8n"
    echo "  5) Mailu mail server"
    echo "  6) Nextcloud"
    echo "  7) Zentyal"
    echo "  0) None (base system only)"
    echo
    
    read -p "Enter selection (0-7): " app_choice
    
    case "$app_choice" in
        1) SELECTED_APP="docker" ;;
        2) SELECTED_APP="coolify" ;;
        3) SELECTED_APP="dokploy" ;;
        4) SELECTED_APP="n8n" ;;
        5) SELECTED_APP="mailu" ;;
        6) SELECTED_APP="nextcloud" ;;
        7) SELECTED_APP="zentyal" ;;
        0) SELECTED_APP="none" ;;
        *) log_error "Invalid selection"; select_application ;;
    esac
}

show_configuration_summary() {
    echo
    log_warning "⚠️  NOTE: Fresh install does NOT wipe existing data/services"
    log_warning "It applies configuration on top of current system state"
    echo
    log_info "Configuration Summary:"
    echo "  Hostname: $HOSTNAME"
    echo "  Timezone: $TIMEZONE"
    echo "  SSH Port: $SSH_PORT"
    echo "  Block root SSH login: $DISABLE_ROOT_LOGIN"
    echo "  Create user: $CREATE_USER ${USERNAME:-}"
    echo "  Disable password auth: $DISABLE_PWD_AUTH"
    echo "  Enable firewall: $ENABLE_FIREWALL"
    echo "  Create swap: $CREATE_SWAP ${SWAP_SIZE:-}GB"
    echo "  Auto updates: $AUTO_UPDATES"
    echo "  Application: ${SELECTED_APP^^}"
    echo
    
    read -p "Proceed with provisioning? (yes/no): " CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_warning "Provisioning cancelled"
        exit 0
    fi
}

system_update() {
    log_info "Updating system packages..."
    apt-get update
    apt-get upgrade -y
    apt-get dist-upgrade -y
    log_success "System update completed"
}

configure_hostname() {
    log_info "Configuring hostname: $HOSTNAME"
    hostnamectl set-hostname "$HOSTNAME"
    sed -i "s/127.0.1.1.*/127.0.1.1 $HOSTNAME/" /etc/hosts || echo "127.0.1.1 $HOSTNAME" >> /etc/hosts
    log_success "Hostname configured"
}

configure_timezone() {
    log_info "Configuring timezone: $TIMEZONE"
    timedatectl set-timezone "$TIMEZONE"
    log_success "Timezone configured"
}

create_user() {
    [[ "$CREATE_USER" != "yes" ]] && return
    
    log_info "Creating user: $USERNAME"
    
    if id "$USERNAME" &>/dev/null; then
        log_warning "User $USERNAME already exists"
        return
    fi
    
    adduser --disabled-password --gecos "" "$USERNAME"
    usermod -aG sudo "$USERNAME"
    
    log_success "User $USERNAME created with sudo privileges"
    
    read -p "Add SSH keys from GitHub? (yes/no): " USE_GITHUB_KEYS
    if [[ "$USE_GITHUB_KEYS" == "yes" ]]; then
        read -p "Enter GitHub username: " GITHUB_USER
        setup_github_ssh_keys "$USERNAME" "$GITHUB_USER"
    else
        log_info "Set password for $USERNAME:"
        passwd "$USERNAME"
    fi
}

setup_github_ssh_keys() {
    local target_user=$1
    local github_user=$2
    local user_home="/home/$target_user"
    local ssh_dir="$user_home/.ssh"
    local auth_keys="$ssh_dir/authorized_keys"
    
    log_info "Fetching SSH keys from GitHub for user: $github_user"
    
    mkdir -p "$ssh_dir"
    chmod 700 "$ssh_dir"
    
    if curl -fsSL "https://github.com/${github_user}.keys" -o "$auth_keys"; then
        chmod 600 "$auth_keys"
        chown -R "$target_user:$target_user" "$ssh_dir"
        log_success "GitHub SSH keys installed for $target_user"
    else
        log_error "Failed to fetch SSH keys from GitHub - setting password instead"
        log_info "Set password for $target_user:"
        passwd "$target_user"
    fi
}

configure_ssh() {
    log_info "Configuring SSH..."
    
    SSH_CONFIG="/etc/ssh/sshd_config"
    cp "$SSH_CONFIG" "$SSH_CONFIG.backup.$(date +%s)"
    
    sed -i "s/^#Port 22/Port $SSH_PORT/" "$SSH_CONFIG"
    sed -i "s/^Port 22/Port $SSH_PORT/" "$SSH_CONFIG"
    
    if [[ "$DISABLE_ROOT_LOGIN" == "yes" ]]; then
        sed -i 's/^#PermitRootLogin.*/PermitRootLogin prohibit-password/' "$SSH_CONFIG"
        sed -i 's/^PermitRootLogin.*/PermitRootLogin prohibit-password/' "$SSH_CONFIG"
        log_info "Root SSH login restricted to key-based auth only (Coolify compatible)"
    else
        sed -i 's/^#PermitRootLogin.*/PermitRootLogin yes/' "$SSH_CONFIG"
        sed -i 's/^PermitRootLogin.*/PermitRootLogin yes/' "$SSH_CONFIG"
    fi
    
    sed -i 's/^#PubkeyAuthentication.*/PubkeyAuthentication yes/' "$SSH_CONFIG"
    
    if [[ "$DISABLE_PWD_AUTH" == "yes" ]]; then
        sed -i 's/^#PasswordAuthentication.*/PasswordAuthentication no/' "$SSH_CONFIG"
        sed -i 's/^PasswordAuthentication.*/PasswordAuthentication no/' "$SSH_CONFIG"
        log_warning "Password authentication disabled - ensure SSH keys configured"
    fi
    
    sed -i 's/^#PermitEmptyPasswords.*/PermitEmptyPasswords no/' "$SSH_CONFIG"
    sed -i 's/^#X11Forwarding.*/X11Forwarding no/' "$SSH_CONFIG"
    sed -i 's/^#MaxAuthTries.*/MaxAuthTries 3/' "$SSH_CONFIG"
    sed -i 's/^#MaxSessions.*/MaxSessions 5/' "$SSH_CONFIG"
    
    if sshd -t; then
        systemctl restart ssh
        log_success "SSH configured on port $SSH_PORT"
    else
        log_error "SSH configuration error - manual recovery needed"
        log_warning "Backup files are in: /etc/ssh/sshd_config.backup.*"
        log_warning "Restore manually: cp /etc/ssh/sshd_config.backup.TIMESTAMP /etc/ssh/sshd_config"
        exit 1
    fi
}

configure_firewall() {
    [[ "$ENABLE_FIREWALL" != "yes" ]] && return
    
    log_info "Configuring UFW firewall..."
    apt-get install -y ufw
    
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow "$SSH_PORT"/tcp
    
    case "$SELECTED_APP" in
        docker) ufw allow 9000/tcp ;;
        coolify)
            ufw allow 3000/tcp
            ufw allow 3001/tcp
            ;;
        dokploy) ufw allow 3000/tcp ;;
        n8n) ufw allow 5678/tcp ;;
        mailu)
            ufw allow 25/tcp
            ufw allow 465/tcp
            ufw allow 587/tcp
            ufw allow 110/tcp
            ufw allow 995/tcp
            ufw allow 143/tcp
            ufw allow 993/tcp
            ufw allow 80/tcp
            ufw allow 443/tcp
            ;;
        nextcloud)
            ufw allow 80/tcp
            ufw allow 443/tcp
            ;;
        zentyal)
            ufw allow 443/tcp
            ufw allow 80/tcp
            ;;
    esac
    
    ufw --force enable
    log_success "UFW firewall enabled"
}

create_swap() {
    [[ "$CREATE_SWAP" != "yes" ]] && return
    
    log_info "Creating ${SWAP_SIZE}GB swap..."
    SWAP_FILE="/swapfile"
    fallocate -l "${SWAP_SIZE}G" "$SWAP_FILE"
    chmod 600 "$SWAP_FILE"
    mkswap "$SWAP_FILE"
    swapon "$SWAP_FILE"
    echo "$SWAP_FILE none swap sw 0 0" >> /etc/fstab
    log_success "Swap created and configured"
}

configure_auto_updates() {
    [[ "$AUTO_UPDATES" != "yes" ]] && return
    
    log_info "Configuring unattended security updates..."
    apt-get install -y unattended-upgrades apt-listchanges
    dpkg-reconfigure -plow unattended-upgrades
    log_success "Unattended updates configured"
}

harden_system() {
    log_info "Applying system hardening..."
    
    # Check if hardening already applied
    if grep -q "# VPS-TOOLS HARDENING" /etc/sysctl.conf 2>/dev/null; then
        log_warning "System hardening already applied - skipping"
        return
    fi
    
    cat >> /etc/sysctl.conf << 'EOF'

# VPS-TOOLS HARDENING - Added by vps-build.sh
net.ipv4.ip_forward = 0
net.ipv6.conf.all.forwarding = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv4.icmp_echo_ignore_broadcasts = 1
net.ipv4.tcp_syncookies = 1
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1
EOF
    
    sysctl -p > /dev/null
    apt-get autoremove -y
    apt-get autoclean -y
    log_success "System hardening completed"
}

install_docker() {
    [[ "$SELECTED_APP" != "docker" ]] && return
    
    log_info "Installing Docker..."
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    systemctl enable docker
    systemctl start docker
    
    [[ "$CREATE_USER" == "yes" ]] && usermod -aG docker "$USERNAME"
    
    log_info "Installing Portainer..."
    docker volume create portainer_data
    docker run -d -p 9000:9000 -p 8000:8000 --name portainer --restart=always -v /var/run/docker.sock:/var/run/docker.sock -v portainer_data:/data portainer/portainer-ce:latest
    
    log_success "Docker and Portainer installed"
    log_info "Access Portainer at: https://$(hostname -I | awk '{print $1}'):9000"
}

install_coolify() {
    [[ "$SELECTED_APP" != "coolify" ]] && return
    
    log_info "Installing Coolify..."
    apt-get install -y curl
    curl -fsSL https://cdn.coollabs.io/coolify/install.sh | bash
    
    log_success "Coolify installed"
    log_info "Access Coolify at: https://$(hostname -I | awk '{print $1}'):3000"
}

install_dokploy() {
    [[ "$SELECTED_APP" != "dokploy" ]] && return
    
    log_info "Installing Docker (required for Dokploy)..."
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    
    log_info "Installing Dokploy..."
    docker volume create dokploy
    docker run -d -p 3000:3000 --name dokploy --restart=always -v /var/run/docker.sock:/var/run/docker.sock -v dokploy:/home/app/data ajnart/dokploy:latest
    
    log_success "Dokploy installed"
    log_info "Access Dokploy at: https://$(hostname -I | awk '{print $1}'):3000"
}

install_n8n() {
    [[ "$SELECTED_APP" != "n8n" ]] && return
    
    log_info "Installing Docker (required for n8n)..."
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    
    log_info "Installing n8n..."
    docker volume create n8n_data
    docker run -d -p 5678:5678 --name n8n --restart=always -v n8n_data:/home/node/.n8n n8nio/n8n
    
    log_success "n8n installed"
    log_info "Access n8n at: https://$(hostname -I | awk '{print $1}'):5678"
}

install_mailu() {
    [[ "$SELECTED_APP" != "mailu" ]] && return
    
    log_info "Installing Docker (required for Mailu)..."
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    
    log_info "Installing Mailu mail server..."
    
    read -p "Enter mail domain (e.g., mail.example.com): " MAIL_DOMAIN
    read -p "Enter admin email (e.g., admin@example.com): " ADMIN_EMAIL
    
    mkdir -p /opt/mailu
    cd /opt/mailu
    
    curl -o docker-compose.yml https://raw.githubusercontent.com/Mailu/Mailu/master/compose/docker-compose.yml
    
    log_success "Mailu downloaded"
    log_warning "Manual configuration required:"
    log_warning "  1. Edit /opt/mailu/docker-compose.yml"
    log_warning "  2. Set domain: $MAIL_DOMAIN"
    log_warning "  3. Set admin: $ADMIN_EMAIL"
    log_warning "  4. Run: cd /opt/mailu && docker-compose up -d"
    log_warning "See: https://github.com/Mailu/Mailu for full configuration"
}

install_nextcloud() {
    [[ "$SELECTED_APP" != "nextcloud" ]] && return
    
    log_info "Installing Docker (required for Nextcloud)..."
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable docker
    systemctl start docker
    
    log_info "Installing Nextcloud..."
    docker volume create nextcloud_data
    docker run -d -p 80:80 --name nextcloud --restart=always -v nextcloud_data:/var/www/html nextcloud:latest
    
    log_success "Nextcloud installed"
    log_info "Access Nextcloud at: http://$(hostname -I | awk '{print $1}')"
    log_warning "Complete setup wizard on first access"
}

install_zentyal() {
    [[ "$SELECTED_APP" != "zentyal" ]] && return
    
    log_info "Installing Zentyal..."
    
    echo "deb http://packages.zentyal.org/zentyal-7.0 focal main" > /etc/apt/sources.list.d/zentyal.list
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 3B37B77C6BE06718
    
    apt-get update
    apt-get install -y zentyal
    
    systemctl enable zentyal
    systemctl start zentyal
    
    log_success "Zentyal installed"
    log_info "Access Zentyal at: https://$(hostname -I | awk '{print $1}'):443"
}

show_summary() {
    echo
    log_success "=== VPS Provisioning Complete ==="
    echo
    echo "Core Configuration:"
    echo "  Hostname: $(hostname)"
    echo "  Timezone: $(timedatectl | grep 'Time zone')"
    echo "  SSH Port: $SSH_PORT"
    [[ "$CREATE_USER" == "yes" ]] && echo "  User: $USERNAME"
    [[ "$ENABLE_FIREWALL" == "yes" ]] && echo "  Firewall: UFW enabled"
    [[ "$CREATE_SWAP" == "yes" ]] && echo "  Swap: ${SWAP_SIZE}GB"
    
    [[ "$SELECTED_APP" != "none" ]] && {
        echo
        echo "Installed Application: ${SELECTED_APP^^}"
    }
    
    echo
    log_warning "Next Steps:"
    echo "  • Test SSH: ssh -p $SSH_PORT user@$(hostname -I | awk '{print $1}')"
    echo "  • Review firewall: sudo ufw status numbered"
    echo "  • Review logs: sudo journalctl -f"
}

main() {
    check_root
    log_info "Ubuntu 24.04 VPS Provisioning Script v${SCRIPT_VERSION}"
    echo
    
    select_mode
    
    if [[ "$MODE" == "fresh" ]]; then
        collect_user_input
        select_application
        show_configuration_summary
        
        system_update
        configure_hostname
        configure_timezone
        create_user
        configure_ssh
        configure_firewall
        create_swap
        configure_auto_updates
        harden_system
        
        install_docker
        install_coolify
        install_dokploy
        install_n8n
        install_mailu
        install_nextcloud
        install_zentyal
        
        show_summary
    fi
    
    if [[ "$MODE" == "troubleshoot" ]]; then
        show_troubleshoot_menu
    fi
}

main "$@"