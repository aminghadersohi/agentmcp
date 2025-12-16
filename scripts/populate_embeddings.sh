#!/bin/bash
# Populate embeddings for agents, skills, and commands
# This script connects to the docker containers to generate and store embeddings

set -e

# Configuration - can be overridden with env vars
EMBEDDING_URL="${EMBEDDING_URL:-http://localhost:18081/embed}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-agentmcp-postgres}"
DB_USER="${DB_USER:-mcp}"
DB_NAME="${DB_NAME:-mcp_serve}"
DB_PORT="${DB_PORT:-54320}"

echo "Populating embeddings..."
echo "Embedding service: $EMBEDDING_URL"
echo "Postgres container: $POSTGRES_CONTAINER"
echo ""

# Check if embedding service is up
if ! curl -s "${EMBEDDING_URL%/embed}/health" > /dev/null 2>&1; then
    echo "Warning: Embedding service may not be reachable"
fi

# Force mode regenerates all embeddings
FORCE_MODE=false
if [ "$1" == "--force" ]; then
    FORCE_MODE=true
    echo "Force mode: regenerating all embeddings"
fi

populate_embeddings() {
    local table=$1
    local text_column=$2
    local where_clause=$3

    echo ""
    echo "=== Populating $table embeddings ==="

    if [ "$FORCE_MODE" == "true" ]; then
        query="SELECT id, name, $text_column FROM $table;"
    else
        query="SELECT id, name, $text_column FROM $table WHERE embedding IS NULL;"
    fi

    items=$(docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -t -c "$query" 2>/dev/null)

    local count=0
    local failed=0

    while IFS='|' read -r id name text; do
        # Trim whitespace
        id=$(echo "$id" | xargs)
        name=$(echo "$name" | xargs)
        text=$(echo "$text" | xargs | head -c 2000)  # Limit text length

        if [ -z "$id" ]; then continue; fi

        echo "  Processing: $name"

        # Escape for JSON
        text=$(echo "$text" | sed 's/"/\\"/g' | tr '\n' ' ')

        # Get embedding from service
        response=$(curl -s -X POST "$EMBEDDING_URL" \
            -H "Content-Type: application/json" \
            -d "{\"texts\": [\"$text\"]}" 2>/dev/null)

        embedding=$(echo "$response" | jq -r '.embeddings[0] | @json' 2>/dev/null)

        if [ "$embedding" != "null" ] && [ -n "$embedding" ] && [ "$embedding" != "" ]; then
            # Update database
            docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -c \
                "UPDATE $table SET embedding = '$embedding' WHERE id = '$id';" > /dev/null
            echo "    OK"
            ((count++))
        else
            echo "    FAILED"
            ((failed++))
        fi
    done <<< "$items"

    echo "  $table: Updated $count, Failed $failed"
}

# Populate agents
populate_embeddings "agents" "description || ' ' || COALESCE(array_to_string(skills, ' '), '')" ""

# Populate skills
populate_embeddings "skills" "description || ' ' || COALESCE(content, '')" ""

# Populate commands
populate_embeddings "commands" "description || ' ' || COALESCE(prompt, '')" ""

echo ""
echo "=== Summary ==="
docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -c "
    SELECT 'agents' as table_name,
           COUNT(*) FILTER (WHERE embedding IS NOT NULL) as with_embedding,
           COUNT(*) FILTER (WHERE embedding IS NULL) as without_embedding
    FROM agents
    UNION ALL
    SELECT 'skills',
           COUNT(*) FILTER (WHERE embedding IS NOT NULL),
           COUNT(*) FILTER (WHERE embedding IS NULL)
    FROM skills
    UNION ALL
    SELECT 'commands',
           COUNT(*) FILTER (WHERE embedding IS NOT NULL),
           COUNT(*) FILTER (WHERE embedding IS NULL)
    FROM commands;
"
