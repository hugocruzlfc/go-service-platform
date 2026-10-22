// Package task composes the task module.
package task

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/task/application"
	"service-platform/internal/task/infrastructure/postgres"
)

// NewService creates the task service backed by PostgreSQL.
func NewService(databasePool *pgxpool.Pool) *application.TaskService {
	taskRepository := postgres.NewTaskRepository(databasePool)
	return application.NewTaskService(taskRepository)
}
