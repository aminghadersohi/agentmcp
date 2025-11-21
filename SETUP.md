# MCP Serve - Complete Setup Guide

This guide walks you through setting up MCP Serve from scratch.

## Prerequisites

### Install Go

MCP Serve requires Go 1.22 or later.

**macOS**:
```bash
brew install go
```

**Linux**:
```bash
# Download and install
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
```

**Windows**:
Download installer from https://go.dev/dl/

Verify installation:
```bash
go version
# Should show: go version go1.22.0 or later
```

## Option 1: Use Pre-built Binaries (Recommended)

### Download

Visit [Releases](https://github.com/yourusername/mcp-serve/releases/latest) and download for your platform.

**Or use command line**:

```bash
# macOS (Apple Silicon)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-darwin-arm64
chmod +x mcp-serve

# macOS (Intel)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-darwin-amd64
chmod +x mcp-serve

# Linux (AMD64)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64
chmod +x mcp-serve

# Linux (ARM64)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-arm64
chmod +x mcp-serve
```

### Test

```bash
./mcp-serve -version
```

## Option 2: Build from Source

### Clone Repository

```bash
git clone https://github.com/yourusername/mcp-serve.git
cd mcp-serve
```

### Download Dependencies

```bash
go mod download
go mod verify
```

### Build

```bash
# Quick build
go build -o mcp-serve .

# Or use Makefile
make build

# Or build for all platforms
./build.sh
```

### Test

```bash
# Run tests
go test -v ./...

# Run tests with coverage
make test-cov

# Run the binary
./mcp-serve -version
```

## Option 3: Docker

### Pull Image

```bash
docker pull ghcr.io/yourusername/mcp-serve:latest
```

### Or Build Locally

```bash
docker build -t mcp-serve:local .
```

### Run

```bash
docker run -v $(pwd)/agents:/app/agents ghcr.io/yourusername/mcp-serve:latest
```

## Basic Configuration

### 1. Create Agents Directory

```bash
mkdir -p agents
```

### 2. Create Your First Agent

Create `agents/helper.yaml`:

```yaml
---
name: helper
version: 1.0.0
description: A helpful assistant

model: sonnet

tools:
  - Read
  - Write

prompt: |
  You are a helpful assistant that provides clear and concise answers.
```

### 3. Run the Server

**stdio mode (for local MCP clients)**:
```bash
./mcp-serve -agents ./agents -transport stdio
```

**HTTP/SSE mode (for remote access)**:
```bash
./mcp-serve -agents ./agents -transport sse -port 8080
```

**With hot reload**:
```bash
./mcp-serve -agents ./agents -transport stdio -watch
```

## Advanced Configuration

### Environment Variables

Create a `.env` file:
```bash
MCP_AGENTS_DIR=./agents
MCP_TRANSPORT=stdio
MCP_PORT=8080
MCP_WATCH=true
```

Load and run:
```bash
source .env
./mcp-serve
```

### Configuration File

Copy the example config:
```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml`:
```yaml
agents_dir: ./agents
transport: stdio
watch: true
```

**Note**: Command line flags override environment variables, which override config file.

### With API Key

```bash
./mcp-serve -agents ./agents -transport sse -api-key mysecretkey
```

Or set environment variable:
```bash
export MCP_API_KEY=mysecretkey
./mcp-serve -agents ./agents -transport sse
```

## Production Deployment

### systemd (Linux)

```bash
# Navigate to deployment directory
cd deployment

# Run installer (requires sudo)
sudo ./install.sh

# Start service
sudo systemctl start mcp-serve

# Enable auto-start
sudo systemctl enable mcp-serve

# Check status
sudo systemctl status mcp-serve

# View logs
sudo journalctl -u mcp-serve -f
```

### Docker Compose

```bash
# Copy and edit docker-compose.yml
cp docker-compose.yml docker-compose.prod.yml

# Start
docker-compose -f docker-compose.prod.yml up -d

# View logs
docker-compose -f docker-compose.prod.yml logs -f

# Stop
docker-compose -f docker-compose.prod.yml down
```

### Oracle Cloud

```bash
# Set up SSH key and get instance IP
export INSTANCE_IP=your-instance-ip
export SSH_KEY=~/.ssh/id_rsa

# Run deployment script
cd deployment
./deploy-oracle-cloud.sh

# Check status
ssh -i $SSH_KEY ubuntu@$INSTANCE_IP 'sudo systemctl status mcp-serve'
```

## Connecting MCP Clients

### Claude Code (stdio)

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "agents": {
      "command": "/path/to/mcp-serve",
      "args": [
        "-agents",
        "/path/to/agents",
        "-transport",
        "stdio"
      ]
    }
  }
}
```

### HTTP/SSE Client

```json
{
  "mcpServers": {
    "agents": {
      "url": "http://localhost:8080",
      "headers": {
        "Authorization": "Bearer your-api-key"
      }
    }
  }
}
```

## Verification

### Test MCP Tools

With server running, the MCP client should see three tools:

1. **list_agents**: List all agents
2. **get_agent**: Get agent by name
3. **search_agents**: Search agents

### Manual Testing (HTTP/SSE mode)

```bash
# List agents
curl http://localhost:8080/mcp -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {"name": "list_agents"},
  "id": 1
}'

