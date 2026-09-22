package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"

	"gorm.io/gorm"
)

type DatabaseHealthService struct {
	db      *gorm.DB
	cfg     config.Config
	backups *BackupService
	ai      *AIConfigurationService
}

func NewDatabaseHealthService(db *gorm.DB, cfg config.Config, backups *BackupService) *DatabaseHealthService {
	return &DatabaseHealthService{db: db, cfg: cfg, backups: backups}
}

func (s *DatabaseHealthService) SetAIConfigurationService(aiConfiguration *AIConfigurationService) {
	s.ai = aiConfiguration
}

type DatabaseHealth struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Ping is intentionally cheap enough for /health and Docker healthchecks.
// The full integrity check remains available through diagnostics.
func (s *DatabaseHealthService) Ping(ctx context.Context) DatabaseHealth {
	if err := s.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return DatabaseHealth{Status: "error", Detail: err.Error()}
	}
	return DatabaseHealth{Status: "ok"}
}

func (s *DatabaseHealthService) Check(ctx context.Context) DatabaseHealth {
	var result string
	if err := s.db.WithContext(ctx).Raw("PRAGMA integrity_check").Scan(&result).Error; err != nil {
		return DatabaseHealth{Status: "error", Detail: err.Error()}
	}
	if result != "ok" {
		return DatabaseHealth{Status: "error", Detail: result}
	}
	if err := s.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return DatabaseHealth{Status: "error", Detail: err.Error()}
	}
	return DatabaseHealth{Status: "ok"}
}

type Diagnostics struct {
	AppVersion          string               `json:"app_version"`
	Environment         string               `json:"environment"`
	SchemaVersion       int                  `json:"schema_version"`
	DatabaseStatus      DatabaseHealth       `json:"database_status"`
	DatabasePath        string               `json:"database_path"`
	DatabaseSize        int64                `json:"database_size"`
	LastBackup          *database.BackupInfo `json:"last_backup"`
	AIProvider          string               `json:"ai_provider"`
	AIModel             string               `json:"ai_model"`
	AITimeouts          map[string]int       `json:"ai_timeouts_seconds"`
	CourseCount         int64                `json:"course_count"`
	LessonCount         int64                `json:"lesson_count"`
	LearningTurnCount   int64                `json:"learning_turn_count"`
	CognitiveStateCount int64                `json:"cognitive_state_count"`
	GeneratedAt         time.Time            `json:"generated_at"`
}

func (s *DatabaseHealthService) Diagnostics(ctx context.Context) (*Diagnostics, error) {
	info, err := os.Stat(s.cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("stat database: %w", err)
	}
	var metadata model.SystemMetadata
	if err := s.db.WithContext(ctx).First(&metadata, 1).Error; err != nil {
		return nil, fmt.Errorf("read schema metadata: %w", err)
	}
	var courseCount, lessonCount, turnCount, stateCount int64
	queries := []struct {
		table string
		dest  *int64
	}{
		{"courses", &courseCount}, {"lessons", &lessonCount}, {"learning_turns", &turnCount}, {"cognitive_states", &stateCount},
	}
	for _, query := range queries {
		if err := s.db.WithContext(ctx).Table(query.table).Count(query.dest).Error; err != nil {
			return nil, fmt.Errorf("count %s: %w", query.table, err)
		}
	}
	aiProvider, aiModel := s.cfg.AIProvider, s.cfg.DeepSeekModel
	if s.ai != nil {
		if configuration, configErr := s.ai.Get(ctx); configErr == nil {
			aiProvider, aiModel = configuration.Provider, configuration.Model
		}
	}
	return &Diagnostics{
		AppVersion: s.cfg.AppVersion, Environment: s.cfg.Environment, SchemaVersion: metadata.SchemaVersion,
		DatabaseStatus: s.Check(ctx), DatabasePath: s.cfg.DatabasePath, DatabaseSize: info.Size(), LastBackup: s.backups.Last(),
		AIProvider: aiProvider, AIModel: aiModel,
		AITimeouts: map[string]int{
			"evaluation": s.cfg.AITimeoutSeconds, "challenge_generation": s.cfg.ChallengeGenerationTimeoutSeconds,
			"curriculum_draft": s.cfg.CurriculumDraftTimeoutSeconds, "source_credibility": s.cfg.SourceCredibilityTimeoutSeconds,
			"domain_skeleton": s.cfg.DomainSkeletonTimeoutSeconds, "domain_starter_blueprint": s.cfg.DomainStarterBlueprintTimeoutSeconds,
			"domain_initial_world": s.cfg.DomainInitialWorldTimeoutSeconds,
		},
		CourseCount: courseCount, LessonCount: lessonCount, LearningTurnCount: turnCount, CognitiveStateCount: stateCount, GeneratedAt: time.Now().UTC(),
	}, nil
}
