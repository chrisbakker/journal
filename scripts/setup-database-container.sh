#!/bin/bash

# Docker/Podman Database Setup Script
# This script sets up a PostgreSQL database using Docker or Podman

set -e

# Configuration
CONTAINER_NAME="${CONTAINER_NAME:-journal-postgres}"
DB_NAME="${DB_NAME:-journal}"
DB_USER="${DB_USER:-journal}"
DB_PASSWORD="${DB_PASSWORD:-journaldev}"
DB_PORT="${DB_PORT:-5432}"

# Detect container runtime
if command -v podman &> /dev/null; then
    RUNTIME="podman"
elif command -v docker &> /dev/null; then
    RUNTIME="docker"
else
    echo "❌ Error: Neither podman nor docker found"
    echo "Please install Docker or Podman"
    exit 1
fi

echo "🚀 Journal Database Setup (using $RUNTIME)"
echo "=========================================="
echo ""

# Check if container already exists
if $RUNTIME ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "⚠️  Container '$CONTAINER_NAME' already exists"
    read -p "Do you want to remove it and start fresh? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Stopping and removing existing container..."
        $RUNTIME stop "$CONTAINER_NAME" 2>/dev/null || true
        $RUNTIME rm "$CONTAINER_NAME" 2>/dev/null || true
    else
        echo "Using existing container"
        $RUNTIME start "$CONTAINER_NAME" 2>/dev/null || true
        echo "✅ Container started"
        exit 0
    fi
fi

echo "Step 1: Starting PostgreSQL container with pgvector..."
$RUNTIME run -d \
    --name "$CONTAINER_NAME" \
    -e POSTGRES_PASSWORD="$DB_PASSWORD" \
    -e POSTGRES_USER="$DB_USER" \
    -e POSTGRES_DB="$DB_NAME" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -p "${DB_PORT}:5432" \
    pgvector/pgvector:pg16

echo "✅ Container '$CONTAINER_NAME' started"

echo ""
echo "Step 2: Waiting for PostgreSQL to be ready..."
sleep 3

for i in {1..30}; do
    if $RUNTIME exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" > /dev/null 2>&1; then
        echo "✅ PostgreSQL is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ PostgreSQL failed to start"
        exit 1
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "Step 3: Installing extensions..."
$RUNTIME exec "$CONTAINER_NAME" psql -U postgres -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS vector" > /dev/null
echo "✅ pgvector extension installed"
$RUNTIME exec "$CONTAINER_NAME" psql -U postgres -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS citext" > /dev/null
echo "✅ citext extension installed"

echo ""
echo "✅ Database setup complete!"
echo ""
echo "Connection details:"
echo "  postgresql://$DB_USER:$DB_PASSWORD@localhost:$DB_PORT/$DB_NAME?sslmode=disable"
echo ""
echo "Container management:"
echo "  Stop:    $RUNTIME stop $CONTAINER_NAME"
echo "  Start:   $RUNTIME start $CONTAINER_NAME"
echo "  Remove:  $RUNTIME rm -f $CONTAINER_NAME"
echo ""
echo "Next steps:"
echo "  1. Run migrations: make db-migrate-up"
echo "  2. Create default user: ./scripts/create-default-user.sh"
echo "  3. Start the server: make run"
