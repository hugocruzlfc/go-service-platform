package memory

import (
	"context"
	"errors"
	"sort"
	"sync"

	"go-service-platform/internal/task/domain"
)

var ErrTaskAlreadyExists = errors.New("task already exists")

type TaskRepository struct {
	mu    sync.RWMutex
	tasks map[string]domain.Task
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[string]domain.Task),
	}
}

func (r *TaskRepository) Create(
	ctx context.Context,
	task domain.Task,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}

	r.tasks[task.ID] = task

	return nil
}

func (r *TaskRepository) List(
	ctx context.Context,
) ([]domain.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]domain.Task, 0, len(r.tasks))

	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	return tasks, nil
}