# Get specific agent
curl http://localhost:8080/mcp -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get_agent",
    "arguments": {"name": "helper"}
  },
  "id": 2
}'
```

## Troubleshooting

### "go: command not found"

Go is not installed or not in PATH.
- Install Go: https://go.dev/dl/
- Add to PATH: `export PATH=$PATH:/usr/local/go/bin`

### "cannot find package"

Dependencies not downloaded.
```bash
go mod download
```

### "permission denied"

Binary not executable.
```bash
chmod +x mcp-serve
```

### "address already in use"

Port is already taken (SSE mode).
```bash
# Use different port
./mcp-serve -agents ./agents -transport sse -port 8081

# Or find and kill process using port
lsof -ti:8080 | xargs kill
```

### "no agents loaded"

- Check agents directory exists: `ls -la agents/`
- Verify YAML files: `cat agents/*.yaml`
- Check file extensions: `.yaml` or `.yml`
- Look at server logs for parsing errors

### Agent not showing up

- Ensure agent has `name` field
- Check YAML syntax is valid
- Restart server (or use `-watch` flag)
- Check logs for errors

## Updating

### Binary

Download latest release and replace:
```bash
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x mcp-serve
```

### Docker

```bash
docker pull ghcr.io/yourusername/mcp-serve:latest
docker-compose up -d
```

### From Source

```bash
git pull
go build -o mcp-serve .
```

### systemd Service

```bash
# Download new binary
sudo curl -L -o /opt/mcp-serve/mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64
sudo chmod +x /opt/mcp-serve/mcp-serve

# Restart service
sudo systemctl restart mcp-serve
```

## Development Setup

### IDE Setup

**VS Code**:
- Install "Go" extension
- Open workspace: `code .`

**GoLand**:
- Open project directory
- Wait for indexing

### Running Tests

```bash
# All tests
go test -v ./...

# Specific test
go test -v -run TestLoadAgents

# With coverage
go test -v -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Making Changes

```bash
# Create branch
git checkout -b feature/my-feature

# Make changes
# Edit files...

# Run tests
go test -v ./...

# Format code
go fmt ./...

# Build
go build -o mcp-serve .

# Test locally
./mcp-serve -agents ./agents -transport stdio
```

## Next Steps

- Read [QUICKSTART.md](QUICKSTART.md) for usage guide
- See [README.md](README.md) for full documentation
- Check [CONTRIBUTING.md](CONTRIBUTING.md) to contribute
- Browse [agents/](agents/) for examples

## Get Help

- GitHub Issues: Report bugs or request features
- Discussions: Ask questions and share ideas
- Documentation: Full docs in README.md

## Useful Commands Reference

```bash
# Build
go build -o mcp-serve .
make build

# Test
go test -v ./...
make test

# Run
./mcp-serve -agents ./agents -transport stdio
./mcp-serve -agents ./agents -transport sse -port 8080

# Docker
docker build -t mcp-serve .
docker run -v $(pwd)/agents:/app/agents mcp-serve

# systemd
sudo systemctl start mcp-serve
sudo systemctl status mcp-serve
sudo journalctl -u mcp-serve -f

# Development
go fmt ./...
go vet ./...
make lint
```

Happy building! 🚀
