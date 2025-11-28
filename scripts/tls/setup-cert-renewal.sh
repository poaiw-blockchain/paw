#!/bin/bash

################################################################################
# Setup Automatic Certificate Renewal
#
# This script configures automatic renewal of Let's Encrypt certificates
# using systemd timers or cron jobs.
#
# Usage:
#   ./setup-cert-renewal.sh [--method systemd|cron]
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
METHOD="${1:-auto}"

echo -e "${GREEN}PAW Chain - Certificate Auto-Renewal Setup${NC}"
echo "============================================"
echo ""

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} certbot is not installed"
    exit 1
fi

# Auto-detect method
if [ "$METHOD" = "auto" ]; then
    if command -v systemctl &> /dev/null && systemctl list-unit-files | grep -q certbot.timer; then
        METHOD="systemd"
    else
        METHOD="cron"
    fi
fi

echo -e "${YELLOW}[INFO]${NC} Using renewal method: $METHOD"
echo ""

# Function to setup systemd timer
setup_systemd() {
    echo -e "${BLUE}Setting up systemd timer for certificate renewal...${NC}"

    # Check if certbot timer exists
    if systemctl list-unit-files | grep -q certbot.timer; then
        echo -e "${YELLOW}[INFO]${NC} certbot.timer already exists"

        # Enable and start the timer
        sudo systemctl enable certbot.timer
        sudo systemctl start certbot.timer

        echo -e "${GREEN}[SUCCESS]${NC} certbot timer enabled and started"

        # Show timer status
        echo ""
        echo "Timer status:"
        systemctl status certbot.timer --no-pager

        echo ""
        echo "Next run:"
        systemctl list-timers certbot.timer --no-pager
    else
        echo -e "${YELLOW}[INFO]${NC} Creating custom systemd timer"

        # Create service file
        sudo tee /etc/systemd/system/certbot-renewal.service > /dev/null << 'EOF'
[Unit]
Description=Certbot Renewal
After=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --deploy-hook "/usr/bin/systemctl reload nginx"
StandardOutput=journal
StandardError=journal
EOF

        # Create timer file
        sudo tee /etc/systemd/system/certbot-renewal.timer > /dev/null << 'EOF'
[Unit]
Description=Certbot Renewal Timer

[Timer]
OnCalendar=daily
RandomizedDelaySec=1h
Persistent=true

[Install]
WantedBy=timers.target
EOF

        # Reload systemd and enable timer
        sudo systemctl daemon-reload
        sudo systemctl enable certbot-renewal.timer
        sudo systemctl start certbot-renewal.timer

        echo -e "${GREEN}[SUCCESS]${NC} Custom certbot renewal timer created and started"

        echo ""
        echo "Timer status:"
        systemctl status certbot-renewal.timer --no-pager
    fi
}

# Function to setup cron job
setup_cron() {
    echo -e "${BLUE}Setting up cron job for certificate renewal...${NC}"

    CRON_JOB="0 3 * * * /usr/bin/certbot renew --quiet --deploy-hook '/usr/sbin/nginx -s reload'"

    # Check if cron job already exists
    if crontab -l 2>/dev/null | grep -q "certbot renew"; then
        echo -e "${YELLOW}[INFO]${NC} Certbot cron job already exists"
        echo ""
        echo "Current certbot cron jobs:"
        crontab -l | grep certbot
    else
        # Add cron job
        (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -

        echo -e "${GREEN}[SUCCESS]${NC} Cron job added successfully"
        echo ""
        echo "Cron job:"
        echo "$CRON_JOB"
        echo ""
        echo "The renewal will run daily at 3:00 AM"
    fi
}

# Create renewal hook script
create_renewal_hook() {
    echo ""
    echo -e "${BLUE}Creating renewal hook script...${NC}"

    HOOK_DIR="/etc/letsencrypt/renewal-hooks/deploy"
    sudo mkdir -p "$HOOK_DIR"

    sudo tee "$HOOK_DIR/reload-nginx.sh" > /dev/null << 'EOF'
#!/bin/bash

# Reload NGINX after certificate renewal
if command -v systemctl &> /dev/null; then
    systemctl reload nginx
elif command -v service &> /dev/null; then
    service nginx reload
else
    nginx -s reload
fi

# Log the reload
echo "[$(date)] Certificate renewed and NGINX reloaded" >> /var/log/certbot-renewal.log
EOF

    sudo chmod +x "$HOOK_DIR/reload-nginx.sh"

    echo -e "${GREEN}[SUCCESS]${NC} Renewal hook created: $HOOK_DIR/reload-nginx.sh"
}

# Setup monitoring script
create_monitoring_script() {
    echo ""
    echo -e "${BLUE}Creating certificate monitoring script...${NC}"

    MONITOR_SCRIPT="/usr/local/bin/check-cert-expiry.sh"

    sudo tee "$MONITOR_SCRIPT" > /dev/null << 'EOF'
#!/bin/bash

# Check all Let's Encrypt certificates for expiry
CERT_DIR="/etc/letsencrypt/live"
WARNING_DAYS=30
CRITICAL_DAYS=7

if [ ! -d "$CERT_DIR" ]; then
    echo "Certificate directory not found: $CERT_DIR"
    exit 1
fi

echo "Checking certificate expiry..."
echo ""

for cert_path in "$CERT_DIR"/*; do
    if [ -d "$cert_path" ]; then
        cert_name=$(basename "$cert_path")
        cert_file="$cert_path/fullchain.pem"

        if [ -f "$cert_file" ]; then
            expiry_date=$(openssl x509 -in "$cert_file" -noout -enddate | cut -d= -f2)
            expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null)
            current_epoch=$(date +%s)
            days_until_expiry=$(( ($expiry_epoch - $current_epoch) / 86400 ))

            echo "Certificate: $cert_name"
            echo "  Expiry: $expiry_date"
            echo "  Days until expiry: $days_until_expiry"

            if [ $days_until_expiry -lt $CRITICAL_DAYS ]; then
                echo "  Status: CRITICAL - Renew immediately!"
            elif [ $days_until_expiry -lt $WARNING_DAYS ]; then
                echo "  Status: WARNING - Should renew soon"
            else
                echo "  Status: OK"
            fi
            echo ""
        fi
    fi
done
EOF

    sudo chmod +x "$MONITOR_SCRIPT"

    echo -e "${GREEN}[SUCCESS]${NC} Monitoring script created: $MONITOR_SCRIPT"
    echo ""
    echo "Run manually: sudo $MONITOR_SCRIPT"
}

# Main setup
case "$METHOD" in
    systemd)
        setup_systemd
        ;;
    cron)
        setup_cron
        ;;
    *)
        echo -e "${RED}[ERROR]${NC} Unknown method: $METHOD"
        echo "Valid methods: systemd, cron"
        exit 1
        ;;
esac

create_renewal_hook
create_monitoring_script

echo ""
echo -e "${GREEN}[SUCCESS]${NC} Certificate auto-renewal setup complete!"
echo ""
echo "Summary:"
echo "  - Renewal method: $METHOD"
echo "  - Renewal frequency: Daily"
echo "  - NGINX reload: Automatic"
echo ""
echo "Testing:"
echo "  - Test renewal: sudo certbot renew --dry-run"
echo "  - Check status: $METHOD status certbot"
echo "  - View logs: sudo journalctl -u certbot"
echo ""
