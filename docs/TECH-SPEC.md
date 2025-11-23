

````markdown
# Journal Web App — Technical Specification & Developer Guide

## Overview

A clean, minimal journal web app inspired by a physical notebook.

- **Frontend:** Vite + TypeScript + Quill  
- **Backend:** Go (Gin + pgx + sqlc)  
- **Database:** PostgreSQL (JSONB + `bytea`)  
- **Architecture:** Single-user (Phase 1) → Multi-user (later)  
- **Auto-save:** Debounced 2000 ms + on blur + beforeunload  
- **Display:** Saved HTML view with click-to-edit (Quill)  
- **Timezone:** `America/New_York` initially (user-selectable later)

---

## Architecture

### Components
| Layer | Technology | Purpose |
|-------|-------------|----------|
| **Frontend** | Vite + TypeScript + Quill | SPA journal UI |
| **Backend** | Go + Gin | REST API + static file serving |
| **Data Layer** | pgx + sqlc | Type-safe Postgres access |
| **Database** | PostgreSQL | Journal data storage |
| **Migrations** | golang-migrate | Schema version control |

### Data Flow

1. User edits entry → SPA debounces 2 s.
2. `PATCH /api/entries/:id` with new data.
3. Server normalizes, re-renders HTML, saves.
4. SPA refreshes the entry card on success.

---

## Data Model

### `users`
```sql
create table users (
  id            uuid primary key default gen_random_uuid(),
  email         citext unique not null,
  display_name  text,
  timezone      text not null default 'America/New_York',
  created_at    timestamptz not null default now()
);
````

### `entries`

```sql
create table entries (
  id                 uuid primary key default gen_random_uuid(),
  user_id            uuid not null references users(id) on delete cascade,
  title              text not null default '',
  body_delta         jsonb not null,
  body_html          text not null,
  render_version     int  not null default 1,
  attendees_original text not null default '',
  attendees          text[] not null default '{}',
  type               text not null check (type in ('meeting','notes','other')),
  day_year           int  not null,
  day_month          int  not null,
  day_day            int  not null,
  archived           boolean not null default false,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);
```

### `attachments`

```sql
create table attachments (
  id          uuid primary key default gen_random_uuid(),
  user_id     uuid not null references users(id) on delete cascade,
  entry_id    uuid not null references entries(id) on delete cascade,
  filename    text not null,
  mime_type   text not null,
  size_bytes  bigint not null,
  data        bytea not null,
  created_at  timestamptz not null default now()
);
```

### Indexes

```sql
create index entries_day_idx
  on entries (user_id, day_year, day_month, day_day, archived, created_at);
create index entries_created_desc_idx
  on entries (user_id, created_at desc);
