package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"learnos/internal/auth"
	"learnos/internal/model"
	"learnos/internal/repository"
	"learnos/internal/service"
)

func newSeriesRouter(t *testing.T) (*gin.Engine, service.MediaService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "series.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.MediaItem{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	repo := repository.NewMediaRepository(db)
	media := service.NewMediaService(repo)
	handler := &Handler{media: media}
	router := gin.New()
	group := router.Group("/api/v1/library")
	group.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), auth.Principal{UserID: 1, Username: "owner"}))
		c.Next()
	})
	group.GET("/media/series/:externalID", handler.mediaSeries)
	group.GET("/media/series/:externalID/episodes", handler.mediaEpisodes)
	return router, *media
}

const seriesExternalID = "11111111111111111111111111111111"

func seedSeries(t *testing.T, media service.MediaService) {
	t.Helper()
	items := []model.MediaItem{
		{UserID: 1, Library: "电视剧", Source: "jellyfin", SourceID: "host", ExternalID: "11111111111111111111111111111111", Type: "series", Title: "测试剧"},
	}
	for season := 1; season <= 2; season++ {
		for episode := 1; episode <= season; episode++ {
			items = append(items, model.MediaItem{
				UserID: 1, Library: "电视剧", Source: "jellyfin", SourceID: "host",
				ExternalID: fmt.Sprintf("%s%02d%02d", seriesExternalID, season, episode),
				Type: "episode", Title: "第" + string(rune('0'+episode)) + "集",
				SeriesExternalID: seriesExternalID, SeasonExternalID: "season1", SeasonNumber: season, EpisodeNumber: episode,
				PremiereDate: "2011-04-1" + string(rune('0'+episode)), RuntimeSeconds: 3600,
			})
		}
	}
	if err := media.Upsert(t.Context(), 1, items[:1]); err != nil {
		t.Fatal(err)
	}
	for start := 1; start < len(items); start += 50 {
		end := start + 50
		if end > len(items) {
			end = len(items)
		}
		if err := media.Upsert(t.Context(), 1, items[start:end]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMediaSeriesAPIExposesSeasonsAndEpisodes(t *testing.T) {
	router, media := newSeriesRouter(t)
	seedSeries(t, media)

	request := func(target string) (int, map[string]any) {
		t.Helper()
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
		var body map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %s", res.Body)
		}
		return res.Code, body
	}

	code, body := request("/api/v1/library/media/series/" + seriesExternalID)
	if code != 200 {
		t.Fatalf("series status=%d body=%v", code, body)
	}
	data := body["data"].(map[string]any)
	if data["series"].(map[string]any)["Title"] != "测试剧" {
		t.Fatalf("series title wrong: %v", data["series"])
	}
	seasons := data["seasons"].([]any)
	if len(seasons) != 2 {
		t.Fatalf("expected two seasons: %v", seasons)
	}
	if data["episode_count"].(float64) != 3 {
		t.Fatalf("episode_count=%v", data["episode_count"])
	}

	code, body = request("/api/v1/library/media/series/" + seriesExternalID + "/episodes?season=2")
	if code != 200 {
		t.Fatalf("episodes status=%d", code)
	}
	items := body["data"].(map[string]any)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("season 2 should hold 2 episodes: %v", items)
	}
	first := items[0].(map[string]any)
	if first["episode_number"].(float64) != 1 || first["season_number"].(float64) != 2 {
		t.Fatalf("episode numbering wrong: %v", first)
	}

	code, _ = request("/api/v1/library/media/series/nothex")
	if code != 400 {
		t.Fatalf("invalid id should be rejected: %d", code)
	}
	code, _ = request("/api/v1/library/media/series/abc123")
	if code != 404 {
		t.Fatalf("unknown series should be 404: %d", code)
	}
	code, _ = request("/api/v1/library/media/series/" + seriesExternalID + "/episodes?season=x")
	if code != 400 {
		t.Fatalf("invalid season should be rejected: %d", code)
	}
}