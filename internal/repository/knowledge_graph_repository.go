package repository

import (
	"context"
	"fmt"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type KnowledgeGraphRepository struct {
	db *gorm.DB
}

func NewKnowledgeGraphRepository(db *gorm.DB) *KnowledgeGraphRepository {
	return &KnowledgeGraphRepository{db: db}
}

func (r *KnowledgeGraphRepository) ListUnitsByCourse(ctx context.Context, courseID uint) ([]model.CourseUnit, error) {
	var units []model.CourseUnit
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order ASC, id ASC").
		Find(&units).Error; err != nil {
		return nil, fmt.Errorf("list course units for graph: %w", err)
	}
	return units, nil
}

func (r *KnowledgeGraphRepository) ListLessonsByCourse(ctx context.Context, courseID uint) ([]model.Lesson, error) {
	var lessons []model.Lesson
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("unit_id ASC, sort_order ASC, id ASC").
		Find(&lessons).Error; err != nil {
		return nil, fmt.Errorf("list lessons for graph: %w", err)
	}
	return lessons, nil
}

func (r *KnowledgeGraphRepository) ListRelationsByCourse(ctx context.Context, courseID uint) ([]model.LessonRelation, error) {
	var relations []model.LessonRelation
	if err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("from_lesson_id ASC, to_lesson_id ASC, relation_type ASC, id ASC").
		Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("list lesson relations for graph: %w", err)
	}
	return relations, nil
}

func (r *KnowledgeGraphRepository) ListRelationsForLesson(ctx context.Context, courseID, lessonID uint) ([]model.LessonRelation, error) {
	var relations []model.LessonRelation
	if err := r.db.WithContext(ctx).
		Where("course_id = ? AND (from_lesson_id = ? OR to_lesson_id = ?)", courseID, lessonID, lessonID).
		Order("from_lesson_id ASC, to_lesson_id ASC, relation_type ASC, id ASC").
		Find(&relations).Error; err != nil {
		return nil, fmt.Errorf("list lesson relations: %w", err)
	}
	return relations, nil
}

func (r *KnowledgeGraphRepository) FindLessonByCourse(ctx context.Context, courseID, lessonID uint) (*model.Lesson, error) {
	var lesson model.Lesson
	if err := r.db.WithContext(ctx).
		Where("course_id = ? AND id = ?", courseID, lessonID).
		First(&lesson).Error; err != nil {
		return nil, fmt.Errorf("find lesson in course: %w", err)
	}
	return &lesson, nil
}
