package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"service-platform/internal/task/application"
	"service-platform/internal/task/domain"
)

type taskHandler struct {
	service *application.TaskService
	logger  *slog.Logger
}

func newTaskHandler(
	service *application.TaskService,
	logger *slog.Logger,
) *taskHandler {
	return &taskHandler{
		service: service,
		logger:  logger,
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
		writeError(h.logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(h.logger, w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	task, err := h.service.CreateTask(r.Context(), request.Title)
	if err != nil {
		if errors.Is(err, domain.ErrTaskIDRequired) ||
			errors.Is(err, domain.ErrTaskTitleRequired) ||
			errors.Is(err, domain.ErrTaskTitleTooLong) {
			writeError(h.logger, w, http.StatusBadRequest, "VALIDATION_ERROR", validationErrorMessage(err))
			return
		}

		h.logger.Error("creating task", slog.Any("error", err))

		writeError(h.logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeJSON(h.logger, w, http.StatusCreated, task)
}

func (h *taskHandler) listTasks(
	w http.ResponseWriter,
	r *http.Request,
) {
	tasks, err := h.service.ListTasks(r.Context())
	if err != nil {
		h.logger.Error("listing tasks", slog.Any("error", err))

		writeError(h.logger, w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	writeJSON(h.logger, w, http.StatusOK, tasks)
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

func writeError(logger *slog.Logger, w http.ResponseWriter, status int, code, message string) {
	writeJSON(logger, w, status, errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(logger *slog.Logger, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("encoding JSON response", slog.Any("error", err))
	}
}
