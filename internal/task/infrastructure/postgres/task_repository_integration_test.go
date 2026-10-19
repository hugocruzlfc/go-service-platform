package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"service-platform/internal/task/domain"
)

func TestTaskRepositoryCreateAndListIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("creating database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("connecting to database: %v", err)
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
		_, cleanupErr := pool.Exec(ctx, "DELETE FROM tasks WHERE id = $1", task.ID)
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
