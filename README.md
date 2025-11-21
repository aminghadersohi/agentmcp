# MCP Serve

> Ultra-lightweight MCP (Model Context Protocol) server for serving AI agent definitions

[![CI](https://github.com/yourusername/mcp-serve/actions/workflows/ci.yml/badge.svg)](https://github.com/yourusername/mcp-serve/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)

**MCP Serve** is a minimalist MCP server that serves specialized AI agent definitions from YAML files. With a ~10MB memory footprint and zero hosting cost, it's perfect for sharing and managing custom agents across your team.

## Features

- **Tiny Footprint**: ~10MB RAM, 8-12MB binary
- **Fast**: <100ms cold start, <10ms request latency
- **Simple**: ~400 lines of Go code, 2 dependencies
- **Flexible**: stdio (local) or HTTP/SSE (remote) transport
- **Zero Cost**: Deploy on Oracle Cloud Always Free tier
- **Hot Reload**: Watch agent files for automatic updates
- **Production Ready**: Docker, systemd, comprehensive tests

## Quick Start

### Installation

**macOS (Homebrew)**:
```bash
# Coming soon
brew install mcp-serve
```

**Download Binary**:
```bash
# Linux (AMD64)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-linux-amd64
chmod +x mcp-serve

# macOS (Apple Silicon)
curl -L -o mcp-serve https://github.com/yourusername/mcp-serve/releases/latest/download/mcp-serve-darwin-arm64
chmod +x mcp-serve
```

**Docker**:
```bash
docker pull ghcr.io/yourusername/mcp-serve:latest
```

### Usage

**Local (stdio)**:
```bash
# Create agents directory
mkdir agents

# Add your agent YAML files to agents/

# Run server
./mcp-serve -agents ./agents -transport stdio
```

**Remote (HTTP/SSE)**:
```bash
./mcp-serve -agents ./agents -transport sse -port 8080
```

**Docker**:
```bash
docker run -v $(pwd)/agents:/app/agents ghcr.io/yourusername/mcp-serve:latest
```

## Agent Definition Format

Create agent definitions in YAML:

```yaml
# agents/frontend-developer.yaml
---
name: frontend-developer
version: 1.0.0
description: Expert frontend engineer specializing in React and TypeScript

model: sonnet  # or opus, haiku

tools:
  - Read
  - Write
  - Grep
  - Glob
  - Edit
  - Bash

metadata:
  author: Your Name
  tags:
    - frontend
    - react
    - typescript
  created: 2025-01-15T10:00:00Z

prompt: |
  You are an expert frontend engineer with deep knowledge of modern web development.

  ## Core Expertise
  - React 18+ with hooks and suspense
  - TypeScript with strict mode
  - CSS Grid, Flexbox, responsive design
  - Performance optimization

  ## Working Principles
  - Component reusability
  - Type safety
  - Mobile-first design
  - Accessibility from the start
```

See [agents/](agents/) for more examples.

## MCP Tools

MCP Serve exposes three tools:

### `list_agents`
List all available agents, optionally filtered by tags.

```json
{
  "name": "list_agents",
  "arguments": {
    "tags": ["frontend", "react"]
  }
}
```

### `get_agent`
Get complete agent definition by name.

```json
{
  "name": "get_agent",
  "arguments": {
    "name": "frontend-developer"
  }
}
```

### `search_agents`
Search agents by keyword in name, description, or tags.

```json
{
  "name": "search_agents",
  "arguments": {
    "query": "typescript"
  }
}
```

## Configuration

### Command Line Flags

```bash
./mcp-serve \
  -agents ./agents \           # Path to agents directory
  -transport stdio \           # Transport: stdio or sse
  -port 8080 \                 # HTTP port (sse mode only)
  -api-key your-secret-key \   # Optional API key
  -watch                       # Enable hot reload
```

### Environment Variables

```bash
export MCP_AGENTS_DIR=./agents
export MCP_TRANSPORT=stdio
export MCP_PORT=8080
export MCP_API_KEY=your-secret-key
export MCP_WATCH=true
```

### Configuration File

Copy `config.yaml.example` to `config.yaml`:

```yaml
agents_dir: ./agents
transport: stdio
port: 8080
watch: true

# Optional Git integration
git:
  repo: https://github.com/yourusername/agents.git
  branch: main
  pull_on_startup: true
```

## Deployment

### Docker Compose

```bash
docker-compose up -d
```

### Systemd (Linux)

```bash
# Copy files
sudo cp mcp-serve /opt/mcp-serve/
sudo cp agents/* /opt/mcp-serve/agents/

# Install service
sudo cp deployment/mcp-serve.service /etc/systemd/system/
sudo systemctl enable mcp-serve
sudo systemctl start mcp-serve
```

Or use the automated installer:

```bash
cd deployment
sudo ./install.sh
```

### Oracle Cloud Always Free

Deploy to Oracle Cloud's perpetual free tier (4 ARM cores, 24GB RAM):

```bash
cd deployment
INSTANCE_IP=your-instance-ip ./deploy-oracle-cloud.sh
```

### Fly.io

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Deploy
fly launch
fly deploy
```

## Development

### Prerequisites

- Go 1.22+
- Make (optional)

### Build from Source

```bash
git clone https://github.com/yourusername/mcp-serve.git
cd mcp-serve
go build -o mcp-serve .
```

### Run Tests

```bash
go test -v ./...
```

### Build for All Platforms

```bash
./build.sh
```

This creates binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Project Structure

```
mcp-serve/
├── main.go               # Server implementation
├── main_test.go          # Unit tests
├── go.mod                # Go dependencies
├── agents/               # Example agents
│   ├── frontend-developer.yaml
│   ├── backend-engineer.yaml
│   ├── devops-engineer.yaml
│   └── code-reviewer.yaml
├── deployment/           # Deployment files
│   ├── mcp-serve.service
│   ├── install.sh
│   └── deploy-oracle-cloud.sh
├── Dockerfile            # Container image
├── docker-compose.yml    # Docker Compose config
├── build.sh              # Multi-platform build script
└── .github/workflows/    # CI/CD pipelines
    ├── ci.yml
    └── release.yml
```

## Performance

Target metrics:

| Metric | Target | Typical |
|--------|--------|---------|
| Binary Size | <12MB | ~10MB |
| Memory (Idle) | <10MB | ~8MB |
| Memory (Load) | <20MB | ~15MB |
| Cold Start | <100ms | ~50ms |
| Request Latency | <10ms | ~2-5ms |
| Throughput | >1000 req/s | ~5000 req/s |

## Security

- **Authentication**: Optional API key via `-api-key` flag or `MCP_API_KEY` env var
- **YAML Safety**: Uses safe YAML parsing (no code execution)
- **File Validation**: Validates agent schema before loading
- **Resource Limits**: Memory and CPU limits via Docker/systemd

## Cost Analysis

| Deployment | Monthly Cost | Notes |
|------------|--------------|-------|
| **Oracle Cloud Always Free** | **$0** | Perpetual free tier, 24GB RAM |
| **AWS Lambda Free Tier** | **$0** | <1M requests/month |
| **Fly.io Hobby** | $5 | Auto-scaling, global |
| **Railway** | $5 | Usage-based |
| Local (dev) | $0 | stdio transport |

**Recommended**: Oracle Cloud Always Free for production.

## Roadmap

- [x] Core MCP server with 3 tools
- [x] stdio and HTTP/SSE transports
- [x] File watcher for hot reload
- [x] Docker support
- [x] Systemd service
- [x] Deployment scripts
- [x] CI/CD pipelines
- [ ] Agent validation (JSON Schema)
- [ ] Fuzzy search
- [ ] Webhooks for Git updates
- [ ] Multi-repo support
- [ ] Prometheus metrics endpoint
- [ ] Agent composition/imports

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## FAQ

**Q: Why Go instead of Rust?**
A: Faster development, excellent MCP SDK, good enough performance. Go produces 8-12MB binaries vs Rust's 2-3MB, but for serving YAML files, the difference doesn't matter.

**Q: Why not use a database?**
A: Agent definitions are static files that change infrequently. Git + filesystem provides version control, easy editing, zero overhead, and simple backup.

**Q: Can I run multiple instances?**
A: Yes! The server is read-only, so you can run multiple instances behind a load balancer. Each instance uses <10MB RAM.

**Q: How do I update agents?**
A: Edit YAML files and either restart the server or enable `-watch` for hot reload.

**Q: What about agent execution?**
A: Out of scope. Agents execute in MCP clients (like Claude Code), not the server. The server only serves definitions.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [mcp-go](https://github.com/mark3labs/mcp-go)
- Inspired by the Quake engine's legendary efficiency
- Philosophy: "Perfection is achieved not when there is nothing more to add, but when there is nothing left to take away." - Antoine de Saint-Exupéry

## Links

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)
- [Documentation](https://github.com/yourusername/mcp-serve/wiki)
- [Issues](https://github.com/yourusername/mcp-serve/issues)

---

**Built with ❤️ for the MCP community**
