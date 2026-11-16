package service

import (
	"context"
	"errors"
	"strings"

	"learnos/internal/model"
	"learnos/internal/repository"
)

var ErrMediaInput = errors.New("invalid media item")

type MediaService struct{ repo *repository.MediaRepository }

func NewMediaService(repo *repository.MediaRepository) *MediaService { return &MediaService{repo} }
func (s *MediaService) Upsert(ctx context.Context, user uint, items []model.MediaItem) error {
	if len(items) < 1 || len(items) > 50 {
		return ErrMediaInput
	}
	for i := range items {
		item := &items[i]
		valid := item.Source == "calibre" && item.Type == "book" || item.Source == "jellyfin" && (item.Type == "movie" || item.Type == "series")
		if !valid || item.SourceID == "" || len(item.SourceID) > 255 || item.ExternalID == "" || len(item.ExternalID) > 1024 || strings.TrimSpace(item.Title) == "" || len(item.Title) > 1024 {
			return ErrMediaInput
		}
		item.UserID = user
	}
	return s.repo.Upsert(ctx, items)
}
func (s *MediaService) List(ctx context.Context, user uint, kind, source, search string, page int, showR18 bool) ([]model.MediaItem, int64, error) {
	return s.repo.List(ctx, user, kind, source, search, page, showR18)
}
