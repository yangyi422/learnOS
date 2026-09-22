package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"
)

var (
	ErrExplorationDirectionNotFound = errors.New("exploration direction not found")
	ErrExplorationQuestionNotFound  = errors.New("exploration question not found")
	ErrExplorationInvalidStatus     = errors.New("invalid exploration status")
	ErrExplorationNoUnknown         = errors.New("no unfamiliar knowledge available")
)

const (
	ReasonCodeGraphNeighbor        = "graph_neighbor"
	ReasonCodePrerequisiteGap      = "prerequisite_gap"
	ReasonCodeDownstream           = "downstream"
	ReasonCodeRelated              = "related"
	ReasonCodeApplication          = "application"
	ReasonCodeReasoningPattern     = "reasoning_pattern"
	ReasonCodeMisconceptionPattern = "misconception_pattern"
	ReasonCodeCrossCourseBridge    = "cross_course_bridge"
	ReasonCodeUnfamiliarBoundary   = "unfamiliar_boundary"
)

type ExplorationService struct {
	courses        *repository.CourseRepository
	graphs         *repository.KnowledgeGraphRepository
	learning       *repository.LearningRepository
	cognitive      *repository.CognitiveRepository
	misconceptions *repository.MisconceptionRepository
	directions     *repository.ExplorationRepository
	provider       ai.ExplorationProvider
	unfamiliarMu   sync.Mutex
	lastUnfamiliar map[uint]uint
}

func NewExplorationService(courses *repository.CourseRepository, graphs *repository.KnowledgeGraphRepository, learning *repository.LearningRepository, cognitive *repository.CognitiveRepository, misconceptions *repository.MisconceptionRepository, directions *repository.ExplorationRepository) *ExplorationService {
	return &ExplorationService{courses: courses, graphs: graphs, learning: learning, cognitive: cognitive, misconceptions: misconceptions, directions: directions, lastUnfamiliar: map[uint]uint{}}
}

func (s *ExplorationService) SetExplorationProvider(provider ai.ExplorationProvider) {
	s.provider = provider
}

type ExplorationCourseView struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ExplorationLessonView struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	CourseID uint   `json:"course_id"`
}

