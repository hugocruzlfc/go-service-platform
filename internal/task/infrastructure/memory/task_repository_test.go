package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"service-platform/internal/task/domain"
)

func TestTaskRepositoryCreateAndList(t *testing.T) {
	repository := NewTaskRepository()
	ctx := context.Background()

	first, err := domain.NewTask(
		"task-2",
		"Learn Go",
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := domain.NewTask(
		"task-1",
		"Build an API",
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("creating first task: %v", err)
	}

	if err := repository.Create(ctx, second); err != nil {
		t.Fatalf("creating second task: %v", err)
	}

	tasks, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("listing tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("expected first ID %q, got %q", "task-1", tasks[0].ID)
	}
}

func TestTaskRepositoryRejectsDuplicateID(t *testing.T) {
	repository := NewTaskRepository()
	ctx := context.Background()

	task, err := domain.NewTask("task-1", "Learn Go", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	err = repository.Create(ctx, task)

	if !errors.Is(err, ErrTaskAlreadyExists) {
		t.Errorf("expected ErrTaskAlreadyExists, got %v", err)
	}
}

func TestTaskRepositoryRejectsCancelledContext(t *testing.T) {
	repository := NewTaskRepository()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tasks, err := repository.List(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	if tasks != nil {
		t.Errorf("expected nil tasks, got %v", tasks)
	}
}
