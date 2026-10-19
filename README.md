# Go Service Platform

Backend service platform built incrementally with Go.

## Prerequisites

- Go 1.27.2
- Docker with Docker Compose

## Start PostgreSQL

The local development database runs in Docker Compose. Start it before running database migrations or connecting the API to PostgreSQL:

```bash
docker compose up -d postgres
```

Check the container and its health status:

```bash
docker compose ps
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

## Run the first migration

Once PostgreSQL is healthy, execute the initial migration:

```bash
docker compose exec -T postgres \
  psql -U service_platform -d service_platform \
  < migrations/001_create_tasks.sql
```

The current command is intentionally explicit while we learn the migration workflow. A versioned migration runner will be added before introducing more migrations.

## Run the API

The API currently uses the in-memory repository. Start it with:

```bash
go run ./cmd/api
```

By default it listens on `:8080`. To use another port:

```bash
PORT=9090 go run ./cmd/api
```

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

## Verify the project

```bash
gofmt -l .
go test ./...
go test -race ./...
go vet ./...
```
