package service

import (
	"context"
	"errors"
	"learnos/internal/links"
	"learnos/internal/model"
	"learnos/internal/repository"
	"strings"
	"time"
)

var ErrInvalidLifeInput = errors.New("invalid life input")
var LifeDomains = []string{"finance", "health", "relationships", "work", "learning", "experiences", "home", "rhythm"}

func ValidLifeDomain(d string) bool {
	if d == "" {
		return true
	}
	for _, v := range LifeDomains {
		if d == v {
			return true
		}
	}
	return false
}
func lifeDate(d string) bool {
	t, err := time.Parse("2006-01-02", d)
	return err == nil && t.Format("2006-01-02") == d && t.Year() > 0
}
func lifeText(v string, max int) bool { return len([]rune(v)) <= max }
func keyPointer(key string) *string {
	if key == "" {
		return nil
	}
	return &key
}

type LifeService struct{ repo *repository.LifeRepository }

func NewLifeService(repo *repository.LifeRepository) *LifeService { return &LifeService{repo: repo} }

type LifeEventInput struct {
	CreationKey     string `json:"creation_key"`
	Title           string `json:"title"`
	OccurredOn      string `json:"occurred_on"`
	Description     string `json:"description"`
	PrimaryDomain   string `json:"primary_domain"`
	SecondaryDomain string `json:"secondary_domain"`
	Milestone       bool   `json:"milestone"`
	GoalID          *uint  `json:"goal_id"`
	ExternalURL     string `json:"external_url"`
	LinkName        string `json:"link_name"`
	SourceType      string `json:"source_type"`
	SourceID        *uint  `json:"source_id"`
}

func (s *LifeService) SaveEvent(ctx context.Context, user, id uint, in LifeEventInput) (*model.LifeEvent, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.ExternalURL = strings.TrimSpace(in.ExternalURL)
	in.LinkName = strings.TrimSpace(in.LinkName)
	if in.Title == "" || !lifeText(in.Title, 200) || !lifeText(in.Description, 10000) || !lifeDate(in.OccurredOn) || !ValidLifeDomain(in.PrimaryDomain) || !ValidLifeDomain(in.SecondaryDomain) || (in.SecondaryDomain != "" && (in.PrimaryDomain == "" || in.SecondaryDomain == in.PrimaryDomain)) || (in.GoalID != nil && *in.GoalID == 0) || !links.SafeExternal(in.ExternalURL) || !lifeText(in.ExternalURL, 2048) || !lifeText(in.LinkName, 160) || !validRequestKey(in.CreationKey) {
		return nil, ErrInvalidLifeInput
	}
	if (in.SourceID == nil) != (in.SourceType == "") || (in.SourceID != nil && (*in.SourceID == 0 || (in.SourceType != "project" && in.SourceType != "course" && in.SourceType != "inbox"))) {
		return nil, ErrInvalidLifeInput
	}
	if id == 0 && in.SourceID == nil && in.CreationKey == "" {
		return nil, ErrInvalidLifeInput
	}
	e := model.LifeEvent{CreationKey: keyPointer(in.CreationKey), Title: in.Title, OccurredOn: in.OccurredOn, Description: in.Description, PrimaryDomain: in.PrimaryDomain, SecondaryDomain: in.SecondaryDomain, Milestone: in.Milestone, GoalID: in.GoalID, ExternalURL: in.ExternalURL, LinkName: in.LinkName, SourceType: in.SourceType, SourceID: in.SourceID}
	return s.repo.SaveEvent(ctx, user, id, e)
}
func (s *LifeService) Events(ctx context.Context, user uint, domain string, milestones bool) ([]model.LifeEvent, error) {
	if !ValidLifeDomain(domain) {
		return nil, ErrInvalidLifeInput
	}
	return s.repo.Events(ctx, user, domain, milestones)
}
func (s *LifeService) Event(ctx context.Context, user, id uint) (*model.LifeEvent, error) {
	return s.repo.Event(ctx, user, id)
}
func (s *LifeService) DeleteEvent(ctx context.Context, user, id uint) error {
	return s.repo.DeleteEvent(ctx, user, id)
}
func (s *LifeService) Source(ctx context.Context, user, id uint, kind string) (*repository.LifeSource, error) {
	return s.repo.Source(ctx, user, id, kind)
}
func (s *LifeService) Graduate(ctx context.Context, user, id uint) error {
	return s.repo.Graduate(ctx, user, id)
}

