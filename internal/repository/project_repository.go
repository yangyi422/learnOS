package repository

import (
	"context"
	"errors"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type ProjectRepository struct{ db *gorm.DB }

var ErrTaskOrder = errors.New("invalid task order")
var ErrTaskArchive = errors.New("only completed tasks can be archived; restore archived tasks before editing")
var ErrTaskConflict = errors.New("task changed since it was loaded")

func NewProjectRepository(db *gorm.DB) *ProjectRepository { return &ProjectRepository{db: db} }

func (r *ProjectRepository) List(ctx context.Context, userID uint) ([]model.Project, error) {
	var projects []model.Project
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC, id DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	type countRow struct {
		ProjectID uint
		Status    string
		Total     int64
		Archived  int64
	}
	var counts []countRow
	if err := r.db.WithContext(ctx).Model(&model.ProjectTask{}).
		Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ?", userID).
		Select("project_tasks.project_id, project_tasks.status, COUNT(*) AS total, SUM(CASE WHEN project_tasks.archived_at IS NOT NULL THEN 1 ELSE 0 END) AS archived").
		Group("project_tasks.project_id, project_tasks.status").Scan(&counts).Error; err != nil {
		return nil, err
	}
	byID := map[uint]*model.Project{}
	for i := range projects {
		byID[projects[i].ID] = &projects[i]
	}
	for _, row := range counts {
		project := byID[row.ProjectID]
		if project == nil {
			continue
		}
		project.ArchivedTaskCount += row.Archived
		if row.Status == "done" {
			project.DoneTaskCount += row.Total
		} else {
			project.OpenTaskCount += row.Total
		}
		if row.Status == "doing" {
			project.DoingTaskCount += row.Total
		}
		if row.Status == "next" {
			project.NextTaskCount += row.Total
		}
	}
	return projects, nil
}

func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *ProjectRepository) GetProject(ctx context.Context, userID, projectID uint) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error
	return &project, err
}

func (r *ProjectRepository) Update(ctx context.Context, userID, projectID uint, changes map[string]interface{}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old model.Project
		if err := tx.Where("id = ? AND user_id = ?", projectID, userID).First(&old).Error; err != nil {
			return err
		}
		if status, ok := changes["status"].(string); ok {
			if status == "completed" && old.Status != "completed" {
				changes["completed_at"] = time.Now().UTC()
			}
			if status == "active" && old.Status != "active" {
				changes["completed_at"] = nil
			}
		}
		return tx.Model(&old).Updates(changes).Error
	})
}

func (r *ProjectRepository) ListTasks(ctx context.Context, userID uint, projectID *uint, status, due string, limit, offset int, archived ...bool) ([]model.ProjectTask, error) {
	var tasks []model.ProjectTask
	query := r.db.WithContext(ctx).Model(&model.ProjectTask{}).
		Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ?", userID)
	if len(archived) > 0 && archived[0] {
		query = query.Where("project_tasks.archived_at IS NOT NULL")
	} else {
		query = query.Where("project_tasks.archived_at IS NULL")
	}
	if projectID != nil {
		query = query.Where("project_tasks.project_id = ?", *projectID)
	} else if !(len(archived) > 0 && archived[0]) {
		query = query.Where("projects.status <> ?", "archived")
	}
	if status != "" {
		query = query.Where("project_tasks.status = ?", status)
	}
	if due != "" {
		query = query.Where("project_tasks.due_date = ?", due)
	}
	if status == "done" && limit > 0 {
		query = query.Order("project_tasks.sort_order DESC, project_tasks.id DESC")
	} else {
		query = query.Order("project_tasks.sort_order ASC, project_tasks.id ASC")
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

func (r *ProjectRepository) GetTask(ctx context.Context, userID, taskID uint) (*model.ProjectTask, error) {
	var task model.ProjectTask
	err := r.db.WithContext(ctx).Model(&model.ProjectTask{}).
		Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ? AND project_tasks.id = ?", userID, taskID).First(&task).Error
	return &task, err
}

func (r *ProjectRepository) CreateTask(ctx context.Context, userID, projectID uint, task *model.ProjectTask) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.CreateTaskInTx(tx, userID, projectID, task, false)
	})
}

