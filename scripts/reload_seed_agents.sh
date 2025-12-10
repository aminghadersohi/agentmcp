#!/bin/bash
# Reload seed agents into the database
# This script can be run on clean deploys or to update agent definitions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-54320}"
DB_NAME="${DB_NAME:-mcp_serve}"
DB_USER="${DB_USER:-mcp}"
DB_PASSWORD="${DB_PASSWORD:-mcpserve}"

echo "======================================"
echo "  AgentMCP - Reload Seed Agents"
echo "======================================"
echo ""
echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Check if running in Docker or local
if command -v docker &> /dev/null && docker ps | grep -q agentmcp-postgres; then
    echo "✓ Detected Docker environment"
    echo "  Running seed script via Docker..."

    docker exec -i agentmcp-postgres psql -U "$DB_USER" -d "$DB_NAME" < "$PROJECT_ROOT/migrations/002_seed_agents.sql"

elif command -v psql &> /dev/null; then
    echo "✓ Detected local psql"
    echo "  Running seed script locally..."

    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$PROJECT_ROOT/migrations/002_seed_agents.sql"

else
    echo "❌ Error: Neither docker nor psql found"
    echo ""
    echo "Please install PostgreSQL client or ensure Docker is running"
    exit 1
fi

echo ""
echo "✓ Seed agents reloaded successfully!"
echo ""
echo "Agents now in database:"
docker exec agentmcp-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT name, is_system, status FROM agents ORDER BY is_system DESC, name;" 2>/dev/null || \
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT name, is_system, status FROM agents ORDER BY is_system DESC, name;"

echo ""
echo "======================================"
