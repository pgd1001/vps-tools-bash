#!/bin/bash
set -euo pipefail

# VPS Tools Installation Script
# Makes all scripts available system-wide with plugin support
# Usage: bash install.sh [--uninstall]

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly INSTALL_DIR="/opt/vps-tools"
readonly BIN_DIR="/usr/local/bin"
readonly CONFIG_DIR="/etc/vps-tools"
readonly LOG_DIR="/var/log/vps-tools"
readonly PLUGINS_FILE="$CONFIG_DIR/plugins.conf"

source "${SCRIPT_DIR}/lib/output.sh"

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run with sudo"
        exit 1
    fi
}

install() {
    log_info "VPS Tools Installation"
    echo
    
    check_root
    
    # Create directories
    log_info "Creating directories..."
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/custom"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    # Copy all files
    log_info "Copying files to $INSTALL_DIR..."
    cp -r "$SCRIPT_DIR"/* "$INSTALL_DIR/"
    chmod -R 755 "$INSTALL_DIR"
    find "$INSTALL_DIR" -name "*.sh" -exec chmod 755 {} \;
    
    # Install plugins.conf if not exists (preserve user customizations)
    if [[ ! -f "$PLUGINS_FILE" ]]; then
        log_info "Creating plugin registry..."
        cp "$INSTALL_DIR/plugins.conf.default" "$PLUGINS_FILE"
    else
        log_warning "Plugin registry exists - keeping user customizations"
    fi
    
    # Install config.conf if not exists (preserve user customizations)
    if [[ ! -f "$CONFIG_DIR/config.conf" ]]; then
        log_info "Creating configuration file..."
        cp "$INSTALL_DIR/config.conf.default" "$CONFIG_DIR/config.conf"
    else
        log_warning "Config file exists - keeping user customizations"
    fi
    
    # Create main dispatcher script with plugin support
    log_info "Creating dispatcher script..."
    cat > "$BIN_DIR/vps-tools" << 'DISPATCHER'
#!/bin/bash
set -euo pipefail

readonly TOOLS_DIR="/opt/vps-tools"
readonly CONFIG_DIR="/etc/vps-tools"
readonly PLUGINS_FILE="$CONFIG_DIR/plugins.conf"
readonly CONFIG_FILE="$CONFIG_DIR/config.conf"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Inline validation (dispatcher can't source lib/)
_validate_command_name() {
    [[ "$1" =~ ^[a-zA-Z0-9_-]+$ ]]
}
_validate_path() {
    [[ "$1" != *".."* ]] && [[ "$1" =~ ^[a-zA-Z0-9_./-]+$ ]]
}
_sanitize_for_sed() {
    printf '%s' "$1" | sed 's/[\\\/&.*\[\]^$]/\\&/g'
}

# Load plugins from registry
declare -A PLUGIN_COMMANDS
declare -A PLUGIN_DESCRIPTIONS
declare -A PLUGIN_CATEGORIES

load_plugins() {
    if [[ ! -f "$PLUGINS_FILE" ]]; then
        echo -e "${RED}[ERROR]${NC} Plugin registry not found: $PLUGINS_FILE"
        exit 1
    fi
    
    while IFS=: read -r cmd path desc category enabled; do
        # Skip comments and empty lines
        [[ -z "$cmd" || "$cmd" =~ ^# ]] && continue
        
        if [[ "$enabled" == "true" ]]; then
            PLUGIN_COMMANDS[$cmd]="$path"
            PLUGIN_DESCRIPTIONS[$cmd]="$desc"
            PLUGIN_CATEGORIES[$cmd]="$category"
        fi
    done < "$PLUGINS_FILE"
}

# Get plugins by category
get_category_plugins() {
    local category="$1"
    local result=""
    for cmd in "${!PLUGIN_CATEGORIES[@]}"; do
        if [[ "${PLUGIN_CATEGORIES[$cmd]}" == "$category" ]]; then
            result+="$cmd "
        fi
    done
    echo "$result"
}

show_menu() {
    clear
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║              VPS Tools Management Suite                        ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo
    
    local num=1
    declare -gA MENU_COMMANDS
    
    # Provisioning
    echo "PROVISIONING"
    for cmd in $(get_category_plugins "provisioning"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    echo
    
    # Monitoring
    echo "MONITORING & HEALTH"
    for cmd in $(get_category_plugins "monitoring"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    echo
    
    # Security
    echo "SECURITY"
    for cmd in $(get_category_plugins "security"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    echo
    
    # Docker
    echo "DOCKER MANAGEMENT"
    for cmd in $(get_category_plugins "docker"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    echo
    
    # Maintenance
    echo "MAINTENANCE"
    for cmd in $(get_category_plugins "maintenance"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    echo
    
    # Orchestration
    echo "ORCHESTRATION & REPORTING"
    for cmd in $(get_category_plugins "orchestration"); do
        printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
        MENU_COMMANDS[$num]="$cmd"
        ((num++))
    done
    
    # Custom plugins
    local custom_plugins=$(get_category_plugins "custom")
    if [[ -n "$custom_plugins" ]]; then
        echo
        echo "CUSTOM SCRIPTS"
        for cmd in $custom_plugins; do
            printf "  %2d) %s\n" $num "${PLUGIN_DESCRIPTIONS[$cmd]}"
            MENU_COMMANDS[$num]="$cmd"
            ((num++))
        done
    fi
    
    echo
    echo "UTILITIES"
    printf "  %2d) View Documentation\n" $num
    MENU_COMMANDS[$num]="docs"
    ((num++))
    printf "  %2d) Install Cron Jobs\n" $num
    MENU_COMMANDS[$num]="cron"
    ((num++))
    printf "  %2d) Plugin Manager\n" $num
    MENU_COMMANDS[$num]="plugin"
    ((num++))
    echo
    echo "   0) Exit"
    echo
}

run_plugin() {
    local cmd="$1"
    shift
    
    local path="${PLUGIN_COMMANDS[$cmd]:-}"
    
    if [[ -z "$path" ]]; then
        echo -e "${RED}[ERROR]${NC} Unknown or disabled command: $cmd"
        return 1
    fi
    
    local full_path="$TOOLS_DIR/$path"
    
    if [[ ! -f "$full_path" ]]; then
        echo -e "${RED}[ERROR]${NC} Script not found: $full_path"
        return 1
    fi
    
    sudo bash "$full_path" "$@"
}

plugin_manager() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo -e "${BLUE}=== Enabled Plugins ===${NC}"
            printf "%-15s %-40s %s\n" "COMMAND" "DESCRIPTION" "CATEGORY"
            echo "─────────────────────────────────────────────────────────────────────"
            for cmd in "${!PLUGIN_COMMANDS[@]}"; do
                printf "%-15s %-40s %s\n" "$cmd" "${PLUGIN_DESCRIPTIONS[$cmd]}" "${PLUGIN_CATEGORIES[$cmd]}"
            done | sort
            echo
            echo "Total: ${#PLUGIN_COMMANDS[@]} enabled plugins"
            echo
            echo "To disable: Edit $PLUGINS_FILE and set enabled to 'false'"
            ;;
        
        add)
            local cmd="${1:-}"
            local path="${2:-}"
            local desc="${3:-Custom script}"

            if [[ -z "$cmd" || -z "$path" ]]; then
                echo "Usage: vps-tools plugin add <command> <path> [description]"
                echo "Example: vps-tools plugin add my-backup custom/my-backup.sh 'My backup script'"
                return 1
            fi

            if ! _validate_command_name "$cmd"; then
                echo -e "${RED}[✗]${NC} Invalid command name: $cmd (alphanumeric, hyphens, underscores only)"
                return 1
            fi

            if ! _validate_path "$path"; then
                echo -e "${RED}[✗]${NC} Invalid path: $path"
                return 1
            fi

            # Strip colons from description to prevent registry corruption
            desc="${desc//:/}"

            if ! grep -q "^$cmd:" "$PLUGINS_FILE"; then
                echo "$cmd:$path:$desc:custom:true" >> "$PLUGINS_FILE"
                echo -e "${GREEN}[✓]${NC} Added plugin: $cmd -> $path"
            else
                echo -e "${YELLOW}[⚠]${NC} Plugin '$cmd' already exists"
            fi
            ;;
        
        enable)
            local cmd="${1:-}"
            if [[ -z "$cmd" ]]; then
                echo "Usage: vps-tools plugin enable <command>"
                return 1
            fi
            if ! _validate_command_name "$cmd"; then
                echo -e "${RED}[✗]${NC} Invalid command name: $cmd"
                return 1
            fi
            local safe_cmd
            safe_cmd=$(_sanitize_for_sed "$cmd")
            if grep -q "^$cmd:" "$PLUGINS_FILE"; then
                sed -i "s/^${safe_cmd}:\(.*\):false$/${safe_cmd}:\1:true/" "$PLUGINS_FILE"
                echo -e "${GREEN}[✓]${NC} Enabled: $cmd"
            else
                echo -e "${RED}[✗]${NC} Plugin not found: $cmd"
            fi
            ;;
        
        disable)
            local cmd="${1:-}"
            if [[ -z "$cmd" ]]; then
                echo "Usage: vps-tools plugin disable <command>"
                return 1
            fi
            if ! _validate_command_name "$cmd"; then
                echo -e "${RED}[✗]${NC} Invalid command name: $cmd"
                return 1
            fi
            local safe_cmd
            safe_cmd=$(_sanitize_for_sed "$cmd")
            if grep -q "^$cmd:" "$PLUGINS_FILE"; then
                sed -i "s/^${safe_cmd}:\(.*\):true$/${safe_cmd}:\1:false/" "$PLUGINS_FILE"
                echo -e "${YELLOW}[⚠]${NC} Disabled: $cmd"
            else
                echo -e "${RED}[✗]${NC} Plugin not found: $cmd"
            fi
            ;;
        
        *)
            echo "Plugin Manager Commands:"
            echo "  vps-tools plugin list              List enabled plugins"
            echo "  vps-tools plugin add CMD PATH DESC Add custom plugin"
            echo "  vps-tools plugin enable CMD        Enable a plugin"
            echo "  vps-tools plugin disable CMD       Disable a plugin"
            ;;
    esac
}

config_manager() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo -e "${BLUE}=== VPS Tools Configuration ===${NC}"
            if [[ -f "$CONFIG_FILE" ]]; then
                grep -v '^#' "$CONFIG_FILE" | grep -v '^$' | while IFS= read -r line; do
                    local key="${line%%=*}"
                    local value="${line#*=}"
                    printf "  %-25s = %s\n" "$key" "$value"
                done
            else
                echo "Config file not found: $CONFIG_FILE"
            fi
            ;;
        
        get)
            local key="${1:-}"
            if [[ -z "$key" ]]; then
                echo "Usage: vps-tools config get <KEY>"
                return 1
            fi
            if [[ -f "$CONFIG_FILE" ]]; then
                grep "^$key=" "$CONFIG_FILE" | cut -d= -f2-
            fi
            ;;
        
        set)
            local key="${1:-}"
            local value="${2:-}"
            if [[ -z "$key" ]]; then
                echo "Usage: vps-tools config set <KEY> <VALUE>"
                return 1
            fi
            if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
                echo -e "${RED}[✗]${NC} Invalid config key: $key"
                return 1
            fi
            if [[ -f "$CONFIG_FILE" ]]; then
                local safe_key safe_value
                safe_key=$(_sanitize_for_sed "$key")
                safe_value=$(_sanitize_for_sed "$value")
                if grep -q "^${key}=" "$CONFIG_FILE"; then
                    sed -i "s|^${safe_key}=.*|${safe_key}=${safe_value}|" "$CONFIG_FILE"
                    echo -e "${GREEN}[✓]${NC} Updated: $key=$value"
                else
                    echo "$key=$value" >> "$CONFIG_FILE"
                    echo -e "${GREEN}[✓]${NC} Added: $key=$value"
                fi
            else
                echo -e "${RED}[✗]${NC} Config file not found"
                return 1
            fi
            ;;
        
        *)
            echo "Config Manager Commands:"
            echo "  vps-tools config list          List all settings"
            echo "  vps-tools config get KEY       Get setting value"
            echo "  vps-tools config set KEY VAL   Set setting value"
            ;;
    esac
}

show_help() {
    echo "VPS Tools - Plugin-based VPS Management"
    echo
    echo "Usage: vps-tools [command] [options]"
    echo
    echo "Commands:"
    for cmd in "${!PLUGIN_COMMANDS[@]}"; do
        printf "  %-18s %s\n" "$cmd" "${PLUGIN_DESCRIPTIONS[$cmd]}"
    done | sort
    echo
    echo "Plugin Management:"
    echo "  plugin list        List enabled plugins"
    echo "  plugin add         Add custom plugin"
    echo "  plugin enable      Enable a plugin"
    echo "  plugin disable     Disable a plugin"
    echo
    echo "Configuration:"
    echo "  config list        List all settings"
    echo "  config get KEY     Get setting value"
    echo "  config set KEY VAL Set setting value"
    echo
    echo "API Server:"
    echo "  api start          Start REST API server"
    echo "  api stop           Stop REST API server"
    echo "  api status         Show API server status"
    echo
    echo "Other:"
    echo "  help               Show this help"
    echo
    echo "Plugin registry: $PLUGINS_FILE"
    echo "Custom scripts: $TOOLS_DIR/custom/"
    echo "Config file: $CONFIG_FILE"
}

interactive_menu() {
    while true; do
        show_menu
        
        read -p "Select option: " choice
        
        if [[ "$choice" == "0" ]]; then
            exit 0
        fi
        
        local cmd="${MENU_COMMANDS[$choice]:-}"
        
        if [[ -z "$cmd" ]]; then
            echo -e "${RED}Invalid option${NC}"
            read -p "Press enter to continue..."
            continue
        fi
        
        case "$cmd" in
            docs)
                less "$TOOLS_DIR/USAGE.md"
                ;;
            cron)
                if [[ -f "$TOOLS_DIR/vps-tools-cron.conf" ]]; then
                    sudo cp "$TOOLS_DIR/vps-tools-cron.conf" /etc/cron.d/vps-tools
                    sudo chmod 644 /etc/cron.d/vps-tools
                    echo -e "${GREEN}[✓]${NC} Cron jobs installed"
                    read -p "Press enter to continue..."
                else
                    echo -e "${RED}[ERROR]${NC} Cron config not found"
                    read -p "Press enter to continue..."
                fi
                ;;
            plugin)
                plugin_manager list
                read -p "Press enter to continue..."
                ;;
            *)
                run_plugin "$cmd"
                echo
                read -p "Press enter to continue..."
                ;;
        esac
    done
}

# Main entry point
load_plugins

if [[ $# -gt 0 ]]; then
    case "$1" in
        help|--help|-h)
            show_help
            ;;
        plugin)
            shift
            plugin_manager "$@"
            ;;
        config)
            shift
            config_manager "$@"
            ;;
        api)
            shift
            sudo bash "$TOOLS_DIR/api/vps-api-server.sh" "$@"
            ;;
        *)
            run_plugin "$@"
            ;;
    esac
else
    interactive_menu
fi
DISPATCHER
    
    chmod 755 "$BIN_DIR/vps-tools"
    log_success "Dispatcher created with plugin support"
    
    # Create convenience symlinks
    log_info "Creating convenience shortcuts..."
    ln -sf "$INSTALL_DIR/vps-build.sh" "$BIN_DIR/vps-build" 2>/dev/null || true
    ln -sf "$BIN_DIR/vps-tools" "$BIN_DIR/vps-health" 2>/dev/null || true
    
    # Set proper permissions
    chown -R root:root "$INSTALL_DIR"
    chmod 755 "$LOG_DIR"
    
    echo
    log_success "Installation Complete!"
    echo
    echo "Available commands:"
    echo "  • vps-tools              - Interactive menu"
    echo "  • vps-tools help         - List all commands"
    echo "  • vps-tools plugin list  - List plugins"
    echo "  • vps-build              - Direct VPS setup"
    echo
    echo "Plugin System:"
    echo "  • Registry: $PLUGINS_FILE"
    echo "  • Custom scripts: $INSTALL_DIR/custom/"
    echo
    echo "Documentation:"
    echo "  • $INSTALL_DIR/README.md"
    echo "  • $INSTALL_DIR/USAGE.md"
    echo
}

uninstall() {
    log_warning "Uninstalling VPS Tools..."
    
    check_root
    
    read -p "Remove all VPS Tools? This cannot be undone. (yes/no): " confirm
    
    if [[ "$confirm" != "yes" ]]; then
        log_warning "Cancelled"
        return
    fi
    
    # Remove cron jobs
    [[ -f /etc/cron.d/vps-tools ]] && rm -f /etc/cron.d/vps-tools
    
    # Remove symlinks
    rm -f "$BIN_DIR/vps-tools"
    rm -f "$BIN_DIR/vps-build"
    rm -f "$BIN_DIR/vps-health"
    
    # Remove installation
    rm -rf "$INSTALL_DIR"
    
    log_success "Uninstalled"
    echo "Config and logs remain in:"
    echo "  • $CONFIG_DIR/"
    echo "  • $LOG_DIR/"
    echo
}

main_installer() {
    if [[ "${1:-}" == "--uninstall" ]]; then
        uninstall
    else
        install
    fi
}

main_installer "$@"