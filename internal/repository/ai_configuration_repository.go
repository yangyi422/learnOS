package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"learnos/internal/model"
)

type AIConfigurationRepository struct{ db *gorm.DB }

func NewAIConfigurationRepository(db *gorm.DB) *AIConfigurationRepository {
	return &AIConfigurationRepository{db: db}
}

func (r *AIConfigurationRepository) LastSuccessfulCallAt(ctx context.Context) (*time.Time, error) {
	var run model.AIEvaluationRun
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.AIEvaluationRunStatusSuccess).
		Order("created_at DESC").First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	value := run.CreatedAt.UTC()
	return &value, nil
}

func (r *AIConfigurationRepository) Find(ctx context.Context) (*model.AIConfiguration, error) {
	var configuration model.AIConfiguration
	if err := r.db.WithContext(ctx).First(&configuration, 1).Error; err != nil {
		return nil, err
	}
	return &configuration, nil
}

func (r *AIConfigurationRepository) Save(ctx context.Context, configuration *model.AIConfiguration) error {
	var existing model.AIConfiguration
	err := r.db.WithContext(ctx).First(&existing, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		configuration.ID = 1
		return r.db.WithContext(ctx).Create(configuration).Error
	}
	if err != nil {
		return err
	}
	configuration.ID = 1
	return r.db.WithContext(ctx).Save(configuration).Error
}
