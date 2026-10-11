package service

import (
	"context"
	"learnos/internal/model"
	"testing"
)

func TestTaskArchiveLifecycleAndAtomicBatch(t *testing.T) {
	s := projectTestService(t)
	ctx := context.Background()
	p, _ := s.Create(ctx, 1, "Project")
	q, _ := s.Create(ctx, 1, "Other")
	makeTask := func(project uint, status string) *model.ProjectTask {
		t.Helper()
		task, err := s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project, Title: "task", Description: "keep", Status: status})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	a, b, c := makeTask(p.ID, "done"), makeTask(q.ID, "done"), makeTask(p.ID, "doing")
	if err := s.ArchiveTasks(ctx, 1, []uint{a.ID, c.ID}, true); err == nil {
		t.Fatal("accepted unfinished batch")
	}
	saved, _ := s.GetTask(ctx, 1, a.ID)
	if saved.ArchivedAt != nil {
		t.Fatal("partial archive")
	}
	if err := s.ArchiveTasks(ctx, 2, []uint{a.ID}, true); err == nil {
		t.Fatal("foreign archive")
	}
	if err := s.ArchiveTasks(ctx, 1, []uint{a.ID}, true); err != nil {
		t.Fatal(err)
	}
	saved, _ = s.GetTask(ctx, 1, a.ID)
	if saved.ArchivedAt == nil || saved.Status != "done" || saved.Description != a.Description || saved.ProjectID != a.ProjectID || !saved.CompletedAt.Equal(*a.CompletedAt) || !saved.ActualCompletedAt.Equal(*a.ActualCompletedAt) {
		t.Fatalf("changed facts: %+v", saved)
	}
	at := *saved.ArchivedAt
	if err := s.ArchiveTasks(ctx, 1, []uint{a.ID, b.ID}, true); err != nil {
		t.Fatal(err)
	}
	saved, _ = s.GetTask(ctx, 1, a.ID)
	if !saved.ArchivedAt.Equal(at) {
		t.Fatal("rearchive changed date")
	}
	tasks, _ := s.ListTasks(ctx, 1, nil, "done", "", 0, 0)
	if len(tasks) != 0 {
		t.Fatal("default includes archived")
	}
	tasks, _ = s.ListTasks(ctx, 1, &p.ID, "done", "", 1, 0, true)
	if len(tasks) != 1 || tasks[0].ID != a.ID {
		t.Fatal("archive project filter")
	}
	projects, _ := s.List(ctx, 1)
	for _, project := range projects {
		if project.DoneTaskCount != 1 || project.ArchivedTaskCount != 1 {
			t.Fatal("lost history counts")
		}
	}
	status := "next"
	if _, err := s.EditTask(ctx, 1, a.ID, ProjectTaskPatch{Status: &status}); err == nil {
		t.Fatal("reopened archived task")
	}
	if _, err := s.MoveTask(ctx, 1, a.ID, "next", 0, 0, nil); err == nil {
		t.Fatal("moved archived task")
	}
	if err := s.ArchiveTasks(ctx, 1, []uint{a.ID}, false); err != nil {
		t.Fatal(err)
	}
	tasks, _ = s.ListTasks(ctx, 1, &p.ID, "done", "", 0, 0)
	if len(tasks) != 1 || tasks[0].ArchivedAt != nil || !tasks[0].CompletedAt.Equal(*a.CompletedAt) {
		t.Fatal("restore lost completion")
	}
	if err := s.ArchiveTasks(ctx, 1, []uint{a.ID, a.ID}, true); err == nil {
		t.Fatal("duplicate IDs")
	}
}
