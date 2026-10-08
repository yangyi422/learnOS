package httpapi

import (
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLifeAPIAndAuthentication(t *testing.T) {
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "life.db")})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	defer sql.Close()
	life := service.NewLifeService(repository.NewLifeRepository(db))
	h := NewHandler(nil, nil, life)
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}
	router := NewRouter(config.Config{Environment: "development"}, h, web)
	call := func(method, path, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	call("POST", "/api/v1/life/events", `{"title":" ","occurred_on":"2026-10-08"}`, 400)
	event := `{"creation_key":"api-event","title":"搬家","occurred_on":"2020-03-02","primary_domain":"home","secondary_domain":"work","milestone":true}`
	call("POST", "/api/v1/life/events", event, 201)
	call("POST", "/api/v1/life/events", event, 201)
	if got := call("GET", "/api/v1/life/events?domain=work&milestones=true", "", 200); !strings.Contains(got, "搬家") {
		t.Fatal(got)
	}
	call("GET", "/api/v1/life/events?domain=unknown", "", 400)
	call("GET", "/api/v1/life/events?milestones=yes", "", 400)
	call("PUT", "/api/v1/life/events/1", `{"title":"搬家纪念","occurred_on":"2020-03-03"}`, 200)
	call("GET", "/api/v1/life/events/1", "", 200)
	call("POST", "/api/v1/life/goals", `{"creation_key":"goal","title":"自己的家","status":"considering"}`, 200)
	call("PUT", "/api/v1/life/goals/1", `{"title":"自己的家","status":"active","status_date":"2026-10-08","status_reason":"开始准备"}`, 200)
	call("POST", "/api/v1/life/goals/1/entries", `{"creation_key":"entry","occurred_on":"2026-09-02","content":"考虑通勤"}`, 201)
	if got := call("GET", "/api/v1/life/goals/1", "", 200); !strings.Contains(got, "开始准备") || !strings.Contains(got, "考虑通勤") {
		t.Fatal(got)
	}
	foreign := model.LifeEvent{UserID: 999, Title: "private", OccurredOn: "2020-01-01"}
	db.Create(&foreign)
	call("GET", "/api/v1/life/events/2", "", 404)
	item := model.InboxItem{UserID: 0, Content: "原文", Status: "inbox", SourceType: "manual"}
	db.Create(&item)
	input := `{"creation_key":"inbox","title":"重逢","description":"原文","occurred_on":"2026-10-08"}`
	call("POST", "/api/v1/inbox/1/convert-to-life-event", input, 201)
	call("POST", "/api/v1/inbox/1/convert-to-life-event", input, 201)
	db.First(&item, 1)
	if item.Status != "processed" || item.ProcessedToType != "life_event" {
		t.Fatal(item)
	}
	call("DELETE", "/api/v1/life/events/1", "", 200)
	call("GET", "/api/v1/life/events/1", "", 404)
	protected := NewRouter(config.Config{Environment: "production", Username: "test", PasswordHash: "unused"}, h, web)
	for _, path := range []string{"/api/v1/life/events", "/api/v1/life/goals", "/api/v1/life/sources/project/1"} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 401 {
			t.Fatalf("unauthorized %s returned %d", path, res.Code)
		}
	}
}
