package repository

import (
	"context"
	"errors"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

// ErrDailyFocusConflict reports that the stored focus revision no longer
// matches the revision the caller read.
var ErrDailyFocusConflict = errors.New("daily focus changed since it was loaded")

type DailyFocusRepository struct{ db *gorm.DB }

func NewDailyFocusRepository(db *gorm.DB) *DailyFocusRepository {
	return &DailyFocusRepository{db: db}
}

// DailyFocusTask is the ownership-checked projection of a task and its project
// used both for candidate validation and for read-time state derivation. It is
// always loaded through a join on projects.user_id, so another user's task can
// never be returned.
type DailyFocusTask struct {
	TaskID         uint
	Title          string
	Status         string
	Priority       string
	PlanDate       *string
	DueDate        *string
	CompletedAt    *time.Time
	ArchivedAt     *time.Time
	UpdatedAt      time.Time
	ProjectID      uint
	ProjectTitle   string
	ProjectContext string
	ProjectStatus  string
}

func dailyFocusScope(ctx context.Context, db *gorm.DB, userID uint, date, scene string) *gorm.DB {
	return db.WithContext(ctx).Where("user_id = ? AND focus_date = ? AND context = ?", userID, date, scene)
}

// Get returns the focus for one user, day and scene. gorm.ErrRecordNotFound
// means no focus is set.
func (r *DailyFocusRepository) Get(ctx context.Context, userID uint, date, scene string) (*model.DailyFocus, error) {
	var focus model.DailyFocus
	if err := dailyFocusScope(ctx, r.db, userID, date, scene).First(&focus).Error; err != nil {
		return nil, err
	}
	return &focus, nil
}

// ListByDate returns every focus of one user on one calendar day. The scope is
// small by construction: at most one row per scene.
func (r *DailyFocusRepository) ListByDate(ctx context.Context, userID uint, date string) ([]model.DailyFocus, error) {
	focuses := make([]model.DailyFocus, 0, 2)
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND focus_date = ?", userID, date).
		Order("context ASC").
		Find(&focuses).Error; err != nil {
		return nil, err
	}
	return focuses, nil
}

// Save points the day/scene focus at taskID. expectedRevision > 0 requires the
// stored revision to match; 0 skips the version check. Re-setting the same task
// is idempotent and does not bump the revision. Only the daily_focuses row is
// written: task and project state are never changed here.
func (r *DailyFocusRepository) Save(ctx context.Context, userID uint, date, scene string, taskID uint, expectedRevision int) (*model.DailyFocus, error) {
	var saved model.DailyFocus
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.DailyFocus
		err := dailyFocusScope(ctx, tx, userID, date, scene).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if expectedRevision > 0 {
				// The caller held a revision but no focus exists any more.
				return ErrDailyFocusConflict
			}
			created := model.DailyFocus{UserID: userID, FocusDate: date, Context: scene, TaskID: &taskID, Revision: 1}
			if createErr := tx.Create(&created).Error; createErr != nil {
				// A concurrent writer may have won the unique scope first; retry
				// as the conditional update instead of failing the request.
				var raced model.DailyFocus
				if readErr := dailyFocusScope(ctx, tx, userID, date, scene).First(&raced).Error; readErr != nil {
					return createErr
				}
				if updateErr := updateDailyFocusTask(tx, &raced, taskID, expectedRevision); updateErr != nil {
					return updateErr
				}
				saved = raced
				return nil
			}
			saved = created
			return nil
		}
		if err != nil {
			return err
		}
		if updateErr := updateDailyFocusTask(tx, &current, taskID, expectedRevision); updateErr != nil {
			return updateErr
		}
		saved = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

func updateDailyFocusTask(tx *gorm.DB, focus *model.DailyFocus, taskID uint, expectedRevision int) error {
	if expectedRevision > 0 && focus.Revision != expectedRevision {
		return ErrDailyFocusConflict
	}
	if focus.TaskID != nil && *focus.TaskID == taskID {
		return nil
	}
	result := tx.Model(&model.DailyFocus{}).
		Where("id = ? AND revision = ?", focus.ID, focus.Revision).
		Updates(map[string]interface{}{"task_id": taskID, "revision": focus.Revision + 1})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDailyFocusConflict
	}
	return tx.First(focus, focus.ID).Error
}

// Clear removes the day/scene focus row. Clearing an absent focus succeeds so
// that the operation is idempotent; a stale expectedRevision only conflicts
// while a row is still present.
func (r *DailyFocusRepository) Clear(ctx context.Context, userID uint, date, scene string, expectedRevision int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.DailyFocus
		err := dailyFocusScope(ctx, tx, userID, date, scene).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if expectedRevision > 0 && current.Revision != expectedRevision {
			return ErrDailyFocusConflict
		}
		result := tx.Where("id = ? AND revision = ?", current.ID, current.Revision).Delete(&model.DailyFocus{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrDailyFocusConflict
		}
		return nil
	})
}

// Tasks loads the referenced tasks together with their projects. The join on
// projects.user_id scopes every row to the requesting user; tasks that are gone
// or belong to somebody else are simply absent from the result.
func (r *DailyFocusRepository) Tasks(ctx context.Context, userID uint, taskIDs []uint) (map[uint]DailyFocusTask, error) {
	tasks := make(map[uint]DailyFocusTask, len(taskIDs))
	if len(taskIDs) == 0 {
		return tasks, nil
	}
	rows := make([]DailyFocusTask, 0, len(taskIDs))
	err := r.db.WithContext(ctx).Table("project_tasks").
		Select(`project_tasks.id AS task_id, project_tasks.title AS title, project_tasks.status AS status,
			project_tasks.priority AS priority, project_tasks.plan_date AS plan_date, project_tasks.due_date AS due_date,
			project_tasks.completed_at AS completed_at, project_tasks.archived_at AS archived_at,
			project_tasks.updated_at AS updated_at,
			projects.id AS project_id, projects.title AS project_title,
			projects.context AS project_context, projects.status AS project_status`).
		Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ? AND project_tasks.id IN ?", userID, taskIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row.PlanDate = normalizeFocusDate(row.PlanDate)
		row.DueDate = normalizeFocusDate(row.DueDate)
		tasks[row.TaskID] = row
	}
	return tasks, nil
}

// Task returns one ownership-checked task projection. The boolean is false when
// the task does not exist or belongs to another user.
func (r *DailyFocusRepository) Task(ctx context.Context, userID, taskID uint) (*DailyFocusTask, bool, error) {
	tasks, err := r.Tasks(ctx, userID, []uint{taskID})
	if err != nil {
		return nil, false, err
	}
	task, ok := tasks[taskID]
	if !ok {
		return nil, false, nil
	}
	return &task, true, nil
}

// SQLite may return a DATE column as an RFC3339 timestamp. Keep focus read
// models at calendar-date precision in every API response.
func normalizeFocusDate(value *string) *string {
	if value == nil {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, *value); err == nil {
		formatted := parsed.Format("2006-01-02")
		return &formatted
	}
	return value
}
