package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/task/application"
	"service-platform/internal/task/domain"
)

var _ application.TaskRepository = (*TaskRepository)(nil)

// TaskRepository persists tasks in PostgreSQL.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository creates a PostgreSQL task repository using pool.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{
		pool: pool,
	}
}

func (r *TaskRepository) Create(
	ctx context.Context,
	task domain.Task,
) error {
	const query = `
		INSERT INTO tasks (id, title, completed, created_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.pool.Exec(
		ctx,
		query,
		task.ID,
		task.Title,
		task.Completed,
		task.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// Delete removes a task by ID.
func (r *TaskRepository) Delete(
	ctx context.Context,
	id string,
) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}

	return nil
}

// Get returns a task by ID.
func (r *TaskRepository) Get(
	ctx context.Context,
	id string,
) (domain.Task, error) {
	const query = `
		SELECT id, title, completed, created_at
		FROM tasks
		WHERE id = $1
	`

	var task domain.Task

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
		&task.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task: %w", err)
	}

	return task, nil
}

// Update replaces an existing task.
func (r *TaskRepository) Update(
	ctx context.Context,
	task domain.Task,
) error {
	const query = `
		UPDATE tasks
		SET title = $2, completed = $3
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, task.ID, task.Title, task.Completed)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}

	return nil
}

func (r *TaskRepository) List(
	ctx context.Context,
) ([]domain.Task, error) {
	const query = `
		SELECT id, title, completed, created_at
		FROM tasks
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)

	for rows.Next() {
		var task domain.Task

		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
			&task.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}
