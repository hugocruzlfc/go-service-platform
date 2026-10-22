# AGENTS.md — Go Service Platform

## Project Overview

Build a professional backend service platform in Go as a long-term learning project and portfolio repository.

The developer is a senior Full Stack Engineer experienced in TypeScript, JavaScript, React, Next.js, Node.js, and NestJS. They are learning Go to build production-quality backend services, APIs, and microservices.

Teach Go by building a real project incrementally. Connect new concepts to TypeScript/NestJS when useful, but explain Go-specific behavior accurately.

## Primary Goals

- Learn Go fundamentals through practical implementation.
- Develop a professional, maintainable backend architecture.
- Understand packages, imports, interfaces, pointers, structs, methods, errors, and dependency injection.
- Build HTTP APIs with proper validation, error handling, and tests.
- Add PostgreSQL persistence, migrations, configuration, observability, Docker, and CI progressively.
- Produce a repository suitable for a senior backend / full-stack / AI engineering portfolio.

## Current Project

Module path:

`service-platform`

The repository was initially created with Go 1.27.2 in `go.mod`. Verify the actual installed Go version and project configuration before changing them.

## Current Architecture

The project is organized by module (`task`, `auth`) with layers inside each module:

- `cmd/api`: HTTP API entry point. Loads `.env`, wires dependencies, starts the server.
- `cmd/migrate`: applies pending SQL migrations. Loads `.env`.
- `internal/adapter/httpapi`: router, HTTP handlers, bearer-token middleware, error responses, OpenAPI and Swagger UI endpoints.
- `internal/api`: code generated from `api/openapi.yaml`. Do not edit by hand.
- `internal/config`: loads configuration from environment variables and validates it.
- `internal/migration`: migration runner and the SQL files in `sql/`, embedded into the binary.
- `internal/task/domain`, `internal/task/application`: task entity, validation, and service.
- `internal/task/infrastructure/postgres`: PostgreSQL task repository. `memory` is kept for unit tests.
- `internal/auth/domain`, `internal/auth/application`: users, credentials, sessions, login, registration, refresh, and logout.
- `internal/auth/infrastructure/{argon2,postgres,token}`: password hashing, PostgreSQL repositories, and Ed25519 JWT and refresh-token handling.
- `internal/auth/authentication.go`: wires the auth module and returns its use cases and token verifier. The HTTP routes for auth are generated from `api/openapi.yaml` and implemented in `internal/adapter/httpapi/openapi_auth.go`.

Use `README.md` for the current project structure, configuration variables, and commands.

This section describes the intended architecture. Inspect the repository before relying on a specific file name.

## Existing Functionality

### Domain

`domain.Task` contains:

- `ID string`
- `Title string`
- `Completed bool`
- `CreatedAt time.Time`

`domain.NewTask` validates the task ID and title, trims whitespace, rejects titles exceeding 120 Unicode characters, initializes a task as incomplete, and normalizes the timestamp to UTC.

Domain errors include:

- `ErrTaskIDRequired`
- `ErrTaskTitleRequired`
- `ErrTaskTitleTooLong`

### Application

`application.TaskRepository` defines `Create`, `Get`, `Update`, `Delete`, and `List`.

`application.TaskService` exposes `CreateTask`, `GetTask`, `UpdateTask`, `DeleteTask`, and `ListTasks`.

The service generates task IDs, constructs valid domain entities, delegates persistence to the repository, and wraps infrastructure errors with context.

Go interfaces are implemented implicitly. Keep interfaces small and defined according to the needs of the consuming layer.

### Persistence

- `postgres.TaskRepository` is the production repository, created from a `pgxpool.Pool`.
- `memory.TaskRepository` uses a `map[string]domain.Task` protected by `sync.RWMutex`. It is used by unit tests and loses data when the process exits.
- Migrations are SQL files in `internal/migration/sql`, applied by `cmd/migrate` and tracked in `schema_migrations`.

### Authentication

