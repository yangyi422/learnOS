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

func TestLibraryAPICRUDValidationAndAuthentication(t *testing.T) {
	db, err := database.Open(config.Config{DatabasePath: filepath.Join(t.TempDir(), "library.db")})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	s := service.NewLibraryService(repository.NewLibraryRepository(db))
	h := NewHandler(nil, nil, s)
	web := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	router := NewRouter(config.Config{Environment: "development"}, h, web)
	call := func(method, path, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatal(method, path, res.Code, res.Body.String())
		}
		return res.Body.Bytes()
	}
	body := call("POST", "/api/v1/library/collections", `{"name":"测试专题","template":"wiki","display_mode":"wiki"}`, 200)
	var collection struct{ Data model.LibraryCollection }
	json.Unmarshal(body, &collection)
	base := fmt.Sprintf("/api/v1/library/collections/%d", collection.Data.ID)
	body = call("POST", base+"/pages", `{"title":"根目录页面","body":"中文表格"}`, 200)
	var page struct{ Data model.LibraryPage }
	json.Unmarshal(body, &page)
	pageURL := fmt.Sprintf("%s/pages/%d", base, page.Data.ID)
	call("GET", base, "", 200)
	call("GET", pageURL, "", 200)
	call("GET", base+"/pages?q=中文", "", 200)
	call("GET", base+"/links?ids=1,999", "", 200)
	call("GET", base+"/export", "", 200)
	for _, invalid := range []string{`{"title":"x","sources":[{"url":"javascript:alert(1)"}]}`, `{"title":"x","collection_id":999}`, `{"title":"x"} {}`} {
		call("POST", base+"/pages", invalid, 400)
	}
	call("GET", base+"/pages?section=oops", "", 400)
	call("DELETE", pageURL, `{"confirm":false,"version":1}`, 400)
	call("DELETE", pageURL, `{"confirm":true,"version":999}`, 409)
	call("DELETE", pageURL, `{"confirm":true,"version":1}`, 200)
	call("GET", pageURL, "", 404)
	protected := NewRouter(config.Config{Environment: "production"}, h, web)
	for _, item := range [][2]string{{"GET", "/api/v1/library/collections"}, {"POST", "/api/v1/library/collections"}, {"GET", base}, {"PUT", base}, {"DELETE", base}, {"GET", base + "/export"}, {"GET", base + "/links"}, {"GET", base + "/pages"}, {"POST", base + "/pages"}, {"GET", pageURL}, {"PUT", pageURL}, {"DELETE", pageURL}, {"POST", base + "/sections"}, {"PUT", base + "/sections/1"}, {"DELETE", base + "/sections/1"}} {
		res := httptest.NewRecorder()
		protected.ServeHTTP(res, httptest.NewRequest(item[0], item[1], strings.NewReader(`{}`)))
		if res.Code != 401 {
			t.Fatal(item, res.Code)
		}
	}
}
