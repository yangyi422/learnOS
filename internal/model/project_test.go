package model

import "testing"

func TestTaskCalendarDatesDoNotShiftAcrossUTCOffsets(t *testing.T) {
	plan, due := "2026-10-20T00:00:00+14:00", "2026-10-21T23:00:00-08:00"
	task := ProjectTask{PlanDate: &plan, DueDate: &due}
	if err := task.AfterFind(nil); err != nil || *task.PlanDate != "2026-10-20" || *task.DueDate != "2026-10-21" {
		t.Fatal(task, err)
	}
	task = ProjectTask{}
	if err := task.AfterFind(nil); err != nil || task.PlanDate != nil || task.DueDate != nil {
		t.Fatal(task, err)
	}
}
