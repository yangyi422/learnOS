package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

type NextLessonService struct {
	courses    *repository.CourseRepository
	graphs     *repository.KnowledgeGraphRepository
	curriculum *repository.CurriculumRepository
	cognitive  *repository.CognitiveRepository
	learning   *repository.LearningRepository
}

func NewNextLessonService(
	courses *repository.CourseRepository,
	graphs *repository.KnowledgeGraphRepository,
	curriculum *repository.CurriculumRepository,
	cognitive *repository.CognitiveRepository,
	learning *repository.LearningRepository,
) *NextLessonService {
	return &NextLessonService{courses: courses, graphs: graphs, curriculum: curriculum, cognitive: cognitive, learning: learning}
}

type NextLessonView struct {
	Recommended     *NextLessonLesson              `json:"recommended"`
	Alternatives    []NextLessonLesson             `json:"alternatives"`
	Reason          string                         `json:"reason"`
	NeedsExpansion  bool                           `json:"needs_expansion"`
	NeedsGeneration bool                           `json:"needs_generation"`
	RecommendedUnit *model.CurriculumBlueprintUnit `json:"recommended_unit"`
}

type NextLessonLesson struct {
	ID                     uint               `json:"id"`
	UnitID                 uint               `json:"unit_id"`
	UnitTitle              string             `json:"unit_title"`
	Title                  string             `json:"title"`
	CoreQuestion           string             `json:"core_question"`
	Status                 model.LessonStatus `json:"status"`
	IsCore                 bool               `json:"is_core"`
	ContentRole            model.ContentRole  `json:"content_role"`
	DepthLevel             int                `json:"depth_level"`
	Prerequisites          []string           `json:"prerequisites"`
	PrerequisitesSatisfied bool               `json:"prerequisites_satisfied"`
	Unseen                 bool               `json:"unseen"`
}

type nextLessonCandidate struct {
	lesson        model.Lesson
	unit          model.CourseUnit
	unitIndex     int
	sameUnit      bool
	satisfied     bool
	core          bool
	unseen        bool
	prerequisites []string
}

