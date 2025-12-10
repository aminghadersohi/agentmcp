#!/bin/bash
# Deploy AgentMCP with Docker
# Usage: ./deployment/deploy.sh
#
# Prerequisites:
#   - Docker and docker compose installed
#   - Clone this repo first: git clone https://github.com/aminghadersohi/agentmcp.git

set -e

cd "$(dirname "$0")/.."

echo "AgentMCP Deployment"
echo "==================="
echo ""

# Check docker
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is required"
    exit 1
fi

# Build and start
echo "Building and starting services..."
docker compose -f docker-compose.v2.yml up -d --build

echo ""
echo "Waiting for services to start..."
sleep 5

echo ""
echo "=== Container Status ==="
docker compose -f docker-compose.v2.yml ps

echo ""
echo "Deployment complete!"
echo ""
echo "Useful commands:"
echo "  View logs:  docker compose -f docker-compose.v2.yml logs -f"
echo "  Restart:    docker compose -f docker-compose.v2.yml restart"
echo "  Stop:       docker compose -f docker-compose.v2.yml down"
