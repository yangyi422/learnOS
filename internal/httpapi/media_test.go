package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"learnos/internal/config"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
)

func TestMediaAPIUpsertPreservesPersonalFields(t *testing.T) {
	f := newCaptureAPIFixture(t)
	if err := f.db.AutoMigrate(&model.MediaItem{}); err != nil {
		t.Fatal(err)
	}
	h := &Handler{apiTokens: f.tokens, media: service.NewMediaService(repository.NewMediaRepository(f.db))}
	router := NewRouter(config.Config{}, h, fstest.MapFS{})
	mediaToken, err := f.tokens.CreateMedia(t.Context(), 1, "media fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	captureToken, err := f.tokens.Create(t.Context(), 1, "capture fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	request := func(token, title string, want int) {
		t.Helper()
		body := `[{"Source":"calibre","SourceID":"fixture","ExternalID":"one","Type":"book","Title":"` + title + `"}]`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/library/media-agent/items", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("status=%d want=%d body=%s", res.Code, want, res.Body)
		}
	}
	request(captureToken.Token, "初版", 403)
	request(mediaToken.Token, "初版", 200)
	if err := f.db.Model(&model.MediaItem{}).Where("external_id = ?", "one").Updates(map[string]any{"favorite": true, "notes": "读书笔记"}).Error; err != nil {
		t.Fatal(err)
	}
	request(mediaToken.Token, "修订版", 200)
	var items []model.MediaItem
	if err := f.db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "修订版" || !items[0].Favorite || items[0].Notes != "读书笔记" {
		t.Fatalf("unexpected media: %+v", items)
	}
}

func TestMediaVisibilityBeforePagination(t *testing.T) {
	f := newCaptureAPIFixture(t)
	if err := f.db.AutoMigrate(&model.MediaItem{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewMediaRepository(f.db)
	items := []model.MediaItem{
		{UserID: 1, Source: "calibre", ExternalID: "book", Type: "book"},
		{UserID: 1, Source: "jellyfin", ExternalID: "movie", Type: "movie", Library: "电影"},
		{UserID: 1, Source: "jellyfin", ExternalID: "restricted", Type: "movie", Library: "R18"},
		{UserID: 1, Source: "jellyfin", ExternalID: "legacy", Type: "movie"},
	}
	if err := repo.Upsert(t.Context(), items); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		show bool
		want int
	}{{false, 2}, {true, 4}} {
		rows, total, err := repo.List(t.Context(), 1, "", "", "", 1, test.show)
		if err != nil || len(rows) != test.want || total != int64(test.want) {
			t.Fatalf("show=%v: rows=%d total=%d err=%v", test.show, len(rows), total, err)
		}
	}
}