type ExplorationDirectionView struct {
	ID                uint                   `json:"id"`
	CourseID          uint                   `json:"course_id"` // legacy alias for context_course_id
	ContextCourseID   uint                   `json:"context_course_id"`
	ContextLessonID   *uint                  `json:"context_lesson_id"`
	SourceDomainID    uint                   `json:"source_domain_id"`
	SourceCourseID    uint                   `json:"source_course_id"`
	SourceLessonID    uint                   `json:"source_lesson_id"`
	TargetCourseID    uint                   `json:"target_course_id"`
	TargetLessonID    uint                   `json:"target_lesson_id"`
	DirectionType     string                 `json:"direction_type"`
	Title             string                 `json:"title"`
	Summary           string                 `json:"summary"`
	WhyWorthExploring string                 `json:"why_worth_exploring"`
	Score             float64                `json:"score"`
	ReasonCode        string                 `json:"reason_code"`
	ReasonData        map[string]interface{} `json:"reason_data"`
	Status            string                 `json:"status"`
	GeneratedBy       string                 `json:"generated_by"`
	Provider          string                 `json:"provider"`
	Model             string                 `json:"model"`
	PromptVersion     string                 `json:"prompt_version"`
	SourceDomain      ExplorationCourseView  `json:"source_domain"`
	SourceCourse      ExplorationCourseView  `json:"source_course"`
	SourceLesson      ExplorationLessonView  `json:"source_lesson"`
	TargetCourse      ExplorationCourseView  `json:"target_course"`
	TargetLesson      ExplorationLessonView  `json:"target_lesson"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type ExplorationQuestionView struct {
	ID                uint                   `json:"id"`
	CourseID          uint                   `json:"course_id"`
	SourceDirectionID *uint                  `json:"source_direction_id"`
	SourceLessonID    *uint                  `json:"source_lesson_id"`
	TargetCourseID    uint                   `json:"target_course_id"`
	TargetLessonID    uint                   `json:"target_lesson_id"`
	Question          string                 `json:"question"`
	Context           string                 `json:"context"`
	WhyThisQuestion   string                 `json:"why_this_question"`
	QuestionType      string                 `json:"question_type"`
	Status            string                 `json:"status"`
	Priority          string                 `json:"priority"`
	Origin            string                 `json:"origin"`
	SourceCourse      ExplorationCourseView  `json:"source_course"`
	SourceLesson      *ExplorationLessonView `json:"source_lesson"`
	TargetCourse      ExplorationCourseView  `json:"target_course"`
	TargetLesson      ExplorationLessonView  `json:"target_lesson"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type ExplorationQuestionFilter struct {
	TargetCourseID uint
	Priority       string
}

type ExplorationRadar struct {
	Directions []ExplorationDirectionView `json:"directions"`
}

type explorationWorld struct {
	courses   map[uint]model.Course
	lessons   map[uint]model.Lesson
	relations map[uint][]model.LessonRelation
	states    map[uint]model.CognitiveState
}

type explorationCandidate struct {
	sourceLessonID *uint
	targetCourseID uint
	targetLessonID uint
	directionType  string
	reasonCode     string
	reasonData     map[string]interface{}
	baseScore      float64
	why            string
}

func (s *ExplorationService) GetRadar(ctx context.Context, courseID, lessonID uint, limit int) (*ExplorationRadar, error) {
	if courseID == 0 {
		return nil, ErrInvalidCourseID
	}
	if limit <= 0 {
		limit = 6
	}
	if limit > 20 {
		limit = 20
	}
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := world.courses[courseID]; !ok {
		return nil, ErrCourseNotFound
	}
	sourceLessonID, err := s.resolveSourceLesson(ctx, world, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	existing, err := s.directions.ListDirections(ctx, courseID, nil, 0)
	if err != nil {
		return nil, err
	}
	byIdentity := make(map[string]model.ExplorationDirection, len(existing))
	for _, item := range existing {
		byIdentity[directionIdentity(item.CourseID, item.SourceLessonID, item.TargetCourseID, item.TargetLessonID, item.DirectionType)] = item
	}
	candidates, err := s.generateCandidates(ctx, world, courseID, sourceLessonID)
	if err != nil {
		return nil, err
	}
	directions := make([]model.ExplorationDirection, 0, len(candidates))
	for _, candidate := range candidates {
		direction, include, err := s.materializeCandidate(ctx, candidate, courseID, byIdentity, world)
		if err != nil {
			return nil, err
		}
		if include {
			directions = append(directions, direction)
		}
	}
	selected := diversityPass(directions, limit)
	result := &ExplorationRadar{Directions: make([]ExplorationDirectionView, 0, len(selected))}
	for _, direction := range selected {
		view, err := s.directionView(direction, world)
		if err != nil {
			return nil, err
		}
		result.Directions = append(result.Directions, view)
	}
	return result, nil
}

func (s *ExplorationService) FindUnfamiliar(ctx context.Context, courseID uint) (*ExplorationDirectionView, error) {
	radars, err := s.GetRadar(ctx, courseID, 0, 20)
	if err != nil {
		return nil, err
	}
	s.unfamiliarMu.Lock()
	lastTarget := s.lastUnfamiliar[courseID]
	defer s.unfamiliarMu.Unlock()
	for _, direction := range radars.Directions {
		if direction.DirectionType == model.ExplorationDirectionUnknown && direction.TargetLessonID != lastTarget {
			s.lastUnfamiliar[courseID] = direction.TargetLessonID
			return &direction, nil
		}
	}
	return nil, ErrExplorationNoUnknown
}

func (s *ExplorationService) SetDirectionStatus(ctx context.Context, courseID, directionID uint, status string) (*ExplorationDirectionView, error) {
	if !validDirectionStatus(status) {
		return nil, ErrExplorationInvalidStatus
	}
	direction, err := s.directions.FindDirectionByID(ctx, directionID)
	if err != nil || direction.CourseID != courseID {
		return nil, ErrExplorationDirectionNotFound
	}
	if err := s.directions.UpdateDirectionStatus(ctx, directionID, status); err != nil {
		return nil, err
	}
	direction.Status = status
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	view, err := s.directionView(*direction, world)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ExplorationService) AddQuestion(ctx context.Context, courseID, directionID uint) (*ExplorationQuestionView, error) {
	direction, err := s.directions.FindDirectionByID(ctx, directionID)
	if err != nil || direction.CourseID != courseID {
		return nil, ErrExplorationDirectionNotFound
	}
	if question, err := s.directions.FindOpenQuestionByDirection(ctx, directionID); err == nil {
		if err := s.directions.UpdateDirectionStatus(ctx, directionID, model.ExplorationDirectionSaved); err != nil {
			return nil, err
		}
		world, worldErr := s.loadWorld(ctx)
		if worldErr != nil {
			return nil, worldErr
		}
		view, viewErr := s.questionView(*question, world)
		if viewErr != nil {
			return nil, viewErr
		}
		return &view, nil
	} else if !repository.IsNotFound(err) {
		return nil, err
	}
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	target := world.lessons[direction.TargetLessonID]
	questionType := model.ExplorationQuestionDeepen
	question := target.CoreQuestion
	why := direction.WhyWorthExploring
	switch direction.DirectionType {
	case model.ExplorationDirectionCrossDomain:
		questionType = model.ExplorationQuestionConnect
		question = fmt.Sprintf("%s 中的判断方式，和 %s 有什么联系？", sourceTitle(*direction, world), target.Title)
	case model.ExplorationDirectionUnknown:
		questionType = model.ExplorationQuestionUnfamiliar
	}
	newQuestion := &model.ExplorationQuestion{
		CourseID: courseID, SourceDirectionID: &direction.ID, SourceLessonID: direction.SourceLessonID,
		TargetCourseID: direction.TargetCourseID, TargetLessonID: direction.TargetLessonID,
		Question: question, Context: direction.Summary, WhyThisQuestion: why,
		QuestionType: questionType, Status: model.ExplorationQuestionOpen, Priority: model.ExplorationPriorityNormal, Origin: model.ExplorationQuestionFromDirection,
	}
	if err := s.directions.CreateQuestionAndSaveDirection(ctx, newQuestion, direction.ID); err != nil {
		return nil, err
	}
	view, err := s.questionView(*newQuestion, world)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ExplorationService) ListQuestions(ctx context.Context, courseID uint, status string, limit int, filters ...ExplorationQuestionFilter) ([]ExplorationQuestionView, error) {
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		if repository.IsNotFound(err) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	filter := ExplorationQuestionFilter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.Priority != "" && !validQuestionPriority(filter.Priority) {
		return nil, ErrExplorationInvalidStatus
	}
	questions, err := s.directions.ListQuestions(ctx, courseID, status, limit, filter.TargetCourseID, filter.Priority)
	if err != nil {
		return nil, err
	}
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ExplorationQuestionView, 0, len(questions))
	for _, question := range questions {
		view, err := s.questionView(question, world)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *ExplorationService) SetQuestionStatus(ctx context.Context, courseID, questionID uint, status string) (*ExplorationQuestionView, error) {
	if !validQuestionStatus(status) {
		return nil, ErrExplorationInvalidStatus
	}
	question, err := s.directions.FindQuestionByID(ctx, questionID)
	if err != nil || question.CourseID != courseID {
		return nil, ErrExplorationQuestionNotFound
	}
	if err := s.directions.UpdateQuestionStatus(ctx, questionID, status); err != nil {
		return nil, err
	}
	question.Status = status
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	view, err := s.questionView(*question, world)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ExplorationService) SetQuestionPriority(ctx context.Context, courseID, questionID uint, priority string) (*ExplorationQuestionView, error) {
	if !validQuestionPriority(priority) {
		return nil, ErrExplorationInvalidStatus
	}
	question, err := s.directions.FindQuestionByID(ctx, questionID)
	if err != nil || question.CourseID != courseID {
		return nil, ErrExplorationQuestionNotFound
	}
	if err := s.directions.UpdateQuestionPriority(ctx, questionID, priority); err != nil {
		return nil, err
	}
	question.Priority = priority
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	view, err := s.questionView(*question, world)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ExplorationService) UndoQuestion(ctx context.Context, courseID, directionID uint) (*ExplorationDirectionView, error) {
	direction, err := s.directions.FindDirectionByID(ctx, directionID)
	if err != nil || direction.CourseID != courseID {
		return nil, ErrExplorationDirectionNotFound
	}
	if err := s.directions.UndoQuestionSave(ctx, directionID); err != nil {
		return nil, err
	}
	direction.Status = model.ExplorationDirectionActive
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	view, err := s.directionView(*direction, world)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *ExplorationService) ListHistory(ctx context.Context, courseID uint, limit int) ([]ExplorationDirectionView, error) {
	if limit <= 0 {
		limit = 20
	}
	directions, err := s.directions.ListDirectionHistory(ctx, courseID, limit)
	if err != nil {
		return nil, err
	}
	world, err := s.loadWorld(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ExplorationDirectionView, 0, len(directions))
	for _, direction := range directions {
		view, err := s.directionView(direction, world)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *ExplorationService) loadWorld(ctx context.Context) (*explorationWorld, error) {
	courses, err := s.courses.List(ctx)
	if err != nil {
		return nil, err
	}
	world := &explorationWorld{courses: map[uint]model.Course{}, lessons: map[uint]model.Lesson{}, relations: map[uint][]model.LessonRelation{}, states: map[uint]model.CognitiveState{}}
	for _, course := range courses {
		world.courses[course.ID] = course
		lessons, err := s.graphs.ListLessonsByCourse(ctx, course.ID)
		if err != nil {
			return nil, err
		}
		for _, lesson := range lessons {
			world.lessons[lesson.ID] = lesson
		}
		relations, err := s.graphs.ListRelationsByCourse(ctx, course.ID)
		if err != nil {
			return nil, err
		}
		world.relations[course.ID] = relations
		states, err := s.cognitive.ListByCourse(ctx, course.ID)
		if err != nil {
			return nil, err
		}
		for _, state := range states {
			world.states[state.LessonID] = state
		}
	}
	return world, nil
}

func (s *ExplorationService) resolveSourceLesson(ctx context.Context, world *explorationWorld, courseID, requestedID uint) (*uint, error) {
	if requestedID != 0 {
		lesson, ok := world.lessons[requestedID]
		if !ok || lesson.CourseID != courseID {
			return nil, ErrLessonNotInCourse
		}
		return &requestedID, nil
	}
	turns, err := s.learning.ListLearningTurns(ctx, courseID, 1)
	if err != nil {
		return nil, err
	}
	if len(turns) > 0 {
		if lesson, ok := world.lessons[turns[0].LessonID]; ok && lesson.CourseID == courseID {
			id := lesson.ID
			return &id, nil
		}
	}
	course := world.courses[courseID]
	if course.CurrentLessonID != nil {
		if lesson, ok := world.lessons[*course.CurrentLessonID]; ok && lesson.CourseID == courseID {
			id := lesson.ID
			return &id, nil
		}
	}
	return nil, nil
}

func (s *ExplorationService) generateCandidates(ctx context.Context, world *explorationWorld, courseID uint, sourceLessonID *uint) ([]explorationCandidate, error) {
	result := make([]explorationCandidate, 0)
	directTargets := map[uint]struct{}{}
	if sourceLessonID != nil {
		for _, relation := range world.relations[courseID] {
			if relation.FromLessonID != *sourceLessonID && relation.ToLessonID != *sourceLessonID {
				continue
			}
			targetID := relation.FromLessonID
			if targetID == *sourceLessonID {
				targetID = relation.ToLessonID
			}
			if targetID == *sourceLessonID {
				continue
			}
			directTargets[targetID] = struct{}{}
			reasonCode, why := relationReason(relation, *sourceLessonID)
			result = append(result, explorationCandidate{
				sourceLessonID: sourceLessonID, targetCourseID: courseID, targetLessonID: targetID,
				directionType: model.ExplorationDirectionAdjacent, reasonCode: reasonCode,
				reasonData: map[string]interface{}{"relation_type": relation.RelationType, "relation_id": relation.ID},
				baseScore:  30, why: why,
			})
		}
	}

	patterns, err := s.patternCandidates(ctx, world, courseID, sourceLessonID)
	if err != nil {
		return nil, err
	}
	result = append(result, patterns...)
	result = append(result, curatedCrossDomainCandidates(world, courseID, sourceLessonID)...)

	lessonIDs := make([]uint, 0, len(world.lessons))
	for lessonID := range world.lessons {
		lessonIDs = append(lessonIDs, lessonID)
	}
	sort.Slice(lessonIDs, func(i, j int) bool { return lessonIDs[i] < lessonIDs[j] })
	for _, lessonID := range lessonIDs {
		lesson := world.lessons[lessonID]
		if lesson.CourseID == courseID || (sourceLessonID != nil && lessonID == *sourceLessonID) {
			continue
		}
		if cognitiveLevel(world.states[lessonID]) != model.CognitiveLevelUnseen {
			continue
		}
		if _, direct := directTargets[lessonID]; direct {
			continue
		}
		result = append(result, explorationCandidate{
			sourceLessonID: sourceLessonID, targetCourseID: lesson.CourseID, targetLessonID: lessonID,
			directionType: model.ExplorationDirectionUnknown, reasonCode: ReasonCodeUnfamiliarBoundary,
			reasonData: map[string]interface{}{"source_course_id": courseID}, baseScore: 0,
			why: "它与当前课程没有直接结构关系，可以帮助你把认知边界扩展到另一个领域。",
		})
	}
	return dedupeCandidates(result), nil
}

// curatedCrossDomainCandidates is the finite, deterministic fallback for
// installations that do not have CrossCourseLessonRelation or user-created
// MisconceptionPatternLink rows yet. It describes knowledge-world bridges only;
// it never creates or infers a user's cognitive state or misconception.
func curatedCrossDomainCandidates(world *explorationWorld, courseID uint, sourceLessonID *uint) []explorationCandidate {
	if sourceLessonID == nil {
		return nil
	}
	bridges := []struct {
		sourceTitle string
		targetTitle string
		why         string
	}{
		{
			sourceTitle: "口渴是否是可靠的饮水依据",
			targetTitle: "如何判断一条证据有多可靠",
			why:         "你在饮水判断中需要同时考虑多个变量，这个知识点可以帮助你建立更通用的多因素证据判断框架。",
		},
		{
			sourceTitle: "日常饮水需求应该如何判断",
			targetTitle: "单因素解释的陷阱",
			why:         "你正在判断一个没有固定答案的饮水问题，这个知识点可以帮助你识别把多因素问题归因于单一原因的风险。",
		},
	}
	source, ok := world.lessons[*sourceLessonID]
	if !ok {
		return nil
	}
	result := make([]explorationCandidate, 0, len(bridges))
	for _, bridge := range bridges {
		if source.Title != bridge.sourceTitle {
			continue
		}
		for targetID, target := range world.lessons {
			if target.Title != bridge.targetTitle || target.CourseID == courseID || cognitiveLevel(world.states[targetID]) != model.CognitiveLevelUnseen {
				continue
			}
			result = append(result, explorationCandidate{
				sourceLessonID: sourceLessonID, targetCourseID: target.CourseID, targetLessonID: targetID,
				directionType: model.ExplorationDirectionCrossDomain, reasonCode: ReasonCodeCrossCourseBridge,
				reasonData: map[string]interface{}{"mapping_source": "curated_cross_course_taxonomy", "source_title": source.Title, "target_title": target.Title},
				baseScore:  45, why: bridge.why,
			})
		}
	}
	return result
}

func (s *ExplorationService) patternCandidates(ctx context.Context, world *explorationWorld, courseID uint, sourceLessonID *uint) ([]explorationCandidate, error) {
	var misconceptions []model.Misconception
	var err error
	if sourceLessonID != nil {
		misconceptions, err = s.misconceptions.ListByLesson(ctx, courseID, *sourceLessonID)
	} else {
		misconceptions, err = s.misconceptions.ListByCourse(ctx, courseID)
	}
	if err != nil {
		return nil, err
	}
	activeIDs := make([]uint, 0)
	for _, item := range misconceptions {
		if item.Status == model.MisconceptionStatusActive {
			activeIDs = append(activeIDs, item.ID)
		}
	}
	links, err := s.misconceptions.ListPatternLinks(ctx, activeIDs)
	if err != nil {
		return nil, err
	}
	targets := map[string][]string{
		"single_factor_reasoning": {"单因素解释的陷阱"},
		"correlation_causation":   {"相关不等于因果"},
		"unsupported_assumption":  {"如何判断一条证据有多可靠"},
		"overgeneralization":      {"如何判断一条证据有多可靠"},
		"binary_thinking":         {"单因素解释的陷阱"},
	}
	result := make([]explorationCandidate, 0)
	for _, link := range links {
		for _, title := range targets[link.PatternKey] {
			for targetID, target := range world.lessons {
				if target.Title != title || target.CourseID == courseID {
					continue
				}
				result = append(result, explorationCandidate{
					sourceLessonID: sourceLessonID, targetCourseID: target.CourseID, targetLessonID: targetID,
					directionType: model.ExplorationDirectionCrossDomain, reasonCode: ReasonCodeReasoningPattern,
					reasonData: map[string]interface{}{"pattern_key": link.PatternKey, "pattern_explanation": link.Explanation, "misconception_id": link.MisconceptionID},
					baseScore:  50, why: "你最近的学习中遇到了一个判断问题，这个跨领域知识可以帮助你建立更通用的分析框架。",
				})
			}
		}
	}
	return result, nil
}

func (s *ExplorationService) materializeCandidate(ctx context.Context, candidate explorationCandidate, courseID uint, existing map[string]model.ExplorationDirection, world *explorationWorld) (model.ExplorationDirection, bool, error) {
	key := directionIdentity(courseID, candidate.sourceLessonID, candidate.targetCourseID, candidate.targetLessonID, candidate.directionType)
	if old, ok := existing[key]; ok {
		cooldown := time.Duration(0)
		if old.Status == model.ExplorationDirectionDismissed {
			cooldown = 7 * 24 * time.Hour
		} else if old.Status == model.ExplorationDirectionOpened {
			cooldown = 3 * 24 * time.Hour
		}
		if old.Status == model.ExplorationDirectionActive || old.Status == model.ExplorationDirectionSaved || time.Since(old.UpdatedAt) < cooldown {
			return old, true, nil
		}
	}
	target := world.lessons[candidate.targetLessonID]
	sourceState := model.CognitiveState{}
	if candidate.sourceLessonID != nil {
		sourceState = world.states[*candidate.sourceLessonID]
	}
	targetState := world.states[candidate.targetLessonID]
	score, components := calculateExplorationScore(candidate, sourceState, targetState, candidate.targetCourseID == courseID)
	reasonData := candidate.reasonData
	if reasonData == nil {
		reasonData = map[string]interface{}{}
	}
	reasonData["score_components"] = components
	reasonData["target_current_level"] = cognitiveLevel(targetState)
	reasonData["target_status"] = cognitiveStatus(targetState)
	direction := model.ExplorationDirection{
		CourseID: courseID, SourceLessonID: candidate.sourceLessonID, TargetCourseID: candidate.targetCourseID, TargetLessonID: candidate.targetLessonID,
		DirectionType: candidate.directionType, Title: target.Title, Summary: fmt.Sprintf("%s · %s", world.courses[candidate.targetCourseID].Name, target.Title),
		WhyWorthExploring: candidate.why, Score: score, ReasonCode: candidate.reasonCode, ReasonDataJSON: marshalExplorationJSON(reasonData), Status: model.ExplorationDirectionActive, GeneratedBy: model.ExplorationGeneratedByRule,
	}
	// Unfamiliar candidates are deliberately cheap and deterministic: the radar can
	// contain many of them, so only graph/pattern directions spend an AI call for
	// editorial copy. Their rule text remains the fallback for every direction.
	if s.provider != nil && candidate.directionType != model.ExplorationDirectionUnknown {
		request := ai.ExplorationRequest{
			SourceCourseName: world.courses[courseID].Name,
			TargetCourseName: world.courses[candidate.targetCourseID].Name,
			TargetLesson:     target.Title,
			DirectionType:    candidate.directionType,
			ReasonCode:       candidate.reasonCode,
			ReasonData:       reasonData,
			Why:              candidate.why,
		}
		if candidate.sourceLessonID != nil {
			request.SourceLesson = world.lessons[*candidate.sourceLessonID].Title
		}
		result, meta, aiErr := s.provider.GenerateExploration(ctx, request)
		if aiErr == nil {
			direction.Summary = result.Summary
			direction.WhyWorthExploring = result.WhyWorthExploring
			direction.GeneratedBy = model.ExplorationGeneratedByAI
			direction.Provider = meta.Provider
			direction.Model = meta.Model
			direction.PromptVersion = meta.PromptVersion
			_ = s.learning.SaveAIEvaluationRun(ctx, &model.AIEvaluationRun{
				CourseID: candidate.targetCourseID, LessonID: candidate.targetLessonID,
				Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion,
				RunType: "exploration_direction", Status: model.AIEvaluationRunStatusSuccess,
				AttemptCount: positiveOrDefault(meta.AttemptCount, 1), LatencyMS: meta.LatencyMS,
				InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens, RawResponse: meta.RawResponse,
			})
		} else {
			_ = s.learning.SaveAIEvaluationRun(ctx, &model.AIEvaluationRun{
				CourseID: candidate.targetCourseID, LessonID: candidate.targetLessonID,
				Provider: meta.Provider, Model: meta.Model, PromptVersion: meta.PromptVersion,
				RunType: "exploration_direction", Status: model.AIEvaluationRunStatusFailed,
				AttemptCount: positiveOrDefault(meta.AttemptCount, 1), LatencyMS: meta.LatencyMS,
				InputTokens: meta.InputTokens, OutputTokens: meta.OutputTokens, RawResponse: meta.RawResponse,
				ErrorMessage: ai.ErrorCode(aiErr),
			})
		}
	}
	if old, ok := existing[key]; ok {
		direction.ID = old.ID
		direction.CreatedAt = old.CreatedAt
		if err := s.directions.UpdateDirection(ctx, &direction); err != nil {
			return model.ExplorationDirection{}, false, err
		}
		return direction, true, nil
	}
	if err := s.directions.CreateDirection(ctx, &direction); err != nil {
		return model.ExplorationDirection{}, false, err
	}
	return direction, true, nil
}

func (s *ExplorationService) directionView(direction model.ExplorationDirection, world *explorationWorld) (ExplorationDirectionView, error) {
	contextCourse, ok := world.courses[direction.CourseID]
	if !ok {
		return ExplorationDirectionView{}, fmt.Errorf("exploration direction %d references missing context course", direction.ID)
	}
	if direction.SourceLessonID != nil {
		contextLesson, lessonOK := world.lessons[*direction.SourceLessonID]
		if !lessonOK || contextLesson.CourseID != contextCourse.ID {
			return ExplorationDirectionView{}, fmt.Errorf("exploration direction %d has inconsistent context lesson ownership", direction.ID)
		}
	}
	target, ok := world.lessons[direction.TargetLessonID]
	if !ok {
		return ExplorationDirectionView{}, fmt.Errorf("exploration direction %d references missing target lesson", direction.ID)
	}
	targetCourse, ok := world.courses[direction.TargetCourseID]
	if !ok {
		return ExplorationDirectionView{}, fmt.Errorf("exploration direction %d references missing target course", direction.ID)
	}
	if target.CourseID != targetCourse.ID {
		return ExplorationDirectionView{}, fmt.Errorf("exploration direction %d has inconsistent source ownership", direction.ID)
	}
	reasonData := map[string]interface{}{}
	if strings.TrimSpace(direction.ReasonDataJSON) != "" {
		if err := json.Unmarshal([]byte(direction.ReasonDataJSON), &reasonData); err != nil {
			return ExplorationDirectionView{}, fmt.Errorf("decode exploration reason data: %w", err)
		}
	}
	return ExplorationDirectionView{
		ID: direction.ID, CourseID: direction.CourseID, ContextCourseID: direction.CourseID, ContextLessonID: direction.SourceLessonID,
		SourceDomainID: targetCourse.ID, SourceCourseID: targetCourse.ID, SourceLessonID: target.ID,
		TargetCourseID: direction.TargetCourseID, TargetLessonID: direction.TargetLessonID,
		DirectionType: direction.DirectionType, Title: direction.Title, Summary: direction.Summary, WhyWorthExploring: direction.WhyWorthExploring, Score: direction.Score, ReasonCode: direction.ReasonCode, ReasonData: reasonData,
		Status: direction.Status, GeneratedBy: direction.GeneratedBy, Provider: direction.Provider, Model: direction.Model, PromptVersion: direction.PromptVersion,
		SourceDomain: ExplorationCourseView{ID: targetCourse.ID, Name: targetCourse.Name}, SourceCourse: ExplorationCourseView{ID: targetCourse.ID, Name: targetCourse.Name}, SourceLesson: ExplorationLessonView{ID: target.ID, Title: target.Title, CourseID: target.CourseID},
		TargetCourse: ExplorationCourseView{ID: targetCourse.ID, Name: targetCourse.Name}, TargetLesson: ExplorationLessonView{ID: target.ID, Title: target.Title, CourseID: target.CourseID}, CreatedAt: direction.CreatedAt, UpdatedAt: direction.UpdatedAt,
	}, nil
}

func (s *ExplorationService) questionView(question model.ExplorationQuestion, world *explorationWorld) (ExplorationQuestionView, error) {
	target, ok := world.lessons[question.TargetLessonID]
	if !ok {
		return ExplorationQuestionView{}, fmt.Errorf("exploration question %d references missing target lesson", question.ID)
	}
	targetCourse, ok := world.courses[question.TargetCourseID]
	if !ok {
		return ExplorationQuestionView{}, fmt.Errorf("exploration question %d references missing target course", question.ID)
	}
	sourceCourse, ok := world.courses[question.CourseID]
	if !ok {
		return ExplorationQuestionView{}, fmt.Errorf("exploration question %d references missing source course", question.ID)
	}
	var sourceLesson *ExplorationLessonView
	if question.SourceLessonID != nil {
		if lesson, lessonOK := world.lessons[*question.SourceLessonID]; lessonOK {
			sourceLesson = &ExplorationLessonView{ID: lesson.ID, Title: lesson.Title, CourseID: lesson.CourseID}
		}
	}
	return ExplorationQuestionView{
		ID: question.ID, CourseID: question.CourseID, SourceDirectionID: question.SourceDirectionID, SourceLessonID: question.SourceLessonID,
		TargetCourseID: question.TargetCourseID, TargetLessonID: question.TargetLessonID, Question: question.Question, Context: question.Context, WhyThisQuestion: question.WhyThisQuestion,
		QuestionType: question.QuestionType, Status: question.Status, Priority: normalizedQuestionPriority(question.Priority), Origin: question.Origin,
		SourceCourse: ExplorationCourseView{ID: sourceCourse.ID, Name: sourceCourse.Name}, SourceLesson: sourceLesson,
		TargetCourse: ExplorationCourseView{ID: targetCourse.ID, Name: targetCourse.Name}, TargetLesson: ExplorationLessonView{ID: target.ID, Title: target.Title, CourseID: target.CourseID}, CreatedAt: question.CreatedAt, UpdatedAt: question.UpdatedAt,
	}, nil
}

func calculateExplorationScore(candidate explorationCandidate, sourceState, targetState model.CognitiveState, sameCourse bool) (float64, map[string]float64) {
	components := map[string]float64{}
	components["base"] = candidate.baseScore
	if candidate.directionType == model.ExplorationDirectionCrossDomain && candidate.reasonCode == ReasonCodeReasoningPattern {
		components["reasoning_pattern"] = 25
		components["misconception_pattern"] = 25
	}
	if cognitiveLevel(targetState) == model.CognitiveLevelUnseen {
		components["unseen_target"] = 20
	} else if cognitiveStatus(targetState) == model.CognitiveStatusDeveloping {
		components["developing_target"] = 10
	}
	if sameCourse {
		components["same_course"] = 5
	} else {
		components["different_course"] = 15
	}
	if cognitiveStatus(targetState) == model.CognitiveStatusStable {
		components["stable_penalty"] = -30
	}
	if cognitiveLevel(targetState) == model.CognitiveLevelTransfer {
		components["transfer_penalty"] = -50
	}
	if cognitiveLevel(sourceState) == model.CognitiveLevelTransfer && cognitiveStatus(sourceState) == model.CognitiveStatusStable {
		if candidate.directionType == model.ExplorationDirectionAdjacent {
			components["source_transfer_adjacent_penalty"] = -15
		} else {
			components["source_transfer_boundary_bonus"] = 10
		}
	}
	score := 0.0
	for _, value := range components {
		score += value
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, components
}

func diversityPass(directions []model.ExplorationDirection, limit int) []model.ExplorationDirection {
	sort.SliceStable(directions, func(i, j int) bool {
		if directions[i].Score != directions[j].Score {
			return directions[i].Score > directions[j].Score
		}
		return directions[i].ID < directions[j].ID
	})
	groups := map[string][]model.ExplorationDirection{}
	for _, direction := range directions {
		groups[direction.DirectionType] = append(groups[direction.DirectionType], direction)
	}
	base, remainder := limit/3, limit%3
	quotas := map[string]int{model.ExplorationDirectionAdjacent: base, model.ExplorationDirectionCrossDomain: base, model.ExplorationDirectionUnknown: base}
	if remainder > 0 {
		quotas[model.ExplorationDirectionAdjacent]++
	}
	if remainder > 1 {
		quotas[model.ExplorationDirectionCrossDomain]++
	}
	result := make([]model.ExplorationDirection, 0, limit)
	selected := map[uint]struct{}{}
	for _, directionType := range []string{model.ExplorationDirectionAdjacent, model.ExplorationDirectionCrossDomain, model.ExplorationDirectionUnknown} {
		for index := 0; index < quotas[directionType] && index < len(groups[directionType]); index++ {
			result = append(result, groups[directionType][index])
			selected[groups[directionType][index].ID] = struct{}{}
		}
	}
	for _, direction := range directions {
		if len(result) >= limit {
			break
		}
		if _, ok := selected[direction.ID]; ok {
			continue
		}
		result = append(result, direction)
	}
	return result
}

func relationReason(relation model.LessonRelation, sourceLessonID uint) (string, string) {
	if relation.RelationType == model.LessonRelationPrerequisite && relation.ToLessonID == sourceLessonID {
		return ReasonCodePrerequisiteGap, "它是当前知识的前置基础，适合用来补齐理解链条。"
	}
	if relation.RelationType == model.LessonRelationPrerequisite {
		return ReasonCodeDownstream, "它是当前知识的后续节点，可以把已有理解延伸到更具体的场景。"
	}
	switch relation.RelationType {
	case model.LessonRelationRelated:
		return ReasonCodeRelated, "它与当前知识存在相关关系，可以帮助你从另一个角度比较同一问题。"
	case model.LessonRelationApplication:
		return ReasonCodeApplication, "它把当前知识连接到应用场景，适合检验理解的边界。"
	case model.LessonRelationExtends:
		return ReasonCodeGraphNeighbor, "它是当前知识的深化内容，可以逐步扩展已有理解。"
	default:
		return ReasonCodeGraphNeighbor, "它与当前知识在课程结构中直接相邻，值得继续了解。"
	}
}

func dedupeCandidates(candidates []explorationCandidate) []explorationCandidate {
	seen := map[string]int{}
	result := make([]explorationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := fmt.Sprintf("%d:%d:%s", candidate.targetCourseID, candidate.targetLessonID, candidate.directionType)
		if index, ok := seen[key]; ok {
			if candidate.baseScore > result[index].baseScore {
				result[index] = candidate
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, candidate)
	}
	return result
}

func directionIdentity(courseID uint, sourceLessonID *uint, targetCourseID, targetLessonID uint, directionType string) string {
	source := "none"
	if sourceLessonID != nil {
		source = fmt.Sprintf("%d", *sourceLessonID)
	}
	return fmt.Sprintf("%d:%s:%d:%d:%s", courseID, source, targetCourseID, targetLessonID, directionType)
}

func validDirectionStatus(status string) bool {
	return status == model.ExplorationDirectionActive || status == model.ExplorationDirectionSaved || status == model.ExplorationDirectionDismissed || status == model.ExplorationDirectionOpened
}

func validQuestionStatus(status string) bool {
	switch status {
	case model.ExplorationQuestionOpen, model.ExplorationQuestionExploring, model.ExplorationQuestionLater, model.ExplorationQuestionResolved, model.ExplorationQuestionArchived:
		return true
	default:
		return false
	}
}

func validQuestionPriority(priority string) bool {
	return priority == model.ExplorationPriorityLow || priority == model.ExplorationPriorityNormal || priority == model.ExplorationPriorityHigh
}

func normalizedQuestionPriority(priority string) string {
	if validQuestionPriority(priority) {
		return priority
	}
	return model.ExplorationPriorityNormal
}

func cognitiveLevel(state model.CognitiveState) string {
	if state.LessonID == 0 || state.CurrentLevel == "" {
		return model.CognitiveLevelUnseen
	}
	return state.CurrentLevel
}

func cognitiveStatus(state model.CognitiveState) string {
	if state.LessonID == 0 || state.Status == "" {
		return model.CognitiveStatusUnknown
	}
	return state.Status
}

func sourceTitle(direction model.ExplorationDirection, world *explorationWorld) string {
	if direction.SourceLessonID == nil {
		return world.courses[direction.CourseID].Name
	}
	if lesson, ok := world.lessons[*direction.SourceLessonID]; ok {
		return lesson.Title
	}
	return world.courses[direction.CourseID].Name
}

func marshalExplorationJSON(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}