func (s *NextLessonService) Recommend(ctx context.Context, courseID uint) (*NextLessonView, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	units, err := s.graphs.ListUnitsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessons, err := s.graphs.ListLessonsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	relations, err := s.graphs.ListRelationsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	states, err := s.cognitive.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	stateByLesson := make(map[uint]model.CognitiveState, len(states))
	for _, state := range states {
		stateByLesson[state.LessonID] = state
	}
	answered, err := s.learning.ListAnsweredLessonIDs(ctx, courseID)
	if err != nil {
		return nil, err
	}

	unitIndex := make(map[uint]int, len(units))
	unitByID := make(map[uint]model.CourseUnit, len(units))
	for index, unit := range units {
		unitIndex[unit.ID] = index
		unitByID[unit.ID] = unit
	}
	prerequisites := make(map[uint][]uint)
	for _, relation := range relations {
		if relation.RelationType == model.LessonRelationPrerequisite {
			prerequisites[relation.ToLessonID] = append(prerequisites[relation.ToLessonID], relation.FromLessonID)
		}
	}
	lessonByID := make(map[uint]model.Lesson, len(lessons))
	for _, lesson := range lessons {
		lessonByID[lesson.ID] = lesson
	}

	currentUnitIndex := -1
	if course.CurrentUnitID != nil {
		if index, ok := unitIndex[*course.CurrentUnitID]; ok {
			currentUnitIndex = index
		}
	}
	currentLessonID := uint(0)
	if course.CurrentLessonID != nil {
		currentLessonID = *course.CurrentLessonID
	}
	currentLessonSortOrder := -1
	if currentLessonID != 0 {
		if currentLesson, ok := lessonByID[currentLessonID]; ok {
			currentLessonSortOrder = currentLesson.SortOrder
		}
	}
	candidates := make([]nextLessonCandidate, 0, len(lessons))
	for _, lesson := range lessons {
		if lesson.ID == currentLessonID {
			continue
		}
		if currentLessonSortOrder >= 0 && currentUnitIndex >= 0 && unitIndex[lesson.UnitID] == currentUnitIndex && lesson.SortOrder <= currentLessonSortOrder {
			continue
		}
		unit, ok := unitByID[lesson.UnitID]
		if !ok {
			continue
		}
		prereqIDs := prerequisites[lesson.ID]
		satisfied := true
		prereqTitles := make([]string, 0, len(prereqIDs))
		for _, prerequisiteID := range prereqIDs {
			if prerequisite, exists := lessonByID[prerequisiteID]; exists {
				prereqTitles = append(prereqTitles, prerequisite.Title)
			}
			state, exists := stateByLesson[prerequisiteID]
			if !exists || cognitiveRank(state.CurrentLevel) < cognitiveRank(model.CognitiveLevelUnderstand) || state.Status == model.CognitiveStatusNeedsReview {
				satisfied = false
			}
		}
		unseen := true
		if _, exists := answered[lesson.ID]; exists {
			unseen = false
		}
		if _, exists := stateByLesson[lesson.ID]; exists {
			unseen = false
		}
		candidates = append(candidates, nextLessonCandidate{
			lesson: lesson, unit: unit, unitIndex: unitIndex[unit.ID],
			sameUnit:  currentUnitIndex >= 0 && unitIndex[unit.ID] == currentUnitIndex,
			satisfied: satisfied, core: lesson.IsCore && lesson.ContentRole != model.ContentRoleExtension,
			unseen: unseen, prerequisites: prereqTitles,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.sameUnit != right.sameUnit {
			return left.sameUnit
		}
		if left.satisfied != right.satisfied {
			return left.satisfied
		}
		leftDistance, rightDistance := unitDistance(left.unitIndex, currentUnitIndex), unitDistance(right.unitIndex, currentUnitIndex)
		if leftDistance != rightDistance {
			return leftDistance < rightDistance
		}
		if left.core != right.core {
			return left.core
		}
		if left.lesson.DepthLevel != right.lesson.DepthLevel {
			return left.lesson.DepthLevel < right.lesson.DepthLevel
		}
		if left.unseen != right.unseen {
			return left.unseen
		}
		if left.lesson.SortOrder != right.lesson.SortOrder {
			return left.lesson.SortOrder < right.lesson.SortOrder
		}
		return left.lesson.ID < right.lesson.ID
	})

	result := &NextLessonView{Alternatives: []NextLessonLesson{}, Reason: "当前课程暂无可继续的正式 Lesson。"}
	if len(candidates) > 0 {
		result.Recommended = nextLessonView(candidates[0])
		for _, candidate := range candidates[1:] {
			if len(result.Alternatives) == 3 {
				break
			}
			result.Alternatives = append(result.Alternatives, *nextLessonView(candidate))
		}
		switch {
		case candidates[0].sameUnit && candidates[0].satisfied:
			result.Reason = "这是当前单元中下一个前置知识已满足的节点。"
		case candidates[0].sameUnit:
			result.Reason = fmt.Sprintf("这是当前单元中下一个核心节点；它存在尚未完成的前置知识：%s。", joinPrerequisites(candidates[0].prerequisites))
		case candidates[0].satisfied:
			result.Reason = "当前单元没有更多节点，这是相邻核心单元中前置知识已满足的节点。"
		default:
			result.Reason = "这是当前课程中距离最近的可选节点；它存在尚未完成的前置知识。"
		}
		return result, nil
	}

	if s.curriculum != nil {
		blueprint, blueprintErr := s.curriculum.FindActiveBlueprint(ctx, courseID)
		if blueprintErr == nil {
			blueprintUnits, unitsErr := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
			if unitsErr != nil {
				return nil, unitsErr
			}
			currentSortOrder := -1
			if course.CurrentUnitID != nil {
				if unit, ok := unitByID[*course.CurrentUnitID]; ok {
					currentSortOrder = unit.SortOrder
				}
			}
			for index := range blueprintUnits {
				unit := blueprintUnits[index]
				if currentSortOrder >= 0 && unit.SortOrder <= currentSortOrder {
					continue
				}
				if unit.ExpansionStatus != model.CurriculumUnitExpanded {
					unitCopy := unit
					result.NeedsExpansion = true
					result.RecommendedUnit = &unitCopy
					result.Reason = "当前区域的正式节点已经学习到这里，可以展开下一个知识区域。"
					return result, nil
				}
				result.NeedsGeneration = true
				unitCopy := unit
				result.RecommendedUnit = &unitCopy
				result.Reason = "当前区域的正式节点已经学习到这里，可以从下一个知识区域生成学习节点。"
				return result, nil
			}
		} else if !errors.Is(blueprintErr, gorm.ErrRecordNotFound) {
			return nil, blueprintErr
		}
	}
	return result, nil
}

func nextLessonView(candidate nextLessonCandidate) *NextLessonLesson {
	return &NextLessonLesson{
		ID: candidate.lesson.ID, UnitID: candidate.lesson.UnitID, UnitTitle: candidate.unit.Title,
		Title: candidate.lesson.Title, CoreQuestion: candidate.lesson.CoreQuestion, Status: candidate.lesson.Status,
		IsCore: candidate.lesson.IsCore, ContentRole: candidate.lesson.ContentRole, DepthLevel: candidate.lesson.DepthLevel,
		Prerequisites: candidate.prerequisites, PrerequisitesSatisfied: candidate.satisfied, Unseen: candidate.unseen,
	}
}

func unitDistance(left, right int) int {
	if right < 0 {
		return left
	}
	if left > right {
		return left - right
	}
	return right - left
}

func joinPrerequisites(values []string) string {
	if len(values) == 0 {
		return "相关基础节点"
	}
	result := values[0]
	for _, value := range values[1:] {
		result += "、" + value
	}
	return result
}
