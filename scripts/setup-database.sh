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
DEFAULT_USER_ID="02a0aa58-b88a-46f1-9799-f103e04c0b72"
DEFAULT_EMAIL="user@journal.local"
DEFAULT_NAME="Default User"

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

# Prompt for postgres password once if not set
if [ -z "$POSTGRES_PASSWORD" ]; then
    read -sp "Enter password for PostgreSQL user '$POSTGRES_USER' (or press Enter if no password): " POSTGRES_PASSWORD
    echo ""
    echo ""
fi

# Function to run SQL as postgres user
run_sql_as_postgres() {
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d postgres -c "$1"
}

# Function to run SQL on journal database as postgres user
run_sql_on_journal() {
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d "$DB_NAME" -c "$1"
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
echo ""

# Wait a moment for user to see the message
sleep 1

echo "Running migrations..."
if command -v migrate &> /dev/null; then
    if migrate -path db/migrations -database "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up; then
        echo "✅ Migrations completed successfully"
    else
        echo "❌ Migration failed"
        echo "You can run migrations manually with: make db-migrate-up"
        exit 1
    fi
else
    echo "⚠️  'migrate' command not found"
    echo "Please install golang-migrate and run: make db-migrate-up"
    exit 1
fi

echo ""
echo "Step 5: Creating default user..."
if run_sql_on_journal "INSERT INTO users (id, email, display_name) VALUES ('$DEFAULT_USER_ID', '$DEFAULT_EMAIL', '$DEFAULT_NAME') ON CONFLICT (id) DO NOTHING RETURNING id" | grep -q "$DEFAULT_USER_ID"; then
    echo "✅ Default user created"
elif run_sql_on_journal "SELECT id FROM users WHERE id = '$DEFAULT_USER_ID'" | grep -q "$DEFAULT_USER_ID"; then
    echo "✅ Default user already exists"
else
    echo "❌ Failed to create default user"
    exit 1
fi

echo ""
echo "User details:"
echo "  ID: $DEFAULT_USER_ID"
echo "  Email: $DEFAULT_EMAIL"
echo "  Name: $DEFAULT_NAME"

echo ""
echo "🎉 Setup complete!"
echo ""
echo "Start the server:"
echo "  make run"
echo ""
echo "Then open: http://localhost:8080"
