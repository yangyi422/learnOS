package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
)

var ErrMediaNotFound = errors.New("media item not found")

// mediaVisibility keeps R18 Jellyfin libraries out of every read path; episodes
// inherit the series library, so the same rule covers them.
const mediaVisibility = "source <> ? OR library IN ?"

type MediaRepository struct{ db *gorm.DB }

func NewMediaRepository(db *gorm.DB) *MediaRepository {
	return &MediaRepository{db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}

// SeasonSummary is one row of the season selector.
type SeasonSummary struct {
	SeasonNumber int    `json:"season_number"`
	SeasonID     string `json:"season_external_id,omitempty"`
	Episodes     int64  `json:"episodes"`
}

func (r *MediaRepository) Upsert(ctx context.Context, items []model.MediaItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "source"}, {Name: "source_id"}, {Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"type", "title", "description", "authors", "genres", "tags", "year", "rating", "cover_id", "library",
				"series_external_id", "season_external_id", "season_number", "episode_number", "premiere_date", "runtime_seconds", "updated_at"}),
		}).Create(&items).Error
	})
}

func (r *MediaRepository) List(ctx context.Context, user uint, kind, source, search string, page int, showR18 bool) ([]model.MediaItem, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MediaItem{}).Where("user_id = ?", user)
	if !showR18 {
		q = q.Where(mediaVisibility, "jellyfin", []string{"电影", "电视剧"})
	}
	// Episodes belong to a series page, never to the flat media list.
	if kind == "" {
		q = q.Where("type <> ?", "episode")
	}
	if kind != "" {
		q = q.Where("type = ?", kind)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if search != "" {
		search = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(search)
		q = q.Where("title LIKE ? ESCAPE '!'", "%"+search+"%")
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	items := make([]model.MediaItem, 0)
	err := q.Order("id DESC").Offset((page - 1) * 24).Limit(24).Find(&items).Error
	return items, count, err
}

// Delete removes the explicitly selected local catalog rows of one kind. Series
// episodes are removed in the same transaction, matching their full origin so
// another user's or another source's copy of the series remains untouched.
// No source media, remote catalog entries or shared cover files are removed.
func (r *MediaRepository) Delete(ctx context.Context, user uint, kind string, ids []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var items []model.MediaItem
		if err := tx.Where("user_id = ? AND type = ? AND id IN ?", user, kind, ids).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 || len(items) != len(ids) {
			return ErrMediaNotFound
		}
		if kind == "series" {
			origins := make([]clause.Expression, 0, len(items))
			for _, item := range items {
				origins = append(origins, clause.And(
					clause.Eq{Column: "source", Value: item.Source},
					clause.Eq{Column: "source_id", Value: item.SourceID},
					clause.Eq{Column: "series_external_id", Value: item.ExternalID},
				))
			}
			if err := tx.Where("user_id = ? AND type = ?", user, "episode").Where(clause.Or(origins...)).Delete(&model.MediaItem{}).Error; err != nil {
				return err
			}
		}
		result := tx.Where("user_id = ? AND type = ? AND id IN ?", user, kind, ids).Delete(&model.MediaItem{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(ids)) {
			return ErrMediaNotFound
		}
		return nil
	})
}

// Series loads one series row by its Jellyfin id.
func (r *MediaRepository) Series(ctx context.Context, user uint, externalID string) (model.MediaItem, error) {
	var item model.MediaItem
	err := r.db.WithContext(ctx).Where("user_id = ? AND source = ? AND type = ? AND external_id = ?", user, "jellyfin", "series", externalID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MediaItem{}, ErrMediaNotFound
	}
	return item, err
}

// Seasons groups the series' episodes by season number, ascending.
func (r *MediaRepository) Seasons(ctx context.Context, user uint, seriesExternalID string, showR18 bool) ([]SeasonSummary, error) {
	q := r.db.WithContext(ctx).Model(&model.MediaItem{}).
		Select("season_number, COUNT(*) AS episodes").
		Where("user_id = ? AND source = ? AND type = ? AND series_external_id = ?", user, "jellyfin", "episode", seriesExternalID)
	if !showR18 {
		q = q.Where(mediaVisibility, "jellyfin", []string{"电影", "电视剧"})
	}
	seasons := make([]SeasonSummary, 0)
	if err := q.Group("season_number").Order("season_number ASC").Scan(&seasons).Error; err != nil {
		return nil, err
	}
	return seasons, nil
}

// Episodes returns the series' episodes ordered by season then episode; season < 0
// means every season.
func (r *MediaRepository) Episodes(ctx context.Context, user uint, seriesExternalID string, season int, showR18 bool) ([]model.MediaItem, error) {
	q := r.db.WithContext(ctx).Model(&model.MediaItem{}).
		Where("user_id = ? AND source = ? AND type = ? AND series_external_id = ?", user, "jellyfin", "episode", seriesExternalID)
	if !showR18 {
		q = q.Where(mediaVisibility, "jellyfin", []string{"电影", "电视剧"})
	}
	if season >= 0 {
		q = q.Where("season_number = ?", season)
	}
	items := make([]model.MediaItem, 0)
	err := q.Order("season_number ASC").Order("episode_number ASC").Limit(2000).Find(&items).Error
	return items, err
}
