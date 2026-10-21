package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dailyFocusFixture struct {
	t        *testing.T
	db       *gorm.DB
	ctx      context.Context
	focuses  *DailyFocusService
	projects *ProjectService
}

func newDailyFocusFixture(t *testing.T) *dailyFocusFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "daily-focus.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.DailyFocus{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	return &dailyFocusFixture{
		t:        t,
		db:       db,
		ctx:      context.Background(),
		focuses:  NewDailyFocusService(repository.NewDailyFocusRepository(db)),
		projects: NewProjectService(repository.NewProjectRepository(db)),
	}
}

func (f *dailyFocusFixture) project(userID uint, scene string) *model.Project {
	return f.projectWithStatus(userID, scene, "active")
}

func (f *dailyFocusFixture) projectWithStatus(userID uint, scene, status string) *model.Project {
	f.t.Helper()
	project := model.Project{UserID: userID, Title: scene + " 项目", Status: status, Context: scene}
	if err := f.db.Create(&project).Error; err != nil {
		f.t.Fatal(err)
	}
	return &project
}

func (f *dailyFocusFixture) createTask(projectID uint, title, status string, options ...ProjectTaskInput) *model.ProjectTask {
	f.t.Helper()
	input := ProjectTaskInput{ProjectID: projectID, Title: title, Status: status}
	if len(options) > 0 {
		input.PlanDate = options[0].PlanDate
		input.DueDate = options[0].DueDate
	}
	task, err := f.projects.CreateTaskWithInput(f.ctx, 1, input)
	if err != nil {
		f.t.Fatal(err)
	}
	return task
}

func (f *dailyFocusFixture) reloadTask(userID, taskID uint) model.ProjectTask {
	f.t.Helper()
	var task model.ProjectTask
	if err := f.db.First(&task, taskID).Error; err != nil {
		f.t.Fatal(err)
	}
	return task
}

func (f *dailyFocusFixture) focusRows() int64 {
	f.t.Helper()
	var count int64
	if err := f.db.Model(&model.DailyFocus{}).Count(&count).Error; err != nil {
		f.t.Fatal(err)
	}
	return count
}

func (f *dailyFocusFixture) set(userID uint, date, scene string, taskID uint, expectedRevision int) *DailyFocusMutation {
	f.t.Helper()
	mutation, err := f.focuses.Set(f.ctx, userID, date, scene, taskID, expectedRevision)
	if err != nil {
		f.t.Fatalf("set %s/%s task %d: %v", date, scene, taskID, err)
	}
	return mutation
}

func TestDailyFocusScopeUniquenessAndSceneIndependence(t *testing.T) {
	f := newDailyFocusFixture(t)
	day := "2026-10-21"
	workTask := f.createTask(f.project(1, "work").ID, "完成提案初稿", "doing")
	lifeTask := f.createTask(f.project(1, "life").ID, "傍晚跑步", "next")

	workMutation := f.set(1, day, "work", workTask.ID, 0)
	if workMutation.Revision != 1 || workMutation.Focus == nil || workMutation.Focus.State != "ok" ||
		workMutation.Focus.Date != day || workMutation.Focus.Context != "work" ||
		workMutation.Focus.TaskID == nil || *workMutation.Focus.TaskID != workTask.ID ||
		workMutation.Focus.ProjectTitle == "" || workMutation.Focus.ProjectContext != "work" {
		t.Fatalf("work mutation: %+v", workMutation)
	}
	if count := f.focusRows(); count != 1 {
		t.Fatalf("focus rows after first set: %d", count)
	}
	lifeMutation := f.set(1, day, "life", lifeTask.ID, 0)
	if lifeMutation.Focus == nil || lifeMutation.Focus.State != "ok" || lifeMutation.Revision != 1 || lifeMutation.Focus.Context != "life" {
		t.Fatalf("life mutation: %+v", lifeMutation)
	}
	if count := f.focusRows(); count != 2 {
		t.Fatalf("work and life must be independent rows: %d", count)
	}

	// The composite unique index itself is the last line of defence.
	duplicate := model.DailyFocus{UserID: 1, FocusDate: day, Context: "work", Revision: 1}
	if err := f.db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate day/scene focus was accepted")
	}

	all, err := f.focuses.Day(f.ctx, 1, day, "all")
	if err != nil || len(all) != 2 || all["work"] == nil || all["life"] == nil {
		t.Fatalf("all day view: %+v %v", all, err)
	}
	if all["work"].Title != "完成提案初稿" || all["life"].Title != "傍晚跑步" {
		t.Fatalf("scene titles swapped: %+v", all)
	}
	// A concrete scene is trimmed to itself.
	workOnly, err := f.focuses.Day(f.ctx, 1, day, "work")
	if err != nil || len(workOnly) != 1 || workOnly["work"] == nil {
		t.Fatalf("work view: %+v %v", workOnly, err)
	}
	if _, exists := workOnly["life"]; exists {
		t.Fatalf("life leaked into work view: %+v", workOnly)
	}
	lifeOnly, err := f.focuses.Day(f.ctx, 1, day, "life")
	if err != nil || len(lifeOnly) != 1 || lifeOnly["life"] == nil {
		t.Fatalf("life view: %+v %v", lifeOnly, err)
	}
	// No inheritance: the next day has no focus until one is set.
	nextDay, err := f.focuses.Day(f.ctx, 1, "2026-10-22", "all")
	if err != nil || len(nextDay) != 2 || nextDay["work"] != nil || nextDay["life"] != nil {
		t.Fatalf("next day inherited focus: %+v %v", nextDay, err)
	}
}

