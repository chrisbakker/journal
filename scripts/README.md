# Database Setup

This directory contains a script to set up the PostgreSQL database for the Journal application.

## Prerequisites

- PostgreSQL 16+ with pgvector extension support
- `golang-migrate` CLI tool for running migrations

## Quick Start

The setup script handles everything in one go:

```bash
# Make the script executable
chmod +x scripts/setup-database.sh

# Run the setup (will prompt for postgres password if needed)
./scripts/setup-database.sh
```

The script will:
1. Create the database and user
2. Grant necessary privileges
3. Install required extensions (vector, citext)
4. Run all migrations
5. Create the default user

If you don't have `golang-migrate` installed, you can run migrations manually after the initial setup:

```bash
make db-migrate-up
```

## Configuration

The setup script supports environment variables for customization:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_NAME` | `journal` | Database name |
| `DB_USER` | `journal` | Application database user |
| `DB_PASSWORD` | `journaldev` | Application user password |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `POSTGRES_USER` | `postgres` | PostgreSQL superuser |
| `POSTGRES_PASSWORD` | _(prompts if empty)_ | Superuser password |

Example with custom settings:

```bash
export DB_NAME=myjournal
export DB_USER=myuser
export DB_PASSWORD=securepassword
export POSTGRES_PASSWORD=mypostgrespass
./scripts/setup-database.sh
```

If `POSTGRES_PASSWORD` is not set, the script will prompt you once at the beginning.

## After Setup

The setup script automatically runs migrations and creates the default user. After it completes, you can:

**Start the application:**
```bash
make run
```

Then open http://localhost:8080 in your browser.

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

5. **Create the default user:**
   ```sql
   INSERT INTO users (id, email, display_name) 
   VALUES ('02a0aa58-b88a-46f1-9799-f103e04c0b72', 'user@journal.local', 'Default User');
   ```

## Troubleshooting

### pgvector not available

If you get "extension vector is not available":

- Install pgvector extension:
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

## Security Notes

The default password (`journaldev`) is only suitable for local development. For production:

1. Use a strong password
2. Don't use `POSTGRES_HOST_AUTH_METHOD=trust`
3. Enable SSL/TLS connections
4. Restrict network access
5. Consider using managed PostgreSQL services
