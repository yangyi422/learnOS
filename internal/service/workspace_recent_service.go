package service

import (
	"context"
	"learnos/internal/repository"
)

type WorkspaceRecentService struct {
	facts *repository.WorkspaceRecentRepository
}

func NewWorkspaceRecentService(facts *repository.WorkspaceRecentRepository) *WorkspaceRecentService {
	return &WorkspaceRecentService{facts: facts}
}

type WorkspaceRecent struct {
	Library   []repository.RecentLibraryPage `json:"library"`
	Events    []repository.RecentLifeEvent   `json:"events"`
	Goals     []repository.RecentGoalEntry   `json:"goals"`
	AssetDate string                         `json:"asset_date"`
	Errors    map[string]string              `json:"errors"`
}

func (s *WorkspaceRecentService) Recent(ctx context.Context, user uint) *WorkspaceRecent {
	view := &WorkspaceRecent{Library: []repository.RecentLibraryPage{}, Events: []repository.RecentLifeEvent{}, Goals: []repository.RecentGoalEntry{}, Errors: map[string]string{}}
	if rows, err := s.facts.Library(ctx, user); err != nil {
		view.Errors["library"] = "最近资料暂时无法读取。"
	} else {
		view.Library = rows
	}
	if rows, err := s.facts.Events(ctx, user); err != nil {
		view.Errors["events"] = "生活记录暂时无法读取。"
	} else {
		view.Events = rows
	}
	if rows, err := s.facts.Goals(ctx, user); err != nil {
		view.Errors["goals"] = "目标变化暂时无法读取。"
	} else {
		view.Goals = rows
	}
	if date, err := s.facts.AssetDate(ctx, user); err != nil {
		view.Errors["assets"] = "资产记录日期暂时无法读取。"
	} else {
		view.AssetDate = date
	}
	return view
}