func TestDailyFocusSetDoesNotTouchTaskState(t *testing.T) {
	f := newDailyFocusFixture(t)
	day := "2026-10-21"
	project := f.project(1, "work")
	plan := "2026-10-20"
	due := "2026-10-25"
	first := f.createTask(project.ID, "原焦点", "doing", ProjectTaskInput{PlanDate: &plan, DueDate: &due})
	second := f.createTask(project.ID, "新焦点", "next")
	before := f.reloadTask(1, first.ID)

	if mutation := f.set(1, day, "work", first.ID, 0); mutation.Focus.Title != "原焦点" {
		t.Fatalf("focus title: %+v", mutation.Focus)
	} else if mutation.Focus.TaskUpdatedAt == nil || !mutation.Focus.TaskUpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("task_updated_at does not match the task version: %+v vs %v", mutation.Focus.TaskUpdatedAt, before.UpdatedAt)
	}
	afterSet := f.reloadTask(1, first.ID)
	assertTaskUntouched(t, before, afterSet)
	if afterSet.PlanDate == nil || *afterSet.PlanDate != plan || afterSet.DueDate == nil || *afterSet.DueDate != due {
		t.Fatalf("setting focus changed plan/due dates: %+v", afterSet)
	}

	// Replacing points only the daily_focuses row at the new task; the old task
	// keeps its status, dates, ordering and project.
	secondBefore := f.reloadTask(1, second.ID)
	mutation := f.set(1, day, "work", second.ID, 1)
	if mutation.Revision != 2 || mutation.Focus.TaskID == nil || *mutation.Focus.TaskID != second.ID || mutation.Focus.Title != "新焦点" {
		t.Fatalf("replacement: %+v", mutation)
	}
	assertTaskUntouched(t, before, f.reloadTask(1, first.ID))
	assertTaskUntouched(t, secondBefore, f.reloadTask(1, second.ID))
	if count := f.focusRows(); count != 1 {
		t.Fatalf("replacement created a second row: %d", count)
	}
}

func assertTaskUntouched(t *testing.T, before, after model.ProjectTask) {
	t.Helper()
	if after.Status != before.Status || after.SortOrder != before.SortOrder || after.ProjectID != before.ProjectID ||
		after.Title != before.Title || after.Priority != before.Priority || !after.UpdatedAt.Equal(before.UpdatedAt) ||
		(after.ArchivedAt == nil) != (before.ArchivedAt == nil) {
		t.Fatalf("task changed by focus write:\nbefore=%+v\nafter=%+v", before, after)
	}
	if (before.PlanDate == nil) != (after.PlanDate == nil) || (before.PlanDate != nil && *before.PlanDate != *after.PlanDate) {
		t.Fatalf("plan_date changed: before=%v after=%v", before.PlanDate, after.PlanDate)
	}
	if (before.DueDate == nil) != (after.DueDate == nil) || (before.DueDate != nil && *before.DueDate != *after.DueDate) {
		t.Fatalf("due_date changed: before=%v after=%v", before.DueDate, after.DueDate)
	}
	if (before.CompletedAt == nil) != (after.CompletedAt == nil) {
		t.Fatalf("completed_at changed: before=%v after=%v", before.CompletedAt, after.CompletedAt)
	}
}

