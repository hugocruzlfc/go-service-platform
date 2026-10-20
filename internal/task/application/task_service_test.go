package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"service-platform/internal/task/domain"
)

type fakeTaskRepository struct {
	createdTasks []domain.Task
	gottenTasks  map[string]domain.Task
	listedTasks  []domain.Task
	createErr    error
	getErr       error
	updateErr    error
	listErr      error
	createCtx    context.Context
	getCtx       context.Context
	listCtx      context.Context
}

func (r *fakeTaskRepository) Create(
	ctx context.Context,
	task domain.Task,
) error {
	r.createCtx = ctx

	if r.createErr != nil {
		return r.createErr
	}

	r.createdTasks = append(r.createdTasks, task)
	return nil
}

func (r *fakeTaskRepository) Get(
	ctx context.Context,
	id string,
) (domain.Task, error) {
	r.getCtx = ctx

	if r.getErr != nil {
		return domain.Task{}, r.getErr
	}

	task, exists := r.gottenTasks[id]
	if !exists {
		return domain.Task{}, domain.ErrTaskNotFound
	}

	return task, nil
}

func (r *fakeTaskRepository) Update(
	ctx context.Context,
	task domain.Task,
) error {
	if r.updateErr != nil {
		return r.updateErr
	}

	if r.gottenTasks == nil {
		r.gottenTasks = make(map[string]domain.Task)
	}
	r.gottenTasks[task.ID] = task
	return nil
}

func (r *fakeTaskRepository) List(
	ctx context.Context,
) ([]domain.Task, error) {
	r.listCtx = ctx

	if r.listErr != nil {
		return nil, r.listErr
	}

	return r.listedTasks, nil
}

func TestTaskServiceCreateTask(t *testing.T) {
	repository := &fakeTaskRepository{}
	service := NewTaskService(repository)
	ctx := t.Context()

	task, err := service.CreateTask(ctx, "  Learn Go  ")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if task.ID == "" {
		t.Error("expected the service to generate a task ID")
	}

	if task.Title != "Learn Go" {
		t.Errorf("expected title %q, got %q", "Learn Go", task.Title)
	}

	if task.Completed {
		t.Error("expected a new task to be incomplete")
	}

	if len(repository.createdTasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(repository.createdTasks))
	}

	if repository.createdTasks[0] != task {
		t.Errorf("expected repository to receive task %#v, got %#v", task, repository.createdTasks[0])
	}

	if repository.createCtx != ctx {
		t.Error("expected the service to pass the context to the repository")
	}
}

func TestTaskServiceListTasks(t *testing.T) {
	expectedTasks := []domain.Task{
		{
			ID:        "task-1",
			Title:     "Learn Go",
			CreatedAt: time.Date(2026, time.October, 19, 10, 0, 0, 0, time.UTC),
		},
	}
	repository := &fakeTaskRepository{listedTasks: expectedTasks}
	service := NewTaskService(repository)
	ctx := t.Context()

	tasks, err := service.ListTasks(ctx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(tasks) != len(expectedTasks) {
		t.Fatalf("expected %d tasks, got %d", len(expectedTasks), len(tasks))
	}

	if tasks[0] != expectedTasks[0] {
		t.Errorf("expected tasks %#v, got %#v", expectedTasks, tasks)
	}

	if repository.listCtx != ctx {
		t.Error("expected the service to pass the context to the repository")
	}
}

func TestTaskServiceGetTask(t *testing.T) {
	expectedTask := domain.Task{ID: "task-1", Title: "Learn Go"}
	repository := &fakeTaskRepository{
		gottenTasks: map[string]domain.Task{"task-1": expectedTask},
	}
	service := NewTaskService(repository)
	ctx := t.Context()

	task, err := service.GetTask(ctx, " task-1 ")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task != expectedTask {
		t.Errorf("expected task %#v, got %#v", expectedTask, task)
	}
	if repository.getCtx != ctx {
		t.Error("expected the service to pass the context to the repository")
	}
}

func TestTaskServiceGetTaskReturnsNotFound(t *testing.T) {
	service := NewTaskService(&fakeTaskRepository{})

	_, err := service.GetTask(t.Context(), "missing")

	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("expected domain.ErrTaskNotFound, got %v", err)
	}
}

func TestTaskServiceUpdateTask(t *testing.T) {
	completed := true
	title := "Build APIs"
	expectedTask := domain.Task{ID: "task-1", Title: "Learn Go"}
	repository := &fakeTaskRepository{
		gottenTasks: map[string]domain.Task{"task-1": expectedTask},
	}
	service := NewTaskService(repository)

	task, err := service.UpdateTask(t.Context(), " task-1 ", TaskUpdate{
		Title:     &title,
		Completed: &completed,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if task.Title != title || !task.Completed {
		t.Errorf("expected updated task, got %#v", task)
	}
}

func TestTaskServiceUpdateTaskRejectsEmptyUpdate(t *testing.T) {
	repository := &fakeTaskRepository{
		gottenTasks: map[string]domain.Task{"task-1": {ID: "task-1", Title: "Learn Go"}},
	}
	service := NewTaskService(repository)

	_, err := service.UpdateTask(t.Context(), "task-1", TaskUpdate{})

	if !errors.Is(err, domain.ErrTaskUpdateEmpty) {
		t.Errorf("expected domain.ErrTaskUpdateEmpty, got %v", err)
	}
}

func TestTaskServiceCreateTaskWrapsRepositoryError(t *testing.T) {
	repositoryError := errors.New("repository unavailable")
	service := NewTaskService(&fakeTaskRepository{createErr: repositoryError})

	_, err := service.CreateTask(t.Context(), "Learn Go")

	if !errors.Is(err, repositoryError) {
		t.Errorf("expected wrapped repository error, got %v", err)
	}

	if err == repositoryError {
		t.Error("expected the service to add context to the repository error")
	}
}

func TestTaskServiceListTasksWrapsRepositoryError(t *testing.T) {
	repositoryError := errors.New("repository unavailable")
	service := NewTaskService(&fakeTaskRepository{listErr: repositoryError})

	_, err := service.ListTasks(t.Context())

	if !errors.Is(err, repositoryError) {
		t.Errorf("expected wrapped repository error, got %v", err)
	}

	if err == repositoryError {
		t.Error("expected the service to add context to the repository error")
	}
}
