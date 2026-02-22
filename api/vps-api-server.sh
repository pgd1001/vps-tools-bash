#!/bin/bash
set -euo pipefail

# VPS Tools - REST API Server
# Lightweight HTTP API for remote management
# Usage: vps-tools api start|stop|status

readonly TOOLS_DIR="${TOOLS_DIR:-/opt/vps-tools}"
readonly CONFIG_DIR="${CONFIG_DIR:-/etc/vps-tools}"
readonly CONFIG_FILE="$CONFIG_DIR/config.conf"
readonly PID_FILE="/var/run/vps-api.pid"
readonly LOG_FILE="/var/log/vps-tools/api.log"

source "${TOOLS_DIR}/lib/output.sh"
source "${TOOLS_DIR}/lib/validate.sh"

# Load configuration
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        source "$CONFIG_FILE"
    fi
    API_PORT="${API_PORT:-8080}"
    API_BIND="${API_BIND:-127.0.0.1}"
    API_TOKEN="${API_TOKEN:-}"
}

# Check if API server is running
is_running() {
    [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

# Verify token
check_auth() {
    local auth_header="$1"
    
    # No token configured = no auth required
    [[ -z "$API_TOKEN" ]] && return 0
    
    # Check Bearer token
    if [[ "$auth_header" == "Bearer $API_TOKEN" ]]; then
        return 0
    fi
    
    return 1
}

# Parse HTTP request
parse_request() {
    local line
    local method=""
    local path=""
    local auth=""
    local content_length=0
    local body=""
    
    # Read request line
    read -r line
    method=$(echo "$line" | awk '{print $1}')
    path=$(echo "$line" | awk '{print $2}')
    
    # Read headers
    while read -r line; do
        line="${line%%$'\r'}"
        [[ -z "$line" ]] && break
        
        case "$line" in
            Authorization:*) auth="${line#*: }" ;;
            Content-Length:*) content_length="${line#*: }" ;;
        esac
    done
    
    # Read body if present
    if [[ $content_length -gt 0 ]]; then
        read -r -n "$content_length" body
    fi
    
    echo "$method|$path|$auth|$body"
}

# Send HTTP response
send_response() {
    local status="$1"
    local content_type="${2:-application/json}"
    local body="$3"
    
    echo "HTTP/1.1 $status"
    echo "Content-Type: $content_type"
    echo "Content-Length: ${#body}"
    echo "Connection: close"
    echo ""
    echo "$body"
}

# Handle API request
handle_request() {
    local request
    request=$(parse_request)
    
    local method path auth body
    IFS='|' read -r method path auth body <<< "$request"
    
    # Log request
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $method $path" >> "$LOG_FILE"
    
    # Check authentication
    if ! check_auth "$auth"; then
        send_response "401 Unauthorized" "application/json" '{"error":"Unauthorized"}'
        return
    fi
    
    # Route request
    case "$method $path" in
        "GET /api/health"|"GET /health")
            local output
            output=$(bash "$TOOLS_DIR/monitoring/vps-health-monitor.sh" --output=json 2>/dev/null || echo '{"status":"error"}')
            send_response "200 OK" "application/json" "$output"
            ;;
        
        "GET /api/plugins"|"GET /plugins")
            local plugins_json="["
            local first=true
            while IFS=: read -r cmd path desc category enabled; do
                [[ -z "$cmd" || "$cmd" =~ ^# ]] && continue
                $first || plugins_json+=","
                first=false
                plugins_json+="{\"command\":\"$(_json_escape "$cmd")\",\"path\":\"$(_json_escape "$path")\",\"description\":\"$(_json_escape "$desc")\",\"category\":\"$(_json_escape "$category")\",\"enabled\":$enabled}"
            done < "$CONFIG_DIR/plugins.conf"
            plugins_json+="]"
            send_response "200 OK" "application/json" "$plugins_json"
            ;;
        
        "GET /api/config"|"GET /config")
            local config_json="{"
            local first=true
            while IFS= read -r line; do
                [[ -z "$line" || "$line" =~ ^# ]] && continue
                local key="${line%%=*}"
                local value="${line#*=}"
                $first || config_json+=","
                first=false
                config_json+="\"$(_json_escape "$key")\":\"$(_json_escape "$value")\""
            done < "$CONFIG_FILE"
            config_json+="}"
            send_response "200 OK" "application/json" "$config_json"
            ;;
        
        "POST /api/run/"*)
            local cmd="${path#/api/run/}"
            if [[ -z "$cmd" ]]; then
                send_response "400 Bad Request" "application/json" '{"error":"Missing command"}'
            elif ! validate_command_name "$cmd"; then
                send_response "400 Bad Request" "application/json" '{"error":"Invalid command name"}'
            else
                # Look up command in plugin registry
                local script_path=""
                while IFS=: read -r pcmd ppath pdesc pcategory penabled; do
                    [[ -z "$pcmd" || "$pcmd" =~ ^# ]] && continue
                    if [[ "$pcmd" == "$cmd" && "$penabled" == "true" ]]; then
                        script_path="$ppath"
                        break
                    fi
                done < "$CONFIG_DIR/plugins.conf"

                if [[ -z "$script_path" ]]; then
                    send_response "404 Not Found" "application/json" '{"error":"Command not found or disabled"}'
                elif [[ ! -f "$TOOLS_DIR/$script_path" ]]; then
                    send_response "500 Internal Server Error" "application/json" '{"error":"Script not found"}'
                else
                    local output
                    output=$(bash "$TOOLS_DIR/$script_path" --output=json 2>&1 || echo '{"error":"Command failed"}')
                    send_response "200 OK" "application/json" "$output"
                fi
            fi
            ;;
        
        "GET /"|"GET /api")
            local info='{
                "name": "VPS Tools API",
                "version": "1.2.0",
                "endpoints": [
                    {"method": "GET", "path": "/api/health", "description": "System health check"},
                    {"method": "GET", "path": "/api/plugins", "description": "List plugins"},
                    {"method": "GET", "path": "/api/config", "description": "Get configuration"},
                    {"method": "POST", "path": "/api/run/{cmd}", "description": "Run command"}
                ]
            }'
            send_response "200 OK" "application/json" "$info"
            ;;
        
        *)
            send_response "404 Not Found" "application/json" '{"error":"Not found"}'
            ;;
    esac
}

