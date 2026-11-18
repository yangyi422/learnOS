package httpapi

import (
	"encoding/json"
	"fmt"
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

func TestMediaBatchDeleteValidatesSelectionAndOwnership(t *testing.T) {
	tooMany := make([]uint, 101)
	for i := range tooMany {
		tooMany[i] = uint(i + 1)
	}
	largeBody, _ := json.Marshal(map[string]any{"ids": tooMany})
	for _, tc := range []struct {
		name, body        string
		status, remaining int
	}{
		{"selected movies", `{"ids":[1,2]}`, 200, 3},
		{"one movie", `{"ids":[2]}`, 200, 4},
		{"empty", `{"ids":[]}`, 400, 5},
		{"missing field", `{}`, 400, 5},
		{"null", `{"ids":null}`, 400, 5},
		{"duplicate", `{"ids":[1,1]}`, 400, 5},
		{"zero", `{"ids":[0]}`, 400, 5},
		{"negative", `{"ids":[-1]}`, 400, 5},
		{"string id", `{"ids":["1"]}`, 400, 5},
		{"fraction", `{"ids":[1.5]}`, 400, 5},
		{"invalid json", `{"ids":[`, 400, 5},
		{"too many", string(largeBody), 400, 5},
		{"oversized body", `{"ids":[1],"padding":"` + strings.Repeat("x", 9000) + `"}`, 400, 5},
		{"unknown rolls back whole batch", `{"ids":[1,99999]}`, 404, 5},
		{"foreign owner rolls back whole batch", `{"ids":[1,4]}`, 404, 5},
		{"non movie rolls back whole batch", `{"ids":[1,5]}`, 404, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCaptureAPIFixture(t)
			if err := f.db.AutoMigrate(&model.MediaItem{}); err != nil {
				t.Fatal(err)
			}
			rows := []model.MediaItem{
				{ID: 1, UserID: 0, Type: "movie", Source: "jellyfin", ExternalID: "one", Favorite: true, Notes: "笔记"},
				{ID: 2, UserID: 0, Type: "movie", Source: "jellyfin", ExternalID: "two"},
				{ID: 3, UserID: 0, Type: "movie", Source: "jellyfin", ExternalID: "keep"},
				{ID: 4, UserID: 2, Type: "movie", Source: "jellyfin", ExternalID: "foreign"},
				{ID: 5, UserID: 0, Type: "book", Source: "calibre", ExternalID: "book"},
			}
			if err := f.db.Create(&rows).Error; err != nil {
				t.Fatal(err)
			}
			h := &Handler{media: service.NewMediaService(repository.NewMediaRepository(f.db))}
			router := NewRouter(config.Config{Environment: "development"}, h, fstest.MapFS{})
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/library/media/batch-delete", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body)
			}
			var remaining int64
			if err := f.db.Model(&model.MediaItem{}).Count(&remaining).Error; err != nil || remaining != int64(tc.remaining) {
				t.Fatalf("remaining=%d want=%d err=%v", remaining, tc.remaining, err)
			}
			if tc.status == 200 {
				for _, id := range []uint{3, 4, 5} {
					var row model.MediaItem
					if err := f.db.First(&row, id).Error; err != nil {
						t.Fatalf("unselected row %d removed: %v", id, err)
					}
				}
				var response struct {
					Data struct {
						Deleted int `json:"deleted"`
					} `json:"data"`
				}
				if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || response.Data.Deleted != 5-tc.remaining {
					t.Fatalf("wrong deleted count: %s", res.Body)
				}
			}
		})
	}
}

func TestMediaBatchDeleteRequiresWebAuthentication(t *testing.T) {
	f := newCaptureAPIFixture(t)
	h := &Handler{media: service.NewMediaService(repository.NewMediaRepository(f.db)), apiTokens: f.tokens}
	router := NewRouter(config.Config{Environment: "production"}, h, fstest.MapFS{})
	token, err := f.tokens.CreateMedia(t.Context(), 1, "sync only", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, credential := range []string{"", "Bearer " + token.Token} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/library/media/batch-delete", strings.NewReader(`{"ids":[1]}`))
		req.Header.Set("Authorization", credential)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated delete status=%d", res.Code)
		}
	}
}

func TestMediaBatchDeleteRollsBackDatabaseErrors(t *testing.T) {
	f := newCaptureAPIFixture(t)
	if err := f.db.AutoMigrate(&model.MediaItem{}); err != nil {
		t.Fatal(err)
	}
	for id := uint(1); id <= 2; id++ {
		if err := f.db.Create(&model.MediaItem{ID: id, UserID: 0, Type: "movie", ExternalID: fmt.Sprint(id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := f.db.Exec(`CREATE TRIGGER reject_movie_delete BEFORE DELETE ON media_items WHEN OLD.id = 2 BEGIN SELECT RAISE(ABORT, 'secret-internal-marker'); END`).Error; err != nil {
		t.Fatal(err)
	}
	h := &Handler{media: service.NewMediaService(repository.NewMediaRepository(f.db))}
	router := NewRouter(config.Config{Environment: "development"}, h, fstest.MapFS{})
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/media/batch-delete", strings.NewReader(`{"ids":[1,2]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(res, req)
	if res.Code != 500 || strings.Contains(res.Body.String(), "secret-internal-marker") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body)
	}
	var count int64
	if err := f.db.Model(&model.MediaItem{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("partial deletion: count=%d err=%v", count, err)
	}
}
