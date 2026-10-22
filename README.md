# Go Service Platform

Backend service platform built incrementally with Go.

## Prerequisites

- Go 1.27.2
- Docker with Docker Compose

## Project structure

```text
cmd/
  api/                 Entry point of the HTTP API; wires dependencies and loads .env
  migrate/             Entry point that applies pending database migrations
internal/
  adapter/httpapi/     HTTP router, handlers, and Swagger UI/OpenAPI endpoints
  api/                 Code generated from api/openapi.yaml (do not edit)
  auth/                Authentication: domain, application, and infrastructure
                       (argon2 password hashing, PostgreSQL, JWT tokens)
  config/              Loads configuration from environment variables
  migration/           Migration runner and the embedded SQL files in sql/
  task/                Task module: domain, application, and infrastructure
                       (in-memory and PostgreSQL repositories)
api/openapi.yaml       OpenAPI contract, source of truth for the generated code
```

Each module follows the same layers. `domain` holds the entities and rules,
`application` holds the use cases and the interfaces they need, and
`infrastructure` implements those interfaces with concrete technology.

## Configuration

The API reads its configuration from environment variables. For local
development, put them in a `.env` file in the project root. `cmd/api` loads
that file automatically at startup. Variables already set in the shell take
precedence over the file. The file is ignored by git and must not be committed.

Required for the API:

```bash
echo "AUTH_JWT_PRIVATE_KEY_BASE64=$(openssl genpkey -algorithm ed25519 -outform DER | tail -c 32 | base64)" > .env
```

The command above overwrites `.env`. To add more variables later, edit the file
instead of running it again.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `AUTH_JWT_PRIVATE_KEY_BASE64` | Yes, for `cmd/api` | none | Ed25519 key in base64: a 32-byte seed or a 64-byte private key. Used to sign access tokens. |
| `AUTH_JWT_ISSUER` | No | `service-platform` | Value of the `iss` claim in access tokens. |
| `AUTH_JWT_AUDIENCE` | No | `service-platform-api` | Value of the `aud` claim in access tokens. |
| `AUTH_ACCESS_TOKEN_TTL` | No | `15m` | Access token lifetime as a positive Go duration, for example `10m`. |
| `DATABASE_URL` | No | local Docker Compose database | PostgreSQL URL. Must use the `postgres` or `postgresql` scheme. |
| `PORT` | No | `8080` | HTTP port the API listens on. |
| `TEST_DATABASE_URL` | Only for integration tests | none | PostgreSQL URL used by the `-tags=integration` tests. |

`cmd/migrate` does not read `.env`. It uses `DATABASE_URL` from the shell, or
the local default when it is not set. With the Docker Compose database running,
the default is correct and you do not need to set it.

## Go module and dependency commands

This project uses Go Modules. The module definition is stored in `go.mod`,
while `go.sum` stores checksums for downloaded modules. Commit both files.

### Clone and initialize the repository

```bash
git clone <repository-url>
cd service-platform
go version
go mod download
```

`go mod download` downloads the modules declared by `go.mod`. The Go toolchain
also downloads missing modules automatically when running commands such as
`go build`, `go test`, or `go run`.

### Add an application dependency

```bash
go get github.com/example/library@v1.2.3
go mod tidy
```

Import the package from its module path in Go code, then run `go mod tidy` to
remove unused modules and add any required modules discovered from imports.
Avoid editing `go.mod` or `go.sum` manually.

### Update an existing dependency

```bash
go get github.com/example/library@v1.3.0
go mod tidy
```

Review the resulting `go.mod`, `go.sum`, and test results before committing.

### Install or run Go tools

Project tools should be registered in `go.mod` when possible. This repository
registers `govulncheck` as a tool dependency:

```bash
go get -tool golang.org/x/vuln/cmd/govulncheck@v1.9.0
go tool govulncheck ./...
```

For a standalone executable that is not managed by this repository:

```bash
go install github.com/example/tool/cmd/tool@v1.2.3
```

`go install` installs a binary for the local developer. `go get` changes the
project module requirements, while `go get -tool` records a project tool.

### Useful module inspection commands

```bash
go list -m all
go list -m -u all
go mod graph
go mod why -m github.com/example/library
go mod tidy
```

- `go list -m all` lists the module dependency graph.
- `go list -m -u all` shows available updates.
- `go mod graph` prints module relationships.
- `go mod why -m` explains why a module is required.
- `go mod tidy` synchronizes `go.mod` and `go.sum` with the source code.

## OpenAPI contract

The API contract is defined in [`api/openapi.yaml`](api/openapi.yaml). It is
the source of truth for generated Go models, HTTP server interfaces, and the
client used by contract tests.

Regenerate the checked-in API code after changing the specification:

```bash
go generate ./internal/api
```

The generator is pinned as a Go tool in `go.mod`, so the same version is used
locally and in CI. Generated files must not be edited manually.

To inspect the contract directly:

```bash
sed -n '1,260p' api/openapi.yaml
```

