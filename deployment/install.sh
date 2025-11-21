#!/bin/bash
# Installation script for mcp-serve on Linux systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/opt/mcp-serve"
SERVICE_USER="mcp"
SERVICE_GROUP="mcp"
BINARY_URL="https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64"

echo -e "${GREEN}MCP Serve Installation Script${NC}"
echo "=============================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (use sudo)${NC}"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        BINARY_URL="https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64"
        ;;
    aarch64|arm64)
        BINARY_URL="https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${YELLOW}Step 1: Creating user and group${NC}"
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
    echo -e "${GREEN}Created user: $SERVICE_USER${NC}"
else
    echo -e "${YELLOW}User $SERVICE_USER already exists${NC}"
fi

echo ""
echo -e "${YELLOW}Step 2: Creating installation directory${NC}"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/agents"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"
echo -e "${GREEN}Created directory: $INSTALL_DIR${NC}"

echo ""
echo -e "${YELLOW}Step 3: Downloading binary${NC}"
echo "Downloading from: $BINARY_URL"
if command -v curl &> /dev/null; then
    curl -L -o "$INSTALL_DIR/mcp-serve" "$BINARY_URL"
elif command -v wget &> /dev/null; then
    wget -O "$INSTALL_DIR/mcp-serve" "$BINARY_URL"
else
    echo -e "${RED}Error: Neither curl nor wget found. Please install one.${NC}"
    exit 1
fi

chmod +x "$INSTALL_DIR/mcp-serve"
chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/mcp-serve"
echo -e "${GREEN}Binary downloaded and installed${NC}"

echo ""
echo -e "${YELLOW}Step 4: Installing systemd service${NC}"
cp mcp-serve.service /etc/systemd/system/
systemctl daemon-reload
echo -e "${GREEN}Service installed${NC}"

echo ""
echo -e "${YELLOW}Step 5: Creating example agent files${NC}"
if [ ! -f "$INSTALL_DIR/agents/example.yaml" ]; then
    cat > "$INSTALL_DIR/agents/example.yaml" <<EOF
---
name: example-agent
version: 1.0.0
description: Example agent to get you started
model: sonnet
tools:
  - Read
  - Write
metadata:
  author: MCP Serve
  tags:
    - example
prompt: |
  You are an example agent. Replace this with your custom prompt.
EOF
    chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/agents/example.yaml"
    echo -e "${GREEN}Created example agent file${NC}"
fi

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Add your agent YAML files to: $INSTALL_DIR/agents/"
echo "2. Start the service: sudo systemctl start mcp-serve"
echo "3. Enable auto-start: sudo systemctl enable mcp-serve"
echo "4. Check status: sudo systemctl status mcp-serve"
echo "5. View logs: sudo journalctl -u mcp-serve -f"
echo ""
echo "To uninstall, run: sudo systemctl stop mcp-serve && sudo systemctl disable mcp-serve && sudo rm -rf $INSTALL_DIR"
