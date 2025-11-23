#!/bin/bash

# Create Default User Script
# This script creates the default user after migrations have been run

set -e

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-journal}"
DB_USER="${DB_USER:-journal}"
DB_PASSWORD="${DB_PASSWORD:-journaldev}"

DEFAULT_USER_ID="02a0aa58-b88a-46f1-9799-f103e04c0b72"
DEFAULT_EMAIL="user@journal.local"
DEFAULT_NAME="Default User"

echo "🚀 Creating Default Journal User"
echo "================================="
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo "❌ Error: psql command not found"
    echo "Please install PostgreSQL client tools"
    exit 1
fi

# Function to run SQL
run_sql() {
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$1"
}

# Check if users table exists
if ! run_sql "SELECT to_regclass('public.users')" | grep -q users; then
    echo "❌ Error: users table does not exist"
    echo "Please run migrations first: make db-migrate-up"
    exit 1
fi

# Create the default user
echo "Creating default user with ID: $DEFAULT_USER_ID"
run_sql "INSERT INTO users (id, email, display_name) VALUES ('$DEFAULT_USER_ID', '$DEFAULT_EMAIL', '$DEFAULT_NAME') ON CONFLICT (id) DO NOTHING" > /dev/null

# Check if user was created or already exists
if run_sql "SELECT id FROM users WHERE id = '$DEFAULT_USER_ID'" | grep -q "$DEFAULT_USER_ID"; then
    echo "✅ Default user created/verified successfully"
    echo ""
    echo "User details:"
    echo "  ID: $DEFAULT_USER_ID"
    echo "  Email: $DEFAULT_EMAIL"
    echo "  Name: $DEFAULT_NAME"
else
    echo "❌ Failed to create default user"
    exit 1
fi

echo ""
echo "✅ Setup complete!"
