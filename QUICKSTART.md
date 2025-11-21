# MCP Serve - Quick Start Guide

Get up and running with MCP Serve in 5 minutes.

## Step 1: Install

Choose your platform:

### macOS
```bash
# Download for Apple Silicon (M1/M2/M3)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-darwin-arm64

# Or for Intel Macs
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-darwin-amd64

chmod +x mcp-serve
```

### Linux
```bash
# AMD64
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64

# ARM64
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-arm64

chmod +x mcp-serve
```

### Windows
Download from [Releases](https://github.com/yourusername/mcp-serve/releases/latest) and add to PATH.

### Docker
```bash
docker pull ghcr.io/yourusername/mcp-serve:latest
```

## Step 2: Create Your First Agent

Create a directory for agents:
```bash
mkdir agents
```

Create `agents/my-first-agent.yaml`:
```yaml
---
name: my-first-agent
version: 1.0.0
description: My first custom agent

model: sonnet

tools:
  - Read
  - Write

metadata:
  author: Your Name
  tags:
    - example

prompt: |
  You are a helpful assistant that specializes in writing documentation.

  When asked to write docs:
  1. Use clear, concise language
  2. Include code examples
  3. Organize with proper headings
  4. Add helpful tips and warnings
```

## Step 3: Run the Server

**Local (stdio mode)**:
```bash
./mcp-serve -agents ./agents -transport stdio
```

**Remote (HTTP/SSE mode)**:
```bash
./mcp-serve -agents ./agents -transport sse -port 8080
```

**Docker**:
```bash
docker run -v $(pwd)/agents:/app/agents ghcr.io/yourusername/mcp-serve:latest
```

## Step 4: Test It

The server exposes three MCP tools:

### List all agents
```json
{
  "method": "tools/call",
  "params": {
    "name": "list_agents"
  }
}
```

### Get agent details
```json
{
  "method": "tools/call",
  "params": {
    "name": "get_agent",
    "arguments": {
      "name": "my-first-agent"
    }
  }
}
```

### Search agents
```json
{
  "method": "tools/call",
  "params": {
    "name": "search_agents",
    "arguments": {
      "query": "documentation"
    }
  }
}
```

## Step 5: Connect from MCP Client

### Using with Claude Code (stdio)

Add to your MCP client configuration:
```json
{
  "mcpServers": {
    "mcp-serve": {
      "command": "/path/to/mcp-serve",
      "args": ["-agents", "/path/to/agents", "-transport", "stdio"]
    }
  }
}
```

### Using with HTTP/SSE

```json
{
  "mcpServers": {
    "mcp-serve": {
      "url": "http://localhost:8080"
    }
  }
}
```

## Next Steps

### Add More Agents

Create specialized agents for different tasks:

```bash
# Frontend developer
agents/frontend-developer.yaml

# Backend engineer
agents/backend-engineer.yaml

# DevOps specialist
agents/devops-engineer.yaml

# Code reviewer
agents/code-reviewer.yaml
```

### Enable Hot Reload

Watch for file changes:
```bash
./mcp-serve -agents ./agents -transport stdio -watch
```

### Use Environment Variables

```bash
export MCP_AGENTS_DIR=./agents
export MCP_TRANSPORT=stdio
export MCP_WATCH=true

./mcp-serve
```

### Deploy to Production

See deployment guides:
- [Docker Deployment](README.md#docker-compose)
- [Oracle Cloud Free Tier](README.md#oracle-cloud-always-free)
- [Systemd Service](README.md#systemd-linux)

## Common Commands

```bash
# Run with custom config
./mcp-serve -agents ./agents -transport stdio -watch

# Run on different port
./mcp-serve -agents ./agents -transport sse -port 3000

# Run with API key
./mcp-serve -agents ./agents -transport sse -api-key secret123

# Check version
./mcp-serve -version

# Run tests (if building from source)
go test -v ./...
```

## Troubleshooting

### No agents loaded
- Check that `agents/` directory exists
- Verify YAML files have `.yaml` or `.yml` extension
- Check YAML syntax is valid
- Look for errors in server logs

### Server won't start
- Check if port is already in use (sse mode)
- Verify binary has execute permissions
- Check Go version (requires 1.22+)

### Agent not appearing
- Ensure agent has a `name` field
- Check YAML syntax
- Restart server (or use `-watch` flag)

## Get Help

- [Full Documentation](README.md)
- [GitHub Issues](https://github.com/yourusername/mcp-serve/issues)
- [Contributing Guide](CONTRIBUTING.md)

## Examples

See the [agents/](agents/) directory for complete examples:
- `frontend-developer.yaml` - React/TypeScript specialist
- `backend-engineer.yaml` - API/database expert
- `devops-engineer.yaml` - Docker/K8s specialist
- `code-reviewer.yaml` - Code quality expert

Happy building!
