package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"learnos/internal/auth"
	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dailyFocusAPIFixture struct {
	t        *testing.T
	db       *gorm.DB
	focuses  *service.DailyFocusService
	projects *service.ProjectService
	handler  *Handler
	router   *gin.Engine
}

func newDailyFocusAPIFixture(t *testing.T) *dailyFocusAPIFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "daily-focus-api.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.DailyFocus{}, &model.Course{}, &model.InboxItem{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	projectRepository := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepository)
	focusService := service.NewDailyFocusService(repository.NewDailyFocusRepository(db))
	workspaceService := service.NewWorkspaceService(
		projectService,
		service.NewCourseService(repository.NewCourseRepository(db), nil),
		nil,
		service.NewInboxService(repository.NewInboxRepository(db), projectRepository),
		focusService,
	)
	handler := &Handler{dailyFocus: focusService, workspace: workspaceService, projects: projectService}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		userID := uint(1)
		if raw := c.GetHeader("X-Test-User"); raw != "" {
			if parsed, parseErr := strconv.ParseUint(raw, 10, 32); parseErr == nil {
				userID = uint(parsed)
			}
		}
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), auth.Principal{UserID: userID}))
		c.Next()
	})
	router.GET("/workspace/daily-focus", handler.GetDailyFocus)
	router.PUT("/workspace/daily-focus/:date/:context", handler.PutDailyFocus)
	router.GET("/workspace/home", handler.GetWorkspaceHome)
	return &dailyFocusAPIFixture{t: t, db: db, focuses: focusService, projects: projectService, handler: handler, router: router}
}

func (f *dailyFocusAPIFixture) project(userID uint, scene, status string) model.Project {
	f.t.Helper()
	project := model.Project{UserID: userID, Title: scene + " 项目", Status: status, Context: scene}
	if err := f.db.Create(&project).Error; err != nil {
		f.t.Fatal(err)
	}
	return project
}

func (f *dailyFocusAPIFixture) task(projectID uint, title, status string) model.ProjectTask {
	f.t.Helper()
	task := model.ProjectTask{ProjectID: projectID, Title: title, Status: status, Priority: "normal", SortOrder: 1024}
	if err := f.db.Create(&task).Error; err != nil {
		f.t.Fatal(err)
	}
	return task
}

func (f *dailyFocusAPIFixture) call(userID uint, method, path, body string) *httptest.ResponseRecorder {
	f.t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Test-User", strconv.FormatUint(uint64(userID), 10))
	f.router.ServeHTTP(recorder, request)
	return recorder
}

type dailyFocusEnvelope struct {
	Data *struct {
		Date     string                             `json:"date"`
		Context  string                             `json:"context"`
		Focuses  map[string]*service.DailyFocusView `json:"focuses"`
		Focus    *service.DailyFocusView            `json:"focus"`
		Revision int                                `json:"revision"`
	} `json:"data"`
	Error *struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
	} `json:"error"`
}

func decodeDailyFocusEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) dailyFocusEnvelope {
	t.Helper()
	var envelope dailyFocusEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	return envelope
}

func (f *dailyFocusAPIFixture) errorCode(recorder *httptest.ResponseRecorder) string {
	f.t.Helper()
	envelope := decodeDailyFocusEnvelope(f.t, recorder)
	if envelope.Error == nil {
		f.t.Fatalf("expected error envelope, got %s", recorder.Body.String())
	}
	return envelope.Error.Code
}

func (f *dailyFocusAPIFixture) put(userID uint, date, scene, body string) (*httptest.ResponseRecorder, dailyFocusEnvelope) {
	f.t.Helper()
	recorder := f.call(userID, "PUT", "/workspace/daily-focus/"+date+"/"+scene, body)
	return recorder, decodeDailyFocusEnvelope(f.t, recorder)
}

