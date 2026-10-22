package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"service-platform/internal/task/application"
	"service-platform/internal/task/domain"
)

type failingTaskRepository struct {
	err error
}

const testAccessToken = "test-access-token"

// newTaskRequest builds a request carrying a valid bearer token, so task tests
// exercise the handlers rather than the auth middleware.
func newTaskRequest(method string, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+testAccessToken)
	return req
}

func (r failingTaskRepository) Create(
	_ context.Context,
	_ domain.Task,
) error {
	return r.err
}

func (r failingTaskRepository) Delete(
	_ context.Context,
	_ string,
) error {
	return r.err
}

func (r failingTaskRepository) Get(
	_ context.Context,
	_ string,
) (domain.Task, error) {
	return domain.Task{}, r.err
}

func (r failingTaskRepository) Update(
	_ context.Context,
	_ domain.Task,
) error {
	return r.err
}

func (r failingTaskRepository) List(
	_ context.Context,
) ([]domain.Task, error) {
	return nil, r.err
}

func TestCreateTaskRoute(t *testing.T) {
	router := newTestRouter()

	req := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"  Learn Go  "}`),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var task domain.Task

	if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if task.ID == "" {
		t.Error("expected the created task to have an ID")
	}

	if task.Title != "Learn Go" {
		t.Errorf("expected title %q, got %q", "Learn Go", task.Title)
	}

	if task.Completed {
		t.Error("expected a new task to be incomplete")
	}
}

func TestListTasksRoute(t *testing.T) {
	router := newTestRouter()

	createRequest := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"Learn Go"}`),
	)
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("creating task: expected status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	listRequest := newTaskRequest(http.MethodGet, "/tasks", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRecorder.Code)
	}

	var tasks []domain.Task

	if err := json.NewDecoder(listRecorder.Body).Decode(&tasks); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Title != "Learn Go" {
		t.Errorf("expected title %q, got %q", "Learn Go", tasks[0].Title)
	}
}

func TestGetTaskRoute(t *testing.T) {
	router := newTestRouter()

	createRequest := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"Learn Go"}`),
	)
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("creating task: expected status %d, got %d", http.StatusCreated, createRecorder.Code)
	}

	var createdTask domain.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decoding created task: %v", err)
	}

	getRequest := newTaskRequest(http.MethodGet, "/tasks/"+createdTask.ID, nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRecorder.Code)
	}

	var task domain.Task
	if err := json.NewDecoder(getRecorder.Body).Decode(&task); err != nil {
		t.Fatalf("decoding task: %v", err)
	}
	if task != createdTask {
		t.Errorf("expected task %#v, got %#v", createdTask, task)
	}
}

func TestGetTaskRouteReturnsNotFound(t *testing.T) {
	router := newTestRouter()
	req := newTaskRequest(http.MethodGet, "/tasks/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "NOT_FOUND", "task not found")
}

func TestUpdateTaskRoute(t *testing.T) {
	router := newTestRouter()

	createRequest := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"Learn Go"}`),
	)
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	var createdTask domain.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decoding created task: %v", err)
	}

	updateRequest := newTaskRequest(
		http.MethodPatch,
		"/tasks/"+createdTask.ID,
		bytes.NewBufferString(`{"completed":true}`),
	)
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateRecorder.Code)
	}

	var updatedTask domain.Task
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updatedTask); err != nil {
		t.Fatalf("decoding updated task: %v", err)
	}
	if !updatedTask.Completed {
		t.Error("expected task to be completed")
	}
	if updatedTask.Title != createdTask.Title {
		t.Errorf("expected title %q, got %q", createdTask.Title, updatedTask.Title)
	}
}

func TestUpdateTaskRouteRejectsEmptyUpdate(t *testing.T) {
	router := newTestRouter()
	req := newTaskRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "VALIDATION_ERROR", "task update must contain at least one field")
}

func TestDeleteTaskRoute(t *testing.T) {
	router := newTestRouter()

	createRequest := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"Learn Go"}`),
	)
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	var createdTask domain.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decoding created task: %v", err)
	}

	deleteRequest := newTaskRequest(http.MethodDelete, "/tasks/"+createdTask.ID, nil)
	deleteRecorder := httptest.NewRecorder()
	router.ServeHTTP(deleteRecorder, deleteRequest)

	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, deleteRecorder.Code)
	}
	if deleteRecorder.Body.Len() != 0 {
		t.Errorf("expected empty response body, got %q", deleteRecorder.Body.String())
	}

	getRequest := newTaskRequest(http.MethodGet, "/tasks/"+createdTask.ID, nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	assertErrorResponse(t, getRecorder, http.StatusNotFound, "NOT_FOUND", "task not found")
}

func TestDeleteTaskRouteReturnsNotFound(t *testing.T) {
	router := newTestRouter()
	req := newTaskRequest(http.MethodDelete, "/tasks/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "NOT_FOUND", "task not found")
}

func TestCreateTaskRouteRejectsInvalidJSON(t *testing.T) {
	router := newTestRouter()

	req := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":`),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
}

func TestCreateTaskRouteRejectsTrailingJSON(t *testing.T) {
	router := newTestRouter()

	req := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"Learn Go"}{"extra":"data"}`),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
}

func TestCreateTaskRouteRejectsOversizedBody(t *testing.T) {
	router := newTestRouter()

	req := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewReader(bytes.Repeat([]byte("a"), 1<<20+1)),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
}

func TestCreateTaskRouteRejectsEmptyTitle(t *testing.T) {
	router := newTestRouter()

	req := newTaskRequest(
		http.MethodPost,
		"/tasks",
		bytes.NewBufferString(`{"title":"   "}`),
	)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "VALIDATION_ERROR", "task title is required")
}

func TestTaskRoutesHideRepositoryErrors(t *testing.T) {
	repositoryError := errors.New("database connection failed")
	taskService := application.NewTaskService(failingTaskRepository{err: repositoryError})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	readiness := NewReadiness(fakeDatabasePinger{}, logger)
	apiHandler := NewAPIHandler(taskService, nil, readiness, logger)
	router := NewRouter(apiHandler, fakeTokenVerifier{}, logger)

	t.Run("create", func(t *testing.T) {
		req := newTaskRequest(
			http.MethodPost,
			"/tasks",
			bytes.NewBufferString(`{"title":"Learn Go"}`),
		)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})

	t.Run("list", func(t *testing.T) {
		req := newTaskRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})

	t.Run("get", func(t *testing.T) {
		req := newTaskRequest(http.MethodGet, "/tasks/task-1", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})

	t.Run("update", func(t *testing.T) {
		req := newTaskRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(`{"completed":true}`))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})

	t.Run("delete", func(t *testing.T) {
		req := newTaskRequest(http.MethodDelete, "/tasks/task-1", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})
}

func assertErrorResponse(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	expectedStatus int,
	expectedCode string,
	expectedMessage string,
) {
	t.Helper()

	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rec.Code)
	}

	var response errorResponse

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}

	if response.Error.Code != expectedCode {
		t.Errorf("expected error code %q, got %q", expectedCode, response.Error.Code)
	}

	if response.Error.Message != expectedMessage {
		t.Errorf("expected error message %q, got %q", expectedMessage, response.Error.Message)
	}
}
