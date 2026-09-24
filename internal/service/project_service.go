package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidProjectInput = errors.New("invalid project input")
var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type ProjectService struct{ projects *repository.ProjectRepository }

func NewProjectService(projects *repository.ProjectRepository) *ProjectService {
	return &ProjectService{projects: projects}
}

type ProjectPatch struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Icon        *string `json:"icon"`
	Accent      *string `json:"accent"`
}

type ProjectTaskInput struct {
	ProjectID   uint    `json:"project_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
}

type ProjectTaskPatch struct {
	ProjectID    *uint   `json:"project_id"`
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Status       *string `json:"status"`
	Priority     *string `json:"priority"`
	DueDate      *string `json:"due_date"`
	IsNextAction *bool   `json:"is_next_action"`
}

func (s *ProjectService) List(ctx context.Context, userID uint) ([]model.Project, error) {
	projects, err := s.projects.List(ctx, userID)
	if projects == nil {
		projects = []model.Project{}
	}
	for i := range projects {
		if projects[i].Tasks == nil {
			projects[i].Tasks = []model.ProjectTask{}
		}
	}
	return projects, err
}

func (s *ProjectService) Create(ctx context.Context, userID uint, title string) (*model.Project, error) {
	return s.CreateProject(ctx, userID, ProjectPatch{Title: &title})
}

func (s *ProjectService) CreateProject(ctx context.Context, userID uint, input ProjectPatch) (*model.Project, error) {
	if input.Title == nil {
		return nil, ErrInvalidProjectInput
	}
	title := strings.TrimSpace(*input.Title)
	if title == "" || len([]rune(title)) > 160 {
		return nil, ErrInvalidProjectInput
	}
	project := &model.Project{UserID: userID, Title: title, Status: "active", Tasks: []model.ProjectTask{}}
	if input.Description != nil {
		if len([]rune(*input.Description)) > 4000 {
			return nil, ErrInvalidProjectInput
		}
		project.Description = strings.TrimSpace(*input.Description)
	}
	if input.Icon != nil {
		if len([]rune(*input.Icon)) > 8 {
			return nil, ErrInvalidProjectInput
		}
		project.Icon = strings.TrimSpace(*input.Icon)
	}
	if input.Accent != nil {
		if *input.Accent != "" && !accentPattern.MatchString(*input.Accent) {
			return nil, ErrInvalidProjectInput
		}
		project.Accent = *input.Accent
	}
	if err := s.projects.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) Update(ctx context.Context, userID, projectID uint, patch ProjectPatch) error {
	changes := map[string]interface{}{}
	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" || len([]rune(title)) > 160 {
			return ErrInvalidProjectInput
		}
		changes["title"] = title
	}
	if patch.Description != nil {
		if len([]rune(*patch.Description)) > 4000 {
			return ErrInvalidProjectInput
		}
		changes["description"] = strings.TrimSpace(*patch.Description)
	}
	if patch.Icon != nil {
		if len([]rune(*patch.Icon)) > 8 {
			return ErrInvalidProjectInput
		}
		changes["icon"] = strings.TrimSpace(*patch.Icon)
	}
	if patch.Accent != nil {
		if *patch.Accent != "" && !accentPattern.MatchString(*patch.Accent) {
			return ErrInvalidProjectInput
		}
		changes["accent"] = *patch.Accent
	}
	if patch.Status != nil {
		if !oneOf(*patch.Status, "active", "paused", "completed", "archived") {
			return ErrInvalidProjectInput
		}
		changes["status"] = *patch.Status
		if *patch.Status == "archived" {
			changes["archived_at"] = time.Now().UTC()
		} else {
			changes["archived_at"] = nil
		}
	}
	if len(changes) == 0 {
		return ErrInvalidProjectInput
	}
	return s.projects.Update(ctx, userID, projectID, changes)
}

func (s *ProjectService) ListTasks(ctx context.Context, userID uint, projectID *uint, status, due string, limit, offset int) ([]model.ProjectTask, error) {
	if status != "" && !taskStatus(status) {
		return nil, ErrInvalidProjectInput
	}
	if due != "" && !validDueDate(&due) {
		return nil, ErrInvalidProjectInput
	}
	if limit < 0 || limit > 500 {
		return nil, ErrInvalidProjectInput
	}
	if offset < 0 {
		return nil, ErrInvalidProjectInput
	}
	if projectID != nil {
		if _, err := s.projects.GetProject(ctx, userID, *projectID); err != nil {
			return nil, err
		}
	}
	tasks, err := s.projects.ListTasks(ctx, userID, projectID, status, due, limit, offset)
	if tasks == nil {
		tasks = []model.ProjectTask{}
	}
	return tasks, err
}

func (s *ProjectService) CreateTask(ctx context.Context, userID, projectID uint, title string) (*model.ProjectTask, error) {
	return s.CreateTaskWithInput(ctx, userID, ProjectTaskInput{ProjectID: projectID, Title: title})
}

func (s *ProjectService) CreateTaskWithInput(ctx context.Context, userID uint, input ProjectTaskInput) (*model.ProjectTask, error) {
	title := strings.TrimSpace(input.Title)
	if input.ProjectID == 0 || title == "" || len([]rune(title)) > 200 || len([]rune(input.Description)) > 10000 {
		return nil, ErrInvalidProjectInput
	}
	if input.Status == "" {
		input.Status = "inbox"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}
	if !taskStatus(input.Status) || !priority(input.Priority) || !validDueDate(input.DueDate) {
		return nil, ErrInvalidProjectInput
	}
	task := &model.ProjectTask{Title: title, Description: strings.TrimSpace(input.Description), Status: input.Status, Priority: input.Priority, DueDate: input.DueDate}
	if err := s.projects.CreateTask(ctx, userID, input.ProjectID, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *ProjectService) EditTask(ctx context.Context, userID, taskID uint, patch ProjectTaskPatch) (*model.ProjectTask, error) {
	changes := map[string]interface{}{}
	if patch.ProjectID != nil {
		if *patch.ProjectID == 0 {
			return nil, ErrInvalidProjectInput
		}
		if _, err := s.projects.GetProject(ctx, userID, *patch.ProjectID); err != nil {
			return nil, err
		}
		changes["project_id"] = *patch.ProjectID
	}
	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" || len([]rune(title)) > 200 {
			return nil, ErrInvalidProjectInput
		}
		changes["title"] = title
	}
	if patch.Description != nil {
		if len([]rune(*patch.Description)) > 10000 {
			return nil, ErrInvalidProjectInput
		}
		changes["description"] = strings.TrimSpace(*patch.Description)
	}
	if patch.Status != nil {
		if !taskStatus(*patch.Status) {
			return nil, ErrInvalidProjectInput
		}
		changes["status"] = *patch.Status
		changes["is_next_action"] = false
	}
	if patch.Priority != nil {
		if !priority(*patch.Priority) {
			return nil, ErrInvalidProjectInput
		}
		changes["priority"] = *patch.Priority
	}
	if patch.DueDate != nil {
		if !validDueDate(patch.DueDate) {
			return nil, ErrInvalidProjectInput
		}
		if *patch.DueDate == "" {
			changes["due_date"] = nil
		} else {
			changes["due_date"] = *patch.DueDate
		}
	}
	if patch.IsNextAction != nil {
		if *patch.IsNextAction {
			changes["status"] = "next"
		} else {
			changes["is_next_action"] = false
		}
	}
	if len(changes) == 0 {
		return nil, ErrInvalidProjectInput
	}
	return s.projects.UpdateTask(ctx, userID, taskID, changes)
}

func (s *ProjectService) UpdateTask(ctx context.Context, userID, projectID, taskID uint, patch ProjectTaskPatch) error {
	task, err := s.projects.GetTask(ctx, userID, taskID)
	if err != nil {
		return err
	}
	if task.ProjectID != projectID {
		return gorm.ErrRecordNotFound
	}
	_, err = s.EditTask(ctx, userID, taskID, patch)
	return err
}

func (s *ProjectService) MoveTask(ctx context.Context, userID, taskID uint, status string, beforeID, afterID uint, expectedUpdatedAt *time.Time) (*model.ProjectTask, error) {
	if !taskStatus(status) || beforeID == taskID || afterID == taskID || (beforeID != 0 && beforeID == afterID) {
		return nil, ErrInvalidProjectInput
	}
	return s.projects.MoveTask(ctx, userID, taskID, status, beforeID, afterID, expectedUpdatedAt)
}

func (s *ProjectService) DeleteTask(ctx context.Context, userID, projectID, taskID uint) error {
	return s.projects.DeleteTask(ctx, userID, projectID, taskID)
}

func (s *ProjectService) DeleteTaskByID(ctx context.Context, userID, taskID uint) error {
	task, err := s.projects.GetTask(ctx, userID, taskID)
	if err != nil {
		return err
	}
	return s.projects.DeleteTask(ctx, userID, task.ProjectID, taskID)
}

func taskStatus(status string) bool { return oneOf(status, "inbox", "next", "doing", "done") }
func priority(value string) bool    { return oneOf(value, "high", "normal", "low") }
func validDueDate(value *string) bool {
	if value == nil || *value == "" {
		return true
	}
	parsed, err := time.Parse("2006-01-02", *value)
	return err == nil && parsed.Format("2006-01-02") == *value
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
