package service

import (
	"testing"
	"time"

	"learnos/internal/model"
)

func TestCurrentFocusUsesProjectPriorityBeforeLearning(t *testing.T) {
	projects := []model.Project{{ID: 2, Title: "LearnOS"}}
	learning := []WorkspaceLearningItem{{CourseID: 8, CourseName: "营养学", LessonTitle: "口渴是否可靠"}}
	tests := []struct {
		name  string
		today *TodayView
		want  uint
		kind  string
	}{
		{name: "doing task before due or learning", today: &TodayView{Doing: []model.ProjectTask{{ID: 11, ProjectID: 2, Title: "普通进行中", Priority: "normal"}, {ID: 12, ProjectID: 2, Title: "高优先进行中", Priority: "high"}}, Due: []model.ProjectTask{{ID: 20, ProjectID: 2, Priority: "high"}}}, want: 12, kind: "task"},
		{name: "due task before next", today: &TodayView{Due: []model.ProjectTask{{ID: 20, ProjectID: 2, Priority: "normal"}}, Next: []model.ProjectTask{{ID: 30, ProjectID: 2, Priority: "high"}}}, want: 20, kind: "task"},
		{name: "high priority next before learning", today: &TodayView{Next: []model.ProjectTask{{ID: 30, ProjectID: 2, Priority: "high"}}}, want: 30, kind: "task"},
		{name: "learning when only low priority next", today: &TodayView{Next: []model.ProjectTask{{ID: 31, ProjectID: 2, Priority: "low"}}}, want: 8, kind: "learning"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			focus := focusFromToday(test.today, projects, learning)
			if focus.Kind != test.kind {
				t.Fatalf("focus kind = %q, want %q", focus.Kind, test.kind)
			}
			if test.kind == "task" && focus.TaskID != test.want {
				t.Fatalf("task focus id = %d, want %d", focus.TaskID, test.want)
			}
			if test.kind == "learning" && focus.CourseID != test.want {
				t.Fatalf("learning focus id = %d, want %d", focus.CourseID, test.want)
			}
		})
	}

	dueSoon := "2026-09-29"
	focus := focusFromToday(&TodayView{Doing: []model.ProjectTask{
		{ID: 1, ProjectID: 2, Priority: "high", DueDate: ptr("2026-10-01"), UpdatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		{ID: 2, ProjectID: 2, Priority: "high", DueDate: &dueSoon, UpdatedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)},
	}}, projects, learning)
	if focus.TaskID != 2 {
		t.Fatalf("earlier due date should win within the same priority: %+v", focus)
	}
}
