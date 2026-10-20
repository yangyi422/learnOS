package service

import (
	"context"
	"encoding/json"
	"errors"
	"learnos/internal/model"
	"learnos/internal/repository"
	"testing"
)

func TestDailyPlanningDatesInheritanceAndUniqueBuckets(t *testing.T) {
	s := projectTestService(t)
	ctx := context.Background()
	day := "2026-10-20"
	work, _ := s.CreateProject(ctx, 1, ProjectPatch{Title: ptr("work"), Context: ptr("work")})
	life, _ := s.CreateProject(ctx, 1, ProjectPatch{Title: ptr("life"), Context: ptr("life")})
	unc, _ := s.Create(ctx, 1, "unc")
	create := func(project uint, title string, plan, due *string, status string) uint {
		t.Helper()
		task, err := s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: project, Title: title, PlanDate: plan, DueDate: due, Status: status, Priority: "high"})
		if err != nil {
			t.Fatal(err)
		}
		return task.ID
	}
	planned := create(work.ID, "主动计划且今天到期", ptr(day), ptr(day), "next")
	due := create(work.ID, "只有截止日期", nil, ptr(day), "inbox")
	overdue := create(work.ID, "长期逾期", nil, ptr("2020-01-01"), "next")
	previous := create(work.ID, "昨天计划", ptr("2026-10-19"), nil, "inbox")
	both := create(work.ID, "昨天计划也逾期", ptr("2026-10-19"), ptr("2026-10-18"), "inbox")
	create(life.ID, "生活计划", ptr(day), nil, "inbox")
	create(unc.ID, "未分类计划", ptr(day), nil, "inbox")
	done := create(work.ID, "完成计划", ptr(day), nil, "done")
	if err := s.ArchiveTasks(ctx, 1, []uint{done}, true); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"paused", "completed", "archived"} {
		p, _ := s.CreateProject(ctx, 1, ProjectPatch{Title: ptr(status), Context: ptr("work")})
		create(p.ID, status, ptr(day), nil, "doing")
		if err := s.Update(ctx, 1, p.ID, ProjectPatch{Status: &status}); err != nil {
			t.Fatal(err)
		}
	}
	p, tasks, err := s.DailyPlanning(ctx, 1, day, "all")
	if err != nil {
		t.Fatal(err)
	}
	if p.Summary != (PlanningSummary{Planned: 3, DueToday: 1, Overdue: 2, PreviousPlan: 1, Total: 7}) {
		t.Fatalf("summary %+v", p.Summary)
	}
	seen := map[uint]bool{}
	for _, group := range [][]repository.PlanningTask{p.Items, p.DueToday, p.Overdue, p.PreviousPlan} {
		for _, task := range group {
			if seen[task.ID] {
				t.Fatal("duplicate", task.ID)
			}
			seen[task.ID] = true
			if task.ProjectTitle == "" || task.EffectiveContext == "" {
				t.Fatal("missing derived metadata")
			}
		}
	}
	for _, id := range []uint{planned, due, overdue, previous, both} {
		if !seen[id] {
			t.Fatal("missing", id)
		}
	}
	for _, tc := range []struct {
		scene          string
		planned, total int
	}{{"work", 1, 5}, {"life", 1, 1}, {"all", 3, 7}} {
		view, _, e := s.DailyPlanning(ctx, 1, day, tc.scene)
		if e != nil || view.Summary.Planned != tc.planned || view.Summary.Total != tc.total {
			t.Fatal(tc, view, e)
		}
	}
	focus := focusFromPlanning(day, tasks, nil)
	if focus.TaskID != planned {
		t.Fatalf("planned before due and long overdue: %+v", focus)
	}
	// Explicit classification change and task movement are reflected without copying context.
	if err = s.Update(ctx, 1, work.ID, ProjectPatch{Context: ptr("life")}); err != nil {
		t.Fatal(err)
	}
	empty, _, _ := s.DailyPlanning(ctx, 1, day, "work")
	if empty.Summary.Total != 0 {
		t.Fatal(empty)
	}
	if _, err = s.EditTask(ctx, 1, planned, ProjectTaskPatch{ProjectID: &unc.ID}); err != nil {
		t.Fatal(err)
	}
	foreign, _, err := s.DailyPlanning(ctx, 2, day, "all")
	if err != nil || foreign.Summary.Total != 0 {
		t.Fatal("ownership", err)
	}
}

