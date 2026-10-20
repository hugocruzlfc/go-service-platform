package application

import (
	"context"

	"service-platform/internal/task/domain"
)

// TaskRepository persists and retrieves tasks for the application layer.
type TaskRepository interface {
	Create(ctx context.Context, task domain.Task) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (domain.Task, error)
	Update(ctx context.Context, task domain.Task) error
	List(ctx context.Context) ([]domain.Task, error)
}
