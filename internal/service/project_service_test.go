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

func projectTestService(t *testing.T) *ProjectService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "projects.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}); err != nil {
		t.Fatal(err)
	}
	return NewProjectService(repository.NewProjectRepository(db))
}

func TestProjectKanbanWorkflowAndUserIsolation(t *testing.T) {
	svc := projectTestService(t)
	ctx := context.Background()
	project, err := svc.CreateProject(ctx, 1, ProjectPatch{Title: ptr("  LearnOS  "), Description: ptr("完成 v0.1")})
	if err != nil || project.Title != "LearnOS" || project.Description != "完成 v0.1" {
		t.Fatalf("project: %+v %v", project, err)
	}
	second, err := svc.Create(ctx, 1, "像素团团")
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project.ID, Title: "A", Priority: "high", DueDate: ptr("2026-09-24")})
	if err != nil || a.Status != "inbox" || a.Priority != "high" || a.DueDate == nil {
		t.Fatalf("task A: %+v %v", a, err)
	}
	b, err := svc.CreateTask(ctx, 1, second.ID, "B")
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.CreateTask(ctx, 1, project.ID, "C")
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListTasks(ctx, 1, nil, "inbox", "", 0, 0)
	if err != nil || len(list) != 3 {
		t.Fatalf("all inbox: %+v %v", list, err)
	}
	if list[0].ID != a.ID || list[1].ID != b.ID || list[2].ID != c.ID {
		t.Fatalf("global order: %+v", list)
	}
	// Move C before A, then move A through the entire four-column flow.
	if _, err := svc.MoveTask(ctx, 1, c.ID, "inbox", 0, a.ID, nil); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.ListTasks(ctx, 1, nil, "inbox", "", 0, 0)
	if list[0].ID != c.ID || list[1].ID != a.ID {
		t.Fatalf("reorder: %+v", list)
	}
	for step, status := range []string{"next", "doing", "done", "next"} {
		var expected *time.Time
		if step == 0 {
			expected = &a.UpdatedAt
		}
		moved, err := svc.MoveTask(ctx, 1, a.ID, status, 0, 0, expected)
		if err != nil || moved.Status != status {
			t.Fatalf("move to %s: %+v %v", status, moved, err)
		}
		if (status == "done") != (moved.CompletedAt != nil) {
			t.Fatalf("completion timestamp for %s: %+v", status, moved)
		}
	}
	next, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: second.ID, Title: "D", Status: "next"})
	if err != nil {
		t.Fatal(err)
	}
	nextTasks, _ := svc.ListTasks(ctx, 1, nil, "next", "", 0, 0)
	if len(nextTasks) != 2 || nextTasks[0].ID != a.ID || nextTasks[1].ID != next.ID {
		t.Fatalf("multiple next tasks: %+v", nextTasks)
	}
	page, err := svc.ListTasks(ctx, 1, nil, "next", "", 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != next.ID {
		t.Fatalf("task paging: %+v %v", page, err)
	}
	firstDone, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project.ID, Title: "旧完成", Status: "done"})
	if err != nil {
		t.Fatal(err)
	}
	latestDone, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project.ID, Title: "新完成", Status: "done"})
	if err != nil {
		t.Fatal(err)
	}
	if firstDone.CompletedAt == nil || latestDone.CompletedAt == nil {
		t.Fatal("completed task missing completed_at")
	}
	donePage, err := svc.ListTasks(ctx, 1, nil, "done", "", 1, 0)
	if err != nil || len(donePage) != 1 || donePage[0].ID != latestDone.ID {
		t.Fatalf("recent done page: %+v %v", donePage, err)
	}
	donePage, err = svc.ListTasks(ctx, 1, nil, "done", "", 1, 1)
	if err != nil || len(donePage) != 1 || donePage[0].ID != firstDone.ID {
		t.Fatalf("older done page: %+v %v", donePage, err)
	}
	wrongTime := a.CreatedAt.Add(-1)
	if _, err := svc.MoveTask(ctx, 1, a.ID, "doing", 0, 0, &wrongTime); !errors.Is(err, repository.ErrTaskConflict) {
		t.Fatalf("stale move: %v", err)
	}
	if _, err := svc.EditTask(ctx, 1, a.ID, ProjectTaskPatch{Priority: ptr("low"), DueDate: ptr("")}); err != nil {
		t.Fatal(err)
	}
	filtered, _ := svc.ListTasks(ctx, 1, &project.ID, "next", "", 0, 0)
	if len(filtered) != 1 || filtered[0].Priority != "low" || filtered[0].DueDate != nil {
		t.Fatalf("edit and filter: %+v", filtered)
	}
	if err := svc.Update(ctx, 1, second.ID, ProjectPatch{Status: ptr("archived")}); err != nil {
		t.Fatal(err)
	}
	listedProjects, err := svc.List(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, listed := range listedProjects {
		if listed.ID == second.ID && (listed.ArchivedAt == nil || listed.OpenTaskCount != 2) {
			t.Fatalf("archive/counts: %+v", listed)
		}
	}
	all, _ := svc.ListTasks(ctx, 1, nil, "", "", 0, 0)
	for _, task := range all {
		if task.ProjectID == second.ID {
			t.Fatalf("archived task leaked into board: %+v", task)
		}
	}
	if _, err := svc.CreateTask(ctx, 2, project.ID, "intrusion"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user create: %v", err)
	}
	if _, err := svc.MoveTask(ctx, 2, a.ID, "done", 0, 0, nil); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user move: %v", err)
	}
	if err := svc.DeleteTaskByID(ctx, 2, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user delete: %v", err)
	}
	if err := svc.DeleteTaskByID(ctx, 1, b.ID); err != nil {
		t.Fatal(err)
	}
}