func TestDailyFocusCompletionKeepsRowAndCompletionTime(t *testing.T) {
	f := newDailyFocusFixture(t)
	day := "2026-10-21"
	task := f.createTask(f.project(1, "work").ID, "完成提案初稿", "doing")
	f.set(1, day, "work", task.ID, 0)

	if _, err := f.projects.EditTask(f.ctx, 1, task.ID, ProjectTaskPatch{Status: ptr("done")}); err != nil {
		t.Fatal(err)
	}
	completed := f.reloadTask(1, task.ID)
	if completed.CompletedAt == nil {
		t.Fatal("task completion missing completed_at")
	}
	view, err := f.focuses.Day(f.ctx, 1, day, "work")
	if err != nil || view["work"] == nil || view["work"].State != "completed" || view["work"].Title != "完成提案初稿" ||
		view["work"].CompletedAt == nil || !view["work"].CompletedAt.Equal(*completed.CompletedAt) {
		t.Fatalf("completed focus view: %+v %v", view, err)
	}
	// The focus row is kept for the rest of the day, and re-completing must not
	// overwrite the first completion time.
	if count := f.focusRows(); count != 1 {
		t.Fatalf("completed focus row removed: %d", count)
	}
	if _, err := f.projects.EditTask(f.ctx, 1, task.ID, ProjectTaskPatch{Status: ptr("done")}); err != nil {
		t.Fatal(err)
	}
	repeated := f.reloadTask(1, task.ID)
	if repeated.CompletedAt == nil || !repeated.CompletedAt.Equal(*completed.CompletedAt) {
		t.Fatalf("completion time overwritten: first=%v second=%v", completed.CompletedAt, repeated.CompletedAt)
	}
	view, err = f.focuses.Day(f.ctx, 1, day, "work")
	if err != nil || view["work"].State != "completed" {
		t.Fatalf("focus after repeated completion: %+v %v", view, err)
	}
	// A completed task can no longer be chosen as focus.
	if _, err := f.focuses.Set(f.ctx, 1, day, "work", task.ID, 0); !errors.Is(err, ErrDailyFocusTaskInvalid) {
		t.Fatalf("completed task accepted as focus: %v", err)
	}
}

func TestDailyFocusStateDerivation(t *testing.T) {
	day := "2026-10-21"
	t.Run("project paused, completed and archived", func(t *testing.T) {
		for _, status := range []string{"paused", "completed", "archived"} {
			status := status
			t.Run(status, func(t *testing.T) {
				f := newDailyFocusFixture(t)
				project := f.project(1, "work")
				task := f.createTask(project.ID, "保留标题", "doing")
				f.set(1, day, "work", task.ID, 0)
				if err := f.projects.Update(f.ctx, 1, project.ID, ProjectPatch{Status: &status}); err != nil {
					t.Fatal(err)
				}
				view, err := f.focuses.Day(f.ctx, 1, day, "work")
				if err != nil || view["work"] == nil || view["work"].State != "project_inactive" ||
					view["work"].Title != "保留标题" || view["work"].ProjectStatus != status || view["work"].StateReason == "" {
					t.Fatalf("%s focus view: %+v %v", status, view, err)
				}
				// The row is not deleted or moved.
				if count := f.focusRows(); count != 1 {
					t.Fatalf("%s removed focus row: %d", status, count)
				}
			})
		}
	})
	t.Run("project context changed", func(t *testing.T) {
		f := newDailyFocusFixture(t)
		project := f.project(1, "work")
		task := f.createTask(project.ID, "跨场景任务", "doing")
		f.set(1, day, "work", task.ID, 0)
		if err := f.projects.Update(f.ctx, 1, project.ID, ProjectPatch{Context: ptr("life")}); err != nil {
			t.Fatal(err)
		}
		view, err := f.focuses.Day(f.ctx, 1, day, "work")
		if err != nil || view["work"] == nil || view["work"].State != "context_changed" ||
			view["work"].ProjectContext != "life" || view["work"].StateReason == "" || view["work"].Title != "跨场景任务" {
			t.Fatalf("context changed view: %+v %v", view, err)
		}
		// The focus is not moved automatically: the new scene stays empty and
		// the old scene keeps the stale row.
		all, err := f.focuses.Day(f.ctx, 1, day, "all")
		if err != nil || all["work"] == nil || all["life"] != nil {
			t.Fatalf("context change moved focus: %+v %v", all, err)
		}
		// The task is no longer writable as a work focus but is a valid life
		// focus after the project move.
		if _, err := f.focuses.Set(f.ctx, 1, day, "work", task.ID, 0); !errors.Is(err, ErrDailyFocusTaskInvalid) {
			t.Fatalf("cross-scene focus accepted: %v", err)
		}
		if _, err := f.focuses.Set(f.ctx, 1, "2026-10-22", "life", task.ID, 0); err != nil {
			t.Fatalf("life focus after move: %v", err)
		}
	})
	t.Run("task archived", func(t *testing.T) {
		f := newDailyFocusFixture(t)
		task := f.createTask(f.project(1, "life").ID, "归档任务", "next")
		f.set(1, day, "life", task.ID, 0)
		// Archive rules only allow done tasks; simulate the stored anomaly the
		// read model must still classify correctly (archived and not done).
		archivedAt := time.Now().UTC()
		if err := f.db.Model(&model.ProjectTask{}).Where("id = ?", task.ID).Update("archived_at", archivedAt).Error; err != nil {
			t.Fatal(err)
		}
		view, err := f.focuses.Day(f.ctx, 1, day, "life")
		if err != nil || view["life"] == nil || view["life"].State != "archived" || view["life"].StateReason == "" {
			t.Fatalf("archived focus view: %+v %v", view, err)
		}
	})
	t.Run("task deleted keeps row as missing", func(t *testing.T) {
		f := newDailyFocusFixture(t)
		task := f.createTask(f.project(1, "work").ID, "会被删除", "doing")
		f.set(1, day, "work", task.ID, 0)
		if err := f.db.Exec("DELETE FROM project_tasks WHERE id = ?", task.ID).Error; err != nil {
			t.Fatal(err)
		}
		view, err := f.focuses.Day(f.ctx, 1, day, "work")
		if err != nil || view["work"] == nil || view["work"].State != "missing" ||
			view["work"].Title != "" || view["work"].StateReason != "原任务已被删除。" ||
			view["work"].TaskUpdatedAt != nil ||
			view["work"].TaskID == nil || *view["work"].TaskID != task.ID {
			t.Fatalf("missing focus view: %+v %v", view, err)
		}
		// The focus row survives until the user clears it.
		if count := f.focusRows(); count != 1 {
			t.Fatalf("deleted task removed focus row: %d", count)
		}
		if _, err := f.focuses.Clear(f.ctx, 1, day, "work", 0); err != nil {
			t.Fatal(err)
		}
		if count := f.focusRows(); count != 0 {
			t.Fatalf("clearing missing focus kept row: %d", count)
		}
	})
}

func TestDailyFocusIdempotencyClearAndConflicts(t *testing.T) {
	f := newDailyFocusFixture(t)
	day := "2026-10-21"
	project := f.project(1, "work")
	first := f.createTask(project.ID, "第一", "doing")
	second := f.createTask(project.ID, "第二", "next")
	third := f.createTask(project.ID, "第三", "next")

	initial := f.set(1, day, "work", first.ID, 0)
	// Setting the same task again is a successful no-op: revision and row time
	// stay untouched.
	repeated := f.set(1, day, "work", first.ID, 1)
	if repeated.Revision != initial.Revision || !repeated.Focus.UpdatedAt.Equal(initial.Focus.UpdatedAt) {
		t.Fatalf("idempotent set bumped revision: first=%+v repeated=%+v", initial, repeated)
	}
	if count := f.focusRows(); count != 1 {
		t.Fatalf("idempotent set created rows: %d", count)
	}
	// No focus row yet with a non-zero expected revision is a conflict.
	if _, err := f.focuses.Set(f.ctx, 1, "2026-10-23", "work", first.ID, 7); !errors.Is(err, ErrDailyFocusConflict) {
		t.Fatalf("missing row with expected revision: %v", err)
	}
	// Stale revision is rejected and does not change the stored task.
	if _, err := f.focuses.Set(f.ctx, 1, day, "work", second.ID, 99); !errors.Is(err, ErrDailyFocusConflict) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	view, _ := f.focuses.Day(f.ctx, 1, day, "work")
	if view["work"] == nil || view["work"].TaskID == nil || *view["work"].TaskID != first.ID {
		t.Fatalf("conflicting write changed focus: %+v", view)
	}
	// A matching revision and a zero revision both replace the task.
	replaced := f.set(1, day, "work", second.ID, 1)
	if replaced.Revision != 2 {
		t.Fatalf("replacement revision: %d", replaced.Revision)
	}
	replaced = f.set(1, day, "work", third.ID, 0)
	if replaced.Revision != 3 {
		t.Fatalf("unversioned replacement revision: %d", replaced.Revision)
	}
	// Clearing is idempotent and reports revision 0.
	cleared, err := f.focuses.Clear(f.ctx, 1, day, "work", 3)
	if err != nil || cleared.Focus != nil || cleared.Revision != 0 {
		t.Fatalf("clear: %+v %v", cleared, err)
	}
	if count := f.focusRows(); count != 0 {
		t.Fatalf("clear kept row: %d", count)
	}
	again, err := f.focuses.Clear(f.ctx, 1, day, "work", 0)
	if err != nil || again.Focus != nil || again.Revision != 0 {
		t.Fatalf("repeated clear: %+v %v", again, err)
	}
	// Clearing with a stale revision while a row exists conflicts and keeps it.
	kept := f.set(1, day, "work", first.ID, 0)
	if kept.Revision != 1 {
		t.Fatalf("re-set revision: %d", kept.Revision)
	}
	if _, err := f.focuses.Clear(f.ctx, 1, day, "work", 9); !errors.Is(err, ErrDailyFocusConflict) {
		t.Fatalf("stale clear accepted: %v", err)
	}
	if count := f.focusRows(); count != 1 {
		t.Fatalf("conflicting clear removed row: %d", count)
	}
}

