package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrCurriculumBlueprintNotFound         = errors.New("curriculum blueprint not found")
	ErrCurriculumDraftNotFound             = errors.New("curriculum draft not found")
	ErrCurriculumDraftInvalid              = errors.New("invalid curriculum draft")
	ErrCurriculumDraftApplied              = errors.New("curriculum draft already processed")
	ErrCurriculumInvalidScope              = errors.New("invalid curriculum draft scope")
	ErrCurriculumKeyConflict               = errors.New("curriculum key conflict")
	ErrCurriculumUnitNotFound              = errors.New("curriculum blueprint unit not found")
	ErrCurriculumUnitExpansion             = errors.New("curriculum unit expansion failed")
	ErrCurriculumUnitExpansionInProgress   = errors.New("curriculum unit expansion in progress")
	ErrCurriculumUnitAlreadyExpanded       = errors.New("curriculum unit already expanded")
	ErrCurriculumDraftAlreadyPending       = errors.New("curriculum draft already pending")
	ErrCurriculumDraftGenerationInProgress = errors.New("curriculum draft generation in progress")
)

type CurriculumService struct {
	courses        *repository.CourseRepository
	curriculum     *repository.CurriculumRepository
	provider       ai.CurriculumDraftProvider
	domainProvider ai.BlueprintUnitExpansionProvider
}

func NewCurriculumService(courses *repository.CourseRepository, curriculum *repository.CurriculumRepository) *CurriculumService {
	return &CurriculumService{courses: courses, curriculum: curriculum}
}

func (s *CurriculumService) SetCurriculumDraftProvider(provider ai.CurriculumDraftProvider) {
	s.provider = provider
}

func (s *CurriculumService) SetDomainInitializationProvider(provider ai.BlueprintUnitExpansionProvider) {
	s.domainProvider = provider
}

type CurriculumBlueprintView struct {
	Blueprint model.CurriculumBlueprint           `json:"blueprint"`
	Units     []CurriculumBlueprintUnitView       `json:"units"`
	Relations []model.CurriculumBlueprintRelation `json:"relations"`
}

type CurriculumBlueprintUnitView struct {
	Unit    model.CurriculumBlueprintUnit     `json:"unit"`
	Lessons []model.CurriculumBlueprintLesson `json:"lessons"`
}

type CurriculumCoverageLesson struct {
	BlueprintLesson model.CurriculumBlueprintLesson `json:"blueprint_lesson"`
	State           string                          `json:"state"`
	AppliedLesson   *LessonReference                `json:"applied_lesson"`
}

type LessonReference struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	UnitID    uint   `json:"unit_id"`
	UnitTitle string `json:"unit_title"`
}

type CurriculumCoverageUnit struct {
	Unit               model.CurriculumBlueprintUnit `json:"unit"`
	Lessons            []CurriculumCoverageLesson    `json:"lessons"`
	CoreTotal          int                           `json:"core_total"`
	CoreCovered        int                           `json:"core_covered"`
	RecommendedTotal   int                           `json:"recommended_total"`
	RecommendedCovered int                           `json:"recommended_covered"`
	OptionalTotal      int                           `json:"optional_total"`
	OptionalCovered    int                           `json:"optional_covered"`
}

type CurriculumCoverageMetrics struct {
	CoreTotal              int     `json:"core_total"`
	CoreCovered            int     `json:"core_covered"`
	CoreMissing            int     `json:"core_missing"`
	RecommendedTotal       int     `json:"recommended_total"`
	RecommendedCovered     int     `json:"recommended_covered"`
	OptionalTotal          int     `json:"optional_total"`
	OptionalCovered        int     `json:"optional_covered"`
	CoreCoveragePercent    float64 `json:"core_coverage_percent"`
	OverallCoveragePercent float64 `json:"overall_coverage_percent"`
}

type CurriculumCoverageView struct {
	Blueprint   model.CurriculumBlueprint           `json:"blueprint"`
	Units       []CurriculumCoverageUnit            `json:"units"`
	Relations   []model.CurriculumBlueprintRelation `json:"relations"`
	Metrics     CurriculumCoverageMetrics           `json:"metrics"`
	MissingCore []CurriculumCoverageLesson          `json:"missing_core"`
}

type CurriculumDraftView struct {
	Draft     model.CurriculumDraft     `json:"draft"`
	ChangeSet model.CurriculumChangeSet `json:"change_set"`
}