func TestDailyFocusGetAPI(t *testing.T) {
	f := newDailyFocusAPIFixture(t)
	workProject := f.project(1, "work", "active")
	task := f.task(workProject.ID, "写周报", "doing")

	for _, path := range []string{
		"/workspace/daily-focus",
		"/workspace/daily-focus?date=",
		"/workspace/daily-focus?date=2026-02-30",
		"/workspace/daily-focus?date=2026-10-21T00:00:00Z",
		"/workspace/daily-focus?date=20261021",
	} {
		if recorder := f.call(1, "GET", path, ""); recorder.Code != 400 || f.errorCode(recorder) != "DAILY_FOCUS_DATE_INVALID" {
			t.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
		}
	}
	if recorder := f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=unknown", ""); recorder.Code != 400 || f.errorCode(recorder) != "DAILY_FOCUS_CONTEXT_INVALID" {
		t.Fatalf("invalid context: %d %s", recorder.Code, recorder.Body.String())
	}

	// Default context is all and both scenes are present with null values.
	recorder := f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21", "")
	envelope := decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || envelope.Data == nil || envelope.Data.Date != "2026-10-21" || envelope.Data.Context != "all" {
		t.Fatalf("empty day: %d %s", recorder.Code, recorder.Body.String())
	}
	if len(envelope.Data.Focuses) != 2 {
		t.Fatalf("all scene must include both scenes: %s", recorder.Body.String())
	}
	if work, ok := envelope.Data.Focuses["work"]; !ok || work != nil {
		t.Fatalf("empty work focus: %s", recorder.Body.String())
	}
	if life, ok := envelope.Data.Focuses["life"]; !ok || life != nil {
		t.Fatalf("empty life focus: %s", recorder.Body.String())
	}
	// A concrete scene is trimmed to itself.
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=work", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || len(envelope.Data.Focuses) != 1 {
		t.Fatalf("trimmed scene: %d %s", recorder.Code, recorder.Body.String())
	}
	if _, exists := envelope.Data.Focuses["life"]; exists {
		t.Fatalf("life present in work view: %s", recorder.Body.String())
	}

	// After setting a focus the view carries the derived state and task facts.
	setRecorder, setEnvelope := f.put(1, "2026-10-21", "work", `{"task_id": `+strconv.FormatUint(uint64(task.ID), 10)+`}`)
	if setRecorder.Code != 200 || setEnvelope.Data == nil || setEnvelope.Data.Focus == nil ||
		setEnvelope.Data.Focus.State != "ok" || setEnvelope.Data.Focus.Title != "写周报" || setEnvelope.Data.Revision != 1 {
		t.Fatalf("set focus: %d %s", setRecorder.Code, setRecorder.Body.String())
	}
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=all", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || envelope.Data.Focuses["work"] == nil || envelope.Data.Focuses["work"].State != "ok" ||
		envelope.Data.Focuses["work"].ProjectTitle != workProject.Title || envelope.Data.Focuses["life"] != nil {
		t.Fatalf("all view after set: %d %s", recorder.Code, recorder.Body.String())
	}
	// The focus view exposes the referenced task's own updated_at so the client
	// can submit expected_updated_at when completing it.
	var storedTask model.ProjectTask
	if err := f.db.First(&storedTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Focuses["work"].TaskUpdatedAt == nil || !envelope.Data.Focuses["work"].TaskUpdatedAt.Equal(storedTask.UpdatedAt) {
		t.Fatalf("task_updated_at mismatch: %v vs %v (%s)", envelope.Data.Focuses["work"].TaskUpdatedAt, storedTask.UpdatedAt, recorder.Body.String())
	}
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=life", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || envelope.Data.Focuses["life"] != nil {
		t.Fatalf("life view leaks work focus: %d %s", recorder.Code, recorder.Body.String())
	}
	if !decoderHasKey(recorder, "life") {
		// The life key must be present (null) for a concrete scene request.
		t.Fatalf("life key missing: %s", recorder.Body.String())
	}
	// A deleted task turns the stored row into missing with no task version.
	if err := f.db.Exec("DELETE FROM project_tasks WHERE id = ?", task.ID).Error; err != nil {
		t.Fatal(err)
	}
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=work", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || envelope.Data.Focuses["work"] == nil || envelope.Data.Focuses["work"].State != "missing" ||
		envelope.Data.Focuses["work"].TaskUpdatedAt != nil {
		t.Fatalf("missing focus task_updated_at: %d %s", recorder.Code, recorder.Body.String())
	}
}

