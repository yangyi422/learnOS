package service

import (
	"context"

	"learnos/internal/model"
	"learnos/internal/repository"
)

type CourseService struct {
	repository *repository.CourseRepository
}

func NewCourseService(repository *repository.CourseRepository) *CourseService {
	return &CourseService{repository: repository}
}

func (s *CourseService) List(ctx context.Context) ([]model.Course, error) {
	return s.repository.List(ctx)
}

func (s *CourseService) SeedStarterCourse(ctx context.Context) error {
	count, err := s.repository.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	return s.repository.Create(ctx, &model.Course{
		Name:        "营养学",
		Description: "以现实饮食应用为目标的系统营养学课程。",
		Goal:        "建立能够判断、搭配并持续调整个人饮食的知识体系。",
		Status:      model.CourseStatusLearning,
		Progress:    48,
		CurrentUnit: "水与体液平衡",
	})
}
