package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProjectAPIWorkflowAndValidation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "project-api.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}); err != nil {
		t.Fatal(err)
	}
	projects := service.NewProjectService(repository.NewProjectRepository(db))
	router := NewRouter(config.Config{Environment: "development"}, NewHandler(nil, nil, projects), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}})
	call := func(method, path, body string, want int) string {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("%s %s = %d, want %d: %s", method, path, recorder.Code, want, recorder.Body.String())
		}
		return recorder.Body.String()
	}
	call(http.MethodPost, "/api/v1/projects", `{"title":" "}`, http.StatusBadRequest)
	call(http.MethodPost, "/api/v1/projects", `{"title":"First","description":"A goal"}`, http.StatusCreated)
	call(http.MethodPost, "/api/v1/projects", `{"title":"Second"}`, http.StatusCreated)
	created := call(http.MethodPost, "/api/v1/tasks", `{"project_id":1,"title":"First task","priority":"high","due_date":"2026-09-24"}`, http.StatusCreated)
	var taskResponse struct {
		Data model.ProjectTask `json:"data"`
	}
	if err := json.Unmarshal([]byte(created), &taskResponse); err != nil {
		t.Fatal(err)
	}
	if taskResponse.Data.Status != "inbox" || taskResponse.Data.Priority != "high" {
		t.Fatalf("unexpected task: %s", created)
	}
	call(http.MethodPost, "/api/v1/tasks", `{"project_id":2,"title":"Second task","status":"next"}`, http.StatusCreated)
	call(http.MethodPatch, "/api/v1/tasks/1/move", `{"status":"next"}`, http.StatusOK)
	body := call(http.MethodGet, "/api/v1/projects", "", http.StatusOK)
	if !strings.Contains(body, `"description":"A goal"`) || !strings.Contains(body, `"title":"First"`) {
		t.Fatalf("unexpected projects: %s", body)
	}
	if tasks := call(http.MethodGet, "/api/v1/tasks?status=next", "", http.StatusOK); strings.Count(tasks, `"status":"next"`) != 2 {
		t.Fatalf("expected two next tasks: %s", tasks)
	}
	call(http.MethodPatch, "/api/v1/projects/1", `{"status":"archived"}`, http.StatusOK)
	if tasks := call(http.MethodGet, "/api/v1/tasks", "", http.StatusOK); strings.Contains(tasks, `"First task"`) {
		t.Fatalf("archived task in all-projects board: %s", tasks)
	}
	call(http.MethodPatch, "/api/v1/projects/1", `{"status":"active"}`, http.StatusOK)
	completed := call(http.MethodPatch, "/api/v1/tasks/1/move", `{"status":"done"}`, http.StatusOK)
	if !strings.Contains(completed, `"completed_at":"`) {
		t.Fatalf("missing completion timestamp: %s", completed)
	}
	reopened := call(http.MethodPatch, "/api/v1/tasks/1/move", `{"status":"next"}`, http.StatusOK)
	if !strings.Contains(reopened, `"completed_at":null`) {
		t.Fatalf("reopened task remains complete: %s", reopened)
	}
	call(http.MethodPatch, "/api/v1/tasks/1", `{"priority":"low","due_date":""}`, http.StatusOK)
	call(http.MethodDelete, "/api/v1/tasks/1", "", http.StatusOK)
	call(http.MethodDelete, "/api/v1/tasks/1", "", http.StatusNotFound)
	call(http.MethodPost, "/api/v1/projects/999/tasks", `{"title":"No"}`, http.StatusNotFound)
	call(http.MethodPost, "/api/v1/tasks", `{"project_id":999,"title":"No"}`, http.StatusNotFound)
}
