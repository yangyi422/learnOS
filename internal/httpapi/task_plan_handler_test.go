package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"learnos/internal/auth"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestTaskPlanAPINullQueryAndOwnership(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	if err = db.AutoMigrate(&model.Project{}, &model.ProjectTask{}); err != nil {
		t.Fatal(err)
	}
	svc := service.NewProjectService(repository.NewProjectRepository(db))
	p := model.Project{ID: 1, UserID: 1, Title: "work", Status: "active", Context: "work"}
	db.Create(&p)
	life := model.Project{ID: 2, UserID: 1, Title: "life", Status: "active", Context: "life"}
	db.Create(&life)
	foreign := model.Project{ID: 3, UserID: 2, Title: "private", Status: "active", Context: "work"}
	db.Create(&foreign)
	plan := "2026-10-20"
	for _, project := range []uint{2, 2, 3, 1, 1} {
		if _, err = svc.CreateTaskWithInput(context.Background(), map[uint]uint{1: 1, 2: 1, 3: 2}[project], service.ProjectTaskInput{ProjectID: project, Title: "candidate", PlanDate: &plan}); err != nil {
			t.Fatal(err)
		}
	}
	h := &Handler{projects: svc}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), auth.Principal{UserID: 1}))
		c.Next()
	})
	router.GET("/tasks", h.ListTasks)
	router.POST("/tasks", h.CreateTask)
	router.PATCH("/tasks/:id", h.UpdateTask)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(r, req)
		return r
	}
	r := call("GET", "/tasks?context=work&actionable=true&plan_date=2026-10-20&limit=1&offset=1", "")
	var rows struct {
		Data []model.ProjectTask `json:"data"`
	}
	json.Unmarshal(r.Body.Bytes(), &rows)
	if r.Code != 200 || len(rows.Data) != 1 || rows.Data[0].ProjectID != 1 || rows.Data[0].ID != 5 || *rows.Data[0].PlanDate != plan {
		t.Fatal(r.Code, r.Body.String())
	}
	for _, query := range []string{"context=unknown", "plan_date=2026-02-30", "actionable=maybe", "limit=501"} {
		if r = call("GET", "/tasks?"+query, ""); r.Code != 400 {
			t.Fatal(query, r.Code)
		}
	}
	r = call("POST", "/tasks", `{"project_id":1,"title":"new","plan_date":"2026-10-20","due_date":"2026-11-01"}`)
	var created struct {
		Data model.ProjectTask `json:"data"`
	}
	json.Unmarshal(r.Body.Bytes(), &created)
	if r.Code != 201 || created.Data.PlanDate == nil {
		t.Fatal(r.Code, r.Body.String())
	}
	r = call("PATCH", "/tasks/6", `{"plan_date":null}`)
	var result struct {
		Data model.ProjectTask `json:"data"`
	}
	json.Unmarshal(r.Body.Bytes(), &result)
	if r.Code != 200 || result.Data.PlanDate != nil || *result.Data.DueDate != "2026-11-01" {
		t.Fatal(r.Code, r.Body.String())
	}
	r = call("PATCH", "/tasks/6", `{"plan_date":"2026-10-21"}`)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	r = call("PATCH", "/tasks/6", `{"title":"keep date"}`)
	json.Unmarshal(r.Body.Bytes(), &result)
	if r.Code != 200 || result.Data.PlanDate == nil || *result.Data.PlanDate != "2026-10-21" {
		t.Fatal(r.Body.String())
	}
	if r = call("PATCH", "/tasks/3", `{"plan_date":null}`); r.Code != 404 {
		t.Fatal("foreign owner", r.Code)
	}
	if r = call("PATCH", "/tasks/6", `{"plan_date":42}`); r.Code != 400 {
		t.Fatal("invalid shape", r.Code)
	}
	if r = call("PATCH", "/tasks/6", `{"plan_date":"2026-10-21T00:00:00Z"}`); r.Code != 400 {
		t.Fatal("invalid date", r.Code)
	}
}
