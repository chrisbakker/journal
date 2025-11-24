# Docker Deployment

This guide explains how to run the Journal application in a Docker container.

## Quick Start

```bash
# Build and start the container
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

The app will be available at http://localhost:8080

## Prerequisites

Before running the container, ensure you have:

1. **PostgreSQL with pgvector** running (on host or separate container)
2. **Ollama** running on host (for AI features)
3. **Database initialized** using `./scripts/setup-database.sh`

## Configuration

### Using docker-compose.yml

Edit the environment variables in `docker-compose.yml`:

```yaml
environment:
  # Point to your PostgreSQL database
  DATABASE_URL: postgresql://user:pass@host:5432/dbname?sslmode=disable
  
  # Ollama endpoint (use host.docker.internal for host services)
  OLLAMA_BASE_URL: http://host.docker.internal:11434
  
  # Server settings
  SERVER_PORT: "8080"
  SERVER_ENV: production
```

### Using config.yaml

Create a `config.yaml` file in the project root:

```yaml
server:
  port: "8080"
  env: production

database:
  url: postgresql://journal:journaldev@host.docker.internal:5432/journal?sslmode=disable

spa:
  mode: embed
  dir: /app/web/dist

llm:
  provider: ollama
  ollama_base_url: http://host.docker.internal:11434
  embedding_model: nomic-embed-text
  chat_model: llama3.2
  vector_dimensions: 768
  enable_vector_search: true
```

The compose file mounts this as a volume.

## Building the Image

### Using Docker Compose
```bash
docker compose build
```

### Manual Build
```bash
docker build -t journal:latest .
```

## Running Without Compose

```bash
docker run -d \
  --name journal \
  -p 8080:8080 \
  -e DATABASE_URL="postgresql://journal:journaldev@host.docker.internal:5432/journal?sslmode=disable" \
  -e OLLAMA_BASE_URL="http://host.docker.internal:11434" \
  -e SERVER_PORT="8080" \
  -v $(pwd)/config.yaml:/app/config.yaml \
  --add-host host.docker.internal:host-gateway \
  journal:latest
```

## Connecting to Host Services

The container needs to access services running on your host machine (PostgreSQL, Ollama).

### On Linux
Use `host.docker.internal:host-gateway` (added in docker-compose.yml):
```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

### On macOS/Windows
`host.docker.internal` works by default.

## Full Stack with Docker

If you want everything in containers:

```yaml
version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: journal
      POSTGRES_PASSWORD: journaldev
      POSTGRES_DB: journal
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U journal"]
      interval: 5s
      timeout: 5s
      retries: 5

  journal:
    build: .
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://journal:journaldev@postgres:5432/journal?sslmode=disable
      OLLAMA_BASE_URL: http://host.docker.internal:11434
      SERVER_PORT: "8080"
    volumes:
      - ./config.yaml:/app/config.yaml
    extra_hosts:
      - "host.docker.internal:host-gateway"

volumes:
  postgres_data:
```

Then run setup against the containerized database:
```bash
# Start database
docker compose up -d postgres

# Wait for it to be ready
docker compose exec postgres pg_isready -U journal

# Run setup script
export DB_HOST=localhost
./scripts/setup-database.sh

# Start the app
docker compose up -d journal
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | - | PostgreSQL connection string |
| `SERVER_PORT` | `8080` | Port to listen on |
| `SERVER_ENV` | `dev` | Environment (dev/production) |
| `SPA_MODE` | `embed` | Frontend mode (embed/filesystem) |
| `SPA_DIR` | `/app/web/dist` | Frontend files directory |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama API endpoint |
| `LLM_EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name |
| `LLM_CHAT_MODEL` | `llama3.2` | Chat model name |
| `LLM_VECTOR_DIMENSIONS` | `768` | Vector dimensions |
| `LLM_ENABLE_VECTOR_SEARCH` | `true` | Enable AI search |

## Troubleshooting

### Cannot connect to database
- Ensure PostgreSQL is running and accessible
- Check `DATABASE_URL` is correct
- Verify `host.docker.internal` resolves (try `docker.for.mac.localhost` on older Docker)

### Cannot connect to Ollama
- Ensure Ollama is running on host
- Verify it's listening on 0.0.0.0:11434 (not just 127.0.0.1)
- Check `host.docker.internal` is accessible

### Migrations don't run
- Migrations are not run automatically in the container
- Run `./scripts/setup-database.sh` before starting the container
- Or exec into container and run manually:
  ```bash
  docker compose exec journal sh
  # Inside container (if migrate is available)
  # Otherwise run from host pointing to database
  ```

### Config file not found
- Ensure `config.yaml` exists in project root
- Check volume mount in docker-compose.yml
- Verify file permissions

## Production Considerations

1. **Use secrets for sensitive data** (not environment variables)
2. **Enable SSL for database connections** (change `sslmode=disable`)
3. **Set up proper logging** and monitoring
4. **Use health checks** for orchestration
5. **Consider using a reverse proxy** (nginx, traefik)
6. **Back up your data** regularly