func TestProjectKanbanValidation(t *testing.T) {
	svc := projectTestService(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, 1, " "); !errors.Is(err, ErrInvalidProjectInput) {
		t.Fatal(err)
	}
	project, err := svc.Create(ctx, 1, "计划")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project.ID, Title: "任务", DueDate: ptr("2026-02-30")}); !errors.Is(err, ErrInvalidProjectInput) {
		t.Fatal(err)
	}
	if _, err := svc.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project.ID, Title: "任务", Priority: "urgent"}); !errors.Is(err, ErrInvalidProjectInput) {
		t.Fatal(err)
	}
	if err := svc.Update(ctx, 1, project.ID, ProjectPatch{Status: ptr("deleted")}); !errors.Is(err, ErrInvalidProjectInput) {
		t.Fatal(err)
	}
	if _, err := svc.ListTasks(ctx, 1, nil, "todo", "", 0, 0); !errors.Is(err, ErrInvalidProjectInput) {
		t.Fatal(err)
	}
}

func TestProjectTaskOrderPersistsAfterDatabaseReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persisted-projects.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}); err != nil {
		t.Fatal(err)
	}
	svc := NewProjectService(repository.NewProjectRepository(db))
	ctx := context.Background()
	project, err := svc.Create(ctx, 1, "LearnOS")
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.CreateTask(ctx, 1, project.ID, "first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateTask(ctx, 1, project.ID, "second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MoveTask(ctx, 1, second.ID, "inbox", 0, first.ID, nil); err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if connection, err := reopened.DB(); err == nil {
			_ = connection.Close()
		}
	}()
	reloaded := NewProjectService(repository.NewProjectRepository(reopened))
	tasks, err := reloaded.ListTasks(ctx, 1, nil, "inbox", "", 0, 0)
	if err != nil || len(tasks) != 2 || tasks[0].ID != second.ID || tasks[1].ID != first.ID {
		t.Fatalf("order after reopen: %+v %v", tasks, err)
	}
}

func ptr[T any](value T) *T { return &value }
