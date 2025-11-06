#!/bin/bash

# Proxy Test API - Remote Server Deployment Script
# Usage: ./deploy/deploy.sh [server-ip] [username]

set -e

# Default values
SERVER_IP="${1:-100.121.222.76}"
USERNAME="${2:-root}"
PROJECT_NAME="proxy-test-api"
DEPLOY_DIR="/opt/$PROJECT_NAME"
SERVICE_NAME="$PROJECT_NAME"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if required tools are installed
    for tool in ssh scp rsync; do
        if ! command -v $tool &> /dev/null; then
            log_error "$tool is not installed. Please install it first."
            exit 1
        fi
    done
    
    # Check SSH connection
    log_info "Testing SSH connection to $SERVER_IP..."
    if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$USERNAME@$SERVER_IP" "echo 'SSH connection successful'" 2>/dev/null; then
        log_error "Cannot connect to $SERVER_IP. Please check:"
        echo "  1. Server IP address: $SERVER_IP"
        echo "  2. Username: $USERNAME"
        echo "  3. SSH key or password authentication"
        echo "  4. Firewall rules"
        exit 1
    fi
    
    log_success "SSH connection established"
}

# Install dependencies on server
install_dependencies() {
    log_info "Installing dependencies on server..."
    
    ssh "$USERNAME@$SERVER_IP" << 'EOF'
        # Update system
        if command -v apt-get &> /dev/null; then
            sudo apt-get update
            sudo apt-get install -y curl wget git build-essential
        elif command -v yum &> /dev/null; then
            sudo yum update -y
            sudo yum install -y curl wget git gcc gcc-c++ make
        elif command -v dnf &> /dev/null; then
            sudo dnf update -y
            sudo dnf install -y curl wget git gcc gcc-c++ make
        else
            echo "Unsupported package manager"
            exit 1
        fi
        
        # Install Go if not present
        if ! command -v go &> /dev/null; then
            echo "Installing Go..."
            GO_VERSION="1.21.5"
            wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
            sudo tar -C /usr/local -xzf /tmp/go.tar.gz
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            export PATH=$PATH:/usr/local/go/bin
            rm /tmp/go.tar.gz
        fi
        
        # Install additional tools
        if command -v systemctl &> /dev/null; then
            echo "systemd available"
        else
            echo "systemd not available, will use supervisor or manual startup"
        fi
EOF
    
    log_success "Dependencies installed"
}

# Create deployment directory and upload files
deploy_files() {
    log_info "Creating deployment directory and uploading files..."
    
    # Create deployment directory on server
    ssh "$USERNAME@$SERVER_IP" "sudo mkdir -p $DEPLOY_DIR && sudo chown $USERNAME:$USERNAME $DEPLOY_DIR"
    
    # Upload project files
    log_info "Uploading project files..."
    rsync -avz --exclude='.git' --exclude='node_modules' --exclude='*.log' . "$USERNAME@$SERVER_IP:$DEPLOY_DIR/"
    
    # Set proper permissions
    ssh "$USERNAME@$SERVER_IP" "chmod +x $DEPLOY_DIR/cmd/api/api_server $DEPLOY_DIR/cmd/api/api_client"
    
    log_success "Files uploaded successfully"
}