type LifeGoalInput struct {
	RecordChange bool   `json:"record_change"`
	CreationKey  string `json:"creation_key"`
	Title        string `json:"title"`
	Why          string `json:"why"`
	CurrentNote  string `json:"current_note"`
	Status       string `json:"status"`
	Domain       string `json:"domain"`
	ProjectID    *uint  `json:"project_id"`
	ExternalURL  string `json:"external_url"`
	LinkName     string `json:"link_name"`
	StatusDate   string `json:"status_date"`
	StatusReason string `json:"status_reason"`
}

func validLifeStatus(v string) bool {
	return v == "considering" || v == "active" || v == "paused" || v == "achieved" || v == "ended"
}
func (s *LifeService) SaveGoal(ctx context.Context, user, id uint, in LifeGoalInput) (*model.LifeGoal, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Why = strings.TrimSpace(in.Why)
	in.CurrentNote = strings.TrimSpace(in.CurrentNote)
	in.ExternalURL = strings.TrimSpace(in.ExternalURL)
	in.LinkName = strings.TrimSpace(in.LinkName)
	in.StatusReason = strings.TrimSpace(in.StatusReason)
	if in.Title == "" || !lifeText(in.Title, 200) || !lifeText(in.Why, 10000) || !lifeText(in.CurrentNote, 10000) || !validLifeStatus(in.Status) || !ValidLifeDomain(in.Domain) || (in.ProjectID != nil && *in.ProjectID == 0) || !links.SafeExternal(in.ExternalURL) || !lifeText(in.ExternalURL, 2048) || !lifeText(in.LinkName, 160) || !validRequestKey(in.CreationKey) || !lifeText(in.StatusReason, 4000) || (id == 0 && in.CreationKey == "") || (id != 0 && !lifeDate(in.StatusDate)) {
		return nil, ErrInvalidLifeInput
	}
	return s.repo.SaveGoal(ctx, user, id, model.LifeGoal{CreationKey: keyPointer(in.CreationKey), Title: in.Title, Why: in.Why, CurrentNote: in.CurrentNote, Status: in.Status, Domain: in.Domain, ProjectID: in.ProjectID, ExternalURL: in.ExternalURL, LinkName: in.LinkName}, in.StatusDate, in.StatusReason, in.RecordChange)
}
func (s *LifeService) Goals(ctx context.Context, user uint) ([]model.LifeGoal, error) {
	return s.repo.Goals(ctx, user)
}
func (s *LifeService) Goal(ctx context.Context, user, id uint) (*model.LifeGoal, []model.LifeGoalEntry, []model.LifeEvent, error) {
	return s.repo.Goal(ctx, user, id)
}

type LifeEntryInput struct {
	CreationKey string `json:"creation_key"`
	OccurredOn  string `json:"occurred_on"`
	Content     string `json:"content"`
	Reason      string `json:"reason"`
}

func (s *LifeService) AddEntry(ctx context.Context, user, id uint, in LifeEntryInput) (*model.LifeGoalEntry, error) {
	in.Content = strings.TrimSpace(in.Content)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Content == "" || !lifeText(in.Content, 10000) || !lifeText(in.Reason, 4000) || !lifeDate(in.OccurredOn) || in.CreationKey == "" || !validRequestKey(in.CreationKey) {
		return nil, ErrInvalidLifeInput
	}
	return s.repo.AddEntry(ctx, user, id, model.LifeGoalEntry{CreationKey: keyPointer(in.CreationKey), OccurredOn: in.OccurredOn, Content: in.Content, Reason: in.Reason})
}