```

---

## REST API

Base: `/api`

### Entries

| Method     | Endpoint                      | Description                         |
| ---------- | ----------------------------- | ----------------------------------- |
| **GET**    | `/days/:yyyy-:mm-:dd/entries` | List all entries for the given day. |
| **POST**   | `/entries`                    | Create new entry for a day.         |
| **PATCH**  | `/entries/:id`                | Update title/body/attendees/type.   |
| **DELETE** | `/entries/:id`                | Soft delete (`archived=true`).      |

**POST `/api/entries` example**

```json
{
  "title": "Standup Notes",
  "body_delta": { "ops": [ { "insert": "Yesterday: fixed bug #42\n" } ] },
  "attendees_original": "Alice, Bob",
  "type": "meeting",
  "date": "2025-11-20"
}
```

### Attachments

| Method     | Endpoint                   | Description     |
| ---------- | -------------------------- | --------------- |
| **POST**   | `/entries/:id/attachments` | Upload file(s). |
| **GET**    | `/attachments/:id`         | Retrieve file.  |
| **DELETE** | `/attachments/:id`         | Remove file.    |

### Calendar

**GET `/months/:yyyy-:mm/entry-days`**

```json
{ "daysWithEntries": [1, 5, 12, 19] }
```

---

## Rendering Logic

* `body_delta` is **canonical**; `body_html` is derived server-side.
* `render_version` allows re-rendering when HTML generator changes.
* Sanitize HTML before storing.
* Entries are **ordered oldest → newest** by `created_at`.
* **One active editor at a time.**
  Clicking another entry closes the previous editor.

---

## Time & Timezone

* User timezone → `year/month/day` fields.
* Timestamps (`created_at`, `updated_at`) stored as UTC.
* When fetching `/day/YYYY-MM-DD`, interpret that as the user’s **local day**.

---

## Autosave & Error Handling

* Debounce: 2000 ms.
* Also triggers on:

  * Input **blur**.
  * **Before unload** (tab close).
* Inline errors displayed via toast or border highlight.
* Retry automatically after next edit.

---

## Development Environment

### Directory Layout

```
journal/
├── cmd/server/main.go
├── api/
│   ├── handlers.go
│   └── routes.go
├── db/
│   ├── migrations/
│   ├── queries/
│   └── sqlc.yaml
├── web/
│   ├── index.html
│   ├── src/
│   └── dist/
└── go.mod
```

### Configuration

| Env Var            | Description                             |
| ------------------ | --------------------------------------- |
| `PORT`             | HTTP port                               |
| `DATABASE_URL`     | Postgres connection string              |
| `APP_ENV`          | `dev` or `prod`                         |
| `SPA_MODE`         | `fs` (dev) or `embed` (prod)            |
| `SPA_DIR`          | Path to SPA directory when in `fs` mode |
| `DEFAULT_TIMEZONE` | Default TZ for new users                |

### Serving the SPA

* **Dev:** Serve `/web/dist` directly from disk (auto-detect file changes).
* **Prod:** Embed static files with Go’s `embed` and serve via Gin.
* Use env var `SPA_MODE=fs|embed`.

---

## Developer Guide

### 1️⃣ Prerequisites

* Go ≥ 1.22
* Node ≥ 18
* PostgreSQL ≥ 14
* Tools: `sqlc`, `golang-migrate`, `air` (optional live reload)

### 2️⃣ Database Setup

```bash
createdb journal
migrate -path db/migrations -database "postgres://localhost/journal?sslmode=disable" up
```

### 3️⃣ Generating Type-Safe Queries

* Write SQL in `db/queries/*.sql`
* Generate Go code:

  ```bash
  sqlc generate
  ```

### 4️⃣ Running Locally

```bash
go run ./cmd/server
```

or with live reload:

```bash
air
```

SPA dev mode:

```bash
cd web
npm install
npm run dev
```

### 5️⃣ Building for Production

```bash
npm run build
go build -o journal ./cmd/server
```

Docker example (conceptual):

```dockerfile
FROM node:20 AS frontend
WORKDIR /app/web
COPY web/ .
RUN npm ci && npm run build

FROM golang:1.23 AS backend
WORKDIR /app
COPY . .
RUN go build -o journal ./cmd/server

FROM debian:bookworm-slim
COPY --from=backend /app/journal /usr/local/bin/
COPY --from=frontend /app/web/dist /app/web/dist
ENV SPA_MODE=embed
CMD ["journal"]
```

---

## Deployment

### Docker

* One container for Go app (serves API + SPA).
* Postgres as a second container.
* Volume mount or managed PG instance.

### Kubernetes

* Deployments: `journal-api`, `postgres`.
* Secrets: DB URL, optional future API keys.
* ConfigMap: timezone, SPA mode.
* Ingress:

  * `/api/*` → `journal-api`
  * `/*` → SPA.

---

## Roadmap

### Phase 1 (Now)

* Single-user.
* CRUD for entries, attachments.
* Full month calendar view.
* Auto-save, inline errors.
* Postgres schema + migration system.

### Phase 2

* Search (`tsvector` or JSONB GIN).
* Export to JSON/Markdown.
* Basic light/dark theme.

### Phase 3

* Multi-user authentication.
* Activity logging/audit.
* Rate limiting, CSRF.

### Phase 4

* User-configurable timezone.
* Unit and integration testing.

---

## Notes for Future Expansion

* Switch to object storage (S3/R2) for large attachments.
* Support image thumbnails and file previews.
* Add user preferences (theme, editor font size).
* Optional encrypted storage for sensitive entries.

---

## Summary

This design yields:

* Clean separation of backend and frontend.
* JSONB-based schema ready for expansion.
* Postgres performance and reliability.
* Minimal Go code with full type safety (`sqlc` + `pgx`).
* Simple deployment via Docker or Kubernetes.

The initial MVP remains simple yet forward-compatible with multi-user, search, and export features.


