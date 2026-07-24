package repository

import (
	"context"
	"fmt"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type CourseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) List(ctx context.Context) ([]model.Course, error) {
	var courses []model.Course
	if err := r.db.WithContext(ctx).Order("updated_at DESC").Find(&courses).Error; err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	return courses, nil
}

func (r *CourseRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Course{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count courses: %w", err)
	}
	return count, nil
}

func (r *CourseRepository) Create(ctx context.Context, course *model.Course) error {
	if err := r.db.WithContext(ctx).Create(course).Error; err != nil {
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}
