# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Maille is a personal finance/accounting web application (inspired by Beancount) that imports bank transactions from OFX files into PostgreSQL. It parses OFX data, creates double-entry bookkeeping records (transactions + postings), and stores them via bulk `COPY FROM` operations.

## Build & Run Commands

```bash
# Build
go build -o ./tmp/server ./cmd

# Run tests
go test ./...

# Run a single test
go test ./internal/processor -run TestName

# Hot-reload development (via air)
air -c .air.toml
```

## Deployment (Podman-based, managed via mise)

```bash
# Full deploy: build container, create network, start postgres + app
mise run deploy

# Individual steps
mise run build              # podman build the dev container
mise run deploy-network     # create podman network
mise run deploy-postgresql  # start PostgreSQL pod
mise run deploy-pod         # start app pod
mise run set-secret SECRET  # configure database secrets
```

## Environment

- `DATABASE_URL` — PostgreSQL connection string. Local dev default: `postgres://notbeancount:toto@localhost:15542/notbeancount`
- App listens on port **13000**
- PostgreSQL exposed on host port **15542**
- Go **1.25**, PostgreSQL **18**

## Architecture

### Request Flow

`cmd/main.go` → `internal.Run()` → chi router → `web.Upload` handler → `processor.OFXParser.Parse()` → `processor.Processor.Process()` → PostgreSQL

### Key Packages

- **`cmd/`** — Entrypoint. Sets up slog JSON logger, pgxpool, gops agent, and calls `internal.Run()`.
- **`internal/`** — HTTP server setup (`web.go`). Configures chi router with max-bytes and slog middleware. Embeds HTML templates from `internal/template/` via `embed.FS`.
- **`internal/web/`** — HTTP handlers. `Upload` handles GET (render form) and POST (accept OFX file upload, parse, and process).
- **`internal/processor/`** — Core logic:
  - `OFXParser` — Custom streaming OFX parser using `bufio.Scanner` with a custom split function (`splitOnStmtTrn`). Extracts `<STMTTRN>` blocks and parses them into `Transaction` structs. Processes in configurable batch sizes.
  - `Processor` — Inserts transactions and postings into PostgreSQL using `pgx.CopyFrom` for bulk performance.
  - `Bid` — Time-sortable 20-char ID generator (Crockford-like base32, 8 chars timestamp + 12 chars random).

### Database Schema (inferred)

- **`currency`** — `id`, `name`
- **`transaction`** — `id`, `date`, `completed`, `payee`, `external_id`
- **`posting`** — `id`, `transaction_id`, `type` (ASSETS/INCOME/EXPENSES), `account`, `amount`, `currency_id`

### Libraries

- **chi** — HTTP router
- **pgx/pgxpool** — PostgreSQL driver with connection pooling
- **mold** — HTML template engine (embedded templates)
- **oops** — Structured error handling with context
- **slog-chi** — Structured logging middleware
- **gops** — Runtime diagnostics agent

### Development Tooling

- **air** — Hot-reload dev server (config in `.air.toml`, binary built to `tmp/server`)
- **mise** — Task runner and env config (`.mise/config.toml`)
- **podman** — Container runtime (Kubernetes-style YAML in `deploy/`)
