package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// Task validation errors.
	ErrTaskIDRequired    = errors.New("task ID is required")
	ErrTaskTitleRequired = errors.New("task title is required")
	ErrTaskTitleTooLong  = errors.New("task title must not exceed 120 characters")
	ErrTaskNotFound      = errors.New("task not found")
)

// Task represents a task managed by the service.
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
}

// NewTask creates a validated incomplete task and normalizes its timestamp to UTC.
func NewTask(id, title string, createdAt time.Time) (Task, error) {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)

	if id == "" {
		return Task{}, ErrTaskIDRequired
	}

	if title == "" {
		return Task{}, ErrTaskTitleRequired
	}

	if len([]rune(title)) > 120 {
		return Task{}, ErrTaskTitleTooLong
	}

	return Task{
		ID:        id,
		Title:     title,
		Completed: false,
		CreatedAt: createdAt.UTC(),
	}, nil
}
