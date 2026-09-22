package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type CurriculumRepository struct{ db *gorm.DB }

func NewCurriculumRepository(db *gorm.DB) *CurriculumRepository { return &CurriculumRepository{db: db} }

func (r *CurriculumRepository) Transaction(ctx context.Context, fn func(*CurriculumRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&CurriculumRepository{db: tx}) })
}

func (r *CurriculumRepository) FindActiveBlueprint(ctx context.Context, courseID uint) (*model.CurriculumBlueprint, error) {
	var blueprint model.CurriculumBlueprint
	if err := r.db.WithContext(ctx).Where("course_id = ? AND status = ?", courseID, model.CurriculumBlueprintStatusActive).First(&blueprint).Error; err != nil {
		return nil, fmt.Errorf("find active curriculum blueprint: %w", err)
	}
	return &blueprint, nil
}

func (r *CurriculumRepository) FindBlueprintByID(ctx context.Context, id uint) (*model.CurriculumBlueprint, error) {
	var blueprint model.CurriculumBlueprint
	if err := r.db.WithContext(ctx).First(&blueprint, id).Error; err != nil {
		return nil, fmt.Errorf("find curriculum blueprint: %w", err)
	}
	return &blueprint, nil
}

func (r *CurriculumRepository) CreateBlueprint(ctx context.Context, blueprint *model.CurriculumBlueprint) error {
	if err := r.db.WithContext(ctx).Create(blueprint).Error; err != nil {
		return fmt.Errorf("create curriculum blueprint: %w", err)
	}
	return nil
}