func decoderHasKey(recorder *httptest.ResponseRecorder, key string) bool {
	var raw struct {
		Data struct {
			Focuses map[string]json.RawMessage `json:"focuses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		return false
	}
	_, ok := raw.Data.Focuses[key]
	return ok
}

func TestDailyFocusPutValidationAndIdempotency(t *testing.T) {
	f := newDailyFocusAPIFixture(t)
	workProject := f.project(1, "work", "active")
	task := f.task(workProject.ID, "写周报", "doing")
	taskID := strconv.FormatUint(uint64(task.ID), 10)

	for _, tc := range []struct {
		name   string
		date   string
		scene  string
		body   string
		status int
		code   string
	}{
		{name: "invalid date", date: "2026-2-3", scene: "work", body: `{"task_id": 1}`, status: 400, code: "DAILY_FOCUS_DATE_INVALID"},
		{name: "invalid scene", date: "2026-10-21", scene: "all", body: `{"task_id": 1}`, status: 400, code: "DAILY_FOCUS_CONTEXT_INVALID"},
		{name: "unknown scene", date: "2026-10-21", scene: "unclassified", body: `{"task_id": 1}`, status: 400, code: "DAILY_FOCUS_CONTEXT_INVALID"},
		{name: "broken json", date: "2026-10-21", scene: "work", body: `not-json`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "empty body", date: "2026-10-21", scene: "work", body: ``, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "missing task_id", date: "2026-10-21", scene: "work", body: `{"expected_revision": 0}`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "string task_id", date: "2026-10-21", scene: "work", body: `{"task_id": "1"}`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "zero task_id", date: "2026-10-21", scene: "work", body: `{"task_id": 0}`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "fractional task_id", date: "2026-10-21", scene: "work", body: `{"task_id": 1.5}`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
		{name: "negative revision", date: "2026-10-21", scene: "work", body: `{"task_id": 1, "expected_revision": -1}`, status: 400, code: "DAILY_FOCUS_INPUT_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder, _ := f.put(1, tc.date, tc.scene, tc.body)
			if recorder.Code != tc.status || f.errorCode(recorder) != tc.code {
				t.Fatalf("%d %s", recorder.Code, recorder.Body.String())
			}
		})
	}

	// Clearing a day without focus succeeds and reports revision 0.
	recorder, envelope := f.put(1, "2026-10-21", "work", `{"task_id": null}`)
	if recorder.Code != 200 || envelope.Data == nil || envelope.Data.Focus != nil || envelope.Data.Revision != 0 {
		t.Fatalf("clear empty: %d %s", recorder.Code, recorder.Body.String())
	}

	// Set, idempotent re-set, replace and stale-revision conflict.
	recorder, envelope = f.put(1, "2026-10-21", "work", `{"task_id": `+taskID+`}`)
	if recorder.Code != 200 || envelope.Data.Revision != 1 || envelope.Data.Focus == nil || envelope.Data.Focus.State != "ok" {
		t.Fatalf("initial set: %d %s", recorder.Code, recorder.Body.String())
	}
	var putStored model.ProjectTask
	if err := f.db.First(&putStored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Focus.TaskUpdatedAt == nil || !envelope.Data.Focus.TaskUpdatedAt.Equal(putStored.UpdatedAt) {
		t.Fatalf("set response task_updated_at: %v vs %v", envelope.Data.Focus.TaskUpdatedAt, putStored.UpdatedAt)
	}
	recorder, envelope = f.put(1, "2026-10-21", "work", `{"task_id": `+taskID+`, "expected_revision": 1}`)
	if recorder.Code != 200 || envelope.Data.Revision != 1 {
		t.Fatalf("idempotent set: %d %s", recorder.Code, recorder.Body.String())
	}
	second := f.task(workProject.ID, "第二件事", "next")
	secondID := strconv.FormatUint(uint64(second.ID), 10)
	recorder, envelope = f.put(1, "2026-10-21", "work", `{"task_id": `+secondID+`, "expected_revision": 1}`)
	if recorder.Code != 200 || envelope.Data.Revision != 2 || envelope.Data.Focus.Title != "第二件事" {
		t.Fatalf("replace: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder, _ = f.put(1, "2026-10-21", "work", `{"task_id": `+taskID+`, "expected_revision": 1}`)
	if recorder.Code != 409 || f.errorCode(recorder) != "DAILY_FOCUS_CONFLICT" {
		t.Fatalf("stale revision: %d %s", recorder.Code, recorder.Body.String())
	}
	// The rejected write must not change the confirmed focus.
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=work", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if envelope.Data.Focuses["work"] == nil || envelope.Data.Focuses["work"].Title != "第二件事" {
		t.Fatalf("conflict changed focus: %s", recorder.Body.String())
	}
	// A revision that does not exist yet conflicts instead of creating a row.
	recorder, _ = f.put(1, "2026-10-22", "work", `{"task_id": `+taskID+`, "expected_revision": 4}`)
	if recorder.Code != 409 {
		t.Fatalf("expected revision without row: %d %s", recorder.Code, recorder.Body.String())
	}
	// Clearing with the current revision succeeds, then repeats idempotently.
	recorder, envelope = f.put(1, "2026-10-21", "work", `{"task_id": null, "expected_revision": 2}`)
	if recorder.Code != 200 || envelope.Data.Focus != nil || envelope.Data.Revision != 0 {
		t.Fatalf("clear: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder, envelope = f.put(1, "2026-10-21", "work", `{"task_id": null}`)
	if recorder.Code != 200 || envelope.Data.Focus != nil || envelope.Data.Revision != 0 {
		t.Fatalf("repeated clear: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDailyFocusCandidateAndOwnershipAPI(t *testing.T) {
	f := newDailyFocusAPIFixture(t)
	workProject := f.project(1, "work", "active")
	unclassified := f.project(1, "unclassified", "active")
	paused := f.project(1, "work", "paused")
	ownerTask := f.task(workProject.ID, "机密任务", "doing")
	unclassifiedTask := f.task(unclassified.ID, "未分类任务", "doing")
	pausedTask := f.task(paused.ID, "暂停项目任务", "doing")
	doneTask := f.task(workProject.ID, "已完成任务", "done")
	// A candidate from another user is rejected exactly like a missing task.
	foreignProject := f.project(2, "work", "active")
	foreignTask := f.task(foreignProject.ID, "他人任务", "doing")

	recorder, _ := f.put(1, "2026-10-21", "work", `{"task_id": `+strconv.FormatUint(uint64(foreignTask.ID), 10)+`}`)
	if recorder.Code != 400 || f.errorCode(recorder) != "DAILY_FOCUS_TASK_INVALID" {
		t.Fatalf("foreign candidate: %d %s", recorder.Code, recorder.Body.String())
	}
	foreignBody := recorder.Body.String()
	recorder, _ = f.put(1, "2026-10-21", "work", `{"task_id": 999999}`)
	if recorder.Code != 400 || recorder.Body.String() != foreignBody {
		t.Fatalf("foreign and missing candidates must be indistinguishable: %d %s vs %s", recorder.Code, recorder.Body.String(), foreignBody)
	}
	if bytes.Contains([]byte(foreignBody), []byte("他人任务")) || bytes.Contains([]byte(foreignBody), []byte("工作 项目")) {
		t.Fatalf("candidate error leaked task data: %s", foreignBody)
	}
	for name, taskID := range map[string]uint{
		"unclassified":   unclassifiedTask.ID,
		"paused project": pausedTask.ID,
		"done":           doneTask.ID,
	} {
		recorder, _ = f.put(1, "2026-10-21", "work", `{"task_id": `+strconv.FormatUint(uint64(taskID), 10)+`}`)
		if recorder.Code != 400 || f.errorCode(recorder) != "DAILY_FOCUS_TASK_INVALID" {
			t.Fatalf("%s candidate: %d %s", name, recorder.Code, recorder.Body.String())
		}
	}
	// Write: a user cannot see or mutate another user's focus.
	recorder, _ = f.put(1, "2026-10-21", "work", `{"task_id": `+strconv.FormatUint(uint64(ownerTask.ID), 10)+`}`)
	if recorder.Code != 200 {
		t.Fatalf("owner set: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder = f.call(2, "GET", "/workspace/daily-focus?date=2026-10-21&context=all", "")
	envelope := decodeDailyFocusEnvelope(t, recorder)
	if recorder.Code != 200 || envelope.Data.Focuses["work"] != nil {
		t.Fatalf("foreign read: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder, _ = f.put(2, "2026-10-21", "work", `{"task_id": `+strconv.FormatUint(uint64(ownerTask.ID), 10)+`}`)
	if recorder.Code != 400 || f.errorCode(recorder) != "DAILY_FOCUS_TASK_INVALID" {
		t.Fatalf("foreign write: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder = f.call(1, "GET", "/workspace/daily-focus?date=2026-10-21&context=work", "")
	envelope = decodeDailyFocusEnvelope(t, recorder)
	if envelope.Data.Focuses["work"] == nil || envelope.Data.Focuses["work"].Title != "机密任务" {
		t.Fatalf("owner focus disappeared: %s", recorder.Body.String())
	}
}

func TestWorkspaceHomeDailyFocusIntegration(t *testing.T) {
	f := newDailyFocusAPIFixture(t)
	workProject := f.project(1, "work", "active")
	lifeProject := f.project(1, "life", "active")
	workTask := f.task(workProject.ID, "工作焦点", "doing")
	lifeTask := f.task(lifeProject.ID, "生活焦点", "next")
	if _, err := f.focuses.Set(t.Context(), 1, "2026-10-21", "work", workTask.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.focuses.Set(t.Context(), 1, "2026-10-21", "life", lifeTask.ID, 0); err != nil {
		t.Fatal(err)
	}

	type homeEnvelope struct {
		Data  *service.WorkspaceHome `json:"data"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	home := func(query string) (*httptest.ResponseRecorder, homeEnvelope) {
		t.Helper()
		recorder := f.call(1, "GET", "/workspace/home?date=2026-10-21"+query, "")
		var envelope homeEnvelope
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode home: %v %s", err, recorder.Body.String())
		}
		return recorder, envelope
	}

	recorder, envelope := home("&context=all")
	if recorder.Code != 200 || envelope.Data == nil || len(envelope.Data.DailyFocus) != 2 ||
		envelope.Data.DailyFocus["work"] == nil || envelope.Data.DailyFocus["life"] == nil ||
		envelope.Data.DailyFocus["work"].Title != "工作焦点" || envelope.Data.DailyFocus["life"].Title != "生活焦点" {
		t.Fatalf("all home daily focus: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder, envelope = home("&context=work")
	if recorder.Code != 200 || len(envelope.Data.DailyFocus) != 1 || envelope.Data.DailyFocus["work"] == nil {
		t.Fatalf("work home daily focus: %d %s", recorder.Code, recorder.Body.String())
	}
	if _, exists := envelope.Data.DailyFocus["life"]; exists {
		t.Fatalf("life leaked into work home: %s", recorder.Body.String())
	}
	recorder, envelope = home("&context=life")
	if recorder.Code != 200 || len(envelope.Data.DailyFocus) != 1 || envelope.Data.DailyFocus["life"] == nil {
		t.Fatalf("life home daily focus: %d %s", recorder.Code, recorder.Body.String())
	}

	// A failing focus read only writes errors.daily_focus and keeps the rest.
	if err := f.db.Migrator().DropTable(&model.DailyFocus{}); err != nil {
		t.Fatal(err)
	}
	recorder, envelope = home("&context=all")
	if recorder.Code != 200 || envelope.Data == nil || envelope.Data.Errors["daily_focus"] == "" ||
		len(envelope.Data.DailyFocus) != 0 || envelope.Data.Context != "all" {
		t.Fatalf("focus failure leaked into home: %d %s", recorder.Code, recorder.Body.String())
	}
	if envelope.Data.Errors["projects"] != "" || envelope.Data.Errors["planning"] != "" || envelope.Data.Errors["learning"] != "" || envelope.Data.Errors["inbox"] != "" {
		t.Fatalf("focus failure poisoned other sections: %s", recorder.Body.String())
	}

	// The 503 still requires all five core sections to fail; a sixth focus
	// failure must not change that decision either way.
	if err := f.db.Migrator().DropTable(&model.InboxItem{}, &model.Course{}, &model.Project{}); err != nil {
		t.Fatal(err)
	}
	recorder, envelope = home("&context=all")
	if recorder.Code != 503 || envelope.Error == nil || envelope.Error.Code != "WORKSPACE_SECTIONS_UNAVAILABLE" {
		t.Fatalf("all sections failed: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDailyFocusRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "daily-focus-router.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.DailyFocus{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	focusService := service.NewDailyFocusService(repository.NewDailyFocusRepository(db))
	projectService := service.NewProjectService(repository.NewProjectRepository(db))
	handler := NewHandler(nil, nil, focusService, projectService)
	if handler.dailyFocus != focusService {
		t.Fatal("daily focus service was not injected")
	}
	router := NewRouter(config.Config{Environment: "development"}, handler, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}})
	// DevelopmentPrincipal authenticates as user 0.
	project := model.Project{UserID: 0, Title: "工作", Status: "active", Context: "work"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	task := model.ProjectTask{ProjectID: project.ID, Title: "注册任务", Status: "doing", Priority: "normal", SortOrder: 1024}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("PUT", "/api/v1/workspace/daily-focus/2026-10-21/work", bytes.NewBufferString(`{"task_id": `+strconv.FormatUint(uint64(task.ID), 10)+`}`)))
	if recorder.Code != 200 {
		t.Fatalf("PUT route: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/v1/workspace/daily-focus?date=2026-10-21", nil))
	if recorder.Code != 200 || !bytes.Contains(recorder.Body.Bytes(), []byte("注册任务")) {
		t.Fatalf("GET route: %d %s", recorder.Code, recorder.Body.String())
	}
}
