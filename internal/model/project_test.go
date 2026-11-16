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

// An empty DATE column reads back as a Go zero time; it must not reach clients
// as 0001-01-01 (or its 1901-01-01 client fallback).
func TestTaskZeroDatesReadAsUnset(t *testing.T) {
	for _, raw := range []string{"", "  ", "0001-01-01", "0001-01-01T00:00:00Z", "0001-01-01 00:00:00+00:00", "1901-01-01", "1901-01-01T00:00:00Z"} {
		value := raw
		task := ProjectTask{PlanDate: &value, DueDate: &value}
		if err := task.AfterFind(nil); err != nil || task.PlanDate != nil || task.DueDate != nil {
			t.Fatalf("%q should read as unset, got %v / %v", raw, task.PlanDate, task.DueDate)
		}
	}
	kept := "2026-11-16"
	task := ProjectTask{DueDate: &kept}
	if err := task.AfterFind(nil); err != nil || task.DueDate == nil || *task.DueDate != kept {
		t.Fatal(task, err)
	}
}
