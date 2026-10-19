package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"service-platform/internal/task/domain"
)

type TaskService struct {
	repository TaskRepository
}

func NewTaskService(repository TaskRepository) *TaskService {
	return &TaskService{
		repository: repository,
	}
}

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
