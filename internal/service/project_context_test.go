package service

import (
	"context"
	"fmt"
	"testing"

	"learnos/internal/model"
)

func TestProjectContextInheritanceAndFiltering(t *testing.T) {
	s := projectTestService(t)
	ctx := context.Background()
	legacy, err := s.Create(ctx, 1, "历史项目")
	if err != nil || legacy.Context != "unclassified" {
		t.Fatalf("default: %+v %v", legacy, err)
	}
	var work *model.Project
	var workTask *model.ProjectTask
	// Put work after more than five other projects/tasks: filtering must precede limits.
	for i := 0; i < 8; i++ {
		p, e := s.CreateProject(ctx, 1, ProjectPatch{Title: ptr(fmt.Sprint(i)), Context: ptr("life")})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: p.ID, Title: "生活", Status: "doing"}); e != nil {
			t.Fatal(e)
		}
	}
	work, err = s.CreateProject(ctx, 1, ProjectPatch{Title: ptr("工作"), Context: ptr("work")})
	if err != nil {
		t.Fatal(err)
	}
	workTask, err = s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: work.ID, Title: "工作事项", Status: "doing"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: legacy.ID, Title: "未分类", Status: "doing"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		scene string
		count int
	}{{"all", 10}, {"work", 1}, {"life", 8}} {
		projects, e := s.List(ctx, 1, tc.scene)
		if e != nil || len(projects) != tc.count {
			t.Fatalf("projects %s: %d %v", tc.scene, len(projects), e)
		}
		today, e := s.Today(ctx, 1, "2026-10-20", nil, tc.scene)
		if e != nil || len(today.Doing) != tc.count {
			t.Fatalf("today %s: %+v %v", tc.scene, today, e)
		}
	}
	tasks, err := s.projects.ListTasksInContext(ctx, 1, nil, "doing", "", 1, 0, false, "work")
	if err != nil || len(tasks) != 1 || tasks[0].ID != workTask.ID {
		t.Fatalf("filter before limit: %+v %v", tasks, err)
	}
	tasks, err = s.projects.ListTasksInContext(ctx, 1, nil, "doing", "", 1, 1, false, "work")
	if err != nil || len(tasks) != 0 {
		t.Fatalf("filter before offset: %+v %v", tasks, err)
	}
	if err = s.Update(ctx, 1, work.ID, ProjectPatch{Context: ptr("life")}); err != nil {
		t.Fatal(err)
	}
	today, _ := s.Today(ctx, 1, "2026-10-20", nil, "work")
	if len(today.Doing) != 0 {
		t.Fatal("task remained in old context")
	}
	after, _ := s.GetTask(ctx, 1, workTask.ID)
	if after.ID != workTask.ID || after.SortOrder != workTask.SortOrder || after.Status != workTask.Status || !after.UpdatedAt.Equal(workTask.UpdatedAt) {
		t.Fatal("classification changed task facts")
	}
	if err = s.Update(ctx, 1, work.ID, ProjectPatch{Status: ptr("archived")}); err != nil {
		t.Fatal(err)
	}
	today, _ = s.Today(ctx, 1, "2026-10-20", nil, "life")
	if len(today.Doing) != 8 {
		t.Fatal("archived project included")
	}
	// An explicit project query retains the existing archive inspection behavior.
	rows, err := s.projects.ListTasksInContext(ctx, 1, &work.ID, "", "", 0, 0, false, "life")
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	if _, err = s.CreateProject(ctx, 1, ProjectPatch{Title: ptr("bad"), Context: ptr("all")}); err == nil {
		t.Fatal("accepted invalid context")
	}
	if err = s.Update(ctx, 1, legacy.ID, ProjectPatch{Context: ptr("invalid")}); err == nil {
		t.Fatal("accepted invalid patch")
	}
	rows, err = s.projects.ListTasksInContext(ctx, 2, nil, "", "", 0, 0, false, "life")
	if err != nil || len(rows) != 0 {
		t.Fatal("ownership leaked", err)
	}
}
