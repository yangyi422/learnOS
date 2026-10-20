package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestWorkspaceHomeContextAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "home.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	if err = db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.Course{}, &model.InboxItem{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewProjectRepository(db)
	projects := service.NewProjectService(repo)
	for i := 0; i < 8; i++ {
		scene := "life"
		if i == 0 {
			scene = "work"
		}
		if i == 1 {
			scene = "unclassified"
		}
		p := model.Project{UserID: 1, Title: fmt.Sprint(i), Status: "active", Context: scene}
		if err = db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		if _, err = projects.CreateTaskWithInput(context.Background(), 1, service.ProjectTaskInput{ProjectID: p.ID, Title: "task " + scene, Status: "doing"}); err != nil {
			t.Fatal(err)
		}
	}
	ws := service.NewWorkspaceService(projects, service.NewCourseService(repository.NewCourseRepository(db), nil), nil, service.NewInboxService(repository.NewInboxRepository(db), repo))
	h := &Handler{workspace: ws}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), auth.Principal{UserID: 1}))
		c.Next()
	})
	router.GET("/home", h.GetWorkspaceHome)
	for _, tc := range []struct {
		query string
		count int
		kind  string
	}{{"", 5, "task"}, {"&context=all", 5, "task"}, {"&context=work", 1, "task"}, {"&context=life", 5, "task"}} {
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest("GET", "/home?date=2026-10-20"+tc.query, nil))
		var body struct {
			Data service.WorkspaceHome `json:"data"`
		}
		if err = json.Unmarshal(r.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if r.Code != 200 || len(body.Data.Projects) != tc.count || body.Data.Focus.Kind != tc.kind {
			t.Fatalf("%s: %d %s", tc.query, r.Code, r.Body.String())
		}
		if tc.query == "&context=work" && (body.Data.Focus.Title != "task work" || len(body.Data.Today.Doing) != 1) {
			t.Fatal("cross-context task")
		}
	}
	r := httptest.NewRecorder()
	router.ServeHTTP(r, httptest.NewRequest("GET", "/home?context=unknown", nil))
	if r.Code != 400 {
		t.Fatal(r.Code)
	}
	// Changing the only work project removes its task from work, without fallback.
	if err = db.Model(&model.Project{}).Where("context = ?", "work").Update("context", "life").Error; err != nil {
		t.Fatal(err)
	}
	view, err := ws.Home(context.Background(), 1, "2026-10-20", "work")
	if err != nil || len(view.Projects) != 0 || view.Focus.Kind != "empty" {
		t.Fatal(view, err)
	}
	// Secondary data failure must not discard scoped projects/tasks.
	if err = db.Migrator().DropTable(&model.InboxItem{}); err != nil {
		t.Fatal(err)
	}
	view, err = ws.Home(context.Background(), 1, "2026-10-20", "life")
	if err != nil || len(view.Projects) != 5 || view.Errors["inbox"] == "" || view.Focus.Kind != "task" {
		t.Fatal(view, err)
	}
}
