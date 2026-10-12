package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/middleware"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
)

// Opt-in local measurements: inputs remain outside the repository and are never
// printed. Every run uses an independent database and the production session gate.
func TestLibraryImagePerformance(t *testing.T) {
	dir := os.Getenv("LIFEOS_IMAGE_PERF_DIR")
	if dir == "" {
		t.Skip("optional local image measurements")
	}
	cfg := config.Config{Environment: "production", DatabasePath: filepath.Join(t.TempDir(), "images.db")}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	user := model.User{Username: "image-test", DisplayName: "Image test", PasswordHash: "unused", Role: model.UserRoleUser, Status: model.UserStatusActive}
	if err = db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	token := "isolated-image-performance-session"
	sum := sha256.Sum256([]byte(token))
	session := model.Session{UserID: user.ID, TokenHash: hex.EncodeToString(sum[:]), ExpiresAt: time.Now().Add(time.Hour)}
	if err = db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	images := service.NewLibraryImageService(repository.NewLibraryImageRepository(db), cfg.DatabasePath, 0)
	users := service.NewUserService(repository.NewUserRepository(db))
	router := NewRouter(cfg, NewHandler(nil, nil, images), fstest.MapFS{"index.html": {Data: []byte("test")}}, users)
	server := httptest.NewServer(router)
	defer server.Close()
	for _, name := range []string{"sample-0", "sample-1"} {
		data, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		row, e := images.Upload(context.Background(), user.ID, name, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		t.Logf("%s source bytes=%d dimensions=%dx%d MIME=%s", name, row.Size, row.Width, row.Height, row.MIME)
		for _, variant := range []string{"", "?variant=card-v1"} {
			etag := ""
			for i := 0; i < 3; i++ {
				req, _ := http.NewRequest("GET", server.URL+row.URL+variant, nil)
				req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
				if i == 2 {
					req.Header.Set("If-None-Match", etag)
				}
				start := time.Now()
				res, e := server.Client().Do(req)
				if e != nil {
					t.Fatal(e)
				}
				n, e := io.Copy(io.Discard, res.Body)
				res.Body.Close()
				if e != nil {
					t.Fatal(e)
				}
				etag = res.Header.Get("ETag")
				want := http.StatusOK
				if i == 2 {
					want = http.StatusNotModified
				}
				if res.StatusCode != want || (i == 2 && n != 0) {
					t.Fatalf("unexpected conditional response: %d / %d bytes", res.StatusCode, n)
				}
				t.Logf("%s variant=%q request=%d status=%d body=%d elapsed=%s cache=%q MIME=%q etag_present=%t nosniff=%q content_length=%q transfer_encoding=%v CORP=%q CSP=%q disposition=%q", name, variant, i, res.StatusCode, n, time.Since(start), res.Header.Get("Cache-Control"), res.Header.Get("Content-Type"), etag != "", res.Header.Get("X-Content-Type-Options"), res.Header.Get("Content-Length"), res.TransferEncoding, res.Header.Get("Cross-Origin-Resource-Policy"), res.Header.Get("Content-Security-Policy"), res.Header.Get("Content-Disposition"))
			}
		}
	}
}
