//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"service-platform/internal/task/domain"
)

func TestTaskRepositoryCreateAndListIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("creating database pool: %v", err)
	}

	repository := NewTaskRepository(pool)
	task, err := domain.NewTask(
		"integration-task-"+time.Now().UTC().Format("20060102150405.000000000"),
		"Integration test task",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("creating domain task: %v", err)
	}

	t.Cleanup(func() {
		cleanupContext, cancelCleanup := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelCleanup()

		_, cleanupErr := pool.Exec(cleanupContext, "DELETE FROM tasks WHERE id = $1", task.ID)
		if cleanupErr != nil {
			t.Errorf("cleaning up integration task: %v", cleanupErr)
		}

		pool.Close()
	})

	if err := repository.Create(ctx, task); err != nil {
		t.Fatalf("creating task in PostgreSQL: %v", err)
	}

	tasks, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("listing tasks from PostgreSQL: %v", err)
	}

	for _, listedTask := range tasks {
		if listedTask.ID == task.ID {
			if listedTask.Title != task.Title {
				t.Errorf("expected title %q, got %q", task.Title, listedTask.Title)
			}

			if listedTask.Completed != task.Completed {
				t.Errorf("expected completed=%t, got %t", task.Completed, listedTask.Completed)
			}

			return
		}
	}

	t.Fatalf("created task %q was not returned by List", task.ID)
}
