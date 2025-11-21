#!/bin/bash
# Deployment script for Oracle Cloud Always Free tier

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}MCP Serve - Oracle Cloud Deployment${NC}"
echo "===================================="
echo ""

# Configuration
INSTANCE_IP="${INSTANCE_IP:-}"
SSH_KEY="${SSH_KEY:-~/.ssh/id_rsa}"
SSH_USER="ubuntu"
INSTALL_DIR="/opt/mcp-serve"
AGENTS_REPO="${AGENTS_REPO:-}"

# Check required variables
if [ -z "$INSTANCE_IP" ]; then
    echo -e "${RED}Error: INSTANCE_IP environment variable not set${NC}"
    echo "Usage: INSTANCE_IP=your-instance-ip ./deploy-oracle-cloud.sh"
    exit 1
fi

echo -e "${YELLOW}Deploying to: $INSTANCE_IP${NC}"
echo ""

# Test SSH connection
echo -e "${YELLOW}Step 1: Testing SSH connection${NC}"
if ! ssh -i "$SSH_KEY" -o ConnectTimeout=10 "$SSH_USER@$INSTANCE_IP" "echo 'SSH connection successful'"; then
    echo -e "${RED}Error: Cannot connect to instance${NC}"
    exit 1
fi
echo -e "${GREEN}SSH connection OK${NC}"
echo ""

# Update system
echo -e "${YELLOW}Step 2: Updating system${NC}"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << 'EOF'
sudo apt-get update
sudo apt-get upgrade -y
sudo apt-get install -y git curl ca-certificates
EOF
echo -e "${GREEN}System updated${NC}"
echo ""

# Create installation directory
echo -e "${YELLOW}Step 3: Creating installation directory${NC}"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << EOF
sudo mkdir -p $INSTALL_DIR
sudo mkdir -p $INSTALL_DIR/agents
sudo useradd -r -s /bin/false -d $INSTALL_DIR mcp || true
sudo chown -R mcp:mcp $INSTALL_DIR
EOF
echo -e "${GREEN}Directory created${NC}"
echo ""

# Detect architecture and download binary
echo -e "${YELLOW}Step 4: Downloading mcp-serve binary${NC}"
ARCH=$(ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" "uname -m")
case $ARCH in
    x86_64)
        BINARY="mcp-serve-linux-amd64"
        ;;
    aarch64|arm64)
        BINARY="mcp-serve-linux-arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

DOWNLOAD_URL="https://github.com/yourusername/mcp-serve/releases/latest/download/$BINARY"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << EOF
sudo curl -L -o $INSTALL_DIR/mcp-serve "$DOWNLOAD_URL"
sudo chmod +x $INSTALL_DIR/mcp-serve
sudo chown mcp:mcp $INSTALL_DIR/mcp-serve
EOF
echo -e "${GREEN}Binary downloaded${NC}"
echo ""

# Clone agents repository if provided
if [ -n "$AGENTS_REPO" ]; then
    echo -e "${YELLOW}Step 5: Cloning agents repository${NC}"
    ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << EOF
cd $INSTALL_DIR
sudo -u mcp git clone $AGENTS_REPO agents-repo
sudo cp agents-repo/*.yaml agents/ 2>/dev/null || true
EOF
    echo -e "${GREEN}Agents cloned${NC}"
    echo ""
else
    echo -e "${YELLOW}Step 5: Creating example agent${NC}"
    ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << 'EOF'
sudo tee /opt/mcp-serve/agents/example.yaml > /dev/null <<AGENT
---
name: example-agent
version: 1.0.0
description: Example agent
model: sonnet
tools:
  - Read
  - Write
prompt: You are an example agent.
AGENT
sudo chown mcp:mcp /opt/mcp-serve/agents/example.yaml
EOF
    echo -e "${GREEN}Example agent created${NC}"
    echo ""
fi

# Install systemd service
echo -e "${YELLOW}Step 6: Installing systemd service${NC}"
scp -i "$SSH_KEY" mcp-serve.service "$SSH_USER@$INSTANCE_IP:/tmp/"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << 'EOF'
sudo mv /tmp/mcp-serve.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mcp-serve
sudo systemctl start mcp-serve
EOF
echo -e "${GREEN}Service installed and started${NC}"
echo ""

# Configure firewall
echo -e "${YELLOW}Step 7: Configuring firewall${NC}"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" << 'EOF'
# Allow SSH, HTTP, and custom port 8080
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 8080/tcp
sudo ufw --force enable
EOF
echo -e "${GREEN}Firewall configured${NC}"
echo ""

# Check service status
echo -e "${YELLOW}Step 8: Checking service status${NC}"
ssh -i "$SSH_KEY" "$SSH_USER@$INSTANCE_IP" "sudo systemctl status mcp-serve --no-pager"
echo ""

echo -e "${GREEN}Deployment complete!${NC}"
echo ""
echo "Service is running at: http://$INSTANCE_IP:8080"
echo ""
echo "Useful commands:"
echo "  Check status:  ssh -i $SSH_KEY $SSH_USER@$INSTANCE_IP 'sudo systemctl status mcp-serve'"
echo "  View logs:     ssh -i $SSH_KEY $SSH_USER@$INSTANCE_IP 'sudo journalctl -u mcp-serve -f'"
echo "  Restart:       ssh -i $SSH_KEY $SSH_USER@$INSTANCE_IP 'sudo systemctl restart mcp-serve'"
echo "  Update agents: scp -i $SSH_KEY your-agent.yaml $SSH_USER@$INSTANCE_IP:$INSTALL_DIR/agents/"
