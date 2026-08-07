package database

import (
	"fmt"
	"net/url"

	"learnos/internal/config"
	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	values := url.Values{}
	values.Set("_busy_timeout", "5000")
	values.Set("_journal_mode", "WAL")
	values.Set("_foreign_keys", "on")

	dsn := fmt.Sprintf("file:%s?%s", cfg.DatabasePath, values.Encode())
	logLevel := logger.Info
	if cfg.Production() {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := db.AutoMigrate(
		&model.Course{},
		&model.CourseUnit{},
		&model.Lesson{},
		&model.LessonRelation{},
		&model.LearningTurn{},
		&model.MasteryRecord{},
		&model.Misconception{},
		&model.AIEvaluationRun{},
		&model.CognitiveState{},
		&model.CognitiveEvidence{},
		&model.CognitiveStateEvent{},
		&model.AssessmentChallenge{},
		&model.ChallengeAttempt{},
		&model.MisconceptionEvent{},
		&model.MisconceptionPatternLink{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}
