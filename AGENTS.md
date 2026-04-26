# AGENTS.md

## Quick Start

```bash
# Start local infrastructure
docker-compose up -d

# Run api-server
cd apps/api-server && go run cmd/main.go

# Run tests (per-module, not from root due to go.work)
cd apps/api-server && go test ./...
cd apps/outbox-relay && go test ./...
```

## Architecture

- **Monorepo**: Go workspace (`go.work` with Go 1.26.2)
- **Apps**: `api-server`, `stt-worker`, `llm-worker`, `outbox-relay`, `infra-migration`
- **Packages**: `config`, `db`, `mq`, `redis`, `storage`, `stt`, `llm`, `telemetry`

## Dependency Injection

Uses **Google Wire**. After modifying any `wire.go`:
```bash
./scripts/sync-di.sh
```
Regenerates `wire_gen.go` files for all apps.

## Running Apps

Entry points in `apps/<app>/cmd/main.go`:
- `api-server`: `cd apps/api-server && go run cmd/main.go` (port 8080)
- `stt-worker`: `cd apps/stt-worker && go run cmd/main.go`
- `llm-worker`: `cd apps/llm-worker && go run cmd/main.go`
- `outbox-relay`: `cd apps/outbox-relay && go run cmd/main.go`

## Environment Configuration

- Config via `.env` in project root or each app directory
- ENV variable has three modes (defined in `packages/config/env.go`):
  - `EnvMock = "mock"` - 略過 MinIO/STT，使用 mock 資料
  - `EnvLocal = "local"` - 本地開發（預設）
  - `EnvProduction = "production"` - 正式環境

Infrastructure: Postgres (:5432), Redis (:6379), RabbitMQ (:5672), MinIO (:9000/:9001)

## Key Conventions

- Interfaces in `repository/` package, implementations in packages
- Use Wire skill (`go-wire-arch`) when adding repositories/usecases/handlers
- Use TDD skill (`go-tdd`) for test-driven development

## Skills

- `go-tdd`: TDD workflow for Go unit tests
- `go-wire-arch`: Wire DI injection for new components