package service

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
	"learnos/internal/repository"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func reviewTime(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &t
}
func TestMonthlyReviewAggregationCalendarFactsAndSnapshotLifecycle(t *testing.T) {
	life, db := lifeTestService(t)
	s := NewMonthlyReviewService(repository.NewMonthlyReviewRepository(db))
	ctx := context.Background()
	create := func(v interface{}) {
		t.Helper()
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	project := model.Project{UserID: 1, Title: "成果", Status: "completed", CompletedAt: reviewTime("2026-01-15T10:00:00Z")}
	create(&project)
	foreign := model.Project{UserID: 2, Title: "private", Status: "active"}
	create(&foreign)
	for i, date := range []string{"2025-12-31T15:59:59Z", "2025-12-31T16:00:00Z", "2026-01-31T15:59:59Z", "2026-01-31T16:00:00Z"} {
		task := model.ProjectTask{ProjectID: project.ID, Title: date, Status: "done", CompletedAt: reviewTime(date), ActualCompletedAt: reviewTime(date), SortOrder: int64(i + 1), ArchivedAt: reviewTime("2026-02-01T00:00:00Z")}
		create(&task)
	}
	create(&model.ProjectTask{ProjectID: project.ID, Title: "legacy inferred", Status: "done", CompletedAt: reviewTime("2026-01-10T00:00:00Z")})
	create(&model.ProjectTask{ProjectID: foreign.ID, Title: "private task", Status: "done", ActualCompletedAt: reviewTime("2026-01-10T00:00:00Z")})
	course := model.Course{UserID: 1, Name: "真实课程", Status: model.CourseStatusCompleted, CompletedAt: reviewTime("2026-01-20T00:00:00Z")}
	create(&course)
	unit := model.CourseUnit{CourseID: course.ID, Title: "unit"}
	create(&unit)
	lesson := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "已完成课时", Status: model.LessonStatusCompleted, CompletedAt: reviewTime("2026-01-18T00:00:00Z")}
	create(&lesson)
	activity := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "回答活动", Status: model.LessonStatusLearning}
	create(&activity)
	create(&model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "旧完成无日期", Status: model.LessonStatusCompleted})
	for _, id := range []uint{lesson.ID, activity.ID, activity.ID} {
		create(&model.LearningTurn{CourseID: course.ID, UnitID: unit.ID, LessonID: id, Question: "q", UserAnswer: "a", Result: "ok", CreatedAt: *reviewTime("2026-01-18T00:00:00Z")})
	}
	create(&model.ExplorationQuestion{CourseID: course.ID, Question: "真实探索问题", QuestionType: "deepen", WhyThisQuestion: "reason", CreatedAt: *reviewTime("2026-01-19T00:00:00Z")})
	event := model.LifeEvent{UserID: 1, Title: "补录经历", OccurredOn: "2026-01-03", Description: "一月发生", CreatedAt: *reviewTime("2026-03-01T00:00:00Z")}
	create(&event)
	create(&model.LifeEvent{UserID: 1, Title: "项目里程碑", OccurredOn: "2026-01-16", SourceType: "project", SourceID: &project.ID, Milestone: true, Description: "保留生活说明"})
	create(&model.LifeEvent{UserID: 1, Title: "课程里程碑", OccurredOn: "2026-01-20", SourceType: "course", SourceID: &course.ID, Milestone: true})
	goal, err := life.SaveGoal(ctx, 1, 0, goalInput("review-goal"))
	if err != nil {
		t.Fatal(err)
	}
	create(&model.LifeGoalEntry{GoalID: goal.ID, OccurredOn: "2026-01-05", Content: "决定等待", FromStatus: "active", ToStatus: "paused", CreatedAt: *reviewTime("2026-02-01T00:00:00Z")})
	ledger := model.FinanceLedger{UserID: 1, Name: "钱迹测试"}
	create(&ledger)
	create(&model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "income", OccurredAt: "2026-01-01 00:01:00", Currency: "CNY", Kind: "income", AmountMinor: 10})
	create(&model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "expense", OccurredAt: "2026-01-31 23:59:59", Currency: "CNY", Kind: "expense", AmountMinor: 20, Category: "生活", TagsJSON: `["A","B"]`})
	create(&model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "outside", OccurredAt: "2025-12-31 23:59:59", Currency: "CNY", Kind: "expense", AmountMinor: 999})
	assets := NewAssetService(repository.NewAssetRepository(db))
	asset, err := assets.Save(ctx, 1, 0, AssetInput{SnapshotDate: "2026-01-15", Bank: assetString("0"), Debt: assetString("0")})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.Generate(ctx, 1, "2026-01")
	if err != nil {
		t.Fatal(err)
	}
	f := draft.Facts
	if len(f.Actions) != 3 || len(f.Events) != 1 || f.Events[0].Date != "2026-01-03" || len(f.Goals) != 1 || f.Goals[0].Date != "2026-01-05" || len(f.Learning) != 4 {
		t.Fatal("bad monthly sources", f)
	}
	if f.Actions[0].Kind == "project" && !f.Actions[0].Milestone {
		t.Fatal("milestone lost")
	}
	for _, item := range f.Actions {
		if item.Kind == "project" && (!item.Milestone || !strings.Contains(item.Detail, "保留生活说明")) {
			t.Fatal("duplicate merge lost context", item)
		}
	}
	summary, err := NewFinanceService(repository.NewFinanceRepository(db)).Summary(ctx, 1, ledger.ID, FinanceFilter{From: "2026-01-01", To: "2026-01-31", Currency: "CNY", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Finance) != 1 || f.Finance[0].Summary.Income != summary.Income || f.Finance[0].Summary.Expense != summary.Expense || f.Finance[0].Summary.Difference != -10 || len(f.Finance[0].Summary.Tags) != 2 {
		t.Fatal("finance differs", f.Finance)
	}
	if len(f.Assets) != 1 || f.Assets[0].Snapshot.Complete || *f.Assets[0].Snapshot.NetAsset != 0 {
		t.Fatal("partial range", f.Assets)
	}
	input := ReviewSaveInput{FactsID: draft.FactsID, Memorable: "这月值得记住", Thoughts: "自己的思考", NextFocus: "继续观察", ExternalURL: "obsidian://open?vault=Life&file=2026-01"}
	saved, err := s.Save(ctx, 1, "2026-01", input)
	if err != nil || saved.Review.Version != 1 {
		t.Fatal(saved, err)
	}
	replay, err := s.Save(ctx, 1, "2026-01", input)
	if err != nil || replay.Review.ID != saved.Review.ID {
		t.Fatal("replay", replay, err)
	}
	// Mutating/deleting original data must not rewrite a saved archive.
	db.Model(&model.LifeEvent{}).Where("id = ?", event.ID).Update("title", "后来修改")
	db.Delete(&model.ProjectTask{}, 1)
	changed := AssetInput{SnapshotDate: asset.SnapshotDate, Bank: assetString("12.34"), Debt: assetString("0"), Version: asset.Version}
	if _, err = assets.Save(ctx, 1, asset.ID, changed); err != nil {
		t.Fatal(err)
	}
	reopened, err := s.Get(ctx, 1, "2026-01")
	if err != nil || reopened.Facts.Events[0].Title != "补录经历" || *reopened.Facts.Assets[0].Snapshot.AssetTotal != 0 || reopened.Review.Thoughts != input.Thoughts {
		t.Fatal("archive changed", reopened, err)
	}
	next, err := s.Generate(ctx, 1, "2026-01")
	if err != nil || next.Facts.Events[0].Title != "后来修改" || *next.Facts.Assets[0].Snapshot.AssetTotal != 1234 {
		t.Fatal("refresh preview", next, err)
	}
	replace := input
	replace.FactsID = next.FactsID
	replace.Version = saved.Review.Version
	if _, err = s.Save(ctx, 1, "2026-01", replace); !errors.Is(err, ErrReviewInput) {
		t.Fatal("unconfirmed replacement", err)
	}
	replace.ConfirmReplace = true
	refreshed, err := s.Save(ctx, 1, "2026-01", replace)
	if err != nil || refreshed.Review.Thoughts != input.Thoughts || len(refreshed.History) != 2 {
		t.Fatal("lost personal/version", refreshed, err)
	}
	history, err := s.HistorySnapshot(ctx, 1, "2026-01", draft.FactsID)
	if err != nil || history.Facts.Events[0].Title != "补录经历" || history.ArchivedPersonal.Thoughts != input.Thoughts {
		t.Fatal("lost prior version", history, err)
	}
	db.Delete(&event)
	db.Delete(&model.ProjectTask{}, 2)
	reopened, err = s.Get(ctx, 1, "2026-01")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Facts.Events[0].Available {
		t.Fatal("deleted source available")
	}
	unavailable := false
	for _, item := range reopened.Facts.Actions {
		if item.SourceID == 2 && item.Kind == "task" {
			unavailable = !item.Available
		}
	}
	if !unavailable {
		t.Fatal("deleted task available")
	}
	// Editing thoughts uses frozen facts, not regenerated data.
	edit := replace
	edit.Version = refreshed.Review.Version
	edit.Thoughts = "更新心得"
	edit.ConfirmReplace = false
	notes, err := s.Save(ctx, 1, "2026-01", edit)
	if err != nil || notes.Review.Thoughts != edit.Thoughts || notes.Facts.Events[0].Title != "后来修改" {
		t.Fatal(notes, err)
	}
	if _, err = s.Save(ctx, 1, "2026-01", replace); !errors.Is(err, repository.ErrReviewConflict) {
		t.Fatal("stale overwrite", err)
	}
	if _, err = s.HistorySnapshot(ctx, 2, "2026-01", draft.FactsID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("leak", err)
	}
	if doc, err := s.Get(ctx, 2, "2026-01"); err != nil || doc.Review != nil {
		t.Fatal(doc, err)
	}
	feb, err := s.Generate(ctx, 1, "2026-02")
	if err != nil || len(feb.Facts.Assets) != 0 || feb.Facts.Finance[0].Summary != nil || len(feb.Facts.Events) != 0 {
		t.Fatal("empty month fabricated", feb, err)
	}
	export, err := NewExportService(db).JSON(ctx)
	if err != nil || !strings.Contains(string(export), "facts_json") || !strings.Contains(string(export), "自己的思考") {
		t.Fatal("export incomplete", err)
	}
	var count int64
	db.Model(&model.MonthlyReview{}).Where("user_id = ? AND month = ?", 1, "2026-01").Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
}
func TestMonthlyReviewValidationAtomicFailureAndRestore(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DatabasePath: filepath.Join(dir, "review.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s := NewMonthlyReviewService(repository.NewMonthlyReviewRepository(db))
	for _, month := range []string{"2026-00", "2026-13", "26-01", "2026-1", "9999-12", "1899-12"} {
		if _, err = s.Generate(ctx, 1, month); !errors.Is(err, ErrReviewInput) {
			t.Fatal(month, err)
		}
	}
	draft, err := s.Generate(ctx, 1, "2025-12")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, 2, "2025-12", ReviewSaveInput{FactsID: draft.FactsID}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, 1, "2026-01", ReviewSaveInput{FactsID: draft.FactsID}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, 1, "2025-12", ReviewSaveInput{FactsID: draft.FactsID, ExternalURL: "javascript:alert(1)"}); !errors.Is(err, ErrReviewInput) {
		t.Fatal(err)
	}
	db.Exec("CREATE TRIGGER fail_review BEFORE UPDATE ON monthly_review_facts BEGIN SELECT RAISE(ABORT,'synthetic failure'); END")
	if _, err = s.Save(ctx, 1, "2025-12", ReviewSaveInput{FactsID: draft.FactsID}); err == nil {
		t.Fatal("expected rollback")
	}
	var count int64
	db.Model(&model.MonthlyReview{}).Count(&count)
	if count != 0 {
		t.Fatal("partial save", count)
	}
	db.Exec("DROP TRIGGER fail_review")
	saved, err := s.Save(ctx, 1, "2025-12", ReviewSaveInput{FactsID: draft.FactsID})
	if err != nil {
		t.Fatal(err)
	}
	backup, err := database.CreateBackup(ctx, db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	db.Delete(&model.MonthlyReview{}, saved.Review.ID)
	conn, _ := db.DB()
	conn.Close()
	if _, err = RestoreDatabase(cfg.DatabasePath, backup.Path, cfg.BackupDir, 5); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	s = NewMonthlyReviewService(repository.NewMonthlyReviewRepository(db))
	restored, err := s.Get(ctx, 1, "2025-12")
	if err != nil || restored.FactsID != draft.FactsID || restored.Review == nil {
		t.Fatal(restored, err)
	}
	bytes, err := json.Marshal(restored)
	if err != nil || !strings.Contains(string(bytes), `"actions":[]`) {
		t.Fatal(string(bytes), err)
	}
}
func TestMonthlyReviewFutureCompletionCaptureAndReplay(t *testing.T) {
	life, db := lifeTestService(t)
	ctx := context.Background()
	projects := repository.NewProjectRepository(db)
	p := model.Project{UserID: 1, Title: "future project", Status: "active"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	task := model.ProjectTask{Title: "future task", Status: "done"}
	if err := projects.CreateTask(ctx, 1, p.ID, &task); err != nil || task.ActualCompletedAt == nil || !task.ActualCompletedAt.Equal(*task.CompletedAt) {
		t.Fatal(task, err)
	}
	reopened, err := projects.UpdateTask(ctx, 1, task.ID, map[string]interface{}{"status": "doing"})
	if err != nil || reopened.ActualCompletedAt != nil {
		t.Fatal(reopened, err)
	}
	done, err := projects.MoveTask(ctx, 1, task.ID, "done", 0, 0, nil)
	if err != nil || done.ActualCompletedAt == nil {
		t.Fatal(done, err)
	}
	original := *done.ActualCompletedAt
	done, err = projects.UpdateTask(ctx, 1, task.ID, map[string]interface{}{"title": "rename"})
	if err != nil || !done.ActualCompletedAt.Equal(original) {
		t.Fatal("editing forged completion", done, err)
	}
	if err = projects.Update(ctx, 1, p.ID, map[string]interface{}{"status": "completed"}); err != nil {
		t.Fatal(err)
	}
	saved, err := projects.GetProject(ctx, 1, p.ID)
	if err != nil || saved.CompletedAt == nil {
		t.Fatal(saved, err)
	}
	original = *saved.CompletedAt
	if err = projects.Update(ctx, 1, p.ID, map[string]interface{}{"status": "completed", "title": "rename"}); err != nil {
		t.Fatal(err)
	}
	saved, _ = projects.GetProject(ctx, 1, p.ID)
	if !saved.CompletedAt.Equal(original) {
		t.Fatal("replay reset project completion")
	}
	course := model.Course{UserID: 1, Name: "graduate", Status: model.CourseStatusLearning}
	db.Create(&course)
	unit := model.CourseUnit{CourseID: course.ID, Title: "unit"}
	db.Create(&unit)
	lesson := model.Lesson{CourseID: course.ID, UnitID: unit.ID, Title: "lesson", Status: model.LessonStatusLearning}
	db.Create(&lesson)
	db.Model(&course).Update("current_lesson_id", lesson.ID)
	learning := repository.NewLearningRepository(db)
	if _, err = learning.AdvanceLesson(ctx, course.ID, lesson.ID, model.LessonStatusCompleted); err != nil {
		t.Fatal(err)
	}
	db.First(&lesson, lesson.ID)
	if lesson.CompletedAt == nil {
		t.Fatal("missing lesson completion")
	}
	original = *lesson.CompletedAt
	if _, err = learning.AdvanceLesson(ctx, course.ID, lesson.ID, model.LessonStatusCompleted); err != nil {
		t.Fatal(err)
	}
	db.First(&lesson, lesson.ID)
	if !lesson.CompletedAt.Equal(original) {
		t.Fatal("replay reset lesson completion")
	}
	if err = life.Graduate(ctx, 1, course.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&course, course.ID)
	if course.CompletedAt == nil {
		t.Fatal("missing graduation time")
	}
	original = *course.CompletedAt
	if err = life.Graduate(ctx, 1, course.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&course, course.ID)
	if !course.CompletedAt.Equal(original) {
		t.Fatal("replay reset graduation")
	}
	month := localReviewMonth()
	doc, err := NewMonthlyReviewService(repository.NewMonthlyReviewRepository(db)).Generate(ctx, 1, month)
	if err != nil || len(doc.Facts.Actions) != 2 || len(doc.Facts.Learning) != 2 {
		t.Fatal(doc, err)
	}
}
func localReviewMonth() string {
	loc, _ := time.LoadLocation(ReviewTimezone)
	return time.Now().In(loc).Format("2006-01")
}