- `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, and `POST /auth/logout` are public.
- Login returns an Ed25519-signed JWT access token and a refresh token. The refresh token is also set as a `Secure`, `HttpOnly` cookie scoped to `/auth`.
- `httpapi.RequireAccessToken` protects `/tasks` and `/tasks/`. It requires `Authorization: Bearer <token>` and returns `401` for missing or invalid tokens. It does not yet pass the claims to the handlers.

### Configuration

`internal/config.Load` reads environment variables and validates them. `DATABASE_URL` and `AUTH_JWT_PRIVATE_KEY_BASE64` are required. `.env.example` lists every variable.

### HTTP API

Public endpoints:

- `GET /health`, `GET /healthz`, `GET /readyz`, `GET /version`
- `GET /openapi.json`, `GET /docs`
- `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`

Protected endpoints (require a bearer access token):

- `POST /tasks`, `GET /tasks`
- `GET /tasks/{id}`, `PATCH /tasks/{id}`, `DELETE /tasks/{id}`

`cmd/api` wires the concrete dependencies: the PostgreSQL pool, the task module, and the auth module, then creates the router.

The API uses `net/http` and Go's standard library. Do not introduce a web framework without a concrete reason.

## Architecture Principles

Maintain these dependency directions:

- HTTP adapter → application.
- Application → domain and the repository interface it requires.
- Infrastructure → application interface and domain types as needed.
- `cmd/api` → concrete implementations and application wiring.

The domain must not depend on HTTP, PostgreSQL, or infrastructure packages.

Keep the project simple and idiomatic. Do not add layers, interfaces, libraries, or abstractions without a clear benefit.

Prefer standard library functionality when appropriate.

## Engineering Standards

- Use idiomatic Go and `gofmt`.
- Handle errors explicitly; do not silently discard errors.
- Use `errors.Is` and `errors.As` where appropriate.
- Wrap errors with `%w` when adding useful context.
- Use `context.Context` for cancellable operations and propagate it through I/O.
- Protect shared mutable state from data races.
- Validate external input and return appropriate HTTP status codes.
- Avoid exposing internal errors or sensitive implementation details to clients.
- Write unit tests for domain rules and application behavior.
- Write HTTP tests using `net/http/httptest`.
- Use meaningful names and small functions.
- Avoid premature abstractions and unnecessary dependencies.
- Preserve working behavior unless a change is explicitly justified.

## Learning Methodology

The developer wants to understand the code, not merely receive a completed implementation.

For each meaningful step:

1. Briefly explain the objective and why it matters.
2. Inspect the current implementation before editing.
3. Make one coherent, incremental change.
4. Explain important Go syntax and architectural decisions.
5. Provide the commands needed to format, test, and run the project.
6. Verify the changes and explain any failures.
7. Summarize what was learned and identify the next logical step.

Connect concepts to TypeScript/NestJS when helpful, while explaining differences rather than assuming the languages behave identically.

Do not dump an entire production architecture at once. Introduce new concepts when the project needs them.

## Next Steps

First inspect the repository and establish the actual current state.

Then continue with:

1. Link tasks to the authenticated user. The access-token middleware validates the token but discards its claims, so tasks are not yet scoped to a user.
2. Describe the bearer security scheme and the `/auth` routes in `api/openapi.yaml`.
3. Resolve the govulncheck finding `GO-2026-6809` (fixed in Go 1.27.3). CI runs `make vuln`, so this fails until the toolchain is upgraded.
4. Add observability (metrics and tracing) and a Docker image for the API.

Do not assume the current code is correct merely because previous tests passed. Verify it before making architectural decisions.

## Working Agreement

Before making changes, summarize the discovered project state and identify the next smallest useful step.

Prefer targeted edits over rewriting entire files. Do not delete existing tests or working functionality without justification.

Run relevant checks after modifications:

- `gofmt`
- `go test ./...`
- `go vet ./...`
- `go test -race ./...` when relevant

If a command fails, investigate and fix the cause rather than reporting success.

The goal is to build a real, maintainable Go backend while developing a deep understanding of Go and professional backend engineering.
