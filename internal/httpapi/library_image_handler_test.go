package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/librarymedia"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
)

func TestLibraryImageAPIAuthenticationLimitsAndOrigin(t *testing.T) {
	cfg := config.Config{Environment: "development", DatabasePath: filepath.Join(t.TempDir(), "images.db")}
	db, e := database.Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	conn, _ := db.DB()
	defer conn.Close()
	images := service.NewLibraryImageService(repository.NewLibraryImageRepository(db), cfg.DatabasePath, 0)
	handler := NewHandler(nil, nil, images)
	web := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	router := NewRouter(cfg, handler, web)
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 3, 2)))
	upload := func(data []byte, origin string, want int) []byte {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("image", "fake.svg")
		part.Write(data)
		writer.Close()
		req := httptest.NewRequest("POST", "/api/v1/library/images", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatal(res.Code, res.Body.String())
		}
		return res.Body.Bytes()
	}
	response := upload(pngData.Bytes(), "http://example.com", 200)
	var result struct{ Data model.LibraryImage }
	if e = json.Unmarshal(response, &result); e != nil {
		t.Fatal(e)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", result.Data.URL, nil))
	if res.Code != 200 || res.Header().Get("Content-Type") != "image/png" || res.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(res.Header().Get("Cache-Control"), "private") || !bytes.Equal(res.Body.Bytes(), pngData.Bytes()) {
		t.Fatal(res.Code, res.Header())
	}
	for _, variant := range []string{"", "?variant=card-v1"} {
		first := httptest.NewRecorder()
		router.ServeHTTP(first, httptest.NewRequest("GET", result.Data.URL+variant, nil))
		etag := first.Header().Get("ETag")
		if first.Code != 200 || etag == "" || first.Header().Get("Cache-Control") != "private, no-cache" || first.Header().Get("Vary") != "Cookie, Authorization" {
			t.Fatal("unsafe or missing cache policy", first.Code, first.Header())
		}
		if variant != "" && first.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatal("thumbnail MIME")
		}
		for _, value := range []string{etag, "W/" + etag, `"unrelated", ` + etag, "*"} {
			req := httptest.NewRequest("GET", result.Data.URL+variant, nil)
			req.Header.Set("If-None-Match", value)
			cached := httptest.NewRecorder()
			router.ServeHTTP(cached, req)
			if cached.Code != 304 || cached.Body.Len() != 0 {
				t.Fatal("conditional GET", cached.Code)
			}
			private := NewRouter(config.Config{Environment: "production"}, handler, web)
			unauthorized := httptest.NewRecorder()
			private.ServeHTTP(unauthorized, req)
			if unauthorized.Code != 401 {
				t.Fatal("cache bypassed authentication", unauthorized.Code)
			}
		}
	}
	invalidVariant := httptest.NewRecorder()
	router.ServeHTTP(invalidVariant, httptest.NewRequest("GET", result.Data.URL+"?variant=50000", nil))
	if invalidVariant.Code != 400 {
		t.Fatal("unbounded thumbnail variant accepted")
	}
	upload(pngData.Bytes(), "https://evil.example.com", 403)
	upload([]byte("<svg onload='alert(1)'></svg>"), "", 400)
	upload(make([]byte, librarymedia.MaxBytes+1), "", 413)
	private := NewRouter(config.Config{Environment: "production"}, handler, web)
	for _, method := range []string{"GET", "POST"} {
		path := result.Data.URL
		if method == "POST" {
			path = "/api/v1/library/images"
		}
		res = httptest.NewRecorder()
		private.ServeHTTP(res, httptest.NewRequest(method, path, nil))
		if res.Code != 401 {
			t.Fatal("private resource exposed", method, res.Code)
		}
	}
	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest("GET", "/api/v1/library/images/not-valid", nil))
	if res.Code < 400 {
		t.Fatal("invalid ID")
	}
	res = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/library/images/cleanup", strings.NewReader(`{"confirm":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(res, req)
	if res.Code != 400 {
		t.Fatal("unconfirmed cleanup")
	}
}
