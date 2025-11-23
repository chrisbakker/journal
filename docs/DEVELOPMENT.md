# 📓 Journal Web App

A clean, minimal digital journal designed to feel like writing in a physical notebook — quiet, private, and free from clutter.

![Status](https://img.shields.io/badge/status-phase%203%20complete-success)
![Go](https://img.shields.io/badge/go-1.22+-blue)
![TypeScript](https://img.shields.io/badge/typescript-5.0+-blue)
![PostgreSQL](https://img.shields.io/badge/postgresql-16+-blue)
![pgvector](https://img.shields.io/badge/pgvector-0.6.0-blue)

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Node.js 18+
- Podman (or Docker)
- golang-migrate: `brew install golang-migrate`
- sqlc: `brew install sqlc` or `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### One-Command Setup
```bash
./dev.sh setup
```

This will:
- ✅ Start PostgreSQL in a container
- ✅ Run database migrations
- ✅ Create a default test user
- ✅ Generate type-safe query code
- ✅ Install frontend dependencies
- ✅ Download Go dependencies

### Configuration

The application requires configuration before first use. You have two options:

#### Option 1: Web-Based Setup Wizard (Recommended for First-Time Users)

1. Start the application: `make run`
2. Open your browser to `http://localhost:8080`
3. Fill out the configuration form with your database and AI settings
4. Click "Save Configuration"
5. The application will restart automatically

#### Option 2: Manual Configuration

Copy the example configuration file and edit it:

```bash
cp .env.example .env
# Edit .env with your settings
```

**Key Configuration Options:**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `DATABASE_URL` | `postgresql://journal:journaldev@localhost:5432/journal?sslmode=disable` | **Yes** | PostgreSQL connection string |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Yes (for AI) | Ollama server URL |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Yes (for AI) | Embedding model name |
| `CHAT_MODEL` | `llama3.2` | Yes (for AI) | Chat model name |
| `PORT` | `8080` | No | Server port |
| `APP_ENV` | `development` | No | Environment (`development` or `prod`) |
| `ENABLE_VECTOR_SEARCH` | `true` | No | Enable AI-powered search and chat |

See [CONFIGURATION.md](CONFIGURATION.md) for detailed configuration documentation.

### Run the Application

**Development:**
```bash
make run
```

**Production (Standalone Binary):**
```bash
make build
./bin/server
```

**Visit:** http://localhost:8080

The application automatically reloads configuration internally when you save changes via the web UI - no restart needed!

See [INTERNAL-RELOAD.md](INTERNAL-RELOAD.md) for technical details on how configuration reload works.

## ✨ Features

### Phase 1: Core Journal Functionality
- **📅 Calendar-Based Navigation**: Organize by days, not folders
- **✍️ Rich Text Editor**: Powered by Quill with auto-save
- **💾 Auto-Save**: Changes save every 2 seconds automatically
- **📝 Entry Types**: Notes, meetings, or other
- **👥 Attendees**: Track who was in meetings
- **🔄 Soft Deletes**: Recovery from accidental deletions
- **📎 Attachments**: Store files and images
- **🎨 Clean UI**: Minimal, distraction-free interface

### Phase 2: Search & Navigation
- **🔍 Full-Text Search**: Search across titles, body content, and attendees
- **🎯 Smart Navigation**: Thin right panel with journal/search icons
- **⚡ Real-Time Results**: Debounced search with instant feedback
- **📊 Result Counts**: See how many entries match your query
- **🔄 Seamless Switching**: Toggle between calendar and search views

## 🏗️ Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Vite + TypeScript + Quill |
| Backend | Go + Gin framework |
| Database | PostgreSQL 16 |
| ORM | sqlc (type-safe code generation) |
| Container | Podman/Docker |

## 📁 Project Structure

```
journal/
├── cmd/server/          # Main application entry point
├── api/                 # HTTP handlers and routes
├── db/
│   ├── migrations/      # Database schema versions
│   ├── queries/         # SQL queries for sqlc
│   └── sqlc.yaml        # sqlc configuration
├── generated/           # Auto-generated database code
├── web/                 # Frontend application
│   ├── src/
│   │   ├── main.ts      # Main TypeScript app
│   │   └── style.css    # Styles
│   └── dist/            # Built frontend (git-ignored)
├── dev.sh               # Development helper script
├── go.mod               # Go dependencies
└── package.json         # Frontend dependencies
```

## 🛠️ Development Commands

The `dev.sh` script provides helpful commands:

### Database
```bash
./dev.sh db:start         # Start PostgreSQL container
./dev.sh db:stop          # Stop PostgreSQL container
./dev.sh db:migrate:up    # Run migrations
./dev.sh db:shell         # Open psql shell
```

### Frontend
```bash
./dev.sh web:dev          # Start Vite dev server (port 5173)
./dev.sh web:build        # Build for production
```

### Backend
```bash
./dev.sh server:dev       # Start Go server (port 8080)
./dev.sh server:build     # Build binary to bin/journal
```

### Code Generation
```bash
./dev.sh sqlc:generate    # Regenerate database code
```

### Testing
```bash
./dev.sh test:api         # Test API endpoints
```

## 🧪 API Examples

### Create an Entry
```bash
curl -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Entry",
    "body_delta": {"ops": [{"insert": "Hello World!\n"}]},
    "type": "notes",
    "date": "2025-11-20"
  }'
```

### List Entries for a Day
```bash
curl http://localhost:8080/api/days/2025-11-20/entries
```

### Update an Entry
```bash
curl -X PATCH http://localhost:8080/api/entries/{id} \
  -H "Content-Type: application/json" \
  -d '{"title": "Updated Title"}'
```

### Get Calendar Days with Entries
```bash
curl http://localhost:8080/api/months/2025-11/entry-days
```

## 🗄️ Database

### Connection Details
```
Host: localhost
Port: 5432
Database: journal
User: journal
Password: journaldev
```

### Schema
- **users**: User accounts with timezone settings
- **entries**: Journal entries with rich text content
- **attachments**: Files associated with entries

### Access Database
```bash
# Via dev script
./dev.sh db:shell

# Or directly
podman exec -it journal-postgres psql -U journal -d journal
```

## 📊 Current Implementation Status

### ✅ Phase 1 (Complete)
- [x] Single-user mode
- [x] CRUD for entries
- [x] CRUD for attachments
- [x] Calendar navigation
- [x] Auto-save (2s debounce)
- [x] Rich text editing (Quill)
- [x] Entry types (notes/meeting/other)
- [x] Attendees support
- [x] Soft deletes
- [x] PostgreSQL storage
- [x] Type-safe queries (sqlc)

### 🔮 Future Phases

#### Phase 2: Search & Navigation ✅
- [x] Full-text search across titles, body content, and attendees
- [x] Right navigation pane with journal/search icons
- [x] Search panel replacing calendar when activated
- [x] Real-time search with debouncing (300ms)
- [x] Results displayed in main entries pane
- [x] Smooth view switching between calendar and search modes

#### Phase 3: AI-Enhanced Search with RAG ✅
**Intelligent entry retrieval using vector embeddings and semantic search**

##### Features:
- **🤖 Chat Interface with RAG**: 
  - Natural language chat with your journal entries
  - AI assistant powered by Ollama (llama3.2)
  - Context-aware responses using semantic search
  - Chat icon in right navigation panel
  
- **📊 Vector Embeddings**:
  - Automated background service generates embeddings for all entries
  - Uses nomic-embed-text model (768 dimensions)
  - Tracks update status with `vectors_updated_at` timestamp
  - Incremental updates only for new/modified entries

- **🔍 Semantic Search**:
  - Natural language queries beyond keyword matching
  - Vector similarity search using pgvector's cosine distance
  - Returns top 5 most relevant entries as context
  - Hybrid approach combining RAG with LLM chat

- **🔄 Background Vector Processing**:
  - Automated background service runs based on configured interval
  - Batch processing of entries needing embeddings
  - Mutex-protected for thread safety
  - Graceful error handling and logging

##### Configuration:
| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `ollama` | LLM provider (`ollama`) |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama API endpoint |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Model for generating embeddings |
| `CHAT_MODEL` | `llama3.2` | Model for chat completions |
| `VECTOR_DIMENSIONS` | `768` | Embedding vector dimensions |
| `VECTOR_UPDATE_INTERVAL` | `5` | Background job interval (minutes) |
| `ENABLE_VECTOR_SEARCH` | `true` | Enable/disable RAG features |

##### Prerequisites:
```bash
# Install Ollama (macOS)
brew install ollama
ollama serve

# Pull required models
ollama pull nomic-embed-text
ollama pull llama3.2
```

##### Technical Implementation:
- PostgreSQL pgvector extension for efficient vector storage (ivfflat index)
- Background worker using Go's time.Ticker with mutex locking
- Incremental updates tracking via `vectors_updated_at` timestamp
- Batch processing (10 entries per cycle) for efficiency
- HTML tag stripping for clean text embeddings
- RAG pipeline: query embedding → vector search → context injection → LLM response
- PostgreSQL pgvector extension for efficient vector storage
- Background worker using Go's time.Ticker with mutex locking
- Incremental updates tracking via `vectors_updated_at` timestamp
- Batch processing for efficiency with configurable batch sizes

#### Phase 4: Multi-user & Advanced Features
- Multi-user authentication and authorization
- User timezone preferences
- Export functionality (PDF, Markdown)
- Theming support
- Comprehensive testing suite

## 🐛 Troubleshooting

### Database won't start
```bash
podman rm -f journal-postgres
./dev.sh db:start
./dev.sh db:migrate:up
```

### Frontend not loading
```bash
cd web
npm install
npm run build
```

### Go compilation errors
```bash
go mod tidy
./dev.sh sqlc:generate
```

## 📚 Documentation

- [Product Overview](README-PRODUCT.md) - Original product vision
- [Technical Spec](tech-spec.md) - Detailed technical design
- [Implementation Guide](IMPLEMENTATION.md) - Phase 1 completion report

## 🤝 Contributing

This is a personal project implementation of the design specified in the technical documentation.

## 📝 License

MIT License - See LICENSE file for details

## 🎯 Design Philosophy

> "The Journal Web App emphasizes simplicity, clarity, and evolvability. Start with a single user; scale to multi-user without schema change. Prefer plain SQL + type-safe code generation over ORMs. Keep the UI minimal. Serve everything from a single Go binary. Optimize for reliability and low maintenance rather than features."

---

Built with ❤️ using Go, TypeScript, and PostgreSQL