// CreateTaskInTx shares ordering, completion timestamp, and project touch rules
// with Inbox conversion while allowing both writes to commit atomically.
func (r *ProjectRepository) CreateTaskInTx(tx *gorm.DB, userID, projectID uint, task *model.ProjectTask, requireActive bool) error {
	projectQuery := tx.Model(&model.Project{}).Select("id").Where("id = ? AND user_id = ?", projectID, userID)
	if requireActive {
		projectQuery = projectQuery.Where("status = ?", "active")
	}
	var project model.Project
	if err := projectQuery.First(&project).Error; err != nil {
		return err
	}
	var maxOrder int64
	if err := tx.Model(&model.ProjectTask{}).Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ? AND project_tasks.status = ?", userID, task.Status).
		Select("COALESCE(MAX(project_tasks.sort_order), 0)").Scan(&maxOrder).Error; err != nil {
		return err
	}
	task.ProjectID = projectID
	task.SortOrder = maxOrder + 1024
	if task.Status == "done" {
		now := time.Now().UTC()
		task.CompletedAt = &now
		task.ActualCompletedAt = &now
	}
	if err := tx.Create(task).Error; err != nil {
		return err
	}
	return touchProject(tx, projectID)
}

func (r *ProjectRepository) UpdateTask(ctx context.Context, userID, taskID uint, changes map[string]interface{}) (*model.ProjectTask, error) {
	var task model.ProjectTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ownedTask(tx, userID, taskID, &task); err != nil {
			return err
		}
		if task.ArchivedAt != nil {
			return ErrTaskArchive
		}
		if status, ok := changes["status"].(string); ok && status != task.Status {
			var maxOrder int64
			if err := tx.Model(&model.ProjectTask{}).Joins("JOIN projects ON projects.id = project_tasks.project_id").
				Where("projects.user_id = ? AND project_tasks.status = ?", userID, status).
				Select("COALESCE(MAX(project_tasks.sort_order), 0)").Scan(&maxOrder).Error; err != nil {
				return err
			}
			changes["sort_order"] = maxOrder + 1024
			if status == "done" {
				changes["completed_at"] = time.Now().UTC()
				changes["actual_completed_at"] = changes["completed_at"]
			} else {
				changes["completed_at"] = nil
				changes["actual_completed_at"] = nil
			}
		}
		if err := tx.Model(&task).Updates(changes).Error; err != nil {
			return err
		}
		if err := touchProject(tx, task.ProjectID); err != nil {
			return err
		}
		if newProjectID, ok := changes["project_id"].(uint); ok && newProjectID != task.ProjectID {
			if err := touchProject(tx, newProjectID); err != nil {
				return err
			}
		}
		return tx.First(&task, taskID).Error
	})
	return &task, err
}

