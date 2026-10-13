package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewTask(t *testing.T) {
	createdAt := time.Date(
		2026, time.October, 13,
		10, 0, 0, 0,
		time.FixedZone("UTC+2", 2*60*60),
	)

	task, err := NewTask("task-1", "  Learn Go  ", createdAt)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if task.ID != "task-1" {
		t.Errorf("expected ID %q, got %q", "task-1", task.ID)
	}

	if task.Title != "Learn Go" {
		t.Errorf("expected title %q, got %q", "Learn Go", task.Title)
	}

	if task.Completed {
		t.Error("expected a new task to be incomplete")
	}

	if task.CreatedAt.Location() != time.UTC {
		t.Error("expected creation time to be normalized to UTC")
	}
}

func TestNewTaskRejectsEmptyTitle(t *testing.T) {
	_, err := NewTask("task-1", "   ", time.Now())

	if !errors.Is(err, ErrTaskTitleRequired) {
		t.Errorf("expected ErrTaskTitleRequired, got %v", err)
	}
}

func TestNewTaskRejectsEmptyID(t *testing.T) {
	_, err := NewTask(" ", "Learn Go", time.Now())

	if !errors.Is(err, ErrTaskIDRequired) {
		t.Errorf("expected ErrTaskIDRequired, got %v", err)
	}
}

func TestNewTaskRejectsLongTitle(t *testing.T) {
	title := strings.Repeat("a", 121)

	_, err := NewTask("task-1", title, time.Now())

	if !errors.Is(err, ErrTaskTitleTooLong) {
		t.Errorf("expected ErrTaskTitleTooLong, got %v", err)
	}
}