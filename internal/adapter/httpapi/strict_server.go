package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"service-platform/internal/api"
	"service-platform/internal/task/application"
	"service-platform/internal/task/domain"
)

type strictServer struct {
	service   *application.TaskService
	readiness *Readiness
	logger    *slog.Logger
}

var _ api.StrictServerInterface = (*strictServer)(nil)

func newStrictServer(
	service *application.TaskService,
	readiness *Readiness,
	logger *slog.Logger,
) *strictServer {
	return &strictServer{
		service:   service,
		readiness: readiness,
		logger:    logger,
	}
}

func (s *strictServer) GetHealth(
	context.Context,
	api.GetHealthRequestObject,
) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse(api.HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC(),
	}), nil
}

func (s *strictServer) GetHealthz(
	context.Context,
	api.GetHealthzRequestObject,
) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse(api.HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC(),
	}), nil
}

func (s *strictServer) GetReadiness(
	ctx context.Context,
	_ api.GetReadinessRequestObject,
) (api.GetReadinessResponseObject, error) {
	if s.readiness.draining.Load() {
		s.logger.Info("readiness check rejected during shutdown")
		return readinessErrorResponse("service is shutting down"), nil
	}

	if err := s.readiness.pinger.Ping(ctx); err != nil {
		s.logger.Warn("readiness check failed", slog.Any("error", err))
		return readinessErrorResponse("service is not ready"), nil
	}

	return api.GetReadiness200JSONResponse(api.HealthResponse{
		Status: "ready",
		Time:   time.Now().UTC(),
	}), nil
}

func (s *strictServer) ListTasks(
	ctx context.Context,
	_ api.ListTasksRequestObject,
) (api.ListTasksResponseObject, error) {
	tasks, err := s.service.ListTasks(ctx)
	if err != nil {
		s.logger.Error("listing tasks", slog.Any("error", err))
		return api.ListTasks500JSONResponse(internalErrorResponse()), nil
	}

	response := make([]api.Task, 0, len(tasks))
	for _, task := range tasks {
		response = append(response, toAPITask(task))
	}

	return api.ListTasks200JSONResponse(response), nil
}

func (s *strictServer) GetTask(
	ctx context.Context,
	request api.GetTaskRequestObject,
) (api.GetTaskResponseObject, error) {
	task, err := s.service.GetTask(ctx, request.Id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return api.GetTask404JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "task not found",
				},
			}), nil
		}

		s.logger.Error("getting task", slog.String("task_id", request.Id), slog.Any("error", err))
		return api.GetTask500JSONResponse(internalErrorResponse()), nil
	}

	return api.GetTask200JSONResponse(toAPITask(task)), nil
}

func (s *strictServer) UpdateTask(
	ctx context.Context,
	request api.UpdateTaskRequestObject,
) (api.UpdateTaskResponseObject, error) {
	if request.Body == nil {
		return api.UpdateTask400JSONResponse(invalidRequestResponse()), nil
	}

	task, err := s.service.UpdateTask(ctx, request.Id, application.TaskUpdate{
		Title:     request.Body.Title,
		Completed: request.Body.Completed,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return api.UpdateTask404JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "task not found",
				},
			}), nil
		}

		if errors.Is(err, domain.ErrTaskIDRequired) ||
			errors.Is(err, domain.ErrTaskUpdateEmpty) ||
			errors.Is(err, domain.ErrTaskTitleRequired) ||
			errors.Is(err, domain.ErrTaskTitleTooLong) {
			return api.UpdateTask400JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: validationErrorMessage(err),
				},
			}), nil
		}

		s.logger.Error("updating task", slog.String("task_id", request.Id), slog.Any("error", err))
		return api.UpdateTask500JSONResponse(internalErrorResponse()), nil
	}

	return api.UpdateTask200JSONResponse(toAPITask(task)), nil
}

func (s *strictServer) DeleteTask(
	ctx context.Context,
	request api.DeleteTaskRequestObject,
) (api.DeleteTaskResponseObject, error) {
	if err := s.service.DeleteTask(ctx, request.Id); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return api.DeleteTask404JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "task not found",
				},
			}), nil
		}

		if errors.Is(err, domain.ErrTaskIDRequired) {
			return api.DeleteTask404JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "task not found",
				},
			}), nil
		}

		s.logger.Error("deleting task", slog.String("task_id", request.Id), slog.Any("error", err))
		return api.DeleteTask500JSONResponse(internalErrorResponse()), nil
	}

	return api.DeleteTask204Response{}, nil
}

func (s *strictServer) CreateTask(
	ctx context.Context,
	request api.CreateTaskRequestObject,
) (api.CreateTaskResponseObject, error) {
	if request.Body == nil {
		return api.CreateTask400JSONResponse(invalidRequestResponse()), nil
	}

	task, err := s.service.CreateTask(ctx, request.Body.Title)
	if err != nil {
		if errors.Is(err, domain.ErrTaskIDRequired) ||
			errors.Is(err, domain.ErrTaskTitleRequired) ||
			errors.Is(err, domain.ErrTaskTitleTooLong) {
			return api.CreateTask400JSONResponse(api.ErrorResponse{
				Error: api.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: validationErrorMessage(err),
				},
			}), nil
		}

		s.logger.Error("creating task", slog.Any("error", err))
		return api.CreateTask500JSONResponse(internalErrorResponse()), nil
	}

	return api.CreateTask201JSONResponse(toAPITask(task)), nil
}

func (s *strictServer) GetVersion(
	context.Context,
	api.GetVersionRequestObject,
) (api.GetVersionResponseObject, error) {
	return api.GetVersion200JSONResponse(api.VersionResponse{
		Version: "1.0.0",
	}), nil
}

func toAPITask(task domain.Task) api.Task {
	return api.Task{
		Id:        task.ID,
		Title:     task.Title,
		Completed: task.Completed,
		CreatedAt: task.CreatedAt,
	}
}

func readinessErrorResponse(message string) api.GetReadiness503JSONResponse {
	return api.GetReadiness503JSONResponse(api.ErrorResponse{
		Error: api.ErrorDetail{
			Code:    "NOT_READY",
			Message: message,
		},
	})
}
