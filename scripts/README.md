# Database Setup

This directory contains scripts to set up the PostgreSQL database for the Journal application.

## Prerequisites

- PostgreSQL 16+ with pgvector extension support
- For container setup: Docker or Podman

## Quick Start

### Option 1: Using Docker/Podman (Recommended)

This is the easiest way to get started:

```bash
# Make the script executable
chmod +x scripts/setup-database-container.sh

# Run the setup
./scripts/setup-database-container.sh
```

This will:
- Pull the `pgvector/pgvector:pg16` image
- Start a PostgreSQL container with pgvector pre-installed
- Create the database and user
- Install required extensions (vector, citext)
- Expose PostgreSQL on port 5432

### Option 2: Using Existing PostgreSQL

If you already have PostgreSQL installed:

```bash
# Make the script executable
chmod +x scripts/setup-database.sh

# Run the setup
./scripts/setup-database.sh
```

You may need to provide credentials via environment variables:

```bash
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=yourpassword
export DB_HOST=localhost
export DB_PORT=5432
./scripts/setup-database.sh
```

## Configuration

Both scripts support environment variables for customization:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_NAME` | `journal` | Database name |
| `DB_USER` | `journal` | Application database user |
| `DB_PASSWORD` | `journaldev` | Application user password |
| `DB_HOST` | `localhost` | Database host (setup-database.sh only) |
| `DB_PORT` | `5432` | Database port |
| `POSTGRES_USER` | `postgres` | PostgreSQL superuser |
| `POSTGRES_PASSWORD` | _(empty)_ | Superuser password |
| `CONTAINER_NAME` | `journal-postgres` | Container name (setup-database-container.sh only) |

Example with custom settings:

```bash
export DB_NAME=myjournal
export DB_USER=myuser
export DB_PASSWORD=securepassword
./scripts/setup-database-container.sh
```

## After Setup

Once the database is set up, you need to:

1. **Run migrations to create tables:**
   ```bash
   make db-migrate-up
   ```

2. **Create the default user:**
   ```bash
   ./scripts/create-default-user.sh
   ```

3. **Start the application:**
   ```bash
   make run
   ```

## Manual Setup (Advanced)

If you prefer to set up manually or need to customize:

1. **Create the database:**
   ```sql
   CREATE DATABASE journal;
   ```

2. **Create the user:**
   ```sql
   CREATE USER journal WITH PASSWORD 'journaldev';
   GRANT ALL PRIVILEGES ON DATABASE journal TO journal;
   ```

3. **Connect to the journal database and install extensions:**
   ```sql
   \c journal
   CREATE EXTENSION vector;
   CREATE EXTENSION citext;
   GRANT ALL ON SCHEMA public TO journal;
   ```

4. **Run migrations:**
   ```bash
   make db-migrate-up
   ```

## Troubleshooting

### pgvector not available

If you get "extension vector is not available":

- **Docker/Podman:** Make sure you're using `pgvector/pgvector:pg16` image
- **Existing PostgreSQL:** Install pgvector extension:
  ```bash
  # macOS
  brew install pgvector
  
  # Ubuntu/Debian
  sudo apt-get install postgresql-16-pgvector
  ```

### Permission denied

If you get "permission denied to create extension":

- You need to run the extension creation as a PostgreSQL superuser (usually `postgres`)
- The scripts handle this automatically, but if running manually:
  ```bash
  psql -U postgres -d journal -c "CREATE EXTENSION vector;"
  ```

### Connection refused

If the database won't connect:

- Check PostgreSQL is running: `pg_isready -h localhost -p 5432`
- For containers: `docker ps` or `podman ps`
- Check firewall settings
- Verify the port isn't already in use

### Migration fails with "dirty database"

If migrations fail partway:

```bash
# Force the version (replace X with the failed version number)
migrate -path db/migrations -database "your-connection-string" force X

# Then run migrations again
migrate -path db/migrations -database "your-connection-string" up
```

## Container Management

If using the container setup:

```bash
# Stop the database
podman stop journal-postgres

# Start the database
podman start journal-postgres

# View logs
podman logs journal-postgres

# Remove completely
podman rm -f journal-postgres
```

## Security Notes

The default password (`journaldev`) is only suitable for local development. For production:

1. Use a strong password
2. Don't use `POSTGRES_HOST_AUTH_METHOD=trust`
3. Enable SSL/TLS connections
4. Restrict network access
5. Consider using managed PostgreSQL services
