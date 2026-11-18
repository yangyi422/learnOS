package service

import (
	"context"
	"errors"
	"strings"

	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrMediaInput    = errors.New("invalid media item")
	ErrMediaNotFound = errors.New("media item not found")
)

type MediaService struct{ repo *repository.MediaRepository }

func NewMediaService(repo *repository.MediaRepository) *MediaService { return &MediaService{repo} }

func (s *MediaService) Upsert(ctx context.Context, user uint, items []model.MediaItem) error {
	if len(items) < 1 || len(items) > 50 {
		return ErrMediaInput
	}
	for i := range items {
		item := &items[i]
		valid := item.Source == "calibre" && item.Type == "book" ||
			item.Source == "jellyfin" && (item.Type == "movie" || item.Type == "series" || item.Type == "episode")
		if !valid || item.SourceID == "" || len(item.SourceID) > 255 || item.ExternalID == "" || len(item.ExternalID) > 1024 || strings.TrimSpace(item.Title) == "" || len(item.Title) > 1024 {
			return ErrMediaInput
		}
		if item.Type == "episode" {
			// An episode without its series can never be reached in the UI.
			if len(item.SeriesExternalID) == 0 || len(item.SeriesExternalID) > 1024 || item.SeasonNumber < 0 || item.EpisodeNumber < 0 || len(item.PremiereDate) > 32 {
				return ErrMediaInput
			}
		} else if item.SeriesExternalID != "" || item.SeasonExternalID != "" {
			return ErrMediaInput
		}
		item.UserID = user
	}
	return s.repo.Upsert(ctx, items)
}

func (s *MediaService) List(ctx context.Context, user uint, kind, source, search string, page int, showR18 bool) ([]model.MediaItem, int64, error) {
	return s.repo.List(ctx, user, kind, source, search, page, showR18)
}

// Delete only deletes catalog records; a later agent sync may import them again.
// An omitted kind retains the original movie-only API behavior.
func (s *MediaService) Delete(ctx context.Context, user uint, kind string, ids []uint) error {
	if kind == "" {
		kind = "movie"
	}
	if (kind != "movie" && kind != "series" && kind != "book") || len(ids) == 0 || len(ids) > 100 {
		return ErrMediaInput
	}
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			return ErrMediaInput
		}
		seen[id] = true
	}
	err := s.repo.Delete(ctx, user, kind, ids)
	if errors.Is(err, repository.ErrMediaNotFound) {
		return ErrMediaNotFound
	}
	return err
}

// SeriesDetail is the payload behind the TV detail page.
type SeriesDetail struct {
	Series       model.MediaItem            `json:"series"`
	Seasons      []repository.SeasonSummary `json:"seasons"`
	EpisodeCount int64                      `json:"episode_count"`
}

func (s *MediaService) Series(ctx context.Context, user uint, externalID string, showR18 bool) (SeriesDetail, error) {
	series, err := s.repo.Series(ctx, user, externalID)
	if errors.Is(err, repository.ErrMediaNotFound) {
		return SeriesDetail{}, ErrMediaNotFound
	}
	if err != nil {
		return SeriesDetail{}, err
	}
	seasons, err := s.repo.Seasons(ctx, user, externalID, showR18)
	if err != nil {
		return SeriesDetail{}, err
	}
	var total int64
	for _, season := range seasons {
		total += season.Episodes
	}
	return SeriesDetail{Series: series, Seasons: seasons, EpisodeCount: total}, nil
}

// Episodes returns all episodes of a series, or just one season when season >= 0.
func (s *MediaService) Episodes(ctx context.Context, user uint, externalID string, season int, showR18 bool) ([]model.MediaItem, error) {
	if _, err := s.repo.Series(ctx, user, externalID); err != nil {
		if errors.Is(err, repository.ErrMediaNotFound) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}
	return s.repo.Episodes(ctx, user, externalID, season, showR18)
}
