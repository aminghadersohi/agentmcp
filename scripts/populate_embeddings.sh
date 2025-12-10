#!/bin/bash
# Populate embeddings for all agents
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

# Check if embedding service is up
if ! curl -s "$EMBEDDING_URL/../health" > /dev/null 2>&1; then
    echo "Warning: Embedding service at $EMBEDDING_URL may not be reachable"
fi

# Get all agents without embeddings (or all if --force flag)
if [ "$1" == "--force" ]; then
    echo "Force mode: regenerating all embeddings"
    query="SELECT id, name, description, COALESCE(skills::text, '{}'), COALESCE(metadata->'tags'::text, '[]') FROM agents;"
else
    query="SELECT id, name, description, COALESCE(skills::text, '{}'), COALESCE(metadata->'tags'::text, '[]') FROM agents WHERE embedding IS NULL;"
fi

agents=$(docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -t -c "$query")

count=0
failed=0

while IFS='|' read -r id name description skills tags; do
    # Trim whitespace
    id=$(echo "$id" | xargs)
    name=$(echo "$name" | xargs)
    description=$(echo "$description" | xargs)
    skills=$(echo "$skills" | xargs | tr -d '{}' | tr ',' ' ')
    tags=$(echo "$tags" | xargs | tr -d '[]"' | tr ',' ' ')

    if [ -z "$id" ]; then continue; fi

    echo "Processing: $name"

    # Create rich embedding text combining name, description, skills and tags
    text="Agent: $name. $description. Skills: $skills $tags"

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
            "UPDATE agents SET embedding = '$embedding' WHERE id = '$id';" > /dev/null
        echo "  ✓ Updated embedding for $name"
        ((count++))
    else
        echo "  ✗ Failed to get embedding for $name"
        echo "    Response: $response"
        ((failed++))
    fi
done <<< "$agents"

echo ""
echo "Done! Updated: $count, Failed: $failed"

# Show summary
echo ""
echo "Summary:"
docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -c \
    "SELECT COUNT(*) as with_embedding FROM agents WHERE embedding IS NOT NULL;"
docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -p "$DB_PORT" -c \
    "SELECT COUNT(*) as without_embedding FROM agents WHERE embedding IS NULL;"
