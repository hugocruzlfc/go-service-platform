package memory

import (
	"context"
	"errors"
	"sort"
	"sync"

	"service-platform/internal/task/domain"
)

// ErrTaskAlreadyExists indicates that a task ID is already stored.
var ErrTaskAlreadyExists = errors.New("task already exists")

// TaskRepository stores tasks in memory and is safe for concurrent use.
type TaskRepository struct {
	mu    sync.RWMutex
	tasks map[string]domain.Task
}

// NewTaskRepository creates an empty in-memory task repository.
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

// Delete removes a task by ID.
func (r *TaskRepository) Delete(
	ctx context.Context,
	id string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return domain.ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}

// Get returns a task by ID.
func (r *TaskRepository) Get(
	ctx context.Context,
	id string,
) (domain.Task, error) {
	if err := ctx.Err(); err != nil {
		return domain.Task{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return domain.Task{}, domain.ErrTaskNotFound
	}

	return task, nil
}

// Update replaces an existing task.
func (r *TaskRepository) Update(
	ctx context.Context,
	task domain.Task,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return domain.ErrTaskNotFound
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
