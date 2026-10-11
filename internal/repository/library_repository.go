package repository

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
	"strings"
	"time"
)

var ErrLibraryConflict = errors.New("library changed")
var ErrLibrarySectionNotEmpty = errors.New("library section contains pages")

type LibraryRepository struct{ db *gorm.DB }

func NewLibraryRepository(db *gorm.DB) *LibraryRepository {
	return &LibraryRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}

type LibraryCollectionRow struct {
	model.LibraryCollection
	PageCount int64 `json:"page_count"`
}
type LibraryPageRow struct {
	ID           uint      `json:"id"`
	CollectionID uint      `json:"collection_id"`
	SectionID    *uint     `json:"section_id"`
	Title        string    `json:"title"`
	Body         string    `json:"-"`
	Summary      string    `json:"summary"`
	Tags         []string  `json:"tags" gorm:"serializer:json"`
	CoverURL     string    `json:"cover_url"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type LibraryPages struct {
	Items    []LibraryPageRow `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

func libraryOwner(tx *gorm.DB, user, id uint) (model.LibraryCollection, error) {
	var row model.LibraryCollection
	err := tx.Where("id = ? AND user_id = ?", id, user).First(&row).Error
	return row, err
}
func librarySection(tx *gorm.DB, collection uint, section *uint) error {
	if section == nil {
		return nil
	}
	var count int64
	if err := tx.Model(&model.LibrarySection{}).Where("id = ? AND collection_id = ?", *section, collection).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *LibraryRepository) List(ctx context.Context, user uint) ([]LibraryCollectionRow, error) {
	rows := []LibraryCollectionRow{}
	err := r.db.WithContext(ctx).Table("library_collections").Select("library_collections.*, (SELECT COUNT(*) FROM library_pages WHERE collection_id = library_collections.id) AS page_count").Where("user_id = ?", user).Order("updated_at DESC, id DESC").Find(&rows).Error
	return rows, err
}
func (r *LibraryRepository) Collection(ctx context.Context, user, id uint) (model.LibraryCollection, []model.LibrarySection, error) {
	row, err := libraryOwner(r.db.WithContext(ctx), user, id)
	sections := []model.LibrarySection{}
	if err == nil {
		err = r.db.WithContext(ctx).Where("collection_id = ?", id).Order("id ASC").Find(&sections).Error
	}
	return row, sections, err
}
func (r *LibraryRepository) CreateCollection(ctx context.Context, row model.LibraryCollection) (model.LibraryCollection, error) {
	row.Version = 1
	err := r.db.WithContext(ctx).Create(&row).Error
	return row, err
}
func (r *LibraryRepository) UpdateCollection(ctx context.Context, user, id uint, row model.LibraryCollection) (model.LibraryCollection, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, err := libraryOwner(tx, user, id)
		if err != nil {
			return err
		}
		if old.Version != row.Version {
			return ErrLibraryConflict
		}
		q := tx.Model(&old).Updates(map[string]interface{}{"name": row.Name, "description": row.Description, "category": row.Category, "template": row.Template, "display_mode": row.DisplayMode, "cover_url": row.CoverURL, "version": row.Version + 1})
		if q.Error != nil {
			return q.Error
		}
		row, err = libraryOwner(tx, user, id)
		return err
	})
	return row, err
}
func (r *LibraryRepository) DeleteCollection(ctx context.Context, user, id, version uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := libraryOwner(tx, user, id)
		if err != nil {
			return err
		}
		if row.Version != version {
			return ErrLibraryConflict
		}
		if err = tx.Where("collection_id = ?", id).Delete(&model.LibraryPage{}).Error; err != nil {
			return err
		}
		if err = tx.Where("collection_id = ?", id).Delete(&model.LibrarySection{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
}
func (r *LibraryRepository) SaveSection(ctx context.Context, user, col, id uint, name string) (model.LibrarySection, error) {
	row := model.LibrarySection{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		if id == 0 {
			row = model.LibrarySection{CollectionID: col, Name: name}
			return tx.Create(&row).Error
		}
		if err := tx.Where("id = ? AND collection_id = ?", id, col).First(&row).Error; err != nil {
			return err
		}
		return tx.Model(&row).Update("name", name).Error
	})
	row.Name = name
	return row, err
}

// Nonempty sections may only be removed by explicitly moving their pages to root.
func (r *LibraryRepository) DeleteSection(ctx context.Context, user, col, id uint, move bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		var row model.LibrarySection
		if err := tx.Where("id = ? AND collection_id = ?", id, col).First(&row).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.LibraryPage{}).Where("section_id = ? AND collection_id = ?", id, col).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 && !move {
			return ErrLibrarySectionNotEmpty
		}
		if count > 0 {
			if err := tx.Model(&model.LibraryPage{}).Where("section_id = ? AND collection_id = ?", id, col).Updates(map[string]interface{}{"section_id": nil, "version": gorm.Expr("version + 1")}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&row).Error
	})
}
func (r *LibraryRepository) Page(ctx context.Context, user, col, id uint) (model.LibraryPage, error) {
	var row model.LibraryPage
	err := r.db.WithContext(ctx).Where("id = ? AND collection_id = ? AND collection_id IN (SELECT id FROM library_collections WHERE user_id = ?)", id, col, user).First(&row).Error
	return row, err
}
func (r *LibraryRepository) SavePage(ctx context.Context, user, col, id uint, row model.LibraryPage) (model.LibraryPage, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		if err := librarySection(tx, col, row.SectionID); err != nil {
			return err
		}
		row.CollectionID = col
		if id == 0 {
			row.Version = 1
			return tx.Omit("Collection", "Section").Create(&row).Error
		}
		var old model.LibraryPage
		if err := tx.Where("id = ? AND collection_id = ?", id, col).First(&old).Error; err != nil {
			return err
		}
		if old.Version != row.Version {
			return ErrLibraryConflict
		}
		if err := tx.Model(&old).Updates(map[string]interface{}{"title": row.Title, "body": row.Body, "tags": gorm.Expr("?", mustJSON(row.Tags)), "sources": gorm.Expr("?", mustJSON(row.Sources)), "section_id": row.SectionID, "cover_url": row.CoverURL, "version": row.Version + 1}).Error; err != nil {
			return err
		}
		return tx.First(&row, id).Error
	})
	return row, err
}
func (r *LibraryRepository) DeletePage(ctx context.Context, user, col, id, version uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		var row model.LibraryPage
		if err := tx.Where("id = ? AND collection_id = ?", id, col).First(&row).Error; err != nil {
			return err
		}
		if row.Version != version {
			return ErrLibraryConflict
		}
		return tx.Delete(&row).Error
	})
}
func (r *LibraryRepository) Pages(ctx context.Context, user, col uint, q string, section *uint, root bool, page int) (LibraryPages, error) {
	out := LibraryPages{Items: []LibraryPageRow{}, Page: page, PageSize: 50}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		db := tx.Model(&model.LibraryPage{}).Where("collection_id = ?", col)
		if root {
			db = db.Where("section_id IS NULL")
		} else if section != nil {
			if err := librarySection(tx, col, section); err != nil {
				return err
			}
			db = db.Where("section_id = ?", *section)
		}
		if q != "" {
			esc := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q)
			pattern := "%" + esc + "%"
			db = db.Where("(title LIKE ? ESCAPE '\\' OR body LIKE ? ESCAPE '\\' OR tags LIKE ? ESCAPE '\\')", pattern, pattern, pattern)
		}
		if err := db.Count(&out.Total).Error; err != nil {
			return err
		}
		return db.Select("id,collection_id,section_id,title,substr(body,1,240) AS body,tags,cover_url,updated_at").Order("updated_at DESC, id DESC").Offset((page - 1) * 50).Limit(50).Find(&out.Items).Error
	})
	for i := range out.Items {
		out.Items[i].Summary = out.Items[i].Body
	}
	return out, err
}
func (r *LibraryRepository) Export(ctx context.Context, user, col uint) (model.LibraryCollection, []model.LibrarySection, []model.LibraryPage, error) {
	var collection model.LibraryCollection
	sections := []model.LibrarySection{}
	pages := []model.LibraryPage{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		collection, err = libraryOwner(tx, user, col)
		if err != nil {
			return err
		}
		if err = tx.Where("collection_id = ?", col).Order("id").Find(&sections).Error; err != nil {
			return err
		}
		return tx.Where("collection_id = ?", col).Order("id").Find(&pages).Error
	})
	return collection, sections, pages, err
}

func mustJSON(value interface{}) string { data, _ := json.Marshal(value); return string(data) }

func (r *LibraryRepository) LinkStatus(ctx context.Context, user, col uint, ids []uint) (map[uint]bool, error) {
	out := map[uint]bool{}
	for _, id := range ids {
		out[id] = false
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := libraryOwner(tx, user, col); err != nil {
			return err
		}
		var found []uint
		if err := tx.Model(&model.LibraryPage{}).Where("collection_id = ? AND id IN ?", col, ids).Pluck("id", &found).Error; err != nil {
			return err
		}
		for _, id := range found {
			out[id] = true
		}
		return nil
	})
	return out, err
}
