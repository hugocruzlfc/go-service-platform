package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrTaskIDRequired indicates that a task ID was not provided.
	ErrTaskIDRequired = errors.New("task ID is required")
	// ErrTaskTitleRequired indicates that a task title was not provided.
	ErrTaskTitleRequired = errors.New("task title is required")
	// ErrTaskTitleTooLong indicates that a task title exceeds 120 characters.
	ErrTaskTitleTooLong = errors.New("task title must not exceed 120 characters")
	// ErrTaskNotFound indicates that a task does not exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskUpdateEmpty indicates that an update contains no fields.
	ErrTaskUpdateEmpty = errors.New("task update must contain at least one field")
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

// Update applies the provided changes while preserving the task identity and creation time.
func (t *Task) Update(title *string, completed *bool) error {
	if title == nil && completed == nil {
		return ErrTaskUpdateEmpty
	}

	if title != nil {
		normalizedTitle := strings.TrimSpace(*title)
		if normalizedTitle == "" {
			return ErrTaskTitleRequired
		}
		if len([]rune(normalizedTitle)) > 120 {
			return ErrTaskTitleTooLong
		}

		t.Title = normalizedTitle
	}

	if completed != nil {
		t.Completed = *completed
	}

	return nil
}