# Create systemd service
create_systemd_service() {
    log_info "Creating systemd service..."
    
    ssh "$USERNAME@$SERVER_IP" << EOF
        # Create systemd service file
        sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null << SERVICEEOF
[Unit]
Description=Proxy Test API Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=$USERNAME
Group=$USERNAME
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/cmd/api/api_server -port 8080 -data-dir /var/lib/$PROJECT_NAME
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$PROJECT_NAME

# Environment variables
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin
Environment=GOPATH=/opt/go

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/$PROJECT_NAME /tmp

[Install]
WantedBy=multi-user.target
SERVICEEOF

        # Reload systemd
        sudo systemctl daemon-reload
        
        # Enable service
        sudo systemctl enable $SERVICE_NAME
        
        echo "Systemd service created and enabled"
EOF
    
    log_success "Systemd service created"
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    
    ssh "$USERNAME@$SERVER_IP" << 'EOF'
        # Check if ufw is installed and enabled
        if command -v ufw &> /dev/null && sudo ufw status | grep -q "Status: active"; then
            echo "Configuring UFW firewall..."
            sudo ufw allow 8080/tcp comment "Proxy Test API"
            sudo ufw reload
        elif command -v firewall-cmd &> /dev/null && sudo systemctl is-active firewalld &> /dev/null; then
            echo "Configuring firewalld..."
            sudo firewall-cmd --permanent --add-port=8080/tcp
            sudo firewall-cmd --reload
        else
            echo "No active firewall detected or manual configuration required"
            echo "Please manually allow port 8080/tcp in your firewall"
        fi
EOF
    
    log_success "Firewall configured"
}

# Start service
start_service() {
    log_info "Starting service..."
    
    ssh "$USERNAME@$SERVER_IP" "sudo systemctl start $SERVICE_NAME"
    
    # Wait a moment for service to start
    sleep 5
    
    # Check service status
    if ssh "$USERNAME@$SERVER_IP" "sudo systemctl is-active $SERVICE_NAME" | grep -q "active"; then
        log_success "Service started successfully"
    else
        log_error "Service failed to start"
        ssh "$USERNAME@$SERVER_IP" "sudo systemctl status $SERVICE_NAME --no-pager"
        return 1
    fi
}

# Test deployment
test_deployment() {
    log_info "Testing deployment..."
    
    # Wait for service to be ready
    sleep 10
    
    # Test health endpoint
    if curl -f "http://$SERVER_IP:8080/health" >/dev/null 2>&1; then
        log_success "API is responding!"
        
        # Get additional info
        ssh "$USERNAME@$SERVER_IP" "sudo systemctl status $SERVICE_NAME --no-pager -l"
        
        # Show logs
        log_info "Recent logs:"
        ssh "$USERNAME@$SERVER_IP" "sudo journalctl -u $SERVICE_NAME -n 10 --no-pager"
        
    else
        log_error "API is not responding. Check logs:"
        ssh "$USERNAME@$SERVER_IP" "sudo journalctl -u $SERVICE_NAME -n 20 --no-pager"
        return 1
    fi
}

# Show deployment info
show_deployment_info() {
    echo ""
    echo "🎉 Deployment completed successfully!"
    echo ""
    echo "📊 Deployment Information:"
    echo "   Server: $SERVER_IP"
    echo "   Service: $SERVICE_NAME"
    echo "   Port: 8080"
    echo "   Directory: $DEPLOY_DIR"
    echo ""
    echo "🌐 Access URLs:"
    echo "   Health Check: http://$SERVER_IP:8080/health"
    echo "   API Status: http://$SERVER_IP:8080/api/v1/status"
    echo ""
    echo "🔧 Management Commands:"
    echo "   Start: sudo systemctl start $SERVICE_NAME"
    echo "   Stop: sudo systemctl stop $SERVICE_NAME"
    echo "   Restart: sudo systemctl restart $SERVICE_NAME"
    echo "   Status: sudo systemctl status $SERVICE_NAME"
    echo "   Logs: sudo journalctl -u $SERVICE_NAME -f"
    echo ""
    echo "📱 Local Testing:"
    echo "   ./cmd/api/api_client --host $SERVER_IP --port 8080 --action demo"
    echo ""
}

# Main execution
main() {
    echo ""
    echo "🚀 Proxy Test API - Remote Server Deployment"
    echo "============================================"
    echo "Server: $SERVER_IP"
    echo "Username: $USERNAME"
    echo ""
    
    check_prerequisites
    install_dependencies
    deploy_files
    create_systemd_service
    configure_firewall
    start_service
    test_deployment
    show_deployment_info
}

# Run main function
main "$@"