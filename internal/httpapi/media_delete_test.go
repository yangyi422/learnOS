package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestMediaBatchDeleteSupportsBooksAndSeries(t *testing.T) {
	for _, tc := range []struct {
		name, body              string
		status, deleted, failOn int
		removed                 []uint
	}{
		{name: "books", body: `{"type":"book","ids":[2,3]}`, status: 200, deleted: 2, removed: []uint{2, 3}},
		{name: "series with episodes", body: `{"type":"series","ids":[10]}`, status: 200, deleted: 1, removed: []uint{10, 20, 21}},
		{name: "two series with episodes", body: `{"type":"series","ids":[10,11]}`, status: 200, deleted: 2, removed: []uint{10, 11, 20, 21, 22}},
		{name: "mixed book category", body: `{"type":"book","ids":[2,1]}`, status: 404},
		{name: "foreign book", body: `{"type":"book","ids":[2,4]}`, status: 404},
		{name: "mixed series category", body: `{"type":"series","ids":[10,2]}`, status: 404},
		{name: "missing series", body: `{"type":"series","ids":[10,999]}`, status: 404},
		{name: "foreign series", body: `{"type":"series","ids":[10,14]}`, status: 404},
		{name: "episode instead of series", body: `{"type":"series","ids":[20]}`, status: 404},
		{name: "direct episode deletion disabled", body: `{"type":"episode","ids":[20]}`, status: 400},
		{name: "game disabled", body: `{"type":"game","ids":[1]}`, status: 400},
		{name: "unknown kind", body: `{"type":"all","ids":[1]}`, status: 400},
		{name: "non string kind", body: `{"type":1,"ids":[1]}`, status: 400},
		{name: "series delete failure restores episodes", body: `{"type":"series","ids":[10,11]}`, status: 500, failOn: 10},
		{name: "episode delete failure restores batch", body: `{"type":"series","ids":[10,11]}`, status: 500, failOn: 21},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCaptureAPIFixture(t)
			if err := f.db.AutoMigrate(&model.MediaItem{}); err != nil {
				t.Fatal(err)
			}
			rows := []model.MediaItem{
				{ID: 1, Type: "movie", Source: "jellyfin", SourceID: "nas-a", ExternalID: "movie"},
				{ID: 2, Type: "book", Source: "calibre", SourceID: "books", ExternalID: "book-1", Favorite: true, Notes: "保留至确认删除"},
				{ID: 3, Type: "book", Source: "calibre", SourceID: "books", ExternalID: "book-2"},
				{ID: 4, UserID: 2, Type: "book", Source: "calibre", SourceID: "books", ExternalID: "book-1"},
				{ID: 10, Type: "series", Source: "jellyfin", SourceID: "nas-a", ExternalID: "same-series"},
				{ID: 11, Type: "series", Source: "jellyfin", SourceID: "nas-a", ExternalID: "other-series"},
				{ID: 12, Type: "series", Source: "jellyfin", SourceID: "nas-b", ExternalID: "same-series"},
				{ID: 13, Type: "series", Source: "other", SourceID: "nas-a", ExternalID: "same-series"},
				{ID: 14, UserID: 2, Type: "series", Source: "jellyfin", SourceID: "nas-a", ExternalID: "same-series"},
				{ID: 20, Type: "episode", Source: "jellyfin", SourceID: "nas-a", ExternalID: "ep-1", SeriesExternalID: "same-series"},
				{ID: 21, Type: "episode", Source: "jellyfin", SourceID: "nas-a", ExternalID: "ep-2", SeriesExternalID: "same-series"},
				{ID: 22, Type: "episode", Source: "jellyfin", SourceID: "nas-a", ExternalID: "ep-other", SeriesExternalID: "other-series"},
				{ID: 23, Type: "episode", Source: "jellyfin", SourceID: "nas-b", ExternalID: "ep-1", SeriesExternalID: "same-series"},
				{ID: 24, Type: "episode", Source: "other", SourceID: "nas-a", ExternalID: "ep-1", SeriesExternalID: "same-series"},
				{ID: 25, UserID: 2, Type: "episode", Source: "jellyfin", SourceID: "nas-a", ExternalID: "ep-1", SeriesExternalID: "same-series"},
			}
			if err := f.db.Create(&rows).Error; err != nil {
				t.Fatal(err)
			}
			if tc.failOn != 0 {
				trigger := fmt.Sprintf(`CREATE TRIGGER fail_batch BEFORE DELETE ON media_items WHEN OLD.id = %d BEGIN SELECT RAISE(ABORT, 'private-delete-error'); END`, tc.failOn)
				if err := f.db.Exec(trigger).Error; err != nil {
					t.Fatal(err)
				}
			}
			h := &Handler{media: service.NewMediaService(repository.NewMediaRepository(f.db))}
			router := NewRouter(config.Config{Environment: "development"}, h, fstest.MapFS{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/library/media/batch-delete", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status || strings.Contains(res.Body.String(), "private-delete-error") {
				t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body)
			}
			var got []uint
			if err := f.db.Model(&model.MediaItem{}).Order("id ASC").Pluck("id", &got).Error; err != nil {
				t.Fatal(err)
			}
			var want []uint
			for _, row := range rows {
				if !slices.Contains(tc.removed, row.ID) {
					want = append(want, row.ID)
				}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("remaining IDs=%v want=%v", got, want)
			}
			if tc.status == 200 {
				var response struct {
					Data struct {
						Deleted int `json:"deleted"`
					} `json:"data"`
				}
				if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || response.Data.Deleted != tc.deleted {
					t.Fatalf("wrong selected-record count: %s", res.Body)
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
