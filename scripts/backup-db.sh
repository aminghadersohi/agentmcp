#!/bin/bash
# Backup AgentMCP database
# Usage: ./scripts/backup-db.sh [output_file]
#
# Run on the server or locally with docker access

set -e

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUTPUT_FILE="${1:-backups/agentmcp_backup_${TIMESTAMP}.sql}"

# Ensure backups directory exists
mkdir -p "$(dirname "$OUTPUT_FILE")"

echo "Backing up database to $OUTPUT_FILE..."

docker exec agentmcp-postgres pg_dump -U mcp -d mcp_serve --clean --if-exists > "$OUTPUT_FILE"

echo "Backup complete: $OUTPUT_FILE"
echo "Size: $(du -h "$OUTPUT_FILE" | cut -f1)"
