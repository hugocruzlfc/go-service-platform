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
	ctx := t.Context()

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
	ctx := t.Context()

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

func TestTaskRepositoryGet(t *testing.T) {
	repository := NewTaskRepository()
	ctx := t.Context()
	task, err := domain.NewTask("task-1", "Learn Go", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := repository.Get(ctx, "task-1")
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got != task {
		t.Errorf("expected task %#v, got %#v", task, got)
	}
}

func TestTaskRepositoryGetReturnsNotFound(t *testing.T) {
	repository := NewTaskRepository()

	_, err := repository.Get(t.Context(), "missing")

	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("expected domain.ErrTaskNotFound, got %v", err)
	}
}

func TestTaskRepositoryUpdate(t *testing.T) {
	repository := NewTaskRepository()
	ctx := t.Context()
	task, err := domain.NewTask("task-1", "Learn Go", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	task.Title = "Build APIs"
	task.Completed = true
	if err := repository.Update(ctx, task); err != nil {
		t.Fatalf("updating task: %v", err)
	}

	updated, err := repository.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting updated task: %v", err)
	}
	if updated != task {
		t.Errorf("expected task %#v, got %#v", task, updated)
	}
}

func TestTaskRepositoryUpdateReturnsNotFound(t *testing.T) {
	repository := NewTaskRepository()

	err := repository.Update(t.Context(), domain.Task{ID: "missing"})

	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("expected domain.ErrTaskNotFound, got %v", err)
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
