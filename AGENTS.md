# AGENTS.md

## Quick Start

```bash
# Start local infrastructure
docker-compose up -d

# Run api-server
cd apps/api-server && go run cmd/main.go

# Run tests
go test ./...
```

## Architecture

- **Monorepo**: Go workspace (`go.work` with Go 1.26.2)
- **Apps**: `api-server`, `stt-worker`, `llm-worker`, `outbox-relay`, `db-migration`
- **Packages**: `config`, `db`, `mq`, `storage`, `stt`

## Dependency Injection

Uses **Google Wire**. After modifying any `wire.go`:
```bash
./scripts/sync-di.sh
# Or manually: cd apps/<app> && wire
```
Regenerates `wire_gen.go` files.

## Running Apps

Entry points are in `apps/<app>/cmd/main.go`:
- `api-server`: `cd apps/api-server && go run cmd/main.go` (port 8080)
- Workers: `cd apps/<worker> && go run cmd/main.go`

## Tests

Run tests from workspace root or per-module:
```bash
go test ./apps/api-server/...
go test ./apps/outbox-relay/...
```

## Environment

- Config via `.env` (values: DB, Redis, RabbitMQ, MinIO/S3)
- Infrastructure: Postgres (:5432), Redis (:6379), RabbitMQ (:5672), MinIO (:9000/:9001)
- MinIO console: http://localhost:9001 (minioadmin/minioadmin)

## Key Conventions

- Interfaces defined in `repository/` package, implementations in packages
- Use Wire skill (`go-wire-arch`) when adding repositories/usecases/handlers
- Use TDD skill (`go-tdd`) for test-driven development

## Skills

- `go-tdd`: TDD workflow for unit tests
- `go-wire-arch`: Wire DI injection for new components