package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
	"learnos/internal/repository"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func lifeTestService(t *testing.T) (*LifeService, *gorm.DB) {
	t.Helper()
	db, err := database.Open(config.Config{Environment: "development", DatabasePath: filepath.Join(t.TempDir(), "life.db")})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	t.Cleanup(func() { sql.Close() })
	return NewLifeService(repository.NewLifeRepository(db)), db
}
func eventInput(key, date string) LifeEventInput {
	return LifeEventInput{CreationKey: key, Title: "值得记住", OccurredOn: date, Description: "原始内容", PrimaryDomain: "home", SecondaryDomain: "work", Milestone: true}
}
func goalInput(key string) LifeGoalInput {
	return LifeGoalInput{CreationKey: key, Title: "自己的家", Status: "considering", StatusDate: "2026-10-08", Why: "长期生活", CurrentNote: "评估资金"}
}
func TestLifeEventLifecycleFiltersAndIsolation(t *testing.T) {
	s, db := lifeTestService(t)
	ctx := context.Background()
	in := eventInput("event-a", "2020-03-02")
	a, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil || again.ID != a.ID {
		t.Fatalf("replay %+v %v", again, err)
	}
	in.Title = "different"
	if _, err = s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, repository.ErrLifeConflict) {
		t.Fatal(err)
	}
	bIn := eventInput("event-b", "2026-10-08")
	bIn.Milestone = false
	b, err := s.SaveEvent(ctx, 1, 0, bIn)
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.Events(ctx, 1, "", false)
	if err != nil || len(items) != 2 || items[0].ID != b.ID {
		t.Fatal(items, err)
	}
	filtered, err := s.Events(ctx, 1, "work", true)
	if err != nil || len(filtered) != 1 || filtered[0].ID != a.ID {
		t.Fatal(filtered, err)
	}
	in = eventInput("", "2019-01-02")
	in.Title = "编辑完整内容"
	a, err = s.SaveEvent(ctx, 1, a.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	read, err := s.Event(ctx, 1, a.ID)
	if err != nil || read.Title != in.Title || read.OccurredOn != in.OccurredOn {
		t.Fatal(read, err)
	}
	for _, check := range []func() error{func() error { _, e := s.Event(ctx, 2, a.ID); return e }, func() error { _, e := s.SaveEvent(ctx, 2, a.ID, in); return e }, func() error { return s.DeleteEvent(ctx, 2, a.ID) }} {
		if !errors.Is(check(), gorm.ErrRecordNotFound) {
			t.Fatal("owner isolation failed")
		}
	}
	if err = s.DeleteEvent(ctx, 1, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Event(ctx, 1, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.LifeEvent{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
}
func TestLifeValidation(t *testing.T) {
	s, _ := lifeTestService(t)
	ctx := context.Background()
	for _, mutate := range []func(*LifeEventInput){func(i *LifeEventInput) { i.Title = " " }, func(i *LifeEventInput) { i.OccurredOn = "2026-02-30" }, func(i *LifeEventInput) { i.OccurredOn = "2026-1-02" }, func(i *LifeEventInput) { i.PrimaryDomain = "unknown" }, func(i *LifeEventInput) { i.SecondaryDomain = i.PrimaryDomain }, func(i *LifeEventInput) { i.PrimaryDomain = "" }, func(i *LifeEventInput) { i.ExternalURL = "javascript:alert(1)" }, func(i *LifeEventInput) { i.ExternalURL = "data:text/plain,abc" }, func(i *LifeEventInput) { i.ExternalURL = "file:///tmp/notes" }, func(i *LifeEventInput) { i.ExternalURL = "obsidian://open?vault=Wiki&file=%ZZ" }, func(i *LifeEventInput) { i.SourceType = "project" }, func(i *LifeEventInput) { i.CreationKey = "bad/key" }} {
		in := eventInput("invalid", "2026-10-08")
		mutate(&in)
		if _, err := s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, ErrInvalidLifeInput) {
			t.Fatalf("not rejected %+v %v", in, err)
		}
	}
	in := eventInput("link", "2026-10-08")
	in.ExternalURL = "obsidian://open?vault=YY-Wiki&file=Life%2F我的家.md"
	saved, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil || saved.ExternalURL != in.ExternalURL {
		t.Fatal(saved, err)
	}
	in = eventInput("bad-goal", "2026-10-08")
	id := uint(999)
	in.GoalID = &id
	if _, err = s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
}
func TestLifeInboxAtomicReplayAndFailure(t *testing.T) {
	s, db := lifeTestService(t)
	ctx := context.Background()
	item := model.InboxItem{UserID: 1, Content: "过去的一次搬家", Status: "inbox", SourceType: "manual"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	in := eventInput("inbox-a", "2020-05-01")
	in.SourceType = "inbox"
	in.SourceID = &item.ID
	in.Description = item.Content
	if err := db.Exec("CREATE TRIGGER reject_life_event BEFORE INSERT ON life_events BEGIN SELECT RAISE(ABORT, 'test failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveEvent(ctx, 1, 0, in); err == nil {
		t.Fatal("injected failure accepted")
	}
	var read model.InboxItem
	db.First(&read, item.ID)
	if read.Status != "inbox" || read.ProcessedToID != nil {
		t.Fatalf("partial conversion %+v", read)
	}
	db.Exec("DROP TRIGGER reject_life_event")
	a, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil || repeat.ID != a.ID {
		t.Fatal(repeat, err)
	}
	in.CreationKey = "different-key"
	repeat, err = s.SaveEvent(ctx, 1, 0, in)
	if err != nil || repeat.ID != a.ID {
		t.Fatal("source dedupe", repeat, err)
	}
	db.First(&read, item.ID)
	if read.Status != "processed" || read.Content != item.Content || read.ProcessedToType != "life_event" || *read.ProcessedToID != a.ID {
		t.Fatal(read)
	}
	in.Title = "different-event"
	if _, err = s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, repository.ErrLifeConflict) {
		t.Fatal(err)
	}
	projects := repository.NewProjectRepository(db)
	inbox := NewInboxService(repository.NewInboxRepository(db), projects)
	if _, _, err = inbox.ConvertToTask(ctx, 1, item.ID, InboxConvertInput{ProjectID: 999}); !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
		t.Fatal("cross conversion", err)
	}
	if err := s.DeleteEvent(ctx, 1, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, repository.ErrLifeSource) {
		t.Fatal("deleted source recaptured", err)
	}
}
func TestLifeConcurrentInboxConversion(t *testing.T) {
	s, db := lifeTestService(t)
	item := model.InboxItem{UserID: 1, Content: "a", Status: "inbox", SourceType: "manual"}
	db.Create(&item)
	in := eventInput("race", "2026-10-08")
	in.SourceType = "inbox"
	in.SourceID = &item.ID
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.SaveEvent(context.Background(), 1, 0, in) }()
	}
	wg.Wait()
	a, err := s.SaveEvent(context.Background(), 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.LifeEvent{}).Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	var read model.InboxItem
	db.First(&read, item.ID)
	if read.ProcessedToID == nil || *read.ProcessedToID != a.ID {
		t.Fatal(read)
	}
}
func TestLifeGoalHistoryAndRelations(t *testing.T) {
	s, db := lifeTestService(t)
	ctx := context.Background()
	p := model.Project{UserID: 1, Title: "住房计划", Status: "active"}
	db.Create(&p)
	in := goalInput("goal-a")
	in.ProjectID = &p.ID
	g, err := s.SaveGoal(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := s.SaveGoal(ctx, 1, 0, in)
	if err != nil || repeat.ID != g.ID {
		t.Fatal(repeat, err)
	}
	in.Title = "新名字"
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	_, entries, _, err := s.Goal(ctx, 1, g.ID)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	in.Status = "active"
	in.StatusDate = "2027-03-02"
	in.StatusReason = "工作地点变化"
	in.CurrentNote = "重新评估区域"
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	entry := LifeEntryInput{CreationKey: "decision", OccurredOn: "2026-10-08", Content: "先评估通勤", Reason: "长期居住"}
	saved, err := s.AddEntry(ctx, 1, g.ID, entry)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.AddEntry(ctx, 1, g.ID, entry)
	if err != nil || saved.ID != again.ID {
		t.Fatal(again, err)
	}
	eIn := eventInput("goal-event", "2027-03-02")
	eIn.GoalID = &g.ID
	if _, err = s.SaveEvent(ctx, 1, 0, eIn); err != nil {
		t.Fatal(err)
	}
	detail, entries, events, err := s.Goal(ctx, 1, g.ID)
	if err != nil || len(entries) != 2 || len(events) != 1 || entries[0].ToStatus != "active" || entries[0].FromStatus != "considering" || detail.CurrentNote != "重新评估区域" {
		t.Fatal(detail, entries, events, err)
	}
	db.Model(&p).Update("status", "archived")
	detail, _, events, err = s.Goal(ctx, 1, g.ID)
	if err != nil || !detail.ProjectAvailable || len(events) != 1 {
		t.Fatal(detail, events, err)
	}
	db.Delete(&p)
	detail, _, _, err = s.Goal(ctx, 1, g.ID)
	if err != nil || detail.ProjectAvailable {
		t.Fatal(detail, err)
	}
	if _, _, _, err = s.Goal(ctx, 2, g.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err = s.AddEntry(ctx, 2, g.ID, entry); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err = s.SaveGoal(ctx, 2, g.ID, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	in = goalInput("foreign-project")
	foreign := model.Project{UserID: 2, Title: "private", Status: "active"}
	db.Create(&foreign)
	in.ProjectID = &foreign.ID
	if _, err = s.SaveGoal(ctx, 1, 0, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	data, err := NewExportService(db).JSON(ctx)
	if err != nil || !strings.Contains(string(data), `"life_goal_entries"`) || !strings.Contains(string(data), "工作地点变化") {
		t.Fatal(string(data), err)
	}
	md, err := NewExportService(db).Markdown(ctx)
	if err != nil || !strings.Contains(string(md), "先评估通勤") {
		t.Fatal(err)
	}
}
func TestLifeManualSourcesAndGraduation(t *testing.T) {
	s, db := lifeTestService(t)
	ctx := context.Background()
	p := model.Project{UserID: 1, Title: "独立作品", Status: "active"}
	db.Create(&p)
	in := eventInput("project-source", "2026-10-08")
	in.SourceType = "project"
	in.SourceID = &p.ID
	if _, err := s.SaveEvent(ctx, 1, 0, in); !errors.Is(err, repository.ErrLifeSource) {
		t.Fatal(err)
	}
	db.Model(&p).Update("status", "completed")
	event, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil || event.SourceTitle != p.Title {
		t.Fatal(event, err)
	}
	in.CreationKey = "new-key"
	again, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil || again.ID != event.ID {
		t.Fatal(again, err)
	}
	db.Model(&p).Update("status", "archived")
	read, err := s.Event(ctx, 1, event.ID)
	if err != nil || !read.SourceAvailable || read.SourceStatus != "archived" {
		t.Fatal(read, err)
	}
	db.Delete(&p)
	read, err = s.Event(ctx, 1, event.ID)
	if err != nil || read.SourceAvailable || read.SourceTitle != "独立作品" {
		t.Fatal(read, err)
	}
	c := model.Course{UserID: 1, Name: "系统思考", Status: model.CourseStatusLearning}
	db.Create(&c)
	unit := model.CourseUnit{CourseID: c.ID, Title: "单元"}
	db.Create(&unit)
	lesson := model.Lesson{CourseID: c.ID, UnitID: unit.ID, Title: "核心", CoreQuestion: "问题", Status: model.LessonStatusLearning}
	db.Create(&lesson)
	if err = s.Graduate(ctx, 1, c.ID); !errors.Is(err, repository.ErrLifeSource) {
		t.Fatal(err)
	}
	db.Model(&lesson).Update("status", "skipped")
	source, err := s.Source(ctx, 1, c.ID, "course")
	if err != nil || source.Eligible || !source.CanGraduate {
		t.Fatal(source, err)
	}
	blueprint := model.CurriculumBlueprint{CourseID: &c.ID, Status: "active"}
	db.Create(&blueprint)
	bl := model.CurriculumBlueprintLesson{BlueprintID: blueprint.ID, Key: "missing", Title: "缺口"}
	db.Create(&bl)
	if err = s.Graduate(ctx, 1, c.ID); !errors.Is(err, repository.ErrLifeSource) {
		t.Fatal("un-generated curriculum permitted", err)
	}
	db.Model(&bl).Update("applied_lesson_id", lesson.ID)
	if err = s.Graduate(ctx, 1, c.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Graduate(ctx, 1, c.ID); err != nil {
		t.Fatal(err)
	}
	in = eventInput("course-source", "2026-10-08")
	in.SourceType = "course"
	in.SourceID = &c.ID
	event, err = s.SaveEvent(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	source, err = s.Source(ctx, 1, c.ID, "course")
	if err != nil || source.EventID == nil || *source.EventID != event.ID {
		t.Fatal(source, err)
	}
	db.Delete(&c)
	read, err = s.Event(ctx, 1, event.ID)
	if err != nil || read.SourceAvailable || read.SourceTitle != "系统思考" {
		t.Fatal(read, err)
	}
}

func TestLifeBackupRestorePreservesArchive(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DatabasePath: filepath.Join(dir, "life.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 10}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := NewLifeService(repository.NewLifeRepository(db))
	ctx := context.Background()
	goal, err := s.SaveGoal(ctx, 1, 0, goalInput("restore-goal"))
	if err != nil {
		t.Fatal(err)
	}
	in := eventInput("restore-event", "2010-02-03")
	in.GoalID = &goal.ID
	event, err := s.SaveEvent(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AddEntry(ctx, 1, goal.ID, LifeEntryInput{CreationKey: "restore-entry", OccurredOn: "2026-10-08", Content: "保留历史"})
	if err != nil {
		t.Fatal(err)
	}
	backups := NewBackupService(db, cfg)
	snapshot, err := backups.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteEvent(ctx, 1, event.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = backups.RequestRestore(ctx, snapshot.Name, "恢复 "+snapshot.Name); err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.Close()
	restored, err := ApplyPendingRestore(cfg)
	if err != nil || !restored {
		t.Fatal(restored, err)
	}
	db, err = database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sql, _ = db.DB()
	defer sql.Close()
	s = NewLifeService(repository.NewLifeRepository(db))
	read, err := s.Event(ctx, 1, event.ID)
	if err != nil || read.OccurredOn != "2010-02-03" {
		t.Fatal(read, err)
	}
	_, entries, events, err := s.Goal(ctx, 1, goal.ID)
	if err != nil || len(entries) != 1 || len(events) != 1 {
		t.Fatal(entries, events, err)
	}
}
func TestLifeGoalJudgmentHistoryIsOptIn(t *testing.T) {
	s, _ := lifeTestService(t)
	ctx := context.Background()
	in := goalInput("judgment")
	g, err := s.SaveGoal(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	in.CurrentNote = "换一种生活方式"
	in.RecordChange = true
	in.StatusDate = "2027-03-01"
	in.StatusReason = "环境变化"
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	_, entries, _, err := s.Goal(ctx, 1, g.ID)
	if err != nil || len(entries) != 1 || entries[0].Content != in.CurrentNote || entries[0].ToStatus != "" {
		t.Fatal(entries, err)
	}
	in.Title = "标题调整"
	if _, err = s.SaveGoal(ctx, 1, g.ID, in); err != nil {
		t.Fatal(err)
	}
	_, entries, _, _ = s.Goal(ctx, 1, g.ID)
	if len(entries) != 1 {
		t.Fatal(entries)
	}
}

func TestLifeGoalUpdateFailureRollsBackHistory(t *testing.T) {
	s, db := lifeTestService(t)
	ctx := context.Background()
	in := goalInput("rollback-goal")
	goal, err := s.SaveGoal(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("CREATE TRIGGER reject_life_goal BEFORE UPDATE ON life_goals BEGIN SELECT RAISE(ABORT, 'test failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	in.Status = "achieved"
	in.CurrentNote = "达成"
	if _, err = s.SaveGoal(ctx, 1, goal.ID, in); err == nil {
		t.Fatal("update failure ignored")
	}
	read, entries, _, err := s.Goal(ctx, 1, goal.ID)
	if err != nil || read.Status != "considering" || len(entries) != 0 {
		t.Fatal(read, entries, err)
	}
}
