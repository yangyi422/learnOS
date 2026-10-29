package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

// newToolStateAPIFixture mounts the real router with the signed-in user injected
// as user 1, so the handler, service, repository and table are all exercised.
func newToolStateAPIFixture(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tool-state-api.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ToolState{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	handler := &Handler{toolStates: service.NewToolStateService(repository.NewToolStateRepository(db))}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), auth.Principal{UserID: 1, Username: "tester", Role: "admin"}))
		c.Next()
	})
	api := router.Group("/api/v1")
	api.GET("/library-tools/:key/state", handler.GetToolState)
	api.PUT("/library-tools/:key/state", handler.SaveToolState)
	return router
}

func TestToolStateAPIReportsMissingThenSavesWithRevision(t *testing.T) {
	router := newToolStateAPIFixture(t)

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/library-tools/fefw-planner/state", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("empty GET status = %d, want 404", missing.Code)
	}

	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPut, "/api/v1/library-tools/fefw-planner/state",
		bytes.NewBufferString(`{"state":{"version":1,"plans":{"kai":{"c1":{"status":"main"}}}},"revision":0}`)))
	if create.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", create.Code, create.Body.String())
	}

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/library-tools/fefw-planner/state", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("GET status = %d", read.Code)
	}
	if body := read.Body.String(); !bytes.Contains([]byte(body), []byte(`"revision":1`)) {
		t.Fatalf("GET body = %s", body)
	}

	stale := httptest.NewRecorder()
	router.ServeHTTP(stale, httptest.NewRequest(http.MethodPut, "/api/v1/library-tools/fefw-planner/state",
		bytes.NewBufferString(`{"state":{"version":1,"plans":{}},"revision":0}`)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale PUT status = %d, want 409", stale.Code)
	}
}

func TestToolStateAPIRejectsUnknownToolAndBadPayload(t *testing.T) {
	router := newToolStateAPIFixture(t)

	unknown := httptest.NewRecorder()
	router.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/api/v1/library-tools/other/state", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown tool status = %d, want 404", unknown.Code)
	}

	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/api/v1/library-tools/fefw-planner/state",
		bytes.NewBufferString(`{"state":"not-an-object","revision":0}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid payload status = %d, want 400", invalid.Code)
	}
}

// The production router must mount the route behind authentication.
func TestRouterMountsToolStateUnderAuth(t *testing.T) {
	router := NewRouter(config.Config{Environment: "production"}, NewHandler(nil, nil), fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok")},
	})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/library-tools/fefw-planner/state", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", res.Code)
	}
}
