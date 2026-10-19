package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"service-platform/internal/task/application"
	"service-platform/internal/task/domain"
)

type taskHandler struct {
	service *application.TaskService
}

func newTaskHandler(service *application.TaskService) *taskHandler {
	return &taskHandler{
		service: service,
	}
}

type createTaskRequest struct {
	Title string `json:"title"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *taskHandler) createTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request createTaskRequest

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	task, err := h.service.CreateTask(r.Context(), request.Title)
	if err != nil {
		if errors.Is(err, domain.ErrTaskIDRequired) ||
			errors.Is(err, domain.ErrTaskTitleRequired) ||
			errors.Is(err, domain.ErrTaskTitleTooLong) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErrorMessage(err))
			return
		}

		log.Printf("creating task: %v", err)

		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *taskHandler) listTasks(
	w http.ResponseWriter,
	r *http.Request,
) {
	tasks, err := h.service.ListTasks(r.Context())
	if err != nil {
		log.Printf("listing tasks: %v", err)

		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func validationErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrTaskIDRequired):
		return "task ID is required"
	case errors.Is(err, domain.ErrTaskTitleRequired):
		return "task title is required"
	case errors.Is(err, domain.ErrTaskTitleTooLong):
		return "task title must not exceed 120 characters"
	default:
		return "invalid task"
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encoding JSON response: %v", err)
	}
}
