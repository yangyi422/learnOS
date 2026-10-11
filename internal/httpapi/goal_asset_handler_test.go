package httpapi

import (
	"context"
	"fmt"
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

func TestGoalAssetAPILifecycleAndAuthentication(t *testing.T) {
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "reference.db")})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	assetService := service.NewAssetService(repository.NewAssetRepository(db))
	a, err := assetService.Save(context.Background(), 0, 0, service.AssetInput{SnapshotDate: "2026-10-10", Bank: func() *string { v := "0"; return &v }()})
	if err != nil {
		t.Fatal(err)
	}
	goal := model.LifeGoal{UserID: 0, Title: "home", Status: "considering"}
	db.Create(&goal)
	h := NewHandler(nil, nil, assetService)
	web := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	router := NewRouter(config.Config{Environment: "development"}, h, web)
	call := func(method, path, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatal(method, path, res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	path := fmt.Sprintf("/api/v1/life/goals/%d/assets/%d", goal.ID, a.ID)
	list := fmt.Sprintf("/api/v1/life/goals/%d/assets", goal.ID)
	reverse := fmt.Sprintf("/api/v1/finance/assets/%d/goals", a.ID)
	body := `{"judgment_date":"2026-10-10","note":"financial context"}`
	call("POST", path, body, 200)
	call("POST", path, body, 200)
	result := call("GET", list, "", 200)
	if !strings.Contains(result, `"complete":false`) || !strings.Contains(result, `"platform_minor":null`) {
		t.Fatal(result)
	}
	call("GET", reverse, "", 200)
	call("PUT", path, `{"judgment_date":"2026-10-09","note":"updated","version":1}`, 200)
	call("DELETE", fmt.Sprintf("/api/v1/finance/assets/%d?version=1", a.ID), "", 409)
	for _, bad := range []string{`{"judgment_date":"2026-02-30"}`, `{"judgment_date":"2026-10-10","amount":"10"}`, body + body} {
		call("POST", path, bad, 400)
	}
	call("POST", "/api/v1/life/goals/999/assets/999", body, 404)
	call("DELETE", path+"?version=1", "", 409)
	call("DELETE", path+"?version=2", "", 200)
	call("GET", fmt.Sprintf("/api/v1/finance/assets/%d", a.ID), "", 200)
	protected := NewRouter(config.Config{Environment: "production"}, h, web)
	for _, tc := range [][2]string{{"GET", list}, {"GET", reverse}, {"POST", path}, {"PUT", path}, {"DELETE", path + "?version=2"}} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest(tc[0], tc[1], strings.NewReader(body)))
		if res.Code != 401 {
			t.Fatal("missing authentication", tc, res.Code)
		}
	}
}