func (s *CurriculumService) GetCurriculum(ctx context.Context, courseID uint) (*CurriculumBlueprintView, error) {
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return nil, mapCurriculumCourseError(err)
	}
	blueprint, err := s.curriculum.FindActiveBlueprint(ctx, courseID)
	if err != nil {
		return nil, mapCurriculumBlueprintError(err)
	}
	units, err := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	lessons, err := s.curriculum.ListBlueprintLessons(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	relations, err := s.curriculum.ListBlueprintRelations(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	if err := validateBlueprint(lessons, relations); err != nil {
		return nil, err
	}
	byUnit := map[uint][]model.CurriculumBlueprintLesson{}
	for _, lesson := range lessons {
		byUnit[lesson.BlueprintUnitID] = append(byUnit[lesson.BlueprintUnitID], lesson)
	}
	result := &CurriculumBlueprintView{Blueprint: *blueprint, Units: make([]CurriculumBlueprintUnitView, 0, len(units)), Relations: relations}
	for _, unit := range units {
		result.Units = append(result.Units, CurriculumBlueprintUnitView{Unit: unit, Lessons: byUnit[unit.ID]})
	}
	return result, nil
}

func (s *CurriculumService) GetCoverage(ctx context.Context, courseID uint) (*CurriculumCoverageView, error) {
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return nil, mapCurriculumCourseError(err)
	}
	blueprint, err := s.curriculum.FindActiveBlueprint(ctx, courseID)
	if err != nil {
		return nil, mapCurriculumBlueprintError(err)
	}
	units, err := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	lessons, err := s.curriculum.ListBlueprintLessons(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	relations, err := s.curriculum.ListBlueprintRelations(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	courseLessons, err := s.curriculum.ListCourseLessons(ctx, courseID)
	if err != nil {
		return nil, err
	}
	courseUnits, err := s.curriculum.ListCourseUnits(ctx, courseID)
	if err != nil {
		return nil, err
	}
	unitTitles := map[uint]string{}
	for _, unit := range courseUnits {
		unitTitles[unit.ID] = unit.Title
	}
	lessonByID := map[uint]model.Lesson{}
	for _, lesson := range courseLessons {
		lessonByID[lesson.ID] = lesson
	}
	byUnit := map[uint][]model.CurriculumBlueprintLesson{}
	for _, lesson := range lessons {
		byUnit[lesson.BlueprintUnitID] = append(byUnit[lesson.BlueprintUnitID], lesson)
	}
	view := &CurriculumCoverageView{Blueprint: *blueprint, Units: make([]CurriculumCoverageUnit, 0, len(units)), Relations: relations, MissingCore: []CurriculumCoverageLesson{}}
	for _, unit := range units {
		coverageUnit := CurriculumCoverageUnit{Unit: unit, Lessons: []CurriculumCoverageLesson{}}
		for _, blueprintLesson := range byUnit[unit.ID] {
			item := CurriculumCoverageLesson{BlueprintLesson: blueprintLesson, State: "missing"}
			if blueprintLesson.AppliedLessonID != nil {
				if lesson, ok := lessonByID[*blueprintLesson.AppliedLessonID]; ok {
					item.State = "covered"
					item.AppliedLesson = &LessonReference{ID: lesson.ID, Title: lesson.Title, UnitID: lesson.UnitID, UnitTitle: unitTitles[lesson.UnitID]}
				}
			}
			countCoverage(&coverageUnit, &view.Metrics, blueprintLesson.Importance, item.State == "covered")
			coverageUnit.Lessons = append(coverageUnit.Lessons, item)
			if item.State == "missing" && blueprintLesson.Importance == model.CurriculumImportanceCore {
				view.MissingCore = append(view.MissingCore, item)
			}
		}
		view.Units = append(view.Units, coverageUnit)
	}
	view.Metrics.CoreMissing = view.Metrics.CoreTotal - view.Metrics.CoreCovered
	if view.Metrics.CoreTotal > 0 {
		view.Metrics.CoreCoveragePercent = percent(view.Metrics.CoreCovered, view.Metrics.CoreTotal)
	}
	total := view.Metrics.CoreTotal + view.Metrics.RecommendedTotal + view.Metrics.OptionalTotal
	covered := view.Metrics.CoreCovered + view.Metrics.RecommendedCovered + view.Metrics.OptionalCovered
	if total > 0 {
		view.Metrics.OverallCoveragePercent = percent(covered, total)
	}
	return view, nil
}

// ExpandBlueprintUnit generates only Blueprint Lessons for one live Unit. It
// deliberately stops before CurriculumDraft/Apply, so no formal Lesson or
// personal learning state is created by this operation.
func (s *CurriculumService) ExpandBlueprintUnit(ctx context.Context, courseID, blueprintUnitID uint) (*CurriculumBlueprintView, error) {
	if courseID == 0 || blueprintUnitID == 0 {
		return nil, ErrCurriculumUnitNotFound
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return nil, mapCurriculumCourseError(err)
	}
	blueprint, err := s.curriculum.FindActiveBlueprint(ctx, courseID)
	if err != nil {
		return nil, mapCurriculumBlueprintError(err)
	}
	units, err := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	var target *model.CurriculumBlueprintUnit
	for index := range units {
		if units[index].ID == blueprintUnitID {
			target = &units[index]
			break
		}
	}
	if target == nil {
		return nil, ErrCurriculumUnitNotFound
	}
	allLessons, err := s.curriculum.ListBlueprintLessons(ctx, blueprint.ID)
	if err != nil {
		return nil, err
	}
	existingKeys := make([]string, 0, len(allLessons))
	for _, lesson := range allLessons {
		existingKeys = append(existingKeys, lesson.Key)
	}
	if target.ExpansionStatus == model.CurriculumUnitExpanded {
		return s.GetCurriculum(ctx, courseID)
	}
	if target.ExpansionStatus == model.CurriculumUnitExpanding {
		return nil, ErrCurriculumUnitExpansionInProgress
	}
	claimed, err := s.curriculum.ClaimBlueprintUnitExpansion(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		currentUnits, reloadErr := s.curriculum.ListBlueprintUnits(ctx, blueprint.ID)
		if reloadErr != nil {
			return nil, reloadErr
		}
		for _, current := range currentUnits {
			if current.ID != target.ID {
				continue
			}
			switch current.ExpansionStatus {
			case model.CurriculumUnitExpanding:
				return nil, ErrCurriculumUnitExpansionInProgress
			case model.CurriculumUnitExpanded:
				return s.GetCurriculum(ctx, courseID)
			}
		}
		return nil, ErrCurriculumUnitExpansion
	}
	resetStatus := func() {
		if resetErr := s.curriculum.ResetBlueprintUnitExpansion(context.Background(), target.ID); resetErr != nil {
			log.Printf("reset blueprint unit expansion status: unit_id=%d error=%v", target.ID, resetErr)
		}
	}
	if s.domainProvider == nil {
		resetStatus()
		return nil, ai.ErrNotConfigured
	}
	adjacent := make([]ai.DomainUnit, 0, 2)
	targetIndex := -1
	for index := range units {
		if units[index].ID == target.ID {
			targetIndex = index
			break
		}
	}
	for _, index := range []int{targetIndex - 1, targetIndex + 1} {
		if index < 0 || index >= len(units) {
			continue
		}
		adjacent = append(adjacent, ai.DomainUnit{Key: units[index].Key, Title: units[index].Title, Description: units[index].Description, Importance: units[index].Importance})
	}
	result, _, providerErr := s.domainProvider.ExpandBlueprintUnit(ctx, ai.BlueprintUnitExpansionRequest{
		DomainName: blueprint.Domain, LearningGoal: blueprint.LearningGoal, TargetDepth: blueprint.TargetDepth,
		TargetUnit:    ai.DomainUnit{Key: target.Key, Title: target.Title, Description: target.Description, Importance: target.Importance},
		AdjacentUnits: adjacent, ExistingLessonKeys: existingKeys,
	})
	if providerErr != nil {
		resetStatus()
		return nil, providerErr
	}
	if len(result.Lessons) < 5 || len(result.Lessons) > 10 {
		resetStatus()
		return nil, fmt.Errorf("%w: expansion must contain 5 to 10 lessons", ErrCurriculumUnitExpansion)
	}
	if err := s.curriculum.Transaction(ctx, func(tx *repository.CurriculumRepository) error {
		currentUnits, err := tx.ListBlueprintUnits(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		var currentUnit *model.CurriculumBlueprintUnit
		for index := range currentUnits {
			if currentUnits[index].ID == target.ID {
				currentUnit = &currentUnits[index]
				break
			}
		}
		if currentUnit == nil {
			return ErrCurriculumUnitNotFound
		}
		currentLessons, err := tx.ListBlueprintLessons(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		currentRelations, err := tx.ListBlueprintRelations(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		keySet := map[string]bool{}
		maxSortOrder := 0
		for _, lesson := range currentLessons {
			keySet[normalizeCurriculumKey(lesson.Key)] = true
			if lesson.BlueprintUnitID == target.ID && lesson.SortOrder > maxSortOrder {
				maxSortOrder = lesson.SortOrder
			}
		}
		for index, lesson := range result.Lessons {
			key := strings.TrimSpace(lesson.Key)
			if keySet[normalizeCurriculumKey(key)] || key == "" || strings.TrimSpace(lesson.Title) == "" {
				return ErrCurriculumKeyConflict
			}
			keySet[normalizeCurriculumKey(key)] = true
			item := &model.CurriculumBlueprintLesson{
				BlueprintID: blueprint.ID, BlueprintUnitID: target.ID, Key: key, Title: strings.TrimSpace(lesson.Title), Summary: lesson.Summary,
				Importance: lesson.Importance, ContentRole: model.ContentRole(lesson.ContentRole), DepthLevel: lesson.DepthLevel,
				AssessmentTargetLevel: lesson.AssessmentTargetLevel, SortOrder: maxSortOrder + index + 1,
				GroundingStatus: model.CurriculumGroundingProvisional,
			}
			if item.DepthLevel < 1 {
				item.DepthLevel = 1
			}
			if err := tx.CreateBlueprintLesson(ctx, item); err != nil {
				return err
			}
		}
		relationKeys := map[string]bool{}
		for _, existing := range currentRelations {
			relationKeys[existing.FromLessonKey+"\x00"+existing.ToLessonKey+"\x00"+existing.RelationType] = true
		}
		for _, relation := range result.Relations {
			fromKey := strings.TrimSpace(relation.FromLessonKey)
			toKey := strings.TrimSpace(relation.ToLessonKey)
			if fromKey == toKey || !curriculumRelationTypeValid(relation.RelationType) || !keySet[normalizeCurriculumKey(fromKey)] || !keySet[normalizeCurriculumKey(toKey)] {
				return ErrCurriculumDraftInvalid
			}
			key := fromKey + "\x00" + toKey + "\x00" + relation.RelationType
			if relationKeys[key] {
				continue
			}
			relationKeys[key] = true
			if err := tx.CreateBlueprintRelation(ctx, &model.CurriculumBlueprintRelation{BlueprintID: blueprint.ID, FromLessonKey: fromKey, ToLessonKey: toKey, RelationType: relation.RelationType}); err != nil {
				return err
			}
		}
		completed, err := tx.CompleteBlueprintUnitExpansion(ctx, currentUnit.ID)
		if err != nil {
			return err
		}
		if !completed {
			return ErrCurriculumUnitExpansionInProgress
		}
		return nil
	}); err != nil {
		resetStatus()
		return nil, err
	}
	return s.GetCurriculum(ctx, courseID)
}

func (s *CurriculumService) GenerateDraft(ctx context.Context, courseID uint, scope string, limit int) (*CurriculumDraftView, error) {
	return s.generateDraft(ctx, courseID, scope, limit, nil)
}

func (s *CurriculumService) GenerateDraftForUnit(ctx context.Context, courseID, blueprintUnitID uint, scope string, limit int) (*CurriculumDraftView, error) {
	return s.generateDraft(ctx, courseID, scope, limit, &blueprintUnitID)
}

func (s *CurriculumService) generateDraft(ctx context.Context, courseID uint, scope string, limit int, targetUnitID *uint) (*CurriculumDraftView, error) {
	if scope == "" {
		scope = "missing_core"
	}
	if scope != "missing_core" && scope != "missing_recommended" {
		return nil, ErrCurriculumInvalidScope
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}
	coverage, err := s.GetCoverage(ctx, courseID)
	if err != nil {
		return nil, err
	}
	pendingKey := curriculumDraftPendingKey(courseID, coverage.Blueprint.ID, targetUnitID)
	if pending, pendingErr := s.curriculum.FindPendingDraft(ctx, pendingKey); pendingErr == nil {
		if pending.Status == model.CurriculumDraftStatusGenerating {
			return nil, ErrCurriculumDraftGenerationInProgress
		}
		return s.GetDraft(ctx, courseID, pending.ID)
	} else if !repository.IsCurriculumNotFound(pendingErr) {
		return nil, pendingErr
	}
	missing := make([]CurriculumCoverageLesson, 0)
	for _, unit := range coverage.Units {
		if targetUnitID != nil && unit.Unit.ID != *targetUnitID {
			continue
		}
		for _, item := range unit.Lessons {
			if item.State == "missing" && (scope == "missing_core" && item.BlueprintLesson.Importance == model.CurriculumImportanceCore || scope == "missing_recommended" && (item.BlueprintLesson.Importance == model.CurriculumImportanceCore || item.BlueprintLesson.Importance == model.CurriculumImportanceRecommended)) {
				missing = append(missing, item)
			}
		}
	}
	if len(missing) > limit {
		missing = missing[:limit]
	}
	if len(missing) == 0 {
		return nil, ErrCurriculumDraftInvalid
	}
	changeSet, summary := ruleDraft(missing, coverage)
	generatedBy := model.CurriculumGeneratedByRule
	providerMeta := ai.ProviderMeta{Provider: "rule", Model: "rule", PromptVersion: "learnos-curriculum-draft-v1", AttemptCount: 1}
	title := "补充缺失的核心知识"
	if targetUnitID != nil {
		for _, unit := range coverage.Units {
			if unit.Unit.ID == *targetUnitID {
				title = "扩充知识区域：" + unit.Unit.Title
				break
			}
		}
	}
	placeholder := `{"new_units":[],"new_lessons":[],"new_relations":[],"blueprint_mappings":[]}`
	draft := &model.CurriculumDraft{CourseID: courseID, BlueprintID: coverage.Blueprint.ID, BlueprintUnitID: targetUnitID, PendingKey: pendingKey, Title: title, Summary: "正在生成课程草案…", Status: model.CurriculumDraftStatusGenerating, GeneratedBy: generatedBy, Provider: providerMeta.Provider, Model: providerMeta.Model, PromptVersion: providerMeta.PromptVersion, ChangeSetJSON: placeholder}
	if err := s.curriculum.CreateDraft(ctx, draft); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrCurriculumDraftAlreadyPending
		}
		return nil, err
	}
	failGeneration := func() {
		if failErr := s.curriculum.FailDraftGeneration(context.Background(), draft.ID); failErr != nil {
			log.Printf("fail curriculum draft generation: draft_id=%d error=%v", draft.ID, failErr)
		}
	}
	if s.provider != nil {
		request := ai.CurriculumDraftRequest{CourseName: coverage.Blueprint.Name, Scope: scope, Limit: limit, Missing: curriculumMissingForAI(missing), RuleChangeSetJSON: mustJSON(changeSet)}
		result, meta, providerErr := s.provider.GenerateCurriculumDraft(ctx, request)
		providerMeta = meta
		if providerErr != nil {
			log.Printf("curriculum draft ai: course_id=%d blueprint_id=%d missing_count=%d provider=%s model=%s prompt_version=%s status=failed error=%s", courseID, coverage.Blueprint.ID, len(missing), meta.Provider, meta.Model, meta.PromptVersion, ai.ErrorCode(providerErr))
			failGeneration()
			return nil, providerErr
		}
		parsed, parseErr := parseChangeSet(result.ChangeSetJSON)
		if parseErr != nil {
			failGeneration()
			return nil, parseErr
		}
		changeSet, summary, generatedBy = parsed, result.Summary, model.CurriculumGeneratedByAI
	}
	raw, _ := json.Marshal(changeSet)
	if err := s.curriculum.FinalizeDraftGeneration(ctx, draft.ID, providerMeta.Provider, providerMeta.Model, providerMeta.PromptVersion, generatedBy, summary, string(raw)); err != nil {
		failGeneration()
		return nil, err
	}
	draft.Status = model.CurriculumDraftStatusDraft
	draft.Summary = summary
	draft.Provider = providerMeta.Provider
	draft.Model = providerMeta.Model
	draft.PromptVersion = providerMeta.PromptVersion
	draft.GeneratedBy = generatedBy
	draft.ChangeSetJSON = string(raw)
	return &CurriculumDraftView{Draft: *draft, ChangeSet: changeSet}, nil
}

func curriculumDraftPendingKey(courseID, blueprintID uint, unitID *uint) string {
	if unitID == nil {
		return fmt.Sprintf("course:%d:blueprint:%d:all", courseID, blueprintID)
	}
	return fmt.Sprintf("course:%d:blueprint:%d:unit:%d", courseID, blueprintID, *unitID)
}

func normalizeCurriculumKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func (s *CurriculumService) ListDrafts(ctx context.Context, courseID uint) ([]CurriculumDraftView, error) {
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return nil, mapCurriculumCourseError(err)
	}
	drafts, err := s.curriculum.ListDrafts(ctx, courseID)
	if err != nil {
		return nil, err
	}
	result := make([]CurriculumDraftView, 0, len(drafts))
	for _, draft := range drafts {
		changeSet, parseErr := parseChangeSet(draft.ChangeSetJSON)
		if parseErr != nil {
			return nil, parseErr
		}
		result = append(result, CurriculumDraftView{Draft: draft, ChangeSet: changeSet})
	}
	return result, nil
}

func (s *CurriculumService) GetDraft(ctx context.Context, courseID, draftID uint) (*CurriculumDraftView, error) {
	draft, err := s.curriculum.FindDraft(ctx, courseID, draftID)
	if err != nil {
		if repository.IsCurriculumNotFound(err) {
			return nil, ErrCurriculumDraftNotFound
		}
		return nil, err
	}
	changeSet, err := parseChangeSet(draft.ChangeSetJSON)
	if err != nil {
		return nil, err
	}
	return &CurriculumDraftView{Draft: *draft, ChangeSet: changeSet}, nil
}

func (s *CurriculumService) RejectDraft(ctx context.Context, courseID, draftID uint) (*CurriculumDraftView, error) {
	if err := s.curriculum.Transaction(ctx, func(tx *repository.CurriculumRepository) error {
		draft, err := tx.FindDraft(ctx, courseID, draftID)
		if err != nil {
			return ErrCurriculumDraftNotFound
		}
		if draft.Status != model.CurriculumDraftStatusDraft {
			return ErrCurriculumDraftApplied
		}
		return tx.UpdateDraftStatus(ctx, draftID, model.CurriculumDraftStatusDraft, model.CurriculumDraftStatusRejected, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetDraft(ctx, courseID, draftID)
}

func (s *CurriculumService) ApplyDraft(ctx context.Context, courseID, draftID uint) (*CurriculumDraftView, error) {
	err := s.curriculum.Transaction(ctx, func(tx *repository.CurriculumRepository) error {
		draft, err := tx.FindDraft(ctx, courseID, draftID)
		if err != nil {
			return ErrCurriculumDraftNotFound
		}
		if draft.Status != model.CurriculumDraftStatusDraft {
			return ErrCurriculumDraftApplied
		}
		changeSet, err := parseChangeSet(draft.ChangeSetJSON)
		if err != nil {
			return err
		}
		blueprint, err := tx.FindActiveBlueprint(ctx, courseID)
		if err != nil || blueprint.ID != draft.BlueprintID {
			return ErrCurriculumBlueprintNotFound
		}
		blueprintLessons, err := tx.ListBlueprintLessons(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		blueprintRelations, err := tx.ListBlueprintRelations(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		if err := validateBlueprint(blueprintLessons, blueprintRelations); err != nil {
			return err
		}
		courseLessons, err := tx.ListCourseLessons(ctx, courseID)
		if err != nil {
			return err
		}
		courseUnits, err := tx.ListCourseUnits(ctx, courseID)
		if err != nil {
			return err
		}
		courseRelations, err := tx.ListCourseRelations(ctx, courseID)
		if err != nil {
			return err
		}
		blueprintUnitByKey := map[string]model.CurriculumBlueprintUnit{}
		blueprintUnits, err := tx.ListBlueprintUnits(ctx, blueprint.ID)
		if err != nil {
			return err
		}
		for _, unit := range blueprintUnits {
			blueprintUnitByKey[unit.Key] = unit
		}
		courseLessonByID := map[uint]model.Lesson{}
		for _, lesson := range courseLessons {
			courseLessonByID[lesson.ID] = lesson
		}
		unitByBlueprintID := map[uint]model.CourseUnit{}
		for _, unit := range courseUnits {
			if unit.BlueprintUnitID != nil {
				unitByBlueprintID[*unit.BlueprintUnitID] = unit
			}
		}
		for _, blueprintLesson := range blueprintLessons {
			if blueprintLesson.AppliedLessonID == nil {
				continue
			}
			lesson, exists := courseLessonByID[*blueprintLesson.AppliedLessonID]
			if !exists {
				continue
			}
			var linked model.CourseUnit
			for _, unit := range courseUnits {
				if unit.ID == lesson.UnitID {
					linked = unit
					break
				}
			}
			if linked.ID == 0 {
				continue
			}
			if err := tx.LinkCourseUnitToBlueprintUnit(ctx, linked.ID, blueprintLesson.BlueprintUnitID); err != nil {
				return ErrCurriculumKeyConflict
			}
			blueprintID := blueprintLesson.BlueprintUnitID
			linked.BlueprintUnitID = &blueprintID
			// Keep a deterministic representative for newly generated lessons;
			// existing applied lessons may legitimately span several course units.
			if _, exists := unitByBlueprintID[blueprintLesson.BlueprintUnitID]; !exists {
				unitByBlueprintID[blueprintLesson.BlueprintUnitID] = linked
			}
		}
		unitByTemp := map[string]uint{}
		for _, newUnit := range changeSet.NewUnits {
			if newUnit.TempKey == "" || newUnit.Title == "" || newUnit.BlueprintUnitKey == "" {
				return ErrCurriculumDraftInvalid
			}
			blueprintUnit, exists := blueprintUnitByKey[newUnit.BlueprintUnitKey]
			if !exists {
				return ErrCurriculumDraftInvalid
			}
			if existing, ok := unitByBlueprintID[blueprintUnit.ID]; ok {
				unitByTemp[newUnit.TempKey] = existing.ID
				continue
			}
			blueprintUnitID := blueprintUnit.ID
			item := &model.CourseUnit{CourseID: courseID, BlueprintUnitID: &blueprintUnitID, Title: newUnit.Title, Objective: newUnit.Description, SortOrder: newUnit.SortOrder, Status: model.CourseUnitStatusPending}
			if err := tx.CreateCourseUnit(ctx, item); err != nil {
				return err
			}
			unitByTemp[newUnit.TempKey] = item.ID
			unitByBlueprintID[blueprintUnit.ID] = *item
		}
		lessonByTitle := map[string]model.Lesson{}
		lessonByID := map[uint]struct{}{}
		lessonByKey := map[string]uint{}
		for _, item := range courseLessons {
			lessonByTitle[item.Title] = item
			lessonByID[item.ID] = struct{}{}
		}
		for _, item := range blueprintLessons {
			if item.AppliedLessonID != nil {
				lessonByKey[item.Key] = *item.AppliedLessonID
			}
		}
		lessonByTemp := map[string]uint{}
		for _, newLesson := range changeSet.NewLessons {
			if newLesson.TempKey == "" || newLesson.BlueprintLessonKey == "" || newLesson.Title == "" {
				return ErrCurriculumDraftInvalid
			}
			if _, exists := lessonByTitle[newLesson.Title]; exists {
				return ErrCurriculumKeyConflict
			}
			unitID := unitByTemp[newLesson.UnitTempKey]
			if unitID == 0 {
				return ErrCurriculumDraftInvalid
			}
			role := model.ContentRole(newLesson.ContentRole)
			if role == "" {
				role = model.ContentRoleCore
			}
			item := &model.Lesson{CourseID: courseID, UnitID: unitID, Title: newLesson.Title, CoreQuestion: newLesson.CoreQuestion, ExpectedUnderstanding: newLesson.ExpectedUnderstanding, SortOrder: newLesson.SortOrder, Status: model.LessonStatusPending, IsCore: newLesson.IsCore, ContentRole: role, DepthLevel: newLesson.DepthLevel, AssessmentTargetLevel: newLesson.AssessmentTargetLevel, GroundingStatus: model.CurriculumGroundingUngrounded}
			if item.DepthLevel < 1 {
				item.DepthLevel = 1
			}
			if item.AssessmentTargetLevel == "" {
				item.AssessmentTargetLevel = model.AssessmentTargetUnderstand
			}
			if err := tx.CreateCourseLesson(ctx, item); err != nil {
				return err
			}
			lessonByTemp[newLesson.TempKey] = item.ID
			lessonByKey[newLesson.BlueprintLessonKey] = item.ID
			lessonByTitle[item.Title] = *item
			lessonByID[item.ID] = struct{}{}
		}
		resolve := func(key string) uint {
			if id := lessonByTemp[key]; id != 0 {
				return id
			}
			return lessonByKey[key]
		}
		for _, relation := range changeSet.NewRelations {
			from, to := resolve(relation.FromKey), resolve(relation.ToKey)
			if from == 0 || to == 0 {
				return ErrCurriculumDraftInvalid
			}
			relationType := model.LessonRelationType(relation.RelationType)
			if relationType != model.LessonRelationPrerequisite && relationType != model.LessonRelationRelated && relationType != model.LessonRelationExtends && relationType != model.LessonRelationApplication {
				return ErrCurriculumDraftInvalid
			}
			duplicate := false
			for _, existing := range courseRelations {
				if existing.FromLessonID == from && existing.ToLessonID == to && existing.RelationType == relationType {
					duplicate = true
					break
				}
			}
			if !duplicate {
				item := &model.LessonRelation{CourseID: courseID, FromLessonID: from, ToLessonID: to, RelationType: relationType}
				if err := tx.CreateCourseRelation(ctx, item); err != nil {
					return err
				}
				courseRelations = append(courseRelations, *item)
			}
		}
		allLessons, err := tx.ListCourseLessons(ctx, courseID)
		if err != nil {
			return err
		}
		allRelations, err := tx.ListCourseRelations(ctx, courseID)
		if err != nil {
			return err
		}
		if err := ValidateCourseGraph(courseID, allLessons, allRelations); err != nil {
			return err
		}
		for _, mapping := range changeSet.BlueprintMappings {
			lessonID := mapping.AppliedLessonID
			if lessonID == nil {
				lessonIDValue := resolve(mapping.LessonTempKey)
				lessonID = &lessonIDValue
			}
			if lessonID == nil || *lessonID == 0 {
				return ErrCurriculumDraftInvalid
			}
			if _, exists := lessonByID[*lessonID]; !exists {
				return ErrCurriculumDraftInvalid
			}
			var blueprintLesson *model.CurriculumBlueprintLesson
			for index := range blueprintLessons {
				if blueprintLessons[index].Key == mapping.BlueprintLessonKey {
					blueprintLesson = &blueprintLessons[index]
					break
				}
			}
			if blueprintLesson == nil {
				return ErrCurriculumDraftInvalid
			}
			if blueprintLesson.AppliedLessonID != nil && *blueprintLesson.AppliedLessonID != *lessonID {
				return ErrCurriculumKeyConflict
			}
			if err := tx.UpdateBlueprintLessonMapping(ctx, blueprintLesson.ID, *lessonID); err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.UpdateDraftStatus(ctx, draftID, model.CurriculumDraftStatusDraft, model.CurriculumDraftStatusApplied, &now)
	})
	if err != nil {
		return nil, err
	}
	return s.GetDraft(ctx, courseID, draftID)
}

func mapCurriculumCourseError(err error) error {
	if repository.IsCurriculumNotFound(err) {
		return ErrCourseNotFound
	}
	return err
}
func mapCurriculumBlueprintError(err error) error {
	if repository.IsCurriculumNotFound(err) {
		return ErrCurriculumBlueprintNotFound
	}
	return err
}
func percent(covered, total int) float64 { return float64(covered) * 100 / float64(total) }
func countCoverage(unit *CurriculumCoverageUnit, metrics *CurriculumCoverageMetrics, importance string, covered bool) {
	switch importance {
	case model.CurriculumImportanceCore:
		unit.CoreTotal++
		metrics.CoreTotal++
		if covered {
			unit.CoreCovered++
			metrics.CoreCovered++
		}
	case model.CurriculumImportanceRecommended:
		unit.RecommendedTotal++
		metrics.RecommendedTotal++
		if covered {
			unit.RecommendedCovered++
			metrics.RecommendedCovered++
		}
	case model.CurriculumImportanceOptional:
		unit.OptionalTotal++
		metrics.OptionalTotal++
		if covered {
			unit.OptionalCovered++
			metrics.OptionalCovered++
		}
	}
}

func curriculumRelationTypeValid(value string) bool {
	switch value {
	case string(model.LessonRelationPrerequisite), string(model.LessonRelationRelated), string(model.LessonRelationExtends), string(model.LessonRelationApplication):
		return true
	default:
		return false
	}
}

func validateBlueprint(lessons []model.CurriculumBlueprintLesson, relations []model.CurriculumBlueprintRelation) error {
	keys := map[string]struct{}{}
	for _, lesson := range lessons {
		if lesson.Key == "" {
			return ErrCurriculumDraftInvalid
		}
		if _, ok := keys[lesson.Key]; ok {
			return ErrCurriculumKeyConflict
		}
		keys[lesson.Key] = struct{}{}
	}
	indegree := map[string]int{}
	adjacency := map[string][]string{}
	for key := range keys {
		indegree[key] = 0
		adjacency[key] = []string{}
	}
	for _, relation := range relations {
		if _, ok := keys[relation.FromLessonKey]; !ok {
			return ErrCurriculumDraftInvalid
		}
		if _, ok := keys[relation.ToLessonKey]; !ok {
			return ErrCurriculumDraftInvalid
		}
		if relation.RelationType != string(model.LessonRelationPrerequisite) && relation.RelationType != string(model.LessonRelationRelated) && relation.RelationType != string(model.LessonRelationExtends) && relation.RelationType != string(model.LessonRelationApplication) {
			return ErrCurriculumDraftInvalid
		}
		if relation.RelationType == string(model.LessonRelationPrerequisite) {
			indegree[relation.ToLessonKey]++
			adjacency[relation.FromLessonKey] = append(adjacency[relation.FromLessonKey], relation.ToLessonKey)
		}
	}
	queue := []string{}
	for key, degree := range indegree {
		if degree == 0 {
			queue = append(queue, key)
		}
	}
	sort.Strings(queue)
	count := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		count++
		for _, next := range adjacency[current] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}
	if count != len(keys) {
		return ErrPrerequisiteCycle
	}
	return nil
}

func parseChangeSet(raw string) (model.CurriculumChangeSet, error) {
	var changeSet model.CurriculumChangeSet
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&changeSet); err != nil {
		return model.CurriculumChangeSet{}, fmt.Errorf("%w: invalid change set: %v", ErrCurriculumDraftInvalid, err)
	}
	return changeSet, nil
}

func ruleDraft(missing []CurriculumCoverageLesson, coverage *CurriculumCoverageView) (model.CurriculumChangeSet, string) {
	set := model.CurriculumChangeSet{NewUnits: []model.CurriculumDraftUnit{}, NewLessons: []model.CurriculumDraftLesson{}, NewRelations: []model.CurriculumDraftRelation{}, BlueprintMappings: []model.CurriculumBlueprintMapping{}}
	unitSeen := map[string]bool{}
	availableKeys := map[string]bool{}
	for _, unit := range coverage.Units {
		for _, lesson := range unit.Lessons {
			if lesson.State == "covered" || containsMissing(missing, lesson.BlueprintLesson.Key) {
				availableKeys[lesson.BlueprintLesson.Key] = true
			}
		}
	}
	tempByBlueprintKey := map[string]string{}
	for _, item := range missing {
		unitKey := "unit_" + strings.NewReplacer(".", "_", "-", "_").Replace(strings.TrimPrefix(item.BlueprintLesson.Key, "nutrition."))
		unitTemp := unitKey
		if !unitSeen[unitTemp] {
			unitSeen[unitTemp] = true
			var unit model.CurriculumBlueprintUnit
			for _, candidate := range coverage.Units {
				if candidate.Unit.ID == item.BlueprintLesson.BlueprintUnitID {
					unit = candidate.Unit
					break
				}
			}
			set.NewUnits = append(set.NewUnits, model.CurriculumDraftUnit{TempKey: unitTemp, BlueprintUnitKey: unit.Key, Title: unit.Title, Description: unit.Description, SortOrder: unit.SortOrder})
		}
		tempKey := "lesson_" + strings.NewReplacer(".", "_", "-", "_").Replace(strings.TrimPrefix(item.BlueprintLesson.Key, "nutrition."))
		set.NewLessons = append(set.NewLessons, model.CurriculumDraftLesson{TempKey: tempKey, UnitTempKey: unitTemp, BlueprintLessonKey: item.BlueprintLesson.Key, Title: item.BlueprintLesson.Title, Summary: item.BlueprintLesson.Summary, CoreQuestion: item.BlueprintLesson.Title + "需要理解哪些关键因素？", ExpectedUnderstanding: item.BlueprintLesson.Summary, ContentRole: string(item.BlueprintLesson.ContentRole), DepthLevel: item.BlueprintLesson.DepthLevel, AssessmentTargetLevel: item.BlueprintLesson.AssessmentTargetLevel, SortOrder: item.BlueprintLesson.SortOrder, IsCore: item.BlueprintLesson.Importance == model.CurriculumImportanceCore})
		set.BlueprintMappings = append(set.BlueprintMappings, model.CurriculumBlueprintMapping{BlueprintLessonKey: item.BlueprintLesson.Key, LessonTempKey: tempKey})
		tempByBlueprintKey[item.BlueprintLesson.Key] = tempKey
	}
	for _, relation := range coverage.Relations {
		if !availableKeys[relation.FromLessonKey] || !availableKeys[relation.ToLessonKey] {
			continue
		}
		fromKey, toKey := relation.FromLessonKey, relation.ToLessonKey
		if tempKey := tempByBlueprintKey[fromKey]; tempKey != "" {
			fromKey = tempKey
		}
		if tempKey := tempByBlueprintKey[toKey]; tempKey != "" {
			toKey = tempKey
		}
		set.NewRelations = append(set.NewRelations, model.CurriculumDraftRelation{FromKey: fromKey, ToKey: toKey, RelationType: relation.RelationType})
	}
	return set, fmt.Sprintf("根据课程蓝图，建议先补充 %d 个缺失的 %s 知识节点。", len(missing), missing[0].BlueprintLesson.Importance)
}

func containsMissing(items []CurriculumCoverageLesson, key string) bool {
	for _, item := range items {
		if item.BlueprintLesson.Key == key {
			return true
		}
	}
	return false
}

func curriculumMissingForAI(items []CurriculumCoverageLesson) []ai.CurriculumMissingLesson {
	result := make([]ai.CurriculumMissingLesson, 0, len(items))
	for _, item := range items {
		result = append(result, ai.CurriculumMissingLesson{Key: item.BlueprintLesson.Key, UnitKey: fmt.Sprint(item.BlueprintLesson.BlueprintUnitID), Title: item.BlueprintLesson.Title, Summary: item.BlueprintLesson.Summary, Importance: item.BlueprintLesson.Importance, ContentRole: string(item.BlueprintLesson.ContentRole), DepthLevel: item.BlueprintLesson.DepthLevel, AssessmentTargetLevel: item.BlueprintLesson.AssessmentTargetLevel})
	}
	return result
}
func mustJSON(value interface{}) string { raw, _ := json.Marshal(value); return string(raw) }
