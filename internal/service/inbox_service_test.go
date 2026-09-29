package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"learnos/internal/model"
	"learnos/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func inboxTestServices(t *testing.T) (*InboxService, *ProjectService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "workspace.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ProjectTask{}, &model.InboxItem{}); err != nil {
		t.Fatal(err)
	}
	projectRepo := repository.NewProjectRepository(db)
	return NewInboxService(repository.NewInboxRepository(db), projectRepo), NewProjectService(projectRepo), db
}

func TestInboxCaptureConversionAndUserIsolation(t *testing.T) {
	inbox, projects, db := inboxTestServices(t)
	ctx := context.Background()
	project, err := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("LearnOS")})
	if err != nil {
		t.Fatal(err)
	}
	content := "先整理一个较长的收集内容\n" + strings.Repeat("补充说明", 60)
	captured, err := inbox.Create(ctx, 1, content)
	if err != nil {
		t.Fatal(err)
	}
	if captured.Status != model.InboxStatusInbox || captured.SourceType != model.InboxSourceManual {
		t.Fatalf("captured item: %+v", captured)
	}
	urlItem, err := inbox.Create(ctx, 1, "https://example.test/article")
	if err != nil || urlItem.SourceType != model.InboxSourceURL || urlItem.SourceURL == "" {
		t.Fatalf("URL capture: %+v %v", urlItem, err)
	}
	converted, task, err := inbox.ConvertToTask(ctx, 1, captured.ID, InboxConvertInput{ProjectID: project.ID, Status: "next", Priority: "high"})
	if err != nil {
		t.Fatal(err)
	}
	if converted.Status != model.InboxStatusProcessed || converted.ProcessedToType != model.InboxTargetTask || converted.ProcessedToID == nil || *converted.ProcessedToID != task.ID {
		t.Fatalf("processed item: %+v", converted)
	}
	if task.Title != "先整理一个较长的收集内容" || task.Description != content || task.Status != "next" || task.Priority != "high" {
		t.Fatalf("converted task: %+v", task)
	}
	if _, _, err := inbox.ConvertToTask(ctx, 1, captured.ID, InboxConvertInput{ProjectID: project.ID}); !errors.Is(err, repository.ErrInboxAlreadyProcessed) {
		t.Fatalf("second conversion error = %v", err)
	}
	if _, err := inbox.List(ctx, 2, ""); err != nil {
		t.Fatal(err)
	} else if view, _ := inbox.List(ctx, 2, ""); len(view.Items) != 0 {
		t.Fatalf("user 2 can see user 1 inbox: %+v", view.Items)
	}
	if _, _, err := inbox.ConvertToTask(ctx, 2, urlItem.ID, InboxConvertInput{ProjectID: project.ID}); !errors.Is(err, repository.ErrInboxItemNotFound) {
		t.Fatalf("cross-user conversion error = %v", err)
	}
	var taskCount int64
	if err := db.Model(&model.ProjectTask{}).Count(&taskCount).Error; err != nil || taskCount != 1 {
		t.Fatalf("task count = %d, err = %v", taskCount, err)
	}
}

func TestInboxConversionRollsBackWhenProjectUnavailable(t *testing.T) {
	inbox, projects, db := inboxTestServices(t)
	ctx := context.Background()
	project, err := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Paused project")})
	if err != nil {
		t.Fatal(err)
	}
	if err := projects.Update(ctx, 1, project.ID, ProjectPatch{Status: ptr("paused")}); err != nil {
		t.Fatal(err)
	}
	item, err := inbox.Create(ctx, 1, "稍后转成任务")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := inbox.ConvertToTask(ctx, 1, item.ID, InboxConvertInput{ProjectID: project.ID}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("conversion error = %v", err)
	}
	unchanged, err := inbox.items.Find(ctx, 1, item.ID)
	if err != nil || unchanged.Status != model.InboxStatusInbox || unchanged.ProcessedAt != nil {
		t.Fatalf("item was not rolled back: %+v %v", unchanged, err)
	}
	var taskCount int64
	if err := db.Model(&model.ProjectTask{}).Count(&taskCount).Error; err != nil || taskCount != 0 {
		t.Fatalf("rollback task count = %d, err = %v", taskCount, err)
	}
}

func TestTodayUsesSharedActiveProjectBuckets(t *testing.T) {
	_, projects, _ := inboxTestServices(t)
	ctx := context.Background()
	active, err := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Active")})
	if err != nil {
		t.Fatal(err)
	}
	paused, err := projects.CreateProject(ctx, 1, ProjectPatch{Title: ptr("Paused")})
	if err != nil {
		t.Fatal(err)
	}
	if err := projects.Update(ctx, 1, paused.ID, ProjectPatch{Status: ptr("paused")}); err != nil {
		t.Fatal(err)
	}
	date := "2026-09-29"
	doing, err := projects.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: active.ID, Title: "进行中的任务", Status: "doing"})
	if err != nil {
		t.Fatal(err)
	}
	due, err := projects.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: active.ID, Title: "到期任务", Status: "inbox", DueDate: &date})
	if err != nil {
		t.Fatal(err)
	}
	next, err := projects.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: active.ID, Title: "下一步", Status: "next"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projects.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: paused.ID, Title: "暂停项目", Status: "doing"}); err != nil {
		t.Fatal(err)
	}
	view, err := projects.Today(ctx, 1, date, nil)
	if err != nil || len(view.Doing) != 1 || len(view.Due) != 1 || len(view.Next) != 1 {
		t.Fatalf("today buckets: %+v %v", view, err)
	}
	if view.Doing[0].ID != doing.ID || view.Due[0].ID != due.ID || view.Next[0].ID != next.ID {
		t.Fatalf("unexpected task order/buckets: %+v", view)
	}
}