func (r *CurriculumRepository) ListBlueprintUnits(ctx context.Context, blueprintID uint) ([]model.CurriculumBlueprintUnit, error) {
	var items []model.CurriculumBlueprintUnit
	if err := r.db.WithContext(ctx).Where("blueprint_id = ?", blueprintID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list blueprint units: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) ListBlueprintLessons(ctx context.Context, blueprintID uint) ([]model.CurriculumBlueprintLesson, error) {
	var items []model.CurriculumBlueprintLesson
	if err := r.db.WithContext(ctx).Where("blueprint_id = ?", blueprintID).Order("blueprint_unit_id ASC, sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list blueprint lessons: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) FindBlueprintLesson(ctx context.Context, id uint) (*model.CurriculumBlueprintLesson, error) {
	var lesson model.CurriculumBlueprintLesson
	if err := r.db.WithContext(ctx).First(&lesson, id).Error; err != nil {
		return nil, fmt.Errorf("find blueprint lesson: %w", err)
	}
	return &lesson, nil
}

func (r *CurriculumRepository) ListBlueprintRelations(ctx context.Context, blueprintID uint) ([]model.CurriculumBlueprintRelation, error) {
	var items []model.CurriculumBlueprintRelation
	if err := r.db.WithContext(ctx).Where("blueprint_id = ?", blueprintID).Order("from_lesson_key ASC, to_lesson_key ASC, relation_type ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list blueprint relations: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) CreateBlueprintUnit(ctx context.Context, item *model.CurriculumBlueprintUnit) error {
	return r.create(item, "blueprint unit")
}
func (r *CurriculumRepository) CreateBlueprintLesson(ctx context.Context, item *model.CurriculumBlueprintLesson) error {
	return r.create(item, "blueprint lesson")
}
func (r *CurriculumRepository) CreateBlueprintRelation(ctx context.Context, item *model.CurriculumBlueprintRelation) error {
	return r.create(item, "blueprint relation")
}

func (r *CurriculumRepository) UpdateBlueprintUnitExpansionStatus(ctx context.Context, id uint, status string) error {
	if err := r.db.WithContext(ctx).Model(&model.CurriculumBlueprintUnit{}).Where("id = ?", id).Update("expansion_status", status).Error; err != nil {
		return fmt.Errorf("update blueprint unit expansion status: %w", err)
	}
	return nil
}

// ClaimBlueprintUnitExpansion atomically moves an unexpanded unit into the
// expanding state. Only the caller that updates one row owns the AI request.
func (r *CurriculumRepository) ClaimBlueprintUnitExpansion(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.CurriculumBlueprintUnit{}).
		Where("id = ? AND expansion_status = ?", id, model.CurriculumUnitUnexpanded).
		Update("expansion_status", model.CurriculumUnitExpanding)
	if result.Error != nil {
		return false, fmt.Errorf("claim blueprint unit expansion: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *CurriculumRepository) ResetBlueprintUnitExpansion(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&model.CurriculumBlueprintUnit{}).
		Where("id = ? AND expansion_status = ?", id, model.CurriculumUnitExpanding).
		Update("expansion_status", model.CurriculumUnitUnexpanded)
	if result.Error != nil {
		return fmt.Errorf("reset blueprint unit expansion: %w", result.Error)
	}
	return nil
}

func (r *CurriculumRepository) CompleteBlueprintUnitExpansion(ctx context.Context, id uint) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.CurriculumBlueprintUnit{}).
		Where("id = ? AND expansion_status = ?", id, model.CurriculumUnitExpanding).
		Update("expansion_status", model.CurriculumUnitExpanded)
	if result.Error != nil {
		return false, fmt.Errorf("complete blueprint unit expansion: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *CurriculumRepository) create(value interface{}, label string) error {
	if err := r.db.Create(value).Error; err != nil {
		return fmt.Errorf("create curriculum %s: %w", label, err)
	}
	return nil
}

func (r *CurriculumRepository) UpdateBlueprintLessonMapping(ctx context.Context, id uint, lessonID uint) error {
	if err := r.db.WithContext(ctx).Model(&model.CurriculumBlueprintLesson{}).Where("id = ?", id).Updates(map[string]interface{}{"applied_lesson_id": lessonID, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("update blueprint lesson mapping: %w", err)
	}
	return nil
}

func (r *CurriculumRepository) FindCourse(ctx context.Context, courseID uint) (*model.Course, error) {
	var course model.Course
	if err := r.db.WithContext(ctx).First(&course, courseID).Error; err != nil {
		return nil, fmt.Errorf("find course for curriculum: %w", err)
	}
	return &course, nil
}

func (r *CurriculumRepository) ListCourseUnits(ctx context.Context, courseID uint) ([]model.CourseUnit, error) {
	var items []model.CourseUnit
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list course units for curriculum: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) ListCourseLessons(ctx context.Context, courseID uint) ([]model.Lesson, error) {
	var items []model.Lesson
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("unit_id ASC, sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list course lessons for curriculum: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) FindCourseLesson(ctx context.Context, id uint) (*model.Lesson, error) {
	var lesson model.Lesson
	if err := r.db.WithContext(ctx).First(&lesson, id).Error; err != nil {
		return nil, fmt.Errorf("find course lesson for grounding: %w", err)
	}
	return &lesson, nil
}

func (r *CurriculumRepository) ListCourseRelations(ctx context.Context, courseID uint) ([]model.LessonRelation, error) {
	var items []model.LessonRelation
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list course relations for curriculum: %w", err)
	}
	return items, nil
}

func (r *CurriculumRepository) CreateCourseUnit(ctx context.Context, item *model.CourseUnit) error {
	return r.create(item, "course unit")
}

func (r *CurriculumRepository) LinkCourseUnitToBlueprintUnit(ctx context.Context, courseUnitID, blueprintUnitID uint) error {
	result := r.db.WithContext(ctx).Model(&model.CourseUnit{}).
		Where("id = ? AND (blueprint_unit_id IS NULL OR blueprint_unit_id = ?)", courseUnitID, blueprintUnitID).
		Update("blueprint_unit_id", blueprintUnitID)
	if result.Error != nil {
		return fmt.Errorf("link course unit to blueprint unit: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("link course unit to blueprint unit: stable link conflict")
	}
	return nil
}
func (r *CurriculumRepository) CreateCourseLesson(ctx context.Context, item *model.Lesson) error {
	return r.create(item, "course lesson")
}
func (r *CurriculumRepository) CreateCourseRelation(ctx context.Context, item *model.LessonRelation) error {
	return r.create(item, "course relation")
}

func (r *CurriculumRepository) CreateDraft(ctx context.Context, draft *model.CurriculumDraft) error {
	if err := r.db.WithContext(ctx).Create(draft).Error; err != nil {
		return fmt.Errorf("create curriculum draft: %w", err)
	}
	return nil
}

func (r *CurriculumRepository) FindPendingDraft(ctx context.Context, pendingKey string) (*model.CurriculumDraft, error) {
	var draft model.CurriculumDraft
	if err := r.db.WithContext(ctx).
		Where("pending_key = ? AND status IN ?", pendingKey, []string{model.CurriculumDraftStatusGenerating, model.CurriculumDraftStatusDraft}).
		First(&draft).Error; err != nil {
		return nil, fmt.Errorf("find pending curriculum draft: %w", err)
	}
	return &draft, nil
}

func (r *CurriculumRepository) FinalizeDraftGeneration(ctx context.Context, id uint, provider, modelName, promptVersion, generatedBy, summary, changeSet string) error {
	result := r.db.WithContext(ctx).Model(&model.CurriculumDraft{}).
		Where("id = ? AND status = ?", id, model.CurriculumDraftStatusGenerating).
		Updates(map[string]interface{}{
			"status": model.CurriculumDraftStatusDraft, "provider": provider, "model": modelName,
			"prompt_version": promptVersion, "generated_by": generatedBy, "summary": summary,
			"change_set_json": changeSet, "updated_at": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("finalize curriculum draft generation: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("finalize curriculum draft generation: %w", gorm.ErrDuplicatedKey)
	}
	return nil
}

func (r *CurriculumRepository) FailDraftGeneration(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Model(&model.CurriculumDraft{}).
		Where("id = ? AND status = ?", id, model.CurriculumDraftStatusGenerating).
		Updates(map[string]interface{}{"status": model.CurriculumDraftStatusRejected, "updated_at": time.Now()})
	if result.Error != nil {
		return fmt.Errorf("fail curriculum draft generation: %w", result.Error)
	}
	return nil
}

func (r *CurriculumRepository) ListDrafts(ctx context.Context, courseID uint) ([]model.CurriculumDraft, error) {
	var drafts []model.CurriculumDraft
	if err := r.db.WithContext(ctx).Where("course_id = ?", courseID).Order("created_at DESC, id DESC").Find(&drafts).Error; err != nil {
		return nil, fmt.Errorf("list curriculum drafts: %w", err)
	}
	return drafts, nil
}

func (r *CurriculumRepository) FindDraft(ctx context.Context, courseID, draftID uint) (*model.CurriculumDraft, error) {
	var draft model.CurriculumDraft
	if err := r.db.WithContext(ctx).Where("course_id = ? AND id = ?", courseID, draftID).First(&draft).Error; err != nil {
		return nil, fmt.Errorf("find curriculum draft: %w", err)
	}
	return &draft, nil
}

func (r *CurriculumRepository) UpdateDraftStatus(ctx context.Context, id uint, fromStatus, toStatus string, appliedAt *time.Time) error {
	updates := map[string]interface{}{"status": toStatus, "updated_at": time.Now()}
	if appliedAt != nil {
		updates["applied_at"] = appliedAt
	}
	result := r.db.WithContext(ctx).Model(&model.CurriculumDraft{}).Where("id = ? AND status = ?", id, fromStatus).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update curriculum draft status: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("curriculum draft status changed: %w", gorm.ErrDuplicatedKey)
	}
	return nil
}

func IsCurriculumNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
