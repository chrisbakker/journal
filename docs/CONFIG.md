# Configuration Guide

## Overview

The Journal application uses a flexible configuration system that supports multiple sources with a clear precedence order. This allows you to configure the application using a YAML file, environment variables, or a combination of both.

## Configuration Precedence

Configuration values are loaded in the following priority order (highest to lowest):

1. **Environment Variables** (highest priority - overrides everything)
2. **YAML Configuration File** (config.yaml)
3. **Default Values** (lowest priority)

This means that if you set a value in both the config file and as an environment variable, the environment variable will be used.

## Configuration File

By default, the application looks for `config.yaml` in the root directory. You can create this file by copying the example:

```bash
cp config.example.yaml config.yaml
```

### Example Configuration

```yaml
server:
  port: 8080
  environment: development

database:
  url: "postgresql://journal:journaldev@localhost:5432/journal"

llm:
  ollama_base_url: "http://localhost:11434"
  embedding_model: "nomic-embed-text"
  chat_model: "llama3.2"

features:
  enable_vector_search: true
```

## Environment Variables

The following environment variables can be used to override configuration file values:

| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `PORT` | Server port | `8080` |
| `APP_ENV` | Application environment | `production`, `development` |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:pass@host:5432/dbname` |
| `OLLAMA_BASE_URL` | Ollama API base URL | `http://localhost:11434` |
| `EMBEDDING_MODEL` | Model for text embeddings | `nomic-embed-text` |
| `CHAT_MODEL` | Model for chat/completion | `llama3.2` |
| `ENABLE_VECTOR_SEARCH` | Enable/disable vector search | `true`, `false` |

## Docker Deployment

When running in Docker using docker-compose, environment variables are the recommended way to configure the application. The docker-compose.yml file defines all necessary environment variables:

```yaml
services:
  journal:
    environment:
      - PORT=8080
      - APP_ENV=production
      - DATABASE_URL=postgresql://journal:journaldev@host.docker.internal:5432/journal
      - OLLAMA_BASE_URL=http://host.docker.internal:11434
      - EMBEDDING_MODEL=nomic-embed-text
      - CHAT_MODEL=llama3.2
      - ENABLE_VECTOR_SEARCH=true
```

These environment variables will **override** any values in a mounted config.yaml file.

### Optional Config File Mount

You can optionally mount a config.yaml file into the container, but environment variables will still take precedence:

```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro
```

## Local Development

For local development, you have two options:

### Option 1: Use config.yaml only

Create a `config.yaml` file with your local settings and run:

```bash
make run
```

### Option 2: Use environment variables

Set environment variables before running:

```bash
export DATABASE_URL="postgresql://journal:journaldev@localhost:5432/journal"
export OLLAMA_BASE_URL="http://localhost:11434"
make run
```

### Option 3: Combine both

Use config.yaml for base settings and override specific values with environment variables:

```bash
export APP_ENV=development
export PORT=3000
make run
```

## Configuration Loading Process

The application loads configuration using the following process:

1. Initialize default values
2. Look for and load `config.yaml` (if exists)
3. Check for environment variables and override loaded values
4. Validate the final configuration

This is implemented in `config/config.go`:

```go
func Load() *Config {
    // Load from config file first
    cfg := LoadFromFile(configPath)
    
    // Override with environment variables (highest priority)
    if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
        cfg.Database.URL = envDB
    }
    if envOllama := os.Getenv("OLLAMA_BASE_URL"); envOllama != "" {
        cfg.LLM.OllamaBaseURL = envOllama
    }
    // ... more environment variable checks
    
    return cfg
}
```

## Best Practices

1. **Local Development**: Use `config.yaml` for convenience and to keep sensitive data out of version control
2. **Docker Deployment**: Use environment variables in `docker-compose.yml` for clear, explicit configuration
3. **Secrets**: Never commit sensitive values to version control. Use environment variables or secret management tools
4. **Environment-Specific Settings**: Use `APP_ENV` to distinguish between `development`, `staging`, and `production`
5. **Database Connections**: For Docker deployments, use `host.docker.internal` to connect to services on the host machine

## Validation

The configuration is validated on startup. If required values are missing or invalid, the application will fail to start with a descriptive error message.

See `config/validator.go` for validation rules.