func TestDailyFocusCandidateValidationAndUserIsolation(t *testing.T) {
	f := newDailyFocusFixture(t)
	day := "2026-10-21"
	work := f.project(1, "work")
	unclassified := f.project(1, "unclassified")
	paused := f.projectWithStatus(1, "work", "paused")
	archivedProject := f.projectWithStatus(1, "work", "archived")
	foreignProject := f.project(2, "work")

	valid := f.createTask(work.ID, "合法", "doing")
	unclassifiedTask := f.createTask(unclassified.ID, "未分类", "doing")
	pausedTask := f.createTask(paused.ID, "暂停项目", "doing")
	archivedProjectTask := f.createTask(archivedProject.ID, "归档项目", "doing")
	doneTask := f.createTask(work.ID, "已完成", "done")
	archivedTask := f.createTask(work.ID, "已归档", "next")
	foreignTask := model.ProjectTask{ProjectID: foreignProject.ID, Title: "他人任务", Status: "doing"}
	if err := f.db.Create(&foreignTask).Error; err != nil {
		t.Fatal(err)
	}
	foreignTaskID := foreignTask.ID
	if err := f.db.Model(&model.ProjectTask{}).Where("id = ?", archivedTask.ID).Update("archived_at", time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}
	for _, taskID := range []uint{unclassifiedTask.ID, pausedTask.ID, archivedProjectTask.ID, doneTask.ID, archivedTask.ID, foreignTaskID, 0, 9999} {
		if _, err := f.focuses.Set(f.ctx, 1, day, "work", taskID, 0); !errors.Is(err, ErrDailyFocusTaskInvalid) && !errors.Is(err, ErrDailyFocusInputInvalid) {
			t.Fatalf("invalid candidate %d accepted: %v", taskID, err)
		}
	}
	// The owner sets a valid focus.
	f.set(1, day, "work", valid.ID, 0)
	// Another user never sees the owner's focus and cannot write the owner's task.
	foreignView, err := f.focuses.Day(f.ctx, 2, day, "all")
	if err != nil || foreignView["work"] != nil || foreignView["life"] != nil {
		t.Fatalf("foreign user saw focus: %+v %v", foreignView, err)
	}
	if _, err := f.focuses.Set(f.ctx, 2, day, "work", valid.ID, 0); !errors.Is(err, ErrDailyFocusTaskInvalid) {
		t.Fatalf("foreign task write accepted: %v", err)
	}
	// The other user may still set their own focus on their own task.
	if _, err := f.focuses.Set(f.ctx, 2, day, "work", foreignTaskID, 0); err != nil {
		t.Fatalf("owner of foreign task cannot set focus: %v", err)
	}
	// Clearing one user's day never touches the other user's row.
	if _, err := f.focuses.Clear(f.ctx, 2, day, "work", 0); err != nil {
		t.Fatal(err)
	}
	ownerView, err := f.focuses.Day(f.ctx, 1, day, "work")
	if err != nil || ownerView["work"] == nil || ownerView["work"].Title != "合法" {
		t.Fatalf("owner focus disappeared: %+v %v", ownerView, err)
	}
	// A missing foreign task is reported as missing, not as another user's task.
	if err := f.db.Exec("DELETE FROM project_tasks WHERE id = ?", foreignTaskID).Error; err != nil {
		t.Fatal(err)
	}
	foreignView, err = f.focuses.Day(f.ctx, 2, day, "work")
	if err != nil || foreignView["work"] != nil {
		t.Fatalf("deleted foreign focus leaked: %+v %v", foreignView, err)
	}
	ownerView, err = f.focuses.Day(f.ctx, 1, day, "work")
	if err != nil || ownerView["work"] == nil || ownerView["work"].Title != "合法" {
		t.Fatalf("owner focus changed by deleting foreign task: %+v %v", ownerView, err)
	}
}

func TestDailyFocusInputValidation(t *testing.T) {
	f := newDailyFocusFixture(t)
	task := f.createTask(f.project(1, "work").ID, "任务", "doing")
	for _, date := range []string{"", "2026-02-30", "2026-10-21T00:00:00Z", "2026/10/21", "20261021"} {
		if _, err := f.focuses.Day(f.ctx, 1, date, "all"); !errors.Is(err, ErrDailyFocusDateInvalid) {
			t.Fatalf("day date %q: %v", date, err)
		}
		if _, err := f.focuses.Set(f.ctx, 1, date, "work", task.ID, 0); !errors.Is(err, ErrDailyFocusDateInvalid) {
			t.Fatalf("set date %q: %v", date, err)
		}
		if _, err := f.focuses.Clear(f.ctx, 1, date, "work", 0); !errors.Is(err, ErrDailyFocusDateInvalid) {
			t.Fatalf("clear date %q: %v", date, err)
		}
	}
	if _, err := f.focuses.Day(f.ctx, 1, "2026-10-21", "unknown"); !errors.Is(err, ErrDailyFocusContextInvalid) {
		t.Fatalf("unknown scene: %v", err)
	}
	for _, scene := range []string{"all", "unclassified", ""} {
		if _, err := f.focuses.Set(f.ctx, 1, "2026-10-21", scene, task.ID, 0); !errors.Is(err, ErrDailyFocusContextInvalid) {
			t.Fatalf("set scene %q: %v", scene, err)
		}
		if _, err := f.focuses.Clear(f.ctx, 1, "2026-10-21", scene, 0); !errors.Is(err, ErrDailyFocusContextInvalid) {
			t.Fatalf("clear scene %q: %v", scene, err)
		}
	}
	if _, err := f.focuses.Set(f.ctx, 1, "2026-10-21", "work", task.ID, -1); !errors.Is(err, ErrDailyFocusInputInvalid) {
		t.Fatalf("negative revision: %v", err)
	}
	if _, err := f.focuses.Clear(f.ctx, 1, "2026-10-21", "work", -1); !errors.Is(err, ErrDailyFocusInputInvalid) {
		t.Fatalf("negative clear revision: %v", err)
	}
}
