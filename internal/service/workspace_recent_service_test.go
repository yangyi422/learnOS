package service

import (
	"context"
	"encoding/json"
	"learnos/internal/model"
	"learnos/internal/repository"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceRecentBoundedOwnedFactsDatesAndNoAmounts(t *testing.T) {
	_, db := lifeTestService(t)
	create := func(row interface{}) {
		t.Helper()
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	col := model.LibraryCollection{UserID: 1, Name: "合成菜谱", DisplayMode: "list", UpdatedAt: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}
	hidden := model.LibraryCollection{UserID: 2, Name: "其他用户"}
	create(&col)
	create(&hidden)
	var last model.LibraryPage
	for i := 1; i <= 5; i++ {
		last = model.LibraryPage{CollectionID: col.ID, Title: "测试页面", Body: "不应返回正文", UpdatedAt: time.Date(2026, 10, i, 0, 0, 0, 0, time.UTC)}
		create(&last)
	}
	create(&model.LibraryPage{CollectionID: hidden.ID, Title: "不可读取", UpdatedAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)})
	create(&model.LifeEvent{UserID: 1, Title: "补录旧经历", OccurredOn: "2025-12-31"})
	create(&model.LifeEvent{UserID: 1, Title: "今年经历", OccurredOn: "2026-01-01"})
	create(&model.LifeEvent{UserID: 1, Title: "最新经历", OccurredOn: "2026-10-01"})
	create(&model.LifeEvent{UserID: 2, Title: "不可读取", OccurredOn: "2027-01-01"})
	goal := model.LifeGoal{UserID: 1, Title: "买房", Status: "active"}
	create(&goal)
	otherGoal := model.LifeGoal{UserID: 2, Title: "不可读取"}
	create(&otherGoal)
	create(&model.LifeGoalEntry{GoalID: goal.ID, Content: "旧决定", OccurredOn: "2025-12-31"})
	create(&model.LifeGoalEntry{GoalID: goal.ID, Content: "真实阶段变化", OccurredOn: "2026-01-01"})
	create(&model.LifeGoalEntry{GoalID: otherGoal.ID, Content: "不可读取", OccurredOn: "2027-01-01"})
	amount := int64(987654321)
	create(&model.AssetSnapshot{UserID: 1, SnapshotDate: "2025-12-31", BankMinor: &amount, Note: "不应返回财务备注"})
	create(&model.AssetSnapshot{UserID: 1, SnapshotDate: "2026-01-01", BankMinor: &amount})
	create(&model.AssetSnapshot{UserID: 2, SnapshotDate: "2027-01-01", BankMinor: &amount})
	s := NewWorkspaceRecentService(repository.NewWorkspaceRecentRepository(db))
	view := s.Recent(context.Background(), 1)
	if len(view.Errors) > 0 || len(view.Library) != 3 || view.Library[0].ID != last.ID || len(view.Events) != 2 || view.Events[0].Title != "最新经历" || len(view.Goals) != 1 || view.Goals[0].Content != "真实阶段变化" || view.AssetDate != "2026-01-01" {
		t.Fatalf("unexpected recent view: %+v", view)
	}
	payload, _ := json.Marshal(view)
	for _, forbidden := range []string{"不可读取", "不应返回正文", "不应返回财务备注", "987654321", "bank_minor", "net_asset_minor"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("private or extra field leaked: %s", forbidden)
		}
	}
	// Refreshed projections reflect source removals without persisting another copy.
	if err := db.Delete(&last).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("goal_id = ?", goal.ID).Delete(&model.LifeGoalEntry{}).Error; err != nil {
		t.Fatal(err)
	}
	fresh := s.Recent(context.Background(), 1)
	if fresh.Library[0].ID == last.ID || len(fresh.Goals) != 0 {
		t.Fatal("deleted source persisted in recent view")
	}
	empty := s.Recent(context.Background(), 3)
	if len(empty.Library)+len(empty.Events)+len(empty.Goals) != 0 || empty.AssetDate != "" {
		t.Fatal("empty user received invented facts")
	}
}

func TestWorkspaceRecentPartialFailureDoesNotDiscardOtherSources(t *testing.T) {
	_, db := lifeTestService(t)
	if err := db.Create(&model.LifeEvent{UserID: 1, Title: "仍可显示", OccurredOn: "2026-10-01"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.LibraryPage{}); err != nil {
		t.Fatal(err)
	}
	view := NewWorkspaceRecentService(repository.NewWorkspaceRecentRepository(db)).Recent(context.Background(), 1)
	if view.Errors["library"] == "" || len(view.Events) != 1 || len(view.Errors) != 1 {
		t.Fatalf("partial failure lost other sources: %+v", view)
	}
}
