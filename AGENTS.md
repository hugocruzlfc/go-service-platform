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

The project has been developed incrementally using these layers:

- `cmd/api`: application entry point and dependency wiring.
- `internal/adapter/httpapi`: HTTP routing and handlers.
- `internal/task/domain`: task entity, construction, and domain validation.
- `internal/task/application`: TaskService and TaskRepository interface.
- `internal/task/infrastructure/memory`: in-memory repository implementation.

Expected files include:

- `cmd/api/main.go`
- `internal/adapter/httpapi/health_handler.go`
- `internal/adapter/httpapi/version_handler.go`
- `internal/adapter/httpapi/router.go`
- `internal/adapter/httpapi/router_test.go`
- `internal/adapter/httpapi/task_handler.go`
- `internal/task/domain/task.go`
- `internal/task/domain/task_test.go`
- `internal/task/application/task_repository.go`
- `internal/task/application/task_service.go`
- `internal/task/infrastructure/memory/task_repository.go`
- `internal/task/infrastructure/memory/task_repository_test.go`

This is the expected state based on previous work, not a guarantee that every file currently exists or matches the description. Inspect the actual repository before acting.

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

`application.TaskRepository` defines:

- `Create(context.Context, domain.Task) error`
- `List(context.Context) ([]domain.Task, error)`

`application.TaskService` exposes:

- `CreateTask(ctx, title)`
- `ListTasks(ctx)`

The service generates task IDs, constructs valid domain entities, delegates persistence to the repository, and wraps infrastructure errors with context.

Go interfaces are implemented implicitly. Keep interfaces small and defined according to the needs of the consuming layer.

### In-memory infrastructure

`memory.TaskRepository` uses a `map[string]domain.Task` protected by `sync.RWMutex`.

It implements `Create` and `List`, rejects duplicate IDs, checks context cancellation, and sorts returned tasks by ID.

This implementation is intended for development and testing. Data is lost when the process exits.

### HTTP API

Existing endpoints:

- `GET /health`
- `GET /version`
- `POST /tasks`
- `GET /tasks`

The task handler receives `*application.TaskService`, not the memory repository directly.

`main.go` wires the concrete dependencies:

1. Create the memory repository.
2. Create TaskService with the repository.
3. Create the HTTP router with the service.
4. Start the HTTP server.

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

1. Review and strengthen HTTP handler tests for task creation, listing, validation, malformed JSON, and error responses.
2. Add application service tests using a small fake repository.
3. Review dependency boundaries and improve error handling where necessary.
4. Introduce configuration and graceful server shutdown.
5. Add PostgreSQL persistence and database migrations, replacing the in-memory implementation through dependency wiring rather than rewriting the service.
6. Add integration tests, Docker, observability, and CI incrementally.

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
