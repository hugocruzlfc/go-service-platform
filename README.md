# Go Service Platform

Backend service platform built incrementally with Go.

## Prerequisites

- Go 1.27.2
- Docker with Docker Compose

## Development workflow

### 1. Start PostgreSQL

The local development database runs in Docker Compose. Start it before running the API:

```bash
docker compose up -d postgres
```

Check the container and its health status:

```bash
docker compose ps
```

Wait until the `postgres` service shows `healthy`. To follow its logs:

```bash
docker compose logs -f postgres
```

The local connection details are:

```text
Host:     localhost
Port:     5432
Database: service_platform
User:     service_platform
Password: service_platform_dev
```

The database data is stored in the named Docker volume `service-platform_postgres_data` and survives container restarts.

### 2. Start the API

The API connects to PostgreSQL and executes pending migrations automatically during startup:

```bash
go run ./cmd/api
```

Keep this terminal open. Stop the API with `Ctrl+C`; it will perform a graceful shutdown.

By default it listens on `:8080`. To use another port:

```bash
PORT=9090 go run ./cmd/api
```

The application stores applied migration versions in the `schema_migrations` table and only executes pending files.

### 3. Test the API

In another terminal:

```bash
curl --fail --silent http://localhost:8080/health
curl --fail --silent http://localhost:8080/version
curl --fail --silent http://localhost:8080/tasks
```

Create a task:

```bash
curl --fail --silent \
  -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Learn Go with PostgreSQL"}'
```

List tasks again:

```bash
curl --fail --silent http://localhost:8080/tasks
```

### 4. Run tests

The normal test suite does not require Docker:

```bash
go test ./...
go test -race ./...
go vet ./...
```

The PostgreSQL integration test is opt-in and requires the Docker database to be running:

```bash
TEST_DATABASE_URL='postgres://service_platform:service_platform_dev@localhost:5432/service_platform?sslmode=disable' \
  go test -count=1 ./internal/task/infrastructure/postgres
```

## Database migrations

Migration files are stored in `internal/migration/sql` and are embedded into the application binary with Go's `embed` package.

When the API starts, the migration runner:

1. Creates the `schema_migrations` table if it does not exist.
2. Finds migration files in filename order.
3. Skips migrations already recorded in `schema_migrations`.
4. Runs each pending migration inside a transaction.
5. Records the migration version after a successful commit.

To add a migration, create a new SQL file using the next ordered version, for example:

```text
internal/migration/sql/002_add_task_priority.sql
```

Pending migrations are applied automatically the next time the API starts. Do not edit a migration that has already been applied; create a new migration instead.

## Stop PostgreSQL

Stop the container while preserving its data:

```bash
docker compose down
```

To remove the local database volume as well, use:

```bash
docker compose down -v
```

This removes local development data and should not be used casually.
