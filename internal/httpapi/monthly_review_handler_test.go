package httpapi

import (
	"encoding/json"
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

func TestMonthlyReviewAPIAuthenticationImmutableFactsAndValidation(t *testing.T) {
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "reviews.db")})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	s := service.NewMonthlyReviewService(repository.NewMonthlyReviewRepository(db))
	h := NewHandler(nil, nil, s)
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
	call("GET", "/api/v1/life/reviews/2026-01", "", 200)
	reply := call("POST", "/api/v1/life/reviews/2026-01/preview", "", 200)
	var draft struct {
		Data service.ReviewDocument `json:"data"`
	}
	if err = json.Unmarshal([]byte(reply), &draft); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"facts_id":%d,"version":0,"thoughts":"个人心得"}`, draft.Data.FactsID)
	call("PUT", "/api/v1/life/reviews/2026-01", body, 200)
	call("PUT", "/api/v1/life/reviews/2026-01", body, 200)
	call("GET", "/api/v1/life/reviews", "", 200)
	call("GET", fmt.Sprintf("/api/v1/life/reviews/2026-01/history/%d", draft.Data.FactsID), "", 200)
	for _, bad := range []string{`{"facts_id":1,"facts":{"income":999}}`, body + body, `{"facts_id":0}`, `{"facts_id":1,"external_url":"javascript:alert(1)"}`} {
		call("PUT", "/api/v1/life/reviews/2026-01", bad, 400)
	}
	call("POST", "/api/v1/life/reviews/2026-13/preview", "", 400)
	call("GET", "/api/v1/life/reviews/2026-01/history/99999", "", 404)
	var count int64
	db.Model(&model.MonthlyReview{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	protected := NewRouter(config.Config{Environment: "production"}, h, web)
	for _, tc := range [][2]string{{"GET", "/api/v1/life/reviews"}, {"GET", "/api/v1/life/reviews/2026-01"}, {"POST", "/api/v1/life/reviews/2026-01/preview"}, {"PUT", "/api/v1/life/reviews/2026-01"}, {"GET", "/api/v1/life/reviews/2026-01/history/1"}} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest(tc[0], tc[1], strings.NewReader(body)))
		if res.Code != 401 {
			t.Fatal(tc, res.Code)
		}
	}
}
