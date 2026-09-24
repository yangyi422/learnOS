package database

import (
	"path/filepath"
	"testing"

	"learnos/internal/config"
	"learnos/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpenCreatesPreMigrationBackupForExistingDatabase(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "learnos.db")
	legacy, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.AutoMigrate(&model.Course{}); err != nil {
		t.Fatal(err)
	}
	legacyCourse := model.Course{Name: "迁移保留测试", Status: model.CourseStatusLearning}
	if err := legacy.Create(&legacyCourse).Error; err != nil {
		t.Fatal(err)
	}
	legacySQL, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = legacySQL.Close()

	db, err := Open(config.Config{AppVersion: "test", DatabasePath: databasePath, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var metadata model.SystemMetadata
	if err := db.First(&metadata, 1).Error; err != nil {
		t.Fatal(err)
	}
	if metadata.SchemaVersion != model.CurrentSchemaVersion {
		t.Fatalf("unexpected schema version: %d", metadata.SchemaVersion)
	}
	var preserved model.Course
	if err := db.First(&preserved, legacyCourse.ID).Error; err != nil || preserved.Name != legacyCourse.Name {
		t.Fatalf("migration did not preserve existing data: course=%+v err=%v", preserved, err)
	}
	columns := map[any][]string{
		&model.LearningTurn{}:        {"idempotency_key", "evidence_used_json", "confidence", "uncertainty", "recommended_next_action", "transfer_challenge_eligible", "mastery_score_before", "mastery_score_after"},
		&model.AssessmentChallenge{}: {"idempotency_key"},
		&model.ChallengeAttempt{}:    {"idempotency_key", "explanation"},
		&model.CourseUnit{}:          {"blueprint_unit_id"},
		&model.Misconception{}:       {"review_status", "user_note"},
		&model.ExplorationQuestion{}: {"priority"},
		&model.Project{}:             {"user_id", "title", "description", "status", "icon", "accent", "archived_at"},
		&model.ProjectTask{}:         {"project_id", "title", "description", "status", "sort_order", "priority", "due_date", "completed_at", "is_next_action"},
	}
	for table, names := range columns {
		for _, name := range names {
			if !db.Migrator().HasColumn(table, name) {
				t.Fatalf("schema %d migration is missing %T.%s", model.CurrentSchemaVersion, table, name)
			}
		}
	}
	backups, err := ListBackups(filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected one pre-migration backup, got %d", len(backups))
	}
}

func TestProjectSchemaMigrationIsRepeatable(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{AppVersion: "test", DatabasePath: filepath.Join(dir, "projects.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	for attempt := 0; attempt < 2; attempt++ {
		db, err := Open(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !db.Migrator().HasTable(&model.Project{}) || !db.Migrator().HasTable(&model.ProjectTask{}) {
			t.Fatal("project tables missing")
		}
		var indexCount int64
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", "idx_project_tasks_board_order").Scan(&indexCount).Error; err != nil || indexCount != 1 {
			t.Fatalf("board order index: count=%d err=%v", indexCount, err)
		}
		connection, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProjectKanbanMigrationPreservesLegacyRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy-projects.db")
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE projects (id integer PRIMARY KEY, user_id integer NOT NULL, title text NOT NULL, status text NOT NULL, created_at datetime, updated_at datetime)`,
		`CREATE TABLE project_tasks (id integer PRIMARY KEY, project_id integer NOT NULL, title text NOT NULL, status text NOT NULL, is_next_action numeric NOT NULL DEFAULT 0, created_at datetime, updated_at datetime)`,
		`CREATE TABLE system_metadata (id integer PRIMARY KEY, schema_version integer NOT NULL, last_migrated_at datetime, app_version text, created_at datetime, updated_at datetime)`,
		`INSERT INTO system_metadata (id, schema_version, app_version) VALUES (1, 16, 'legacy')`,
		`INSERT INTO projects (id, user_id, title, status) VALUES (10, 1, 'LearnOS', 'active'), (11, 1, '像素团团', 'paused')`,
		`INSERT INTO project_tasks (id, project_id, title, status, is_next_action, created_at, updated_at) VALUES
		 (1, 10, '整理', 'todo', 0, '2026-09-01', '2026-09-01'),
		 (2, 10, '部署', 'todo', 1, '2026-09-02', '2026-09-02'),
		 (3, 10, '编码', 'doing', 0, '2026-09-03', '2026-09-03'),
		 (4, 10, '完成', 'done', 0, '2026-09-04', '2026-09-05'),
		 (5, 11, '动画', 'doing', 1, '2026-09-06', '2026-09-06')`,
		`CREATE UNIQUE INDEX idx_project_tasks_one_next_action ON project_tasks(project_id) WHERE is_next_action = 1`,
	}
	for _, statement := range statements {
		if err := legacy.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	connection, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = connection.Close()
	cfg := config.Config{AppVersion: "test", DatabasePath: path, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	for attempt := 0; attempt < 2; attempt++ {
		db, err := Open(cfg)
		if err != nil {
			t.Fatal(err)
		}
		var projects []model.Project
		var tasks []model.ProjectTask
		if err := db.Find(&projects).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Order("id").Find(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		if len(projects) != 2 || len(tasks) != 5 {
			t.Fatalf("rows lost: projects=%d tasks=%d", len(projects), len(tasks))
		}
		want := []string{"inbox", "next", "doing", "done", "doing"}
		for i, task := range tasks {
			if task.ID != uint(i+1) || task.Status != want[i] || task.SortOrder <= 0 || task.Priority != "normal" {
				t.Fatalf("task %d migrated incorrectly: %+v", i+1, task)
			}
		}
		if tasks[3].CompletedAt == nil {
			t.Fatal("old done task has no inferred completion time")
		}
		var oldIndex int64
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_project_tasks_one_next_action'").Scan(&oldIndex).Error; err != nil || oldIndex != 0 {
			t.Fatalf("old index count=%d err=%v", oldIndex, err)
		}
		if attempt == 0 {
			t.Logf("legacy status mapping: todo->inbox=1, todo+next->next=1, doing->doing=2, done->done=1; projects=2 tasks=5")
		}
		connection, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		_ = connection.Close()
	}
	backups, err := ListBackups(cfg.BackupDir)
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one pre-migration backup, got %d: %v", len(backups), err)
	}
}

func TestOpenMigratesLegacyOwnerlessCourseAndDomainDraft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE courses (id integer PRIMARY KEY, name text NOT NULL, status text NOT NULL)`,
		`INSERT INTO courses (id, name, status) VALUES (1, '旧课程', 'learning')`,
		`CREATE TABLE domain_initialization_drafts (id integer PRIMARY KEY, domain_name text NOT NULL, learning_goal text NOT NULL, target_depth text NOT NULL, status text NOT NULL, generated_by text NOT NULL)`,
		`INSERT INTO domain_initialization_drafts (id, domain_name, learning_goal, target_depth, status, generated_by) VALUES (1, '旧领域', '学习目标', 'basic', 'skeleton_draft', 'mock')`,
	}
	for _, statement := range statements {
		if err := legacy.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	connection, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(config.Config{AppVersion: "test", DatabasePath: path, BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		connection, err := db.DB()
		if err == nil {
			_ = connection.Close()
		}
	}()
	var course model.Course
	if err := db.First(&course, 1).Error; err != nil || course.Name != "旧课程" || course.UserID != 0 {
		t.Fatalf("legacy course not preserved: %+v %v", course, err)
	}
	var draft model.DomainInitializationDraft
	if err := db.First(&draft, 1).Error; err != nil || draft.DomainName != "旧领域" || draft.UserID != 0 {
		t.Fatalf("legacy draft not preserved: %+v %v", draft, err)
	}
}

func TestBackfillCourseUnitBlueprintIDsUsesAppliedLessonIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backfill.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Course{}, &model.CourseUnit{}, &model.Lesson{}, &model.CurriculumBlueprint{}, &model.CurriculumBlueprintUnit{}, &model.CurriculumBlueprintLesson{}); err != nil {
		t.Fatal(err)
	}
	course := model.Course{Name: "稳定关联", Status: model.CourseStatusLearning}
	if err := db.Create(&course).Error; err != nil {
		t.Fatal(err)
	}
	unit := model.CourseUnit{CourseID: course.ID, Title: "标题可以不同", SortOrder: 1}
	if err := db.Create(&unit).Error; err != nil {
		t.Fatal(err)
	}
	lesson := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "Lesson", Status: model.LessonStatusPending, ContentRole: model.ContentRoleCore, DepthLevel: 1}
	if err := db.Create(&lesson).Error; err != nil {
		t.Fatal(err)
	}
	blueprint := model.CurriculumBlueprint{CourseID: &course.ID, Name: "Blueprint", Status: model.CurriculumBlueprintStatusActive}
	if err := db.Create(&blueprint).Error; err != nil {
		t.Fatal(err)
	}
	blueprintUnit := model.CurriculumBlueprintUnit{BlueprintID: blueprint.ID, Key: "stable-key", Title: "完全不同标题", SortOrder: 1}
	if err := db.Create(&blueprintUnit).Error; err != nil {
		t.Fatal(err)
	}
	blueprintLesson := model.CurriculumBlueprintLesson{BlueprintID: blueprint.ID, BlueprintUnitID: blueprintUnit.ID, Key: "lesson-key", Title: "Blueprint Lesson", AppliedLessonID: &lesson.ID, ContentRole: model.ContentRoleCore, DepthLevel: 1}
	if err := db.Create(&blueprintLesson).Error; err != nil {
		t.Fatal(err)
	}
	if err := backfillCourseUnitBlueprintIDs(db); err != nil {
		t.Fatal(err)
	}
	var reloaded model.CourseUnit
	if err := db.First(&reloaded, unit.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.BlueprintUnitID == nil || *reloaded.BlueprintUnitID != blueprintUnit.ID {
		t.Fatalf("stable blueprint unit link was not backfilled: %+v", reloaded)
	}
}