func TestPlanDatePatchNullOmittedValidationAndCompletion(t *testing.T) {
	s := projectTestService(t)
	ctx := context.Background()
	p, _ := s.Create(ctx, 1, "project")
	task, err := s.CreateTaskWithInput(ctx, 1, ProjectTaskInput{ProjectID: p.ID, Title: "task", DueDate: ptr("2026-11-01"), PlanDate: ptr("2026-10-20")})
	if err != nil {
		t.Fatal(err)
	}
	originalOrder := task.SortOrder
	patch := func(body string) error {
		var value ProjectTaskPatch
		if e := json.Unmarshal([]byte(body), &value); e != nil {
			return e
		}
		var e error
		task, e = s.EditTask(ctx, 1, task.ID, value)
		return e
	}
	if err = patch(`{"title":"renamed"}`); err != nil || task.PlanDate == nil || *task.PlanDate != "2026-10-20" {
		t.Fatal(task, err)
	}
	if err = patch(`{"plan_date":null}`); err != nil || task.PlanDate != nil {
		t.Fatal("null did not clear", task, err)
	}
	if err = patch(`{"plan_date":"2026-10-22"}`); err != nil || task.PlanDate == nil || *task.DueDate != "2026-11-01" || task.SortOrder != originalOrder {
		t.Fatal(task, err)
	}
	stale := task.UpdatedAt
	if err = patch(`{"plan_date":"2026-10-23"}`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.EditTask(ctx, 1, task.ID, ProjectTaskPatch{PlanDate: OptionalDate{Set: true, Value: nil}, ExpectedUpdatedAt: &stale}); !errors.Is(err, repository.ErrTaskConflict) {
		t.Fatal("missing conflict", err)
	}
	if _, err = s.EditTask(ctx, 2, task.ID, ProjectTaskPatch{PlanDate: OptionalDate{Set: true, Value: nil}}); err == nil {
		t.Fatal("cross-owner patch")
	}
	for _, bad := range []string{`{"plan_date":""}`, `{"plan_date":"2026-02-30"}`, `{"plan_date":"2026-1-02"}`, `{"plan_date":42}`, `{"plan_date":"2026-10-20T00:00:00Z"}`} {
		var value ProjectTaskPatch
		e := json.Unmarshal([]byte(bad), &value)
		if e == nil {
			_, e = s.EditTask(ctx, 1, task.ID, value)
		}
		if e == nil {
			t.Fatal("accepted", bad)
		}
	}
	currentVersion := task.UpdatedAt
	task, err = s.EditTask(ctx, 1, task.ID, ProjectTaskPatch{PlanDate: OptionalDate{Set: true, Value: ptr("2026-10-24")}, ExpectedUpdatedAt: &currentVersion})
	if err != nil {
		t.Fatal("valid version rejected", err)
	}
	if err = patch(`{"status":"done"}`); err != nil {
		t.Fatal(err)
	}
	completed, actual := *task.CompletedAt, *task.ActualCompletedAt
	if err = patch(`{"status":"done"}`); err != nil || !task.CompletedAt.Equal(completed) || !task.ActualCompletedAt.Equal(actual) {
		t.Fatal("completion timestamp changed", err)
	}
	if err = patch(`{"plan_date":null}`); err != nil || task.CompletedAt == nil || !task.CompletedAt.Equal(completed) {
		t.Fatal("plan clear changed completion", err)
	}
}

func TestPlanningSuggestionDoesNotPromoteLongOverdueOrFuturePlans(t *testing.T) {
	rows := []repository.PlanningTask{
		{ProjectTask: model.ProjectTask{ID: 1, ProjectID: 1, Status: "doing", Priority: "high", DueDate: ptr("2020-01-01")}},
		{ProjectTask: model.ProjectTask{ID: 2, ProjectID: 1, Status: "next", Priority: "high", DueDate: ptr("2020-01-01")}},
		{ProjectTask: model.ProjectTask{ID: 3, ProjectID: 1, Status: "doing", Priority: "high", PlanDate: ptr("2026-10-21")}},
	}
	if focus := focusFromPlanning("2026-10-20", rows, nil); focus.Kind != "empty" {
		t.Fatal(focus)
	}
	rows[0].PlanDate = ptr("2026-10-20")
	if focus := focusFromPlanning("2026-10-20", rows, nil); focus.TaskID != 1 {
		t.Fatal("explicit plan not respected", focus)
	}
}
