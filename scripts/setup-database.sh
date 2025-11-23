#!/bin/bash

# Database Setup Script for Journal App
# This script sets up a PostgreSQL database with pgvector support

set -e

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-journal}"
DB_USER="${DB_USER:-journal}"
DB_PASSWORD="${DB_PASSWORD:-journaldev}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"

echo "🚀 Journal Database Setup"
echo "========================="
echo ""
echo "Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo "❌ Error: psql command not found"
    echo "Please install PostgreSQL client tools"
    exit 1
fi

# Function to run SQL as postgres user
run_sql_as_postgres() {
    PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d postgres -c "$1"
}

# Function to run SQL on journal database as postgres user
run_sql_on_journal() {
    PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d "$DB_NAME" -c "$1"
}

echo "Step 1: Creating database..."
if run_sql_as_postgres "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1; then
    echo "✅ Database '$DB_NAME' already exists"
else
    run_sql_as_postgres "CREATE DATABASE $DB_NAME"
    echo "✅ Database '$DB_NAME' created"
fi

echo ""
echo "Step 2: Creating user..."
if run_sql_as_postgres "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1; then
    echo "✅ User '$DB_USER' already exists"
else
    run_sql_as_postgres "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD'"
    echo "✅ User '$DB_USER' created"
fi

echo ""
echo "Step 3: Granting privileges..."
run_sql_as_postgres "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER"
run_sql_on_journal "GRANT ALL ON SCHEMA public TO $DB_USER"
echo "✅ Privileges granted"

echo ""
echo "Step 4: Installing extensions..."
run_sql_on_journal "CREATE EXTENSION IF NOT EXISTS vector"
echo "✅ pgvector extension installed"
run_sql_on_journal "CREATE EXTENSION IF NOT EXISTS citext"
echo "✅ citext extension installed"

echo ""
echo "✅ Database setup complete!"
echo ""
echo "Connection details:"
echo "  postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable"
echo ""
echo "Next steps:"
echo "  1. Run migrations: make db-migrate-up"
echo "  2. Create default user: ./scripts/create-default-user.sh"
echo "  3. Start the server: make run"
