package repository

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"learnos/internal/model"
	"reflect"
	"time"
)

var ErrLifeConflict = errors.New("life request conflicts with saved data")
var ErrLifeSource = errors.New("life source is not eligible")

type LifeRepository struct{ db *gorm.DB }

func NewLifeRepository(db *gorm.DB) *LifeRepository { return &LifeRepository{db: db} }

type LifeSource struct {
	Type        string `json:"type"`
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Eligible    bool   `json:"eligible"`
	CanGraduate bool   `json:"can_graduate"`
	EventID     *uint  `json:"event_id"`
}

func lifeSource(tx *gorm.DB, user, id uint, kind string) (*LifeSource, error) {
	s := &LifeSource{Type: kind, ID: id}
	switch kind {
	case "project":
		var p model.Project
		if err := tx.Where("id = ? AND user_id = ?", id, user).First(&p).Error; err != nil {
			return nil, err
		}
		s.Title = p.Title
		s.Status = p.Status
		s.Eligible = p.Status == "completed"
	case "course":
		var c model.Course
		if err := tx.Where("id = ? AND user_id = ?", id, user).First(&c).Error; err != nil {
			return nil, err
		}
		s.Title = c.Name
		s.Status = string(c.Status)
		s.Eligible = c.Status == model.CourseStatusCompleted
		var total, unfinished, planned, unexpanded int64
		if err := tx.Model(&model.Lesson{}).Where("course_id = ?", id).Count(&total).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&model.Lesson{}).Where("course_id = ? AND status NOT IN ?", id, []string{"completed", "skipped"}).Count(&unfinished).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&model.CurriculumBlueprintLesson{}).Joins("JOIN curriculum_blueprints b ON b.id = curriculum_blueprint_lessons.blueprint_id AND b.status = 'active'").Joins("LEFT JOIN lessons l ON l.id = curriculum_blueprint_lessons.applied_lesson_id AND l.course_id = b.course_id").Where("b.course_id = ? AND (l.id IS NULL OR l.status NOT IN ?)", id, []string{"completed", "skipped"}).Count(&planned).Error; err != nil {
			return nil, err
		}
		if err := tx.Model(&model.CurriculumBlueprintUnit{}).Joins("JOIN curriculum_blueprints b ON b.id = curriculum_blueprint_units.blueprint_id AND b.status = 'active'").Where("b.course_id = ? AND curriculum_blueprint_units.expansion_status != ?", id, model.CurriculumUnitExpanded).Count(&unexpanded).Error; err != nil {
			return nil, err
		}
		s.CanGraduate = !s.Eligible && total > 0 && unfinished == 0 && planned == 0 && unexpanded == 0
	case "inbox":
		var item model.InboxItem
		if err := tx.Where("id = ? AND user_id = ?", id, user).First(&item).Error; err != nil {
			return nil, err
		}
		s.Title = item.Content
		s.Status = item.Status
		s.Eligible = item.Status == model.InboxStatusInbox
	default:
		return nil, ErrLifeSource
	}
	var event model.LifeEvent
	err := tx.Where("user_id = ? AND source_key = ?", user, fmt.Sprintf("%s:%d", kind, id)).First(&event).Error
	if err == nil {
		s.EventID = &event.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return s, nil
}
func (r *LifeRepository) Source(ctx context.Context, user, id uint, kind string) (*LifeSource, error) {
	return lifeSource(r.db.WithContext(ctx), user, id, kind)
}
func (r *LifeRepository) Graduate(ctx context.Context, user, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := lifeSource(tx, user, id, "course")
		if err != nil {
			return err
		}
		if s.Eligible {
			return nil
		}
		if !s.CanGraduate {
			return ErrLifeSource
		}
		return tx.Model(&model.Course{}).Where("id = ? AND user_id = ?", id, user).Update("status", model.CourseStatusCompleted).Error
	})
}
func validateLifeGoal(tx *gorm.DB, user uint, id *uint) error {
	if id == nil {
		return nil
	}
	var goal model.LifeGoal
	return tx.Where("id = ? AND user_id = ?", *id, user).First(&goal).Error
}
func validateLifeProject(tx *gorm.DB, user uint, id *uint) error {
	if id == nil {
		return nil
	}
	var p model.Project
	return tx.Where("id = ? AND user_id = ?", *id, user).First(&p).Error
}
func (r *LifeRepository) Events(ctx context.Context, user uint, domain string, milestones bool) ([]model.LifeEvent, error) {
	items := []model.LifeEvent{}
	q := r.db.WithContext(ctx).Where("user_id = ?", user)
	if domain != "" {
		q = q.Where("primary_domain = ? OR secondary_domain = ?", domain, domain)
	}
	if milestones {
		q = q.Where("milestone = ?", true)
	}
	err := q.Order("occurred_on DESC, id DESC").Find(&items).Error
	return items, err
}
func (r *LifeRepository) Event(ctx context.Context, user, id uint) (*model.LifeEvent, error) {
	var e model.LifeEvent
	tx := r.db.WithContext(ctx)
	if err := tx.Where("id = ? AND user_id = ?", id, user).First(&e).Error; err != nil {
		return nil, err
	}
	if e.SourceID != nil {
		s, err := lifeSource(tx, user, *e.SourceID, e.SourceType)
		if err == nil {
			e.SourceAvailable = true
			e.SourceStatus = s.Status
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return &e, nil
}
func sameEvent(a, b model.LifeEvent) bool {
	return a.Title == b.Title && a.OccurredOn == b.OccurredOn && a.Description == b.Description && a.PrimaryDomain == b.PrimaryDomain && a.SecondaryDomain == b.SecondaryDomain && a.Milestone == b.Milestone && reflect.DeepEqual(a.GoalID, b.GoalID) && a.ExternalURL == b.ExternalURL && a.LinkName == b.LinkName && a.SourceType == b.SourceType && reflect.DeepEqual(a.SourceID, b.SourceID)
}
func (r *LifeRepository) SaveEvent(ctx context.Context, user, id uint, e model.LifeEvent) (*model.LifeEvent, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		e.UserID = user
		if id == 0 {
			// Replays return the existing fact only for an identical payload. A changed
			// payload must never silently overwrite a previously captured memory.
			var old model.LifeEvent
			if e.CreationKey != nil {
				err := tx.Where("user_id = ? AND creation_key = ?", user, *e.CreationKey).First(&old).Error
				if err == nil {
					if !sameEvent(old, e) {
						return ErrLifeConflict
					}
					e = old
					return nil
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if err := validateLifeGoal(tx, user, e.GoalID); err != nil {
				return err
			}
			if e.SourceID != nil {
				key := fmt.Sprintf("%s:%d", e.SourceType, *e.SourceID)
				e.SourceKey = &key
				err := tx.Where("user_id = ? AND source_key = ?", user, key).First(&old).Error
				if err == nil {
					if sameEvent(old, e) {
						e = old
						return nil
					}
					return ErrLifeConflict
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				source, err := lifeSource(tx, user, *e.SourceID, e.SourceType)
				if err != nil {
					return err
				}
				if !source.Eligible {
					return ErrLifeSource
				}
				e.SourceTitle = source.Title
				if e.SourceType == "inbox" {
					now := time.Now().UTC()
					claim := tx.Model(&model.InboxItem{}).Where("id = ? AND user_id = ? AND status = ?", *e.SourceID, user, model.InboxStatusInbox).Updates(map[string]interface{}{"status": model.InboxStatusProcessed, "processed_at": now})
					if claim.Error != nil {
						return claim.Error
					}
					if claim.RowsAffected != 1 {
						return ErrLifeConflict
					}
				}
			}
			if err := tx.Create(&e).Error; err != nil {
				return err
			}
			if e.SourceType == "inbox" {
				return tx.Model(&model.InboxItem{}).Where("id = ? AND user_id = ?", *e.SourceID, user).Updates(map[string]interface{}{"processed_to_type": "life_event", "processed_to_id": e.ID}).Error
			}
			return nil
		}
		var old model.LifeEvent
		if err := tx.Where("id = ? AND user_id = ?", id, user).First(&old).Error; err != nil {
			return err
		}
		if err := validateLifeGoal(tx, user, e.GoalID); err != nil {
			return err
		}
		old.Title = e.Title
		old.OccurredOn = e.OccurredOn
		old.Description = e.Description
		old.PrimaryDomain = e.PrimaryDomain
		old.SecondaryDomain = e.SecondaryDomain
		old.Milestone = e.Milestone
		old.GoalID = e.GoalID
		old.ExternalURL = e.ExternalURL
		old.LinkName = e.LinkName
		if err := tx.Save(&old).Error; err != nil {
			return err
		}
		e = old
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &e, nil
}
func (r *LifeRepository) DeleteEvent(ctx context.Context, user, id uint) error {
	// Retain Inbox processing metadata and prevent its contents from reappearing.
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, user).Delete(&model.LifeEvent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *LifeRepository) Goals(ctx context.Context, user uint) ([]model.LifeGoal, error) {
	goals := []model.LifeGoal{}
	err := r.db.WithContext(ctx).Where("user_id = ?", user).Order("created_at DESC, id DESC").Find(&goals).Error
	return goals, err
}
func (r *LifeRepository) Goal(ctx context.Context, user, id uint) (*model.LifeGoal, []model.LifeGoalEntry, []model.LifeEvent, error) {
	var g model.LifeGoal
	entries := []model.LifeGoalEntry{}
	events := []model.LifeEvent{}
	tx := r.db.WithContext(ctx)
	if err := tx.Where("id = ? AND user_id = ?", id, user).First(&g).Error; err != nil {
		return nil, nil, nil, err
	}
	if g.ProjectID != nil {
		var p model.Project
		err := tx.Where("id = ? AND user_id = ?", *g.ProjectID, user).First(&p).Error
		if err == nil {
			g.ProjectAvailable = true
			g.ProjectTitle = p.Title
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, err
		}
	}
	if err := tx.Where("goal_id = ?", id).Order("occurred_on DESC, id DESC").Find(&entries).Error; err != nil {
		return nil, nil, nil, err
	}
	err := tx.Where("goal_id = ? AND user_id = ?", id, user).Order("occurred_on DESC, id DESC").Find(&events).Error
	return &g, entries, events, err
}
func sameGoal(a, b model.LifeGoal) bool {
	return a.Title == b.Title && a.Why == b.Why && a.CurrentNote == b.CurrentNote && a.Status == b.Status && a.Domain == b.Domain && reflect.DeepEqual(a.ProjectID, b.ProjectID) && a.ExternalURL == b.ExternalURL && a.LinkName == b.LinkName
}
func (r *LifeRepository) SaveGoal(ctx context.Context, user, id uint, g model.LifeGoal, statusDate, statusReason string, recordChange bool) (*model.LifeGoal, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g.UserID = user
		if id == 0 {
			if g.CreationKey != nil {
				var old model.LifeGoal
				err := tx.Where("user_id = ? AND creation_key = ?", user, *g.CreationKey).First(&old).Error
				if err == nil {
					if !sameGoal(old, g) {
						return ErrLifeConflict
					}
					g = old
					return nil
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if err := validateLifeProject(tx, user, g.ProjectID); err != nil {
				return err
			}
			return tx.Create(&g).Error
		}
		var old model.LifeGoal
		if err := tx.Where("id = ? AND user_id = ?", id, user).First(&old).Error; err != nil {
			return err
		}
		if err := validateLifeProject(tx, user, g.ProjectID); err != nil {
			return err
		}
		if old.Status != g.Status || (recordChange && old.CurrentNote != g.CurrentNote) {
			entry := model.LifeGoalEntry{GoalID: id, OccurredOn: statusDate, Content: g.CurrentNote, Reason: statusReason, FromStatus: old.Status, ToStatus: g.Status}
			if old.Status == g.Status {
				entry.FromStatus = ""
				entry.ToStatus = ""
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		g.ID = id
		g.CreatedAt = old.CreatedAt
		g.CreationKey = old.CreationKey
		return tx.Save(&g).Error
	})
	if err != nil {
		return nil, err
	}
	return &g, nil
}
func (r *LifeRepository) AddEntry(ctx context.Context, user, goalID uint, e model.LifeGoalEntry) (*model.LifeGoalEntry, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateLifeGoal(tx, user, &goalID); err != nil {
			return err
		}
		e.GoalID = goalID
		if e.CreationKey != nil {
			var old model.LifeGoalEntry
			err := tx.Where("goal_id = ? AND creation_key = ?", goalID, *e.CreationKey).First(&old).Error
			if err == nil {
				if old.Content != e.Content || old.Reason != e.Reason || old.OccurredOn != e.OccurredOn {
					return ErrLifeConflict
				}
				e = old
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		// Concurrent duplicate attempts are also stopped by the database constraint.
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&e)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrLifeConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &e, nil
}