The running API exposes the generated contract at `/openapi.json` and the
interactive Swagger UI at `/docs` after the service is started. The YAML
remains the checked-in design contract.

### Useful local development commands

```bash
gofmt -w .
go build ./...
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/api
go run ./cmd/migrate
```

The project `Makefile` provides equivalent grouped commands. Run `make ci` to
execute the local checks used by GitHub Actions.

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

### 2. Run database migrations

Run migrations as a separate process before starting the API:

```bash
go run ./cmd/migrate
```

The migration command records applied versions in the `schema_migrations` table and only executes pending files.

### 3. Start the API

Make sure the `.env` file exists with `AUTH_JWT_PRIVATE_KEY_BASE64` set (see
[Configuration](#configuration)). Without it the API stops at startup.

The API connects to PostgreSQL and starts the HTTP server. It does not run database migrations:

```bash
go run ./cmd/api
```

Keep this terminal open. Stop the API with `Ctrl+C`; it will first mark the
service as not ready, wait briefly for traffic to drain, and then perform a
graceful shutdown.

By default it listens on `:8080`. To use another port:

```bash
PORT=9090 go run ./cmd/api
```

### 4. Test the API

In another terminal:

```bash
curl --fail --silent http://localhost:8080/health
curl --fail --silent http://localhost:8080/healthz
curl --fail --silent http://localhost:8080/readyz
curl --fail --silent http://localhost:8080/version
curl --fail --silent http://localhost:8080/tasks
```

Open the interactive API documentation in a browser:

```text
http://localhost:8080/docs
```

Swagger UI loads the generated OpenAPI document from
`http://localhost:8080/openapi.json`. The documentation page uses the Swagger
UI assets from a CDN, so the browser needs internet access to render the UI.
The raw contract remains available in `api/openapi.yaml` and is embedded into
the generated Go package.

`/health` and `/healthz` are liveness checks. They confirm that the process is
running and do not depend on PostgreSQL. `/readyz` is a readiness check: it
returns `200` when PostgreSQL is reachable and `503` when the service should
not receive traffic.

### Authenticate

Register a user, then log in to get an access token and a refresh token:

```bash
curl --fail --silent \
  -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@example.com","password":"correct-horse-battery-staple"}'

curl --fail --silent \
  -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@example.com","password":"correct-horse-battery-staple"}'
```

Registration returns `201` with the created user. Login returns `200` with
`accessToken`, `refreshToken`, and `expiresAt`. The refresh token is also set
as an `HttpOnly` cookie scoped to `/auth`.

The cookie is marked `Secure`, so clients on plain `http://localhost` do not
send it back. The examples below send the refresh token in the JSON body
instead. Replace `<refresh-token>` with the value returned by login:

```bash
curl --fail --silent \
  -X POST http://localhost:8080/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refreshToken":"<refresh-token>"}'

curl --fail --silent --include \
  -X POST http://localhost:8080/auth/logout \
  -H 'Content-Type: application/json' \
  -d '{"refreshToken":"<refresh-token>"}'
```

Refresh returns `200` with a new token pair. Logout returns `204`. The task
endpoints below do not require the access token yet.

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

Get, update, and delete a task using its returned ID:

```bash
curl --fail --silent http://localhost:8080/tasks/<task-id>

curl --fail --silent \
  -X PATCH http://localhost:8080/tasks/<task-id> \
  -H 'Content-Type: application/json' \
  -d '{"completed":true}'

curl --fail --silent --request DELETE \
  http://localhost:8080/tasks/<task-id>
```

### 5. Run tests

The normal test suite does not require Docker:

```bash
go test ./...
go test -race ./...
go vet ./...
```

The PostgreSQL integration tests are opt-in and require the Docker database to be running:

```bash
TEST_DATABASE_URL='postgres://service_platform:service_platform_dev@localhost:5432/service_platform?sslmode=disable' \
  go test -tags=integration -count=1 ./...
```

### 6. Run the CI checks locally

The `Makefile` provides the same test and lint commands used by the pull request CI:

```bash
make ci
```

Individual checks are also available:

```bash
make fmt
make build
make vet
make test-race
make lint
make vuln
make integration
```

`make lint` requires `golangci-lint`, and `make vuln` requires the pinned
`govulncheck` Go tool. The integration target requires PostgreSQL and the
`TEST_DATABASE_URL` environment variable.

## Database migrations

Migration files are stored in `internal/migration/sql` and are embedded into the application binary with Go's `embed` package.

When `cmd/migrate` runs, the migration runner:

1. Creates the `schema_migrations` table if it does not exist.
2. Finds migration files in filename order.
3. Skips migrations already recorded in `schema_migrations`.
4. Runs each pending migration inside a transaction.
5. Records the migration version after a successful commit.

To add a migration, create a new SQL file using the next ordered version, for example:

```text
internal/migration/sql/002_add_task_priority.sql
```

Run `go run ./cmd/migrate` after adding a migration. Do not edit a migration that has already been applied; create a new migration instead.

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