// MoveTask orders a task among all of the user's tasks in the target status.
// beforeID is the preceding card; afterID is the following card. A zero/zero
// pair appends to the end. Only the moved row changes on an ordinary move.
func (r *ProjectRepository) MoveTask(ctx context.Context, userID, taskID uint, status string, beforeID, afterID uint, expectedUpdatedAt *time.Time) (*model.ProjectTask, error) {
	var task model.ProjectTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ownedTask(tx, userID, taskID, &task); err != nil {
			return err
		}
		if task.ArchivedAt != nil {
			return ErrTaskArchive
		}
		if expectedUpdatedAt != nil && !task.UpdatedAt.Equal(*expectedUpdatedAt) {
			return ErrTaskConflict
		}
		var column []model.ProjectTask
		if err := tx.Model(&model.ProjectTask{}).Joins("JOIN projects ON projects.id = project_tasks.project_id").
			Where("projects.user_id = ? AND project_tasks.status = ? AND project_tasks.id <> ? AND project_tasks.archived_at IS NULL", userID, status, taskID).
			Order("project_tasks.sort_order ASC, project_tasks.id ASC").Find(&column).Error; err != nil {
			return err
		}
		index := len(column)
		if afterID != 0 {
			index = -1
			for i := range column {
				if column[i].ID == afterID {
					index = i
					break
				}
			}
			if index < 0 {
				return ErrTaskOrder
			}
		}
		if beforeID != 0 {
			beforeIndex := -1
			for i := range column {
				if column[i].ID == beforeID {
					beforeIndex = i
					break
				}
			}
			if beforeIndex < 0 || (afterID != 0 && beforeIndex >= index) {
				return ErrTaskOrder
			}
			index = beforeIndex + 1
		}
		var lower, upper int64
		if index > 0 {
			lower = column[index-1].SortOrder
		}
		if index < len(column) {
			upper = column[index].SortOrder
		}
		if index < len(column) && upper-lower <= 1 {
			for i := range column {
				column[i].SortOrder = int64(i+1) * 1024
				if err := tx.Exec("UPDATE project_tasks SET sort_order = ? WHERE id = ?", column[i].SortOrder, column[i].ID).Error; err != nil {
					return err
				}
			}
			lower, upper = 0, 0
			if index > 0 {
				lower = column[index-1].SortOrder
			}
			if index < len(column) {
				upper = column[index].SortOrder
			}
		}
		order := lower + 1024
		if index < len(column) {
			order = lower + (upper-lower)/2
		}
		if task.Status == status && task.SortOrder == order {
			return nil
		}
		changes := map[string]interface{}{"status": status, "sort_order": order}
		if status == "done" && task.Status != "done" {
			changes["completed_at"] = time.Now().UTC()
			changes["actual_completed_at"] = changes["completed_at"]
		}
		if status != "done" && task.Status == "done" {
			changes["completed_at"] = nil
			changes["actual_completed_at"] = nil
		}
		if err := tx.Model(&task).Updates(changes).Error; err != nil {
			return err
		}
		if err := touchProject(tx, task.ProjectID); err != nil {
			return err
		}
		return tx.First(&task, taskID).Error
	})
	return &task, err
}

func (r *ProjectRepository) DeleteTask(ctx context.Context, userID, projectID, taskID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ownedProject(tx, userID, projectID); err != nil {
			return err
		}
		result := tx.Where("id = ? AND project_id = ?", taskID, projectID).Delete(&model.ProjectTask{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return touchProject(tx, projectID)
	})
}

func ownedProject(tx *gorm.DB, userID, projectID uint) error {
	var project model.Project
	return tx.Select("id").Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error
}

func ownedTask(tx *gorm.DB, userID, taskID uint, task *model.ProjectTask) error {
	return tx.Model(&model.ProjectTask{}).Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ? AND project_tasks.id = ?", userID, taskID).First(task).Error
}

func touchProject(tx *gorm.DB, projectID uint) error {
	return tx.Model(&model.Project{}).Where("id = ?", projectID).Update("updated_at", time.Now().UTC()).Error
}

// ArchiveTasks changes visibility only. Validate the entire batch before writing.
func (r *ProjectRepository) ArchiveTasks(ctx context.Context, userID uint, ids []uint, archive bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tasks := make([]model.ProjectTask, len(ids))
		for i, id := range ids {
			if err := ownedTask(tx, userID, id, &tasks[i]); err != nil {
				return err
			}
			if tasks[i].Status != "done" {
				return ErrTaskArchive
			}
		}
		now := time.Now().UTC()
		for _, task := range tasks {
			if (task.ArchivedAt != nil) == archive {
				continue
			}
			var at *time.Time
			if archive {
				at = &now
			}
			if err := tx.Model(&task).Updates(map[string]interface{}{"archived_at": at}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
