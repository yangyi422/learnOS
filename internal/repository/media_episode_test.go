package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"learnos/internal/model"
)

func newMediaRepo(t *testing.T) *MediaRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.MediaItem{}); err != nil {
		t.Fatal(err)
	}
	return NewMediaRepository(db)
}

func TestMediaDeleteEmptySelectionKeepsEpisodes(t *testing.T) {
	repo := newMediaRepo(t)
	ctx := t.Context()
	rows := []model.MediaItem{episode("s1", "a", 1, 1, "S1E1")}
	if err := repo.Upsert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, 1, "series", nil); err != ErrMediaNotFound {
		t.Fatalf("empty selection should be rejected: %v", err)
	}
	items, err := repo.Episodes(ctx, 1, "s1", -1, false)
	if err != nil || len(items) != 1 {
		t.Fatalf("empty selection changed episodes: count=%d err=%v", len(items), err)
	}
}

func episode(series, season string, seasonNo, episodeNo int, title string) model.MediaItem {
	return model.MediaItem{UserID: 1, Library: "电视剧", Source: "jellyfin", SourceID: "host", ExternalID: series + "-" + season + "-" + title,
		Type: "episode", Title: title, SeriesExternalID: series, SeasonExternalID: season, SeasonNumber: seasonNo, EpisodeNumber: episodeNo}
}

func TestEpisodesOrderBySeasonThenEpisode(t *testing.T) {
	repo := newMediaRepo(t)
	ctx := context.Background()
	rows := []model.MediaItem{
		episode("s1", "a", 2, 1, "S2E1"),
		episode("s1", "a", 1, 2, "S1E2"),
		episode("s1", "a", 1, 1, "S1E1"),
	}
	if err := repo.Upsert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	all, err := repo.Episodes(ctx, 1, "s1", -1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Title != "S1E1" || all[1].Title != "S1E2" || all[2].Title != "S2E1" {
		t.Fatalf("order wrong: %+v", all)
	}
	seasonOne, err := repo.Episodes(ctx, 1, "s1", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(seasonOne) != 2 {
		t.Fatalf("season filter wrong: %d", len(seasonOne))
	}
}

func TestSeasonsSummariseAndDedupe(t *testing.T) {
	repo := newMediaRepo(t)
	ctx := context.Background()
	rows := []model.MediaItem{episode("s1", "a", 2, 1, "S2E1"), episode("s1", "a", 1, 1, "S1E1"), episode("s1", "a", 1, 2, "S1E2")}
	if err := repo.Upsert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	// A second scan of the same data must not duplicate rows.
	if err := repo.Upsert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	seasons, err := repo.Seasons(ctx, 1, "s1", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(seasons) != 2 || seasons[0].SeasonNumber != 1 || seasons[0].Episodes != 2 || seasons[1].SeasonNumber != 2 || seasons[1].Episodes != 1 {
		t.Fatalf("season summary wrong: %+v", seasons)
	}
}

func TestEpisodeMetadataUpdateKeepsPersonalFields(t *testing.T) {
	repo := newMediaRepo(t)
	ctx := context.Background()
	row := episode("s1", "a", 1, 1, "旧标题")
	if err := repo.Upsert(ctx, []model.MediaItem{row}); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Model(&model.MediaItem{}).Where("external_id = ?", row.ExternalID).Updates(map[string]any{"favorite": true, "notes": "进度"}).Error; err != nil {
		t.Fatal(err)
	}
	row.Title = "新标题"
	if err := repo.Upsert(ctx, []model.MediaItem{row}); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Episodes(ctx, 1, "s1", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Title != "新标题" || !stored[0].Favorite || stored[0].Notes != "进度" {
		t.Fatalf("metadata update clobbered personal fields: %+v", stored)
	}
}

func TestListHidesEpisodesUnlessRequested(t *testing.T) {
	repo := newMediaRepo(t)
	ctx := context.Background()
	rows := []model.MediaItem{
		{UserID: 1, Library: "电视剧", Source: "jellyfin", SourceID: "host", ExternalID: "series-1", Type: "series", Title: "剧"},
		episode("series-1", "a", 1, 1, "S1E1"),
		{UserID: 1, Library: "电影", Source: "jellyfin", SourceID: "host", ExternalID: "movie-1", Type: "movie", Title: "电影"},
	}
	if err := repo.Upsert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	items, total, err := repo.List(ctx, 1, "", "", "", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("list should hide episodes: total=%d items=%+v", total, items)
	}
	episodes, total, err := repo.List(ctx, 1, "episode", "", "", 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(episodes) != 1 {
		t.Fatalf("explicit episode query should return episodes: total=%d", total)
	}
}
