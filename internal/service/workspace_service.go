package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"learnos/internal/model"
)

type WorkspaceService struct {
	projects  *ProjectService
	courses   *CourseService
	cognitive *CognitiveStateService
	inbox     *InboxService
}

func NewWorkspaceService(projects *ProjectService, courses *CourseService, cognitive *CognitiveStateService, inbox *InboxService) *WorkspaceService {
	return &WorkspaceService{projects: projects, courses: courses, cognitive: cognitive, inbox: inbox}
}

type WorkspaceLearningItem struct {
	CourseID         uint       `json:"course_id"`
	CourseName       string     `json:"course_name"`
	UnitTitle        string     `json:"unit_title"`
	LessonID         uint       `json:"lesson_id"`
	LessonTitle      string     `json:"lesson_title"`
	CoreQuestion     string     `json:"core_question"`
	CurrentLevel     string     `json:"current_level"`
	CognitiveStatus  string     `json:"cognitive_status"`
	LastLearningAt   *time.Time `json:"last_learning_at"`
	HasCurrentLesson bool       `json:"has_current_lesson"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type WorkspaceFocus struct {
	Kind            string  `json:"kind"`
	Title           string  `json:"title"`
	ProjectID       uint    `json:"project_id,omitempty"`
	ProjectTitle    string  `json:"project_title,omitempty"`
	TaskID          uint    `json:"task_id,omitempty"`
	Status          string  `json:"status,omitempty"`
	Priority        string  `json:"priority,omitempty"`
	DueDate         *string `json:"due_date,omitempty"`
	CourseID        uint    `json:"course_id,omitempty"`
	CourseName      string  `json:"course_name,omitempty"`
	UnitTitle       string  `json:"unit_title,omitempty"`
	LessonID        uint    `json:"lesson_id,omitempty"`
	CurrentLevel    string  `json:"current_level,omitempty"`
	CognitiveStatus string  `json:"cognitive_status,omitempty"`
}

type WorkspaceHome struct {
	Date     string                  `json:"date"`
	Focus    WorkspaceFocus          `json:"focus"`
	Today    *TodayView              `json:"today,omitempty"`
	Projects []model.Project         `json:"projects"`
	Learning []WorkspaceLearningItem `json:"learning"`
	Inbox    *InboxSummary           `json:"inbox,omitempty"`
	Errors   map[string]string       `json:"errors"`
}

func (s *WorkspaceService) Home(ctx context.Context, userID uint, date string) (*WorkspaceHome, error) {
	view := &WorkspaceHome{
		Date: date, Projects: []model.Project{}, Learning: []WorkspaceLearningItem{},
		Errors: map[string]string{}, Focus: WorkspaceFocus{Kind: "empty"},
	}
	projects, projectsErr := s.projects.List(ctx, userID)
	if projectsErr != nil {
		view.Errors["projects"] = "项目数据暂时无法加载。"
	} else {
		for _, project := range projects {
			if project.Status == "active" && len(view.Projects) < 5 {
				view.Projects = append(view.Projects, project)
			}
		}
	}

	today, todayErr := s.projects.Today(ctx, userID, date, nil)
	if todayErr != nil {
		view.Errors["today"] = "今日任务暂时无法加载。"
	} else {
		view.Today = today
	}

	learning, learningErr := s.learning(ctx, userID)
	if learningErr != nil {
		view.Errors["learning"] = "学习进度暂时无法加载。"
	} else {
		view.Learning = learning
	}

	inbox, inboxErr := s.inbox.Summary(ctx, userID)
	if inboxErr != nil {
		view.Errors["inbox"] = "收集箱暂时无法加载。"
	} else {
		view.Inbox = inbox
	}

	view.Focus = focusFromToday(today, projects, learning)
	return view, nil
}

func (s *WorkspaceService) learning(ctx context.Context, userID uint) ([]WorkspaceLearningItem, error) {
	courses, err := s.courses.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]WorkspaceLearningItem, 0, len(courses))
	for _, course := range courses {
		item := WorkspaceLearningItem{CourseID: course.ID, CourseName: course.Name, UnitTitle: course.CurrentUnit, CurrentLevel: "unseen", CognitiveStatus: "unknown", UpdatedAt: course.UpdatedAt}
		turns, turnsErr := s.courses.ListLearningTurns(ctx, course.ID, 1)
		if turnsErr != nil {
			return nil, turnsErr
		}
		if len(turns) > 0 {
			last := turns[0].CreatedAt
			item.LastLearningAt = &last
		}
		if course.CurrentLessonID != nil {
			current, currentErr := s.courses.GetCurrentLesson(ctx, course.ID)
			if currentErr == nil {
				item.HasCurrentLesson = true
				item.LessonID = current.Lesson.ID
				item.LessonTitle = current.Lesson.Title
				item.CoreQuestion = current.Lesson.CoreQuestion
				item.UnitTitle = current.Unit.Title
				if s.cognitive != nil {
					state, stateErr := s.cognitive.GetLessonCognitiveState(ctx, course.ID, current.Lesson.ID)
					if stateErr != nil {
						return nil, stateErr
					}
					item.CurrentLevel = state.State.CurrentLevel
					item.CognitiveStatus = state.State.Status
				}
			} else if !errors.Is(currentErr, ErrCurrentLessonNotFound) && !errors.Is(currentErr, ErrCourseNotFound) {
				return nil, currentErr
			}
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if (left.LastLearningAt != nil) != (right.LastLearningAt != nil) {
			return left.LastLearningAt != nil
		}
		if left.LastLearningAt != nil && !left.LastLearningAt.Equal(*right.LastLearningAt) {
			return left.LastLearningAt.After(*right.LastLearningAt)
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.CourseID > right.CourseID
	})
	if len(items) > 2 {
		items = items[:2]
	}
	return items, nil
}

func focusFromToday(today *TodayView, projects []model.Project, learning []WorkspaceLearningItem) WorkspaceFocus {
	projectByID := make(map[uint]string, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project.Title
	}
	if today != nil {
		for _, group := range [][]model.ProjectTask{today.Doing, today.Due, highPriorityNext(today.Next)} {
			if task, ok := focusTask(group); ok {
				return WorkspaceFocus{Kind: "task", Title: task.Title, TaskID: task.ID, ProjectID: task.ProjectID, ProjectTitle: projectByID[task.ProjectID], Status: task.Status, Priority: task.Priority, DueDate: task.DueDate}
			}
		}
	}
	if len(learning) > 0 {
		item := learning[0]
		title := item.LessonTitle
		if title == "" {
			title = "选择下一段学习"
		}
		return WorkspaceFocus{Kind: "learning", Title: title, CourseID: item.CourseID, CourseName: item.CourseName, UnitTitle: item.UnitTitle, LessonID: item.LessonID, CurrentLevel: item.CurrentLevel, CognitiveStatus: item.CognitiveStatus}
	}
	return WorkspaceFocus{Kind: "empty"}
}

func highPriorityNext(tasks []model.ProjectTask) []model.ProjectTask {
	result := make([]model.ProjectTask, 0, len(tasks))
	for _, task := range tasks {
		if task.Priority == "high" {
			result = append(result, task)
		}
	}
	return result
}

func focusTask(tasks []model.ProjectTask) (model.ProjectTask, bool) {
	if len(tasks) == 0 {
		return model.ProjectTask{}, false
	}
	ordered := append([]model.ProjectTask(nil), tasks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if priorityRank(left.Priority) != priorityRank(right.Priority) {
			return priorityRank(left.Priority) < priorityRank(right.Priority)
		}
		leftDue, rightDue := "9999-12-31", "9999-12-31"
		if left.DueDate != nil {
			leftDue = *left.DueDate
		}
		if right.DueDate != nil {
			rightDue = *right.DueDate
		}
		if leftDue != rightDue {
			return leftDue < rightDue
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.ID > right.ID
	})
	return ordered[0], true
}

func priorityRank(priority string) int {
	switch priority {
	case "high":
		return 0
	case "normal":
		return 1
	default:
		return 2
	}
}