# Start server
start_server() {
    load_config
    
    if is_running; then
        log_warning "API server already running (PID: $(cat "$PID_FILE"))"
        return 1
    fi
    
    # Check for required tools
    if ! command -v socat &>/dev/null && ! command -v nc &>/dev/null; then
        log_error "Neither socat nor netcat found. Install with: apt install socat"
        return 1
    fi
    
    log_info "Starting VPS Tools API server..."
    log_info "Bind: $API_BIND:$API_PORT"
    
    if [[ -z "$API_TOKEN" ]]; then
        log_warning "No API_TOKEN set - API is unauthenticated!"
        log_warning "Set token: vps-tools config set API_TOKEN \$(openssl rand -hex 32)"
    fi
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Start server in background
    (
        while true; do
            if command -v socat &>/dev/null; then
                socat TCP-LISTEN:$API_PORT,bind=$API_BIND,fork,reuseaddr EXEC:"$0 --handle-request",pty,stderr 2>/dev/null
            else
                # Fallback to netcat (less reliable)
                while true; do
                    nc -l -p $API_PORT -c "$0 --handle-request" 2>/dev/null || sleep 1
                done
            fi
        done
    ) &
    
    echo $! > "$PID_FILE"
    log_success "API server started (PID: $!)"
    log_info "Test: curl http://$API_BIND:$API_PORT/api"
}

# Stop server
stop_server() {
    if ! is_running; then
        log_warning "API server not running"
        return 1
    fi
    
    local pid=$(cat "$PID_FILE")
    kill "$pid" 2>/dev/null
    rm -f "$PID_FILE"
    log_success "API server stopped"
}

# Show status
show_status() {
    load_config
    
    if is_running; then
        local pid=$(cat "$PID_FILE")
        log_success "API server running (PID: $pid)"
        echo "  Bind: $API_BIND:$API_PORT"
        echo "  Auth: ${API_TOKEN:+enabled}${API_TOKEN:-disabled}"
        echo "  Log:  $LOG_FILE"
    else
        log_warning "API server not running"
    fi
}

# Main
main() {
    local action="${1:-help}"
    
    case "$action" in
        start)
            start_server
            ;;
        stop)
            stop_server
            ;;
        status)
            show_status
            ;;
        restart)
            stop_server 2>/dev/null || true
            sleep 1
            start_server
            ;;
        --handle-request)
            load_config
            handle_request
            ;;
        *)
            echo "VPS Tools API Server"
            echo
            echo "Usage: vps-tools api <command>"
            echo
            echo "Commands:"
            echo "  start     Start API server"
            echo "  stop      Stop API server"
            echo "  status    Show server status"
            echo "  restart   Restart server"
            echo
            echo "Configuration:"
            echo "  vps-tools config set API_PORT 8080"
            echo "  vps-tools config set API_BIND 127.0.0.1"
            echo "  vps-tools config set API_TOKEN <token>"
            ;;
    esac
}

main "$@"
