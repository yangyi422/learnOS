package httpapi

import (
	"learnos/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestWorkspaceRecentRequiresAuthentication(t *testing.T) {
	router := NewRouter(config.Config{Environment: "production", Username: "admin", PasswordHash: testPasswordHash}, NewHandler(nil, nil), fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/workspace/recent", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("private recent API status=%d", recorder.Code)
	}
}
