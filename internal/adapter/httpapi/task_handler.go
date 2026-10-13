package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-service-platform/internal/task/application"
	"go-service-platform/internal/task/domain"
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
	Error string `json:"error"`
}

func (h *taskHandler) createTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request createTaskRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
		})
		return
	}

	task, err := h.service.CreateTask(r.Context(), request.Title)
	if err != nil {
		if errors.Is(err, domain.ErrTaskIDRequired) ||
			errors.Is(err, domain.ErrTaskTitleRequired) ||
			errors.Is(err, domain.ErrTaskTitleTooLong) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: err.Error(),
			})
			return
		}

		log.Printf("creating task: %v", err)

		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
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

		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encoding JSON response: %v", err)
	}
}
