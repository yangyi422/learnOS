package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var allowedReasoningPatternKeys = map[string]string{
	"binary_thinking":         "二元化判断",
	"single_factor_reasoning": "单因素归因",
	"overgeneralization":      "过度泛化",
	"boundary_neglect":        "忽略适用边界",
	"dose_neglect":            "忽略剂量 / 程度",
	"correlation_causation":   "相关与因果混淆",
	"category_confusion":      "概念 / 类别混淆",
	"unsupported_assumption":  "无依据假设",
}

type MisconceptionService struct {
	courses        *repository.CourseRepository
	graphs         *repository.KnowledgeGraphRepository
	misconceptions *repository.MisconceptionRepository
}

func NewMisconceptionService(courses *repository.CourseRepository, graphs *repository.KnowledgeGraphRepository, misconceptions *repository.MisconceptionRepository) *MisconceptionService {
	return &MisconceptionService{courses: courses, graphs: graphs, misconceptions: misconceptions}
}

type MisconceptionView struct {
	model.Misconception
	LessonTitle     string                     `json:"lesson_title"`
	PatternKeys     []string                   `json:"pattern_keys"`
	OccurrenceCount int                        `json:"occurrence_count"`
	Events          []model.MisconceptionEvent `json:"events"`
}

type PatternView struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	ActiveCount   int    `json:"active_count"`
	ResolvedCount int    `json:"resolved_count"`
}

type MisconceptionNetwork struct {
	Patterns       []PatternView       `json:"patterns"`
	Misconceptions []MisconceptionView `json:"misconceptions"`
	Edges          []struct {
		PatternKey      string `json:"pattern_key"`
		MisconceptionID uint   `json:"misconception_id"`
	} `json:"edges"`
}

func (s *MisconceptionService) ListLesson(ctx context.Context, courseID, lessonID uint) ([]MisconceptionView, error) {
	if err := s.validateCourseLesson(ctx, courseID, lessonID); err != nil {
		return nil, err
	}
	items, err := s.misconceptions.ListByLesson(ctx, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	return s.toViews(ctx, items)
}

func (s *MisconceptionService) GetNetwork(ctx context.Context, courseID uint) (*MisconceptionNetwork, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	items, err := s.misconceptions.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	views, err := s.toViews(ctx, items)
	if err != nil {
		return nil, err
	}
	network := &MisconceptionNetwork{Patterns: []PatternView{}, Misconceptions: views, Edges: []struct {
		PatternKey      string `json:"pattern_key"`
		MisconceptionID uint   `json:"misconception_id"`
	}{}}
	counts := map[string]*PatternView{}
	for _, item := range views {
		for _, key := range item.PatternKeys {
			if counts[key] == nil {
				counts[key] = &PatternView{Key: key, Name: allowedReasoningPatternKeys[key]}
			}
			if item.Status == model.MisconceptionStatusActive {
				counts[key].ActiveCount++
			} else if item.Status == model.MisconceptionStatusResolved {
				counts[key].ResolvedCount++
			}
			network.Edges = append(network.Edges, struct {
				PatternKey      string `json:"pattern_key"`
				MisconceptionID uint   `json:"misconception_id"`
			}{PatternKey: key, MisconceptionID: item.ID})
		}
	}
	for _, pattern := range counts {
		network.Patterns = append(network.Patterns, *pattern)
	}
	sortPatterns(network.Patterns)
	return network, nil
}

func (s *MisconceptionService) validateCourseLesson(ctx context.Context, courseID, lessonID uint) error {
	if courseID == 0 {
		return ErrInvalidCourseID
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCourseNotFound
		}
		return err
	}
	if _, err := s.graphs.FindLessonByCourse(ctx, courseID, lessonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLessonNotInCourse
		}
		return err
	}
	return nil
}

func (s *MisconceptionService) toViews(ctx context.Context, items []model.Misconception) ([]MisconceptionView, error) {
	result := make([]MisconceptionView, 0, len(items))
	for _, item := range items {
		links, err := s.misconceptions.ListPatternLinks(ctx, []uint{item.ID})
		if err != nil {
			return nil, err
		}
		events, err := s.misconceptions.ListEventsByMisconception(ctx, item.CourseID, item.ID)
		if err != nil {
			return nil, err
		}
		lesson, err := s.graphs.FindLessonByCourse(ctx, item.CourseID, item.LessonID)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(links))
		for _, link := range links {
			keys = append(keys, link.PatternKey)
		}
		occurrences := 0
		for _, event := range events {
			if event.EventType == model.MisconceptionEventObserved || event.EventType == model.MisconceptionEventReopened {
				occurrences++
			}
		}
		result = append(result, MisconceptionView{Misconception: item, LessonTitle: lesson.Title, PatternKeys: keys, OccurrenceCount: occurrences, Events: events})
	}
	return result, nil
}

func sortPatterns(patterns []PatternView) {
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Key < patterns[i].Key {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
}

func ValidatePatternKeys(patterns []model.ReasoningPattern) error {
	if len(patterns) > 2 {
		return fmt.Errorf("too many reasoning patterns")
	}
	seen := map[string]struct{}{}
	for _, pattern := range patterns {
		key := strings.TrimSpace(pattern.PatternKey)
		if _, ok := allowedReasoningPatternKeys[key]; !ok {
			return fmt.Errorf("unsupported reasoning pattern %q", key)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate reasoning pattern %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}
