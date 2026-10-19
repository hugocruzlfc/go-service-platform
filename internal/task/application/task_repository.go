package application

import (
	"context"

	"service-platform/internal/task/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, task domain.Task) error
	List(ctx context.Context) ([]domain.Task, error)
}
