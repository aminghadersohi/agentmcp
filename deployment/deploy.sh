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
sleep 10

# Check if this is an existing database that needs schema_migrations seeded
echo ""
echo "Checking database migration status..."

# Wait for postgres to be ready
for i in {1..30}; do
    if docker exec agentmcp-postgres pg_isready -U mcp -d mcp_serve -p 54320 -q 2>/dev/null; then
        break
    fi
    echo "  Waiting for postgres... ($i/30)"
    sleep 2
done

# Check if schema_migrations table exists and is populated
MIGRATION_COUNT=$(docker exec agentmcp-postgres psql -U mcp -d mcp_serve -t -c \
    "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'schema_migrations';" 2>/dev/null | tr -d ' ')

if [ "$MIGRATION_COUNT" = "1" ]; then
    APPLIED=$(docker exec agentmcp-postgres psql -U mcp -d mcp_serve -t -c \
        "SELECT COUNT(*) FROM schema_migrations;" 2>/dev/null | tr -d ' ')

    if [ "$APPLIED" = "0" ]; then
        echo "  Existing database detected without migration tracking."
        echo "  Seeding schema_migrations table..."

        # Check which tables exist and seed accordingly
        docker exec agentmcp-postgres psql -U mcp -d mcp_serve -c "
            INSERT INTO schema_migrations (version)
            SELECT '001_initial.sql' WHERE EXISTS (SELECT 1 FROM agents LIMIT 1)
            ON CONFLICT DO NOTHING;

            INSERT INTO schema_migrations (version) VALUES
                ('002_seed_agents_fixed.sql'),
                ('003_populate_skills.sql')
            ON CONFLICT DO NOTHING;

            INSERT INTO schema_migrations (version)
            SELECT '004_skills_commands.sql' WHERE EXISTS (SELECT 1 FROM skills LIMIT 1)
            ON CONFLICT DO NOTHING;

            INSERT INTO schema_migrations (version)
            SELECT '005_seed_skills_commands.sql' WHERE EXISTS (SELECT 1 FROM skills WHERE is_system = true LIMIT 1)
            ON CONFLICT DO NOTHING;
        " 2>/dev/null || true

        echo "  Migration tracking initialized."
    else
        echo "  Migration tracking OK ($APPLIED migrations applied)"
    fi
else
    echo "  Fresh database - migrations will run automatically"
fi

# Restart agentmcp to run any pending migrations
echo ""
echo "Restarting agentmcp to apply migrations..."
docker compose -f docker-compose.v2.yml restart agentmcp
sleep 5

echo ""
echo "=== Container Status ==="
docker compose -f docker-compose.v2.yml ps

echo ""
echo "=== Recent Logs ==="
docker compose -f docker-compose.v2.yml logs --tail=20 agentmcp

echo ""
echo "Deployment complete!"
echo ""
echo "Service available at: http://localhost:18080/sse"
echo ""
echo "Useful commands:"
echo "  View logs:  docker compose -f docker-compose.v2.yml logs -f"
echo "  Restart:    docker compose -f docker-compose.v2.yml restart"
echo "  Stop:       docker compose -f docker-compose.v2.yml down"
