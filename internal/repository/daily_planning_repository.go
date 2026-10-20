package repository

import (
	"context"
	"learnos/internal/model"
)

// PlanningTask adds derived project metadata; no second task or context is stored.
type PlanningTask struct {
	model.ProjectTask `gorm:"embedded"`
	ProjectTitle      string `json:"project_title"`
	EffectiveContext  string `json:"effective_context"`
}

func (r *ProjectRepository) PlanningTasks(ctx context.Context, userID uint, date, scene string) ([]PlanningTask, error) {
	rows := []PlanningTask{}
	q := r.db.WithContext(ctx).Table("project_tasks").
		Select("project_tasks.*, projects.title AS project_title, projects.context AS effective_context").
		Joins("JOIN projects ON projects.id = project_tasks.project_id").
		Where("projects.user_id = ? AND projects.status = ? AND project_tasks.archived_at IS NULL AND project_tasks.status <> ?", userID, "active", "done").
		Where("project_tasks.plan_date <= ? OR project_tasks.due_date <= ? OR project_tasks.status IN ?", date, date, []string{"doing", "next"})
	if scene != "all" {
		q = q.Where("projects.context = ?", scene)
	}
	err := q.Order("project_tasks.sort_order ASC, project_tasks.id ASC").Find(&rows).Error
	return rows, err
}
