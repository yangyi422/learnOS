package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"learnos/internal/ai"
	"learnos/internal/auth"
	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrDomainDraftNotFound = errors.New("domain initialization draft not found")
	ErrDomainDraftInvalid  = errors.New("invalid domain initialization draft")
	ErrDomainDraftState    = errors.New("domain initialization stage is not available")
	ErrDomainAlreadyExists = errors.New("domain already exists")
	ErrDomainInputInvalid  = errors.New("invalid domain input")
	ErrDomainDraftApplied  = errors.New("domain initialization draft already applied")
)

type DomainInitializationService struct {
	drafts   *repository.DomainInitializationRepository
	provider ai.DomainInitializationProvider
}

func NewDomainInitializationService(drafts *repository.DomainInitializationRepository) *DomainInitializationService {
	return &DomainInitializationService{drafts: drafts}
}
func (s *DomainInitializationService) SetProvider(provider ai.DomainInitializationProvider) {
	s.provider = provider
}

type DomainInitializationView struct {
	Draft        model.DomainInitializationDraft `json:"draft"`
	Skeleton     *ai.DomainSkeletonResult        `json:"skeleton,omitempty"`
	Starter      *ai.DomainStarterResult         `json:"starter_blueprint,omitempty"`
	InitialWorld *ai.InitialWorldResult          `json:"initial_world,omitempty"`
}

type CreateDomainRequest struct {
	DomainName   string `json:"domain_name"`
	LearningGoal string `json:"learning_goal"`
	TargetDepth  string `json:"target_depth"`
}

func (s *DomainInitializationService) CreateSkeleton(ctx context.Context, input CreateDomainRequest) (*DomainInitializationView, error) {
	input = normalizeDomainInput(input)
	if err := validateDomainInput(input); err != nil {
		return nil, err
	}
	if _, err := s.drafts.FindCourseByName(ctx, input.DomainName); err == nil {
		return nil, ErrDomainAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if s.provider == nil {
		return nil, ai.ErrNotConfigured
	}
	result, meta, err := s.provider.GenerateDomainSkeleton(ctx, ai.DomainSkeletonRequest{DomainName: input.DomainName, LearningGoal: input.LearningGoal, TargetDepth: input.TargetDepth})
	if err != nil {
		log.Printf("AI domain initialization failed: stage=skeleton provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", meta.Provider, meta.Model, meta.PromptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
		return nil, err
	}
	raw, _ := json.Marshal(result)
	draft := &model.DomainInitializationDraft{DomainName: input.DomainName, LearningGoal: input.LearningGoal, TargetDepth: input.TargetDepth, Status: model.DomainInitializationStatusSkeletonReady, GeneratedBy: generatedBy(meta.Provider), Provider: meta.Provider, Model: meta.Model, SkeletonPromptVersion: meta.PromptVersion, SkeletonJSON: string(raw)}
	if err := s.drafts.Create(ctx, draft); err != nil {
		return nil, err
	}
	return s.view(draft), nil
}

func (s *DomainInitializationService) Get(ctx context.Context, id uint) (*DomainInitializationView, error) {
	draft, err := s.drafts.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDomainDraftNotFound
		}
		return nil, err
	}
	return s.view(draft), nil
}

