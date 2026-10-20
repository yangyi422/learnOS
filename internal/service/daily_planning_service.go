package service

import (
	"context"
	"learnos/internal/model"
	"learnos/internal/repository"
	"time"
)

type PlanningSummary struct {
	Planned      int `json:"planned"`
	DueToday     int `json:"due_today"`
	Overdue      int `json:"overdue"`
	PreviousPlan int `json:"previous_plan"`
	Total        int `json:"total"`
}
type WorkspacePlanning struct {
	Items        []repository.PlanningTask `json:"items"`
	DueToday     []repository.PlanningTask `json:"due_today"`
	Overdue      []repository.PlanningTask `json:"overdue"`
	PreviousPlan []repository.PlanningTask `json:"previous_plan"`
	Summary      PlanningSummary           `json:"summary"`
}

func (s *ProjectService) DailyPlanning(ctx context.Context, userID uint, date, scene string) (*WorkspacePlanning, []repository.PlanningTask, error) {
	if date == "" || !validDueDate(&date) || !oneOf(scene, "all", "work", "life") {
		return nil, nil, ErrInvalidProjectInput
	}
	tasks, err := s.projects.PlanningTasks(ctx, userID, date, scene)
	if err != nil {
		return nil, nil, err
	}
	p := &WorkspacePlanning{Items: []repository.PlanningTask{}, DueToday: []repository.PlanningTask{}, Overdue: []repository.PlanningTask{}, PreviousPlan: []repository.PlanningTask{}}
	for _, t := range tasks {
		// One primary position per task. Overlapping dates remain visible as metadata.
		switch {
		case t.PlanDate != nil && *t.PlanDate == date:
			p.Items = append(p.Items, t)
		case t.DueDate != nil && *t.DueDate < date:
			p.Overdue = append(p.Overdue, t)
		case t.DueDate != nil && *t.DueDate == date:
			p.DueToday = append(p.DueToday, t)
		case t.PlanDate != nil && *t.PlanDate < date:
			p.PreviousPlan = append(p.PreviousPlan, t)
		}
	}
	p.Summary = PlanningSummary{Planned: len(p.Items), DueToday: len(p.DueToday), Overdue: len(p.Overdue), PreviousPlan: len(p.PreviousPlan)}
	p.Summary.Total = p.Summary.Planned + p.Summary.DueToday + p.Summary.Overdue + p.Summary.PreviousPlan
	return p, tasks, nil
}

func focusFromPlanning(date string, tasks []repository.PlanningTask, learning []WorkspaceLearningItem) WorkspaceFocus {
	groups := make([][]model.ProjectTask, 4)
	projects := []model.Project{}
	day, _ := time.Parse("2006-01-02", date)
	longOverdue := day.AddDate(0, 0, -7).Format("2006-01-02")
	for _, t := range tasks {
		projects = append(projects, model.Project{ID: t.ProjectID, Title: t.ProjectTitle})
		planned := t.PlanDate != nil && *t.PlanDate == date
		if t.Status == "doing" && (t.DueDate == nil || *t.DueDate >= longOverdue) && (t.PlanDate == nil || *t.PlanDate <= date) {
			groups[0] = append(groups[0], t.ProjectTask)
		}
		if planned {
			groups[1] = append(groups[1], t.ProjectTask)
		}
		if t.DueDate != nil && *t.DueDate == date {
			groups[2] = append(groups[2], t.ProjectTask)
		}
		if t.Status == "next" && t.Priority == "high" && (t.DueDate == nil || *t.DueDate >= date) && (t.PlanDate == nil || *t.PlanDate == date) {
			groups[3] = append(groups[3], t.ProjectTask)
		}
	}
	for _, group := range groups {
		if task, ok := focusTask(group); ok {
			return focusFromToday(&TodayView{Doing: []model.ProjectTask{task}}, projects, nil)
		}
	}
	return focusFromToday(nil, nil, learning)
}
