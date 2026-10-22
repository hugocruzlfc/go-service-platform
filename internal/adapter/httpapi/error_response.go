package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"service-platform/internal/api"
	"service-platform/internal/task/domain"
)

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func invalidRequestResponse() api.ErrorResponse {
	return api.ErrorResponse{
		Error: api.ErrorDetail{
			Code:    "INVALID_REQUEST",
			Message: "invalid request body",
		},
	}
}

func internalErrorResponse() api.ErrorResponse {
	return api.ErrorResponse{
		Error: api.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		},
	}
}

func validationErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrTaskIDRequired):
		return "task ID is required"
	case errors.Is(err, domain.ErrTaskTitleRequired):
		return "task title is required"
	case errors.Is(err, domain.ErrTaskTitleTooLong):
		return "task title must not exceed 120 characters"
	case errors.Is(err, domain.ErrTaskUpdateEmpty):
		return "task update must contain at least one field"
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
