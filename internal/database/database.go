package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"learnos/internal/config"
	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	existingDatabase := databaseFileExists(cfg.DatabasePath)
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
	var preMigrationBackup BackupInfo
	if existingDatabase && migrationNeedsBackup(db) {
		preMigrationBackup, err = CreateBackup(context.Background(), db, cfg.BackupDir, cfg.BackupRetentionCount)
		if err != nil {
			return nil, fmt.Errorf("pre-migration backup failed: %w", err)
		}
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
		&model.ExplorationDirection{},
		&model.ExplorationQuestion{},
		&model.CurriculumBlueprint{},
		&model.CurriculumBlueprintUnit{},
		&model.CurriculumBlueprintLesson{},
		&model.CurriculumBlueprintRelation{},
		&model.CurriculumDraft{},
		&model.KnowledgeSource{},
		&model.SourceEvidence{},
		&model.GroundingLink{},
		&model.SourceCredibilityAssessment{},
		&model.GroundingReviewEvent{},
		&model.SystemMetadata{},
		&model.AIConfiguration{},
		&model.DomainInitializationDraft{},
	); err != nil {
		if preMigrationBackup.Path != "" {
			return nil, fmt.Errorf("migrate database: %w (pre-migration backup: %s)", err, preMigrationBackup.Path)
		}
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	// Schema 14 permits one blueprint area to span multiple course units.
	// AutoMigrate does not remove the old implicit unique index on SQLite.
	if err := db.Exec("DROP INDEX IF EXISTS uni_course_units_blueprint_unit_id").Error; err != nil {
		return nil, fmt.Errorf("remove legacy blueprint unit uniqueness: %w", err)
	}
	if err := backfillCourseUnitBlueprintIDs(db); err != nil {
		return nil, fmt.Errorf("backfill stable course unit links: %w", err)
	}
	now := time.Now().UTC()
	metadata := model.SystemMetadata{ID: 1, SchemaVersion: model.CurrentSchemaVersion, LastMigratedAt: now, AppVersion: cfg.AppVersion}
	if err := db.Where("id = ?", 1).First(&metadata).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("read schema metadata: %w", err)
		}
		if err := db.Create(&metadata).Error; err != nil {
			return nil, fmt.Errorf("write schema metadata: %w", err)
		}
	} else if err := db.Model(&model.SystemMetadata{}).Where("id = ?", 1).Updates(map[string]interface{}{"schema_version": model.CurrentSchemaVersion, "last_migrated_at": now, "app_version": cfg.AppVersion, "updated_at": now}).Error; err != nil {
		return nil, fmt.Errorf("update schema metadata: %w", err)
	}

	return db, nil
}

// backfillCourseUnitBlueprintIDs derives only unambiguous legacy links from
// BlueprintLesson.AppliedLessonID -> Lesson.UnitID. Titles are deliberately
// not used: renamed or duplicate regions must never be merged by display text.
func backfillCourseUnitBlueprintIDs(db *gorm.DB) error {
	type association struct {
		CourseUnitID    uint
		BlueprintUnitID uint
	}
	var rows []association
	if err := db.Raw(`
		SELECT DISTINCT l.unit_id AS course_unit_id, cbl.blueprint_unit_id AS blueprint_unit_id
		FROM curriculum_blueprint_lessons cbl
		JOIN lessons l ON l.id = cbl.applied_lesson_id
		WHERE cbl.applied_lesson_id IS NOT NULL
	`).Scan(&rows).Error; err != nil {
		return err
	}
	byCourseUnit := map[uint]map[uint]struct{}{}
	byBlueprintUnit := map[uint]map[uint]struct{}{}
	for _, row := range rows {
		if byCourseUnit[row.CourseUnitID] == nil {
			byCourseUnit[row.CourseUnitID] = map[uint]struct{}{}
		}
		if byBlueprintUnit[row.BlueprintUnitID] == nil {
			byBlueprintUnit[row.BlueprintUnitID] = map[uint]struct{}{}
		}
		byCourseUnit[row.CourseUnitID][row.BlueprintUnitID] = struct{}{}
		byBlueprintUnit[row.BlueprintUnitID][row.CourseUnitID] = struct{}{}
	}
	for courseUnitID, blueprintIDs := range byCourseUnit {
		if len(blueprintIDs) != 1 {
			continue
		}
		for blueprintUnitID := range blueprintIDs {
			if len(byBlueprintUnit[blueprintUnitID]) != 1 {
				continue
			}
			if err := db.Model(&model.CourseUnit{}).
				Where("id = ? AND blueprint_unit_id IS NULL", courseUnitID).
				Update("blueprint_unit_id", blueprintUnitID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func databaseFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

func migrationNeedsBackup(db *gorm.DB) bool {
	var tableName string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", "system_metadata").Scan(&tableName).Error; err != nil || tableName == "" {
		return true
	}
	var metadata model.SystemMetadata
	if err := db.First(&metadata, 1).Error; err != nil {
		return true
	}
	return metadata.SchemaVersion < model.CurrentSchemaVersion
}