func (s *DomainInitializationService) RegenerateSkeleton(ctx context.Context, id uint) (*DomainInitializationView, error) {
	draft, err := s.getDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if draft.Status == model.DomainInitializationStatusApplied {
		return nil, ErrDomainDraftApplied
	}
	if s.provider == nil {
		return nil, ai.ErrNotConfigured
	}
	result, meta, err := s.provider.GenerateDomainSkeleton(ctx, ai.DomainSkeletonRequest{DomainName: draft.DomainName, LearningGoal: draft.LearningGoal, TargetDepth: draft.TargetDepth})
	if err != nil {
		log.Printf("AI domain initialization failed: stage=skeleton provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", meta.Provider, meta.Model, meta.PromptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
		return nil, err
	}
	raw, _ := json.Marshal(result)
	if err := s.drafts.Update(ctx, id, map[string]interface{}{"status": model.DomainInitializationStatusSkeletonReady, "provider": meta.Provider, "model": meta.Model, "generated_by": generatedBy(meta.Provider), "skeleton_prompt_version": meta.PromptVersion, "skeleton_json": string(raw), "starter_blueprint_json": "", "initial_world_json": "", "starter_prompt_version": "", "world_prompt_version": "", "applied_course_id": nil, "applied_at": nil}); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *DomainInitializationService) ExpandStarter(ctx context.Context, id uint, selected []string) (*DomainInitializationView, error) {
	draft, err := s.getDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if draft.Status == model.DomainInitializationStatusApplied {
		return nil, ErrDomainDraftApplied
	}
	if draft.Status != model.DomainInitializationStatusSkeletonReady && draft.Status != model.DomainInitializationStatusStarterExpanded {
		return nil, ErrDomainDraftState
	}
	var skeleton ai.DomainSkeletonResult
	if err := json.Unmarshal([]byte(draft.SkeletonJSON), &skeleton); err != nil {
		return nil, ErrDomainDraftInvalid
	}
	if len(selected) == 0 {
		selected = append([]string{}, skeleton.RecommendedStarterKeys...)
	}
	if len(selected) < 1 || len(selected) > 2 || !containsKeys(skeleton, selected) {
		return nil, ErrDomainDraftInvalid
	}
	if s.provider == nil {
		return nil, ai.ErrNotConfigured
	}
	result, meta, err := s.provider.ExpandDomainStarter(ctx, ai.DomainStarterRequest{DomainName: draft.DomainName, LearningGoal: draft.LearningGoal, TargetDepth: draft.TargetDepth, Skeleton: skeleton, SelectedStarterKeys: selected})
	if err != nil {
		log.Printf("AI domain initialization failed: stage=starter_blueprint provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", meta.Provider, meta.Model, meta.PromptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
		return nil, err
	}
	raw, _ := json.Marshal(result)
	if err := s.drafts.Update(ctx, id, map[string]interface{}{"status": model.DomainInitializationStatusStarterExpanded, "provider": meta.Provider, "model": meta.Model, "generated_by": generatedBy(meta.Provider), "starter_prompt_version": meta.PromptVersion, "starter_blueprint_json": string(raw), "initial_world_json": "", "world_prompt_version": ""}); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *DomainInitializationService) GenerateInitialWorld(ctx context.Context, id uint) (*DomainInitializationView, error) {
	draft, err := s.getDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if draft.Status == model.DomainInitializationStatusApplied {
		return nil, ErrDomainDraftApplied
	}
	if draft.Status != model.DomainInitializationStatusStarterExpanded && draft.Status != model.DomainInitializationStatusWorldReady {
		return nil, ErrDomainDraftState
	}
	var starter ai.DomainStarterResult
	if err := json.Unmarshal([]byte(draft.StarterBlueprintJSON), &starter); err != nil {
		return nil, ErrDomainDraftInvalid
	}
	if draft.Status == model.DomainInitializationStatusWorldReady && strings.TrimSpace(draft.InitialWorldJSON) != "" {
		return s.Get(ctx, id)
	}
	if s.provider == nil {
		return nil, ai.ErrNotConfigured
	}
	result, meta, err := s.provider.GenerateInitialWorld(ctx, ai.InitialWorldRequest{DomainName: draft.DomainName, LearningGoal: draft.LearningGoal, TargetDepth: draft.TargetDepth, Starter: starter})
	if err != nil {
		log.Printf("AI domain initialization failed: stage=initial_world provider=%s model=%s prompt_version=%s attempt=%d validation_error=%s", meta.Provider, meta.Model, meta.PromptVersion, positiveOrDefault(meta.AttemptCount, 1), ai.ErrorDetail(err))
		return nil, err
	}
	raw, _ := json.Marshal(result)
	if err := s.drafts.Update(ctx, id, map[string]interface{}{"status": model.DomainInitializationStatusWorldReady, "provider": meta.Provider, "model": meta.Model, "generated_by": generatedBy(meta.Provider), "world_prompt_version": meta.PromptVersion, "initial_world_json": string(raw)}); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *DomainInitializationService) Apply(ctx context.Context, id uint) (*DomainInitializationView, error) {
	draft, err := s.getDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	if draft.Status == model.DomainInitializationStatusApplied {
		return s.Get(ctx, id)
	}
	if draft.Status != model.DomainInitializationStatusWorldReady {
		return nil, ErrDomainDraftState
	}
	err = s.drafts.Transaction(ctx, func(tx *gorm.DB) error {
		var current model.DomainInitializationDraft
		if err := tx.First(&current, id).Error; err != nil {
			return ErrDomainDraftNotFound
		}
		if current.Status == model.DomainInitializationStatusApplied {
			return nil
		}
		if current.Status != model.DomainInitializationStatusWorldReady {
			return ErrDomainDraftState
		}
		var duplicate model.Course
		if err := tx.Where("LOWER(TRIM(name)) = LOWER(?)", strings.TrimSpace(current.DomainName)).First(&duplicate).Error; err == nil {
			return ErrDomainAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var skeleton ai.DomainSkeletonResult
		var starter ai.DomainStarterResult
		var world ai.InitialWorldResult
		if json.Unmarshal([]byte(current.SkeletonJSON), &skeleton) != nil || json.Unmarshal([]byte(current.StarterBlueprintJSON), &starter) != nil || json.Unmarshal([]byte(current.InitialWorldJSON), &world) != nil {
			return ErrDomainDraftInvalid
		}
		if _, err := ai.ParseAndValidateDomainSkeleton(current.SkeletonJSON); err != nil {
			return ErrDomainDraftInvalid
		}
		if _, err := ai.ParseAndValidateDomainStarter(current.StarterBlueprintJSON, skeleton, nil); err != nil {
			return ErrDomainDraftInvalid
		}
		if _, err := ai.ParseAndValidateInitialWorld(current.InitialWorldJSON, starter); err != nil {
			return ErrDomainDraftInvalid
		}
		course := &model.Course{Name: current.DomainName, Description: skeleton.Course.Description, Goal: current.LearningGoal, Status: model.CourseStatusInitializing, Progress: 0}
		if principal, ok := auth.PrincipalFromContext(ctx); ok {
			course.UserID = principal.UserID
		}
		if err := tx.Create(course).Error; err != nil {
			return fmt.Errorf("create initialized course: %w", err)
		}
		blueprint := &model.CurriculumBlueprint{CourseID: &course.ID, Name: skeleton.Blueprint.Name, Domain: skeleton.Blueprint.Domain, Description: skeleton.Course.Description, LearningGoal: current.LearningGoal, TargetDepth: current.TargetDepth, Version: "v0", Status: model.CurriculumBlueprintStatusActive, CreatedBy: model.CurriculumBlueprintCreatedByAI, GroundingStatus: model.CurriculumGroundingProvisional}
		if err := tx.Create(blueprint).Error; err != nil {
			return fmt.Errorf("create initialized blueprint: %w", err)
		}
		blueprintUnitIDs := map[string]uint{}
		courseUnitIDs := map[string]uint{}
		unitTitles := map[uint]string{}
		expanded := map[string]bool{}
		for _, unit := range starter.ExpandedUnits {
			expanded[unit.Key] = true
		}
		for index, unit := range skeleton.Blueprint.Units {
			item := &model.CurriculumBlueprintUnit{BlueprintID: blueprint.ID, Key: unit.Key, Title: unit.Title, Description: unit.Description, SortOrder: index + 1, Importance: unit.Importance, ExpansionStatus: model.CurriculumUnitUnexpanded}
			if expanded[unit.Key] {
				item.ExpansionStatus = model.CurriculumUnitExpanded
			}
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			blueprintUnitID := item.ID
			courseUnit := &model.CourseUnit{
				CourseID:        course.ID,
				BlueprintUnitID: &blueprintUnitID,
				Title:           unit.Title,
				Objective:       unit.Description,
				SortOrder:       index + 1,
				Status:          model.CourseUnitStatusPending,
			}
			if err := tx.Create(courseUnit).Error; err != nil {
				return fmt.Errorf("create initialized course unit: %w", err)
			}
			unitTitles[courseUnit.ID] = courseUnit.Title

			blueprintUnitIDs[unit.Key] = item.ID
			courseUnitIDs[unit.Key] = courseUnit.ID
		}
		blueprintLessons := map[string]model.CurriculumBlueprintLesson{}
		lessonCourseUnitIDs := map[string]uint{}
		for unitIndex, unit := range starter.ExpandedUnits {
			for lessonIndex, lesson := range unit.Lessons {
				item := model.CurriculumBlueprintLesson{BlueprintID: blueprint.ID, BlueprintUnitID: blueprintUnitIDs[unit.Key], Key: lesson.Key, Title: lesson.Title, Summary: lesson.Summary, Importance: lesson.Importance, ContentRole: model.ContentRole(lesson.ContentRole), DepthLevel: lesson.DepthLevel, AssessmentTargetLevel: lesson.AssessmentTargetLevel, SortOrder: lessonIndex + 1, GroundingStatus: model.CurriculumGroundingProvisional}
				if item.DepthLevel < 1 {
					item.DepthLevel = unitIndex + 1
				}
				if err := tx.Create(&item).Error; err != nil {
					return err
				}
				blueprintLessons[lesson.Key] = item
				lessonCourseUnitIDs[lesson.Key] = courseUnitIDs[unit.Key]
			}
		}
		relations := mergeDomainRelations(starter.Relations, world.Relations)
		for _, relation := range relations {
			if err := tx.Create(&model.CurriculumBlueprintRelation{BlueprintID: blueprint.ID, FromLessonKey: relation.FromLessonKey, ToLessonKey: relation.ToLessonKey, RelationType: relation.RelationType}).Error; err != nil {
				return err
			}
		}
		if err := validateBlueprintValues(blueprintLessons, relations); err != nil {
			return err
		}
		lessonIDs := map[string]uint{}
		for _, item := range world.InitialLessons {
			blueprintLesson := blueprintLessons[item.BlueprintLessonKey]
			unitID := lessonCourseUnitIDs[item.BlueprintLessonKey]
			if unitID == 0 {
				return ErrDomainDraftInvalid
			}
			lesson := &model.Lesson{CourseID: course.ID, UnitID: unitID, Title: item.Title, CoreQuestion: item.CoreQuestion, ExpectedUnderstanding: item.ExpectedUnderstanding, SortOrder: blueprintLesson.SortOrder, Status: model.LessonStatusPending, IsCore: item.IsCore, ContentRole: model.ContentRole(item.ContentRole), DepthLevel: item.DepthLevel, AssessmentTargetLevel: item.AssessmentTargetLevel, GroundingStatus: model.CurriculumGroundingProvisional}
			if err := tx.Create(lesson).Error; err != nil {
				return err
			}
			lessonIDs[item.BlueprintLessonKey] = lesson.ID
			if err := tx.Model(&model.CurriculumBlueprintLesson{}).Where("id = ?", blueprintLesson.ID).Update("applied_lesson_id", lesson.ID).Error; err != nil {
				return err
			}
		}
		for _, relation := range world.Relations {
			from, to := lessonIDs[relation.FromLessonKey], lessonIDs[relation.ToLessonKey]
			if from == 0 || to == 0 {
				continue
			}
			if err := tx.Create(&model.LessonRelation{CourseID: course.ID, FromLessonID: from, ToLessonID: to, RelationType: model.LessonRelationType(relation.RelationType)}).Error; err != nil {
				return err
			}
		}
		firstID := lessonIDs[world.RecommendedFirstKey]
		if firstID == 0 {
			return ErrDomainDraftInvalid
		}
		var first model.Lesson
		if err := tx.First(&first, firstID).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(course).Updates(map[string]interface{}{"current_unit": unitTitles[first.UnitID], "current_unit_id": first.UnitID, "current_lesson_id": first.ID, "last_studied_at": nil, "status": model.CourseStatusLearning}).Error; err != nil {
			return err
		}
		if err := tx.Model(&current).Updates(map[string]interface{}{"status": model.DomainInitializationStatusApplied, "applied_course_id": course.ID, "applied_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *DomainInitializationService) getDraft(ctx context.Context, id uint) (*model.DomainInitializationDraft, error) {
	draft, err := s.drafts.Find(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDomainDraftNotFound
		}
		return nil, err
	}
	return draft, nil
}
func (s *DomainInitializationService) view(draft *model.DomainInitializationDraft) *DomainInitializationView {
	view := &DomainInitializationView{Draft: *draft}
	var skeleton ai.DomainSkeletonResult
	if json.Unmarshal([]byte(draft.SkeletonJSON), &skeleton) == nil {
		view.Skeleton = &skeleton
	}
	var starter ai.DomainStarterResult
	if json.Unmarshal([]byte(draft.StarterBlueprintJSON), &starter) == nil {
		view.Starter = &starter
	}
	var world ai.InitialWorldResult
	if json.Unmarshal([]byte(draft.InitialWorldJSON), &world) == nil {
		view.InitialWorld = &world
	}
	return view
}
func normalizeDomainInput(input CreateDomainRequest) CreateDomainRequest {
	input.DomainName = strings.TrimSpace(input.DomainName)
	input.LearningGoal = strings.TrimSpace(input.LearningGoal)
	input.TargetDepth = strings.TrimSpace(input.TargetDepth)
	return input
}
func validateDomainInput(input CreateDomainRequest) error {
	if len([]rune(input.DomainName)) < 2 || len([]rune(input.DomainName)) > 50 || len([]rune(input.LearningGoal)) < 10 || len([]rune(input.LearningGoal)) > 4000 || (input.TargetDepth != "overview" && input.TargetDepth != "foundation" && input.TargetDepth != "systematic") {
		return ErrDomainInputInvalid
	}
	return nil
}
func generatedBy(provider string) string {
	if provider == "mock" || provider == "rule" || provider == "" {
		return model.CurriculumGeneratedByRule
	}
	return model.CurriculumGeneratedByAI
}
func containsKeys(skeleton ai.DomainSkeletonResult, selected []string) bool {
	valid := map[string]bool{}
	for _, item := range skeleton.Blueprint.Units {
		valid[item.Key] = true
	}
	for _, key := range selected {
		if !valid[key] {
			return false
		}
	}
	return true
}
func mergeDomainRelations(groups ...[]ai.DomainLessonRelation) []ai.DomainLessonRelation {
	seen := map[string]bool{}
	result := []ai.DomainLessonRelation{}
	for _, group := range groups {
		for _, relation := range group {
			key := relation.FromLessonKey + "|" + relation.ToLessonKey + "|" + relation.RelationType
			if !seen[key] {
				seen[key] = true
				result = append(result, relation)
			}
		}
	}
	return result
}
func validateBlueprintValues(lessons map[string]model.CurriculumBlueprintLesson, relations []ai.DomainLessonRelation) error {
	items := make([]model.CurriculumBlueprintLesson, 0, len(lessons))
	for _, item := range lessons {
		items = append(items, item)
	}
	converted := make([]model.CurriculumBlueprintRelation, 0, len(relations))
	for _, relation := range relations {
		converted = append(converted, model.CurriculumBlueprintRelation{FromLessonKey: relation.FromLessonKey, ToLessonKey: relation.ToLessonKey, RelationType: relation.RelationType})
	}
	return validateBlueprint(items, converted)
}
