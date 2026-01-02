#!/bin/bash
# Sumariza AI - Server Setup Script
# Run this script on a fresh Ubuntu server to set up everything

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "Please run as root (sudo ./setup.sh)"
    exit 1
fi

# Configuration
DEPLOY_PATH="${DEPLOY_PATH:-/var/www/sumariza-ai}"
DOMAIN="${DOMAIN:-}"

log_info "Starting Sumariza AI server setup..."

# Update system
log_info "Updating system packages..."
apt-get update
apt-get upgrade -y

# Install dependencies
log_info "Installing dependencies..."
apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# Install Chromium (try different package names for different Ubuntu versions)
log_info "Installing Chromium browser..."
if ! command -v chromium &> /dev/null && ! command -v chromium-browser &> /dev/null; then
    # Try chromium first (newer Ubuntu), then chromium-browser (older Ubuntu)
    apt-get install -y chromium || apt-get install -y chromium-browser || {
        log_error "Failed to install Chromium. Please install manually."
        exit 1
    }
fi

# Detect Chromium path
CHROME_PATH=""
if command -v chromium &> /dev/null; then
    CHROME_PATH=$(which chromium)
elif command -v chromium-browser &> /dev/null; then
    CHROME_PATH=$(which chromium-browser)
elif [ -x "/snap/bin/chromium" ]; then
    CHROME_PATH="/snap/bin/chromium"
fi

if [ -z "$CHROME_PATH" ]; then
    log_error "Chromium not found. Please install Chromium manually."
    exit 1
fi

log_info "Chromium found at: ${CHROME_PATH}"

# Install Caddy
log_info "Installing Caddy..."
apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt-get update
apt-get install -y caddy

# Create application directory
log_info "Creating application directory..."
mkdir -p "${DEPLOY_PATH}"/{bin,config,static}

# Set ownership
chown -R www-data:www-data "${DEPLOY_PATH}"

# Create log directory for Caddy
mkdir -p /var/log/caddy
chown caddy:caddy /var/log/caddy

# Copy systemd service file
log_info "Installing systemd service..."
if [ -f "deploy/sumariza-ai.service" ]; then
    cp deploy/sumariza-ai.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable sumariza-ai
    log_info "Systemd service installed and enabled"
else
    log_warn "sumariza-ai.service not found, skipping systemd setup"
fi

# Configure Caddy if domain is provided
if [ -n "${DOMAIN}" ]; then
    log_info "Configuring Caddy for domain: ${DOMAIN}"
    
    # Replace domain in Caddyfile template
    if [ -f "deploy/Caddyfile.template" ]; then
        sed "s/\${DOMAIN}/${DOMAIN}/g" deploy/Caddyfile.template > /etc/caddy/Caddyfile
        systemctl restart caddy
        log_info "Caddy configured and restarted"
    else
        log_warn "Caddyfile.template not found, skipping Caddy setup"
    fi
else
    log_warn "DOMAIN not set, skipping Caddy configuration"
    log_warn "Set DOMAIN environment variable and run again, or configure manually"
fi

# Create or update .env file
log_info "Configuring .env file..."
if [ ! -f "${DEPLOY_PATH}/.env" ]; then
    cat > "${DEPLOY_PATH}/.env" << EOF
PORT=3000
CACHE_TTL_MINUTES=5
CHROME_PATH=${CHROME_PATH}
EOF
    chown www-data:www-data "${DEPLOY_PATH}/.env"
    chmod 600 "${DEPLOY_PATH}/.env"
    log_info "Created new .env file"
else
    # Update CHROME_PATH if not set or different
    if grep -q "^CHROME_PATH=" "${DEPLOY_PATH}/.env"; then
        sed -i "s|^CHROME_PATH=.*|CHROME_PATH=${CHROME_PATH}|" "${DEPLOY_PATH}/.env"
        log_info "Updated CHROME_PATH in .env"
    else
        echo "CHROME_PATH=${CHROME_PATH}" >> "${DEPLOY_PATH}/.env"
        log_info "Added CHROME_PATH to .env"
    fi
fi

# Print summary
echo ""
echo "============================================"
log_info "Setup complete!"
echo "============================================"
echo ""
echo "Next steps:"
echo "  1. Deploy the application binary:"
echo "     scp bin/sumariza-linux ${DEPLOY_PATH}/bin/sumariza"
echo ""
echo "  2. Deploy config and static files:"
echo "     scp -r config/ ${DEPLOY_PATH}/"
echo "     scp -r static/ ${DEPLOY_PATH}/"
echo ""
echo "  3. Start the service:"
echo "     systemctl start sumariza-ai"
echo ""
echo "  4. Check status:"
echo "     systemctl status sumariza-ai"
echo "     journalctl -u sumariza-ai -f"
echo ""

if [ -z "${DOMAIN}" ]; then
    echo "  5. Configure Caddy (optional):"
    echo "     export DOMAIN=your-domain.com"
    echo "     sed 's/\${DOMAIN}/${DOMAIN}/g' deploy/Caddyfile.template > /etc/caddy/Caddyfile"
    echo "     systemctl restart caddy"
    echo ""
fi

echo "============================================"

