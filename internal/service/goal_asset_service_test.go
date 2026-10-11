package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"learnos/internal/model"
	"learnos/internal/repository"
	"testing"
)

func TestGoalAssetReferencesLifecycleIsolationAndPreservation(t *testing.T) {
	life, db := lifeTestService(t)
	s := NewAssetService(repository.NewAssetRepository(db))
	ctx := context.Background()
	g, err := life.SaveGoal(ctx, 1, 0, goalInput("home"))
	if err != nil {
		t.Fatal(err)
	}
	g2, err := life.SaveGoal(ctx, 1, 0, goalInput("second"))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := life.SaveGoal(ctx, 2, 0, goalInput("foreign"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Save(ctx, 1, 0, completeAsset("2025-12-31"))
	if err != nil {
		t.Fatal(err)
	}
	partial, err := s.Save(ctx, 1, 0, AssetInput{SnapshotDate: "2026-01-01", Bank: assetString("0"), Debt: assetString("0")})
	if err != nil {
		t.Fatal(err)
	}
	input := GoalAssetInput{Note: "评估资金条件", JudgmentDate: "2026-10-10"}
	first, err := s.SaveReference(ctx, 1, g.ID, a.ID, input, true)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.SaveReference(ctx, 1, g.ID, a.ID, input, true)
	if err != nil || again.ID != first.ID {
		t.Fatal(again, err)
	}
	different := input
	different.Note = "不能覆盖"
	if _, err = s.SaveReference(ctx, 1, g.ID, a.ID, different, true); !errors.Is(err, repository.ErrLifeConflict) {
		t.Fatal(err)
	}
	if _, err = s.SaveReference(ctx, 1, g2.ID, a.ID, input, true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveReference(ctx, 1, g.ID, partial.ID, input, true); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][3]uint{{2, g.ID, a.ID}, {1, foreign.ID, a.ID}, {1, g.ID, 99999}} {
		if _, err = s.SaveReference(ctx, ids[0], ids[1], ids[2], input, true); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal("unauthorized", ids, err)
		}
	}
	if _, err = s.GoalReferences(ctx, 2, g.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	if _, err = s.ReferencingGoals(ctx, 2, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal(err)
	}
	rows, err := s.GoalReferences(ctx, 1, g.ID)
	if err != nil || len(rows) != 2 || rows[0].Snapshot.Complete || rows[0].Snapshot.PlatformMinor != nil || *rows[0].Snapshot.NetAsset != 0 {
		t.Fatal(rows, err)
	}
	goals, err := s.ReferencingGoals(ctx, 1, a.ID)
	if err != nil || len(goals) != 2 {
		t.Fatal(goals, err)
	}
	different.Version = first.Version
	updated, err := s.SaveReference(ctx, 1, g.ID, a.ID, different, false)
	if err != nil || updated.Version != 2 || !updated.CreatedAt.Equal(first.CreatedAt) {
		t.Fatal(updated, err)
	}
	if _, err = s.SaveReference(ctx, 1, g.ID, a.ID, different, false); !errors.Is(err, repository.ErrAssetChanged) {
		t.Fatal(err)
	}
	edited := completeAsset(a.SnapshotDate)
	edited.Bank = assetString("2.10")
	edited.Version = a.Version
	a, err = s.Save(ctx, 1, a.ID, edited)
	if err != nil {
		t.Fatal(err)
	}
	rows, err = s.GoalReferences(ctx, 1, g.ID)
	if err != nil || rows[1].Note != different.Note || *rows[1].Snapshot.AssetTotal != 230 {
		t.Fatal(rows, err)
	}
	if err = s.Delete(ctx, 1, a.ID, a.Version); !errors.Is(err, repository.ErrAssetReferenced) {
		t.Fatal(err)
	}
	// The database protects direct deletion too, and cleans references on a goal delete.
	if err = db.Delete(&model.AssetSnapshot{}, a.ID).Error; err == nil {
		t.Fatal("missing restrictive foreign key")
	}
	if err = db.Delete(&model.LifeGoal{}, g2.ID).Error; err != nil {
		t.Fatal(err)
	}
	goals, err = s.ReferencingGoals(ctx, 1, a.ID)
	if err != nil || len(goals) != 1 {
		t.Fatal(goals, err)
	}
	if err = s.DeleteReference(ctx, 1, g.ID, a.ID, 1); !errors.Is(err, repository.ErrAssetChanged) {
		t.Fatal(err)
	}
	if err = s.DeleteReference(ctx, 1, g.ID, a.ID, updated.Version); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, 1, a.ID); err != nil {
		t.Fatal("unlink deleted asset", err)
	}
	if err = s.Delete(ctx, 1, a.ID, a.Version); err != nil {
		t.Fatal(err)
	}
	detail, entries, _, err := life.Goal(ctx, 1, g.ID)
	if err != nil || len(entries) != 0 || detail.Status != g.Status || detail.CurrentNote != g.CurrentNote || !detail.UpdatedAt.Equal(g.UpdatedAt) {
		t.Fatal("goal changed", detail, entries, err)
	}
	export, err := NewExportService(db).Snapshot(ctx)
	if err != nil || len(export.GoalAssetReferences) != 1 {
		t.Fatal(export, err)
	}
}
func TestGoalAssetValidationAndAtomicFailure(t *testing.T) {
	life, db := lifeTestService(t)
	s := NewAssetService(repository.NewAssetRepository(db))
	ctx := context.Background()
	g, _ := life.SaveGoal(ctx, 1, 0, goalInput("home"))
	a, _ := s.Save(ctx, 1, 0, completeAsset("2026-01-01"))
	for _, in := range []GoalAssetInput{{JudgmentDate: "2026-02-30"}, {JudgmentDate: "2026-01-01", Note: string([]byte{0})}, {JudgmentDate: ""}} {
		if _, err := s.SaveReference(ctx, 1, g.ID, a.ID, in, true); !errors.Is(err, ErrGoalAssetInput) {
			t.Fatal(err)
		}
	}
	db.Exec("CREATE TRIGGER fail_reference BEFORE INSERT ON goal_asset_references BEGIN SELECT RAISE(ABORT, 'synthetic failure'); END")
	if _, err := s.SaveReference(ctx, 1, g.ID, a.ID, GoalAssetInput{JudgmentDate: "2026-10-10"}, true); err == nil {
		t.Fatal("expected failure")
	}
	rows, err := s.GoalReferences(ctx, 1, g.ID)
	if err != nil || len(rows) != 0 {
		t.Fatal("half completed", rows, err)
	}
	if _, err := s.Get(ctx, 1, a.ID); err != nil {
		t.Fatal(err)
	}
}
