#!/bin/bash
# Restore AgentMCP database from backup
# Usage: ./scripts/restore-db.sh <backup_file>
#
# WARNING: This will overwrite the current database!

set -e

if [ -z "$1" ]; then
    echo "Usage: ./scripts/restore-db.sh <backup_file>"
    echo ""
    echo "Available backups:"
    ls -la backups/*.sql 2>/dev/null || echo "  No backups found in backups/"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "WARNING: This will overwrite the current database!"
echo "Backup file: $BACKUP_FILE"
echo ""
read -p "Are you sure? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo "Restoring database from $BACKUP_FILE..."

docker exec -i agentmcp-postgres psql -U mcp -d mcp_serve < "$BACKUP_FILE"

echo "Restore complete!"
echo ""
echo "Restart agentmcp to pick up changes:"
echo "  docker restart agentmcp"
