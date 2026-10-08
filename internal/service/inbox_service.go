package service

import (
	"context"
	"errors"
	"learnos/internal/links"
	"strings"

	"learnos/internal/model"
	"learnos/internal/repository"
)

var ErrInvalidInboxInput = errors.New("invalid inbox input")

type InboxService struct {
	items    *repository.InboxRepository
	projects *repository.ProjectRepository
}

func NewInboxService(items *repository.InboxRepository, projects *repository.ProjectRepository) *InboxService {
	return &InboxService{items: items, projects: projects}
}

type InboxView struct {
	Items  []model.InboxItem `json:"items"`
	Counts map[string]int64  `json:"counts"`
}

type InboxSummary struct {
	Count int64             `json:"count"`
	Items []model.InboxItem `json:"items"`
}

func (s *InboxService) List(ctx context.Context, userID uint, status string) (*InboxView, error) {
	if status != "" && status != model.InboxStatusInbox && status != model.InboxStatusProcessed && status != model.InboxStatusArchived {
		return nil, ErrInvalidInboxInput
	}
	items, counts, err := s.items.List(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []model.InboxItem{}
	}
	return &InboxView{Items: items, Counts: counts}, nil
}

func (s *InboxService) Summary(ctx context.Context, userID uint) (*InboxSummary, error) {
	items, counts, err := s.items.List(ctx, userID, model.InboxStatusInbox)
	if err != nil {
		return nil, err
	}
	if len(items) > 3 {
		items = items[:3]
	}
	return &InboxSummary{Count: counts[model.InboxStatusInbox], Items: items}, nil
}

func (s *InboxService) Create(ctx context.Context, userID uint, content string) (*model.InboxItem, error) {
	return s.CreateWithKey(ctx, userID, content, "")
}
func (s *InboxService) CreateWithKey(ctx context.Context, userID uint, content, key string) (*model.InboxItem, error) {
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 10000 {
		return nil, ErrInvalidInboxInput
	}
	if !validRequestKey(key) {
		return nil, ErrInvalidInboxInput
	}
	urlValue := validInboxURL(content)
	sourceType := model.InboxSourceManual
	if urlValue != "" {
		sourceType = model.InboxSourceURL
	}
	item := &model.InboxItem{UserID: userID, Content: content, Status: model.InboxStatusInbox, SourceType: sourceType, SourceURL: urlValue}
	if key != "" {
		item.CaptureKey = &key
	}
	if err := s.items.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *InboxService) Update(ctx context.Context, userID, itemID uint, content string) (*model.InboxItem, error) {
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 10000 {
		return nil, ErrInvalidInboxInput
	}
	return s.items.UpdateContent(ctx, userID, itemID, content)
}

func (s *InboxService) Archive(ctx context.Context, userID, itemID uint) error {
	return s.items.Archive(ctx, userID, itemID)
}

func (s *InboxService) Delete(ctx context.Context, userID, itemID uint) error {
	return s.items.Delete(ctx, userID, itemID)
}

type InboxConvertInput struct {
	Title       string  `json:"title"`
	ProjectID   uint    `json:"project_id"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	Description string  `json:"description"`
}

func (s *InboxService) ConvertToTask(ctx context.Context, userID, itemID uint, input InboxConvertInput) (*model.InboxItem, *model.ProjectTask, error) {
	item, err := s.items.Find(ctx, userID, itemID)
	if err != nil {
		return nil, nil, err
	}
	if item.Status == model.InboxStatusArchived {
		return nil, nil, repository.ErrInboxAlreadyArchived
	}
	if item.Status != model.InboxStatusInbox {
		return nil, nil, repository.ErrInboxAlreadyProcessed
	}
	if strings.TrimSpace(input.Title) == "" {
		input.Title = firstLine(item.Content)
	}
	if input.Description == "" && len([]rune(strings.TrimSpace(item.Content))) > 200 {
		input.Description = item.Content
	}
	taskInput, err := normalizeProjectTaskInput(ProjectTaskInput{ProjectID: input.ProjectID, Title: input.Title, Description: input.Description, Status: input.Status, Priority: input.Priority, DueDate: input.DueDate})
	if err != nil {
		return nil, nil, err
	}
	task := model.ProjectTask{Title: taskInput.Title, Description: taskInput.Description, Status: taskInput.Status, Priority: taskInput.Priority, DueDate: taskInput.DueDate}
	return s.items.ConvertToTask(ctx, userID, itemID, task, taskInput.ProjectID, s.projects)
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if line, _, found := strings.Cut(value, "\n"); found {
		value = strings.TrimSpace(line)
	}
	runes := []rune(value)
	if len(runes) > 200 {
		return strings.TrimSpace(string(runes[:200]))
	}
	return value
}

func validInboxURL(value string) string {
	if value != "" && links.SafeExternal(value) {
		return value
	}
	return ""
}
