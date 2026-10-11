package httpapi

import (
	"encoding/json"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAssetAPIValidationAuthenticationAndLifecycle(t *testing.T) {
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "assets.db")})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	h := NewHandler(nil, nil, service.NewAssetService(repository.NewAssetRepository(db)))
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
	call("GET", "/api/v1/finance/assets", "", 200)
	body := `{"snapshot_date":"2025-12-31","bank":"0.10","platform":"0.20","investment":"0","other":"0","debt":"1"}`
	response := call("POST", "/api/v1/finance/assets", body, 200)
	var saved struct {
		Data service.AssetView `json:"data"`
	}
	if err = json.Unmarshal([]byte(response), &saved); err != nil || saved.Data.NetAsset == nil || *saved.Data.NetAsset != -70 {
		t.Fatal(response, err)
	}
	call("POST", "/api/v1/finance/assets", body, 409)
	for _, bad := range []string{`{"snapshot_date":"2026-02-30","bank":"1"}`, `{"snapshot_date":"2026-01-01"}`, `{"snapshot_date":"2026-01-01","bank":12.34}`, `{"snapshot_date":"2026-01-01","bank":"-1"}`, `{"snapshot_date":"2026-01-01","bank":"1","currency":"USD"}`, body + body} {
		call("POST", "/api/v1/finance/assets", bad, 400)
	}
	call("GET", "/api/v1/finance/assets/1", "", 200)
	call("PUT", "/api/v1/finance/assets/1", `{"snapshot_date":"2026-01-01","bank":"1","version":1}`, 200)
	response = call("GET", "/api/v1/finance/assets/1", "", 200)
	if !strings.Contains(response, `"debt_minor":null`) || !strings.Contains(response, `"net_asset_minor":null`) || !strings.Contains(response, `"complete":false`) {
		t.Fatal("null scope lost", response)
	}
	call("PUT", "/api/v1/finance/assets/1", `{"snapshot_date":"2026-01-01","bank":"2","version":1}`, 409)
	call("DELETE", "/api/v1/finance/assets/1?version=1", "", 409)
	call("DELETE", "/api/v1/finance/assets/1?version=2", "", 200)
	call("GET", "/api/v1/finance/assets/1", "", 404)
	protected := NewRouter(config.Config{Environment: "production"}, h, web)
	for _, tc := range []struct{ method, path string }{{"GET", "/api/v1/finance/assets"}, {"POST", "/api/v1/finance/assets"}, {"GET", "/api/v1/finance/assets/1"}, {"PUT", "/api/v1/finance/assets/1"}, {"DELETE", "/api/v1/finance/assets/1?version=1"}} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, strings.NewReader(body)))
		if res.Code != 401 {
			t.Fatal("unprotected", tc, res.Code)
		}
	}
}
