package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"service-platform/internal/task/domain"
)

// TaskService implements task use cases.
type TaskService struct {
	repository TaskRepository
}

// NewTaskService creates a task service using repository as its persistence dependency.
func NewTaskService(repository TaskRepository) *TaskService {
	return &TaskService{
		repository: repository,
	}
}

// CreateTask validates and persists a new task with a generated ID.
func (s *TaskService) CreateTask(
	ctx context.Context,
	title string,
) (domain.Task, error) {
	id, err := generateTaskID()
	if err != nil {
		return domain.Task{}, fmt.Errorf("generate task ID: %w", err)
	}

	task, err := domain.NewTask(id, title, time.Now())
	if err != nil {
		return domain.Task{}, err
	}

	if err := s.repository.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

// GetTask returns a task by ID.
func (s *TaskService) GetTask(
	ctx context.Context,
	id string,
) (domain.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Task{}, domain.ErrTaskIDRequired
	}

	task, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task: %w", err)
	}

	return task, nil
}

// ListTasks returns all tasks from the configured repository.
func (s *TaskService) ListTasks(
	ctx context.Context,
) ([]domain.Task, error) {
	tasks, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func generateTaskID() (string, error) {
	var bytes [16]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
