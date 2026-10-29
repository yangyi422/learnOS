package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/middleware"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type captureAPIFixture struct {
	t      *testing.T
	router *gin.Engine
	tokens *service.APITokenService
	db     *gorm.DB
}

// newCaptureAPIFixture mirrors the production capture chain (bearer token →
// scope guard → handler) with real repositories and a real hashed token.
func newCaptureAPIFixture(t *testing.T) *captureAPIFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "capture.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.APIToken{}, &model.InboxItem{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	user := model.User{Username: "owner", DisplayName: "Owner", PasswordHash: "x", Role: model.UserRoleAdmin, Status: model.UserStatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	users := repository.NewUserRepository(db)
	userService := service.NewUserService(users)
	tokens := service.NewAPITokenService(repository.NewAPITokenRepository(db), users)
	handler := &Handler{
		inbox:     service.NewInboxService(repository.NewInboxRepository(db), repository.NewProjectRepository(db)),
		apiTokens: tokens,
	}
	router := gin.New()
	capture := router.Group("/api/v1/capture")
	capture.Use(
		middleware.TokenOrSession(userService.AuthenticateSession, handler.AuthenticateAPIToken),
		middleware.RequireScope(model.TokenScopeCaptureInbox),
	)
	capture.POST("/inbox", handler.CaptureInboxItem)
	return &captureAPIFixture{t: t, router: router, tokens: tokens, db: db}
}

func (f *captureAPIFixture) post(token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capture/inbox", bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	return res
}

func TestCaptureWithTokenCreatesInboxItem(t *testing.T) {
	f := newCaptureAPIFixture(t)
	created, err := f.tokens.Create(t.Context(), 1, "plugin", 30)
	if err != nil {
		t.Fatal(err)
	}

	res := f.post(created.Token, `{"content":"  记下一个想法  ","capture_key":"utools-abc123"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var item model.InboxItem
	if err := f.db.Where("user_id = 1").First(&item).Error; err != nil {
		t.Fatalf("inbox row missing: %v", err)
	}
	if item.Content != "记下一个想法" || item.Status != model.InboxStatusInbox || item.SourceType != model.InboxSourceManual {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.CaptureKey == nil || *item.CaptureKey != "utools-abc123" {
		t.Fatalf("capture key not stored: %+v", item.CaptureKey)
	}
}

func TestCaptureStoresURLAsURLSource(t *testing.T) {
	f := newCaptureAPIFixture(t)
	created, err := f.tokens.Create(t.Context(), 1, "plugin", 30)
	if err != nil {
		t.Fatal(err)
	}
	if res := f.post(created.Token, `{"content":"https://example.com/a?b=1","capture_key":"utools-url1"}`); res.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var item model.InboxItem
	if err := f.db.Where("user_id = 1").First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.SourceType != model.InboxSourceURL || item.SourceURL != "https://example.com/a?b=1" {
		t.Fatalf("url not preserved: %+v", item)
	}
}

func TestCaptureIsIdempotentPerKeyAndRejectsReuse(t *testing.T) {
	f := newCaptureAPIFixture(t)
	created, err := f.tokens.Create(t.Context(), 1, "plugin", 30)
	if err != nil {
		t.Fatal(err)
	}

	first := f.post(created.Token, `{"content":"同一条","capture_key":"utools-retry1"}`)
	second := f.post(created.Token, `{"content":"同一条","capture_key":"utools-retry1"}`)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("retry status = %d/%d", first.Code, second.Code)
	}
	var count int64
	if err := f.db.Model(&model.InboxItem{}).Where("user_id = 1").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("retry created %d rows, want 1", count)
	}

	// Same key with different content must not silently overwrite.
	if res := f.post(created.Token, `{"content":"另一条","capture_key":"utools-retry1"}`); res.Code != http.StatusConflict {
		t.Fatalf("conflicting reuse status = %d body = %s", res.Code, res.Body.String())
	}
}

func TestCaptureRejectsMissingInvalidAndRevokedCredentials(t *testing.T) {
	f := newCaptureAPIFixture(t)
	created, err := f.tokens.Create(t.Context(), 1, "plugin", 30)
	if err != nil {
		t.Fatal(err)
	}

	if res := f.post("", `{"content":"x","capture_key":"k1"}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", res.Code)
	}
	if res := f.post("los_pat_notarealtoken", `{"content":"x","capture_key":"k1"}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("bogus token status = %d", res.Code)
	}

	if err := f.tokens.Revoke(t.Context(), 1, created.ID); err != nil {
		t.Fatal(err)
	}
	if res := f.post(created.Token, `{"content":"x","capture_key":"k1"}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d", res.Code)
	}

	var count int64
	if err := f.db.Model(&model.InboxItem{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rejected requests created %d rows", count)
	}
}

func TestCaptureRejectsEmptyAndOversizedContent(t *testing.T) {
	f := newCaptureAPIFixture(t)
	created, err := f.tokens.Create(t.Context(), 1, "plugin", 30)
	if err != nil {
		t.Fatal(err)
	}
	if res := f.post(created.Token, `{"content":"   ","capture_key":"k-empty"}`); res.Code != http.StatusBadRequest {
		t.Fatalf("empty content status = %d", res.Code)
	}
	if res := f.post(created.Token, `{"content":"x","capture_key":"bad key with spaces"}`); res.Code != http.StatusBadRequest {
		t.Fatalf("invalid capture key status = %d", res.Code)
	}
}

// The production router must expose the capture path and still refuse anonymous
// callers, while keeping the rest of the API session-only.
func TestRouterMountsCaptureAndTokenRoutes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "router.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.APIToken{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	users := repository.NewUserRepository(db)
	userService := service.NewUserService(users)
	router := NewRouter(config.Config{Environment: "production"}, NewHandler(nil, nil), fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok")},
	}, userService)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/capture/inbox", bytes.NewBufferString(`{"content":"x"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous capture status = %d, want 401", res.Code)
	}
	tokens := httptest.NewRecorder()
	router.ServeHTTP(tokens, httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil))
	if tokens.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous token list status = %d, want 401", tokens.Code)
	}
}
