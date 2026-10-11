package service

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"learnos/internal/config"
	"learnos/internal/database"
	"learnos/internal/model"
	"learnos/internal/repository"
	"path/filepath"
	"strings"
	"testing"
)

func assetString(s string) *string { return &s }
func completeAsset(date string) AssetInput {
	return AssetInput{SnapshotDate: date, Bank: assetString("0.10"), Platform: assetString("0.20"), Investment: assetString("0"), Other: assetString("0"), Debt: assetString("1.00")}
}
func TestAssetSnapshotLifecyclePrecisionScopeAndIsolation(t *testing.T) {
	_, db := lifeTestService(t)
	s := NewAssetService(repository.NewAssetRepository(db))
	ctx := context.Background()
	ledger := model.FinanceLedger{UserID: 1, Name: "unchanged"}
	db.Create(&ledger)
	transaction := model.FinanceTransaction{LedgerID: ledger.ID, ExternalID: "source", AmountMinor: 1234, OccurredAt: "2026-01-01 00:00:00", Currency: "CNY"}
	db.Create(&transaction)
	in := completeAsset("2025-12-31")
	in.Note = "历史记录"
	a, err := s.Save(ctx, 1, 0, in)
	if err != nil || !a.Complete || *a.AssetTotal != 30 || *a.NetAsset != -70 || *a.InvestmentMinor != 0 || a.Version != 1 {
		t.Fatal(a, err)
	}
	encoded, err := json.Marshal(a)
	if err != nil || !strings.Contains(string(encoded), `"investment_minor":"0"`) || !strings.Contains(string(encoded), `"net_asset_minor":"-70"`) {
		t.Fatal(string(encoded), err)
	}
	if _, err = s.Save(ctx, 1, 0, in); !errors.Is(err, repository.ErrAssetDateExists) {
		t.Fatal("duplicate date", err)
	}
	// Copying a prior record's values creates an independent new date.
	template := in
	template.SnapshotDate = "2026-02-01"
	b, err := s.Save(ctx, 1, 0, template)
	if err != nil || b.ID == a.ID {
		t.Fatal(b, err)
	}
	createdAt := b.CreatedAt
	template.Bank = assetString("3.01")
	template.Version = b.Version
	b, err = s.Save(ctx, 1, b.ID, template)
	if err != nil || b.Version != 2 || *b.AssetTotal != 321 || *b.NetAsset != 221 || !b.CreatedAt.Equal(createdAt) {
		t.Fatal(b, err)
	}
	original, err := s.Get(ctx, 1, a.ID)
	if err != nil || *original.BankMinor != 10 || original.SnapshotDate != "2025-12-31" {
		t.Fatal("template modified history", original, err)
	}
	if _, err = s.Save(ctx, 1, b.ID, template); !errors.Is(err, repository.ErrAssetChanged) {
		t.Fatal("stale edit", err)
	}
	template.Version = b.Version
	template.SnapshotDate = a.SnapshotDate
	if _, err = s.Save(ctx, 1, b.ID, template); !errors.Is(err, repository.ErrAssetDateExists) {
		t.Fatal("date edit collision", err)
	}
	partial, err := s.Save(ctx, 1, 0, AssetInput{SnapshotDate: "2026-01-01", Bank: assetString("0"), Debt: assetString("0")})
	if err != nil || partial.Complete || partial.PlatformMinor != nil || *partial.AssetTotal != 0 || *partial.NetAsset != 0 {
		t.Fatal("zero/null distinction", partial, err)
	}
	encoded, _ = json.Marshal(partial)
	if !strings.Contains(string(encoded), `"platform_minor":null`) {
		t.Fatal(string(encoded))
	}
	unknownDebt, err := s.Save(ctx, 1, 0, AssetInput{SnapshotDate: "2026-01-02", Bank: assetString("12.34")})
	if err != nil || unknownDebt.NetAsset != nil {
		t.Fatal(unknownDebt, err)
	}
	debtOnly, err := s.Save(ctx, 1, 0, AssetInput{SnapshotDate: "2026-01-03", Debt: assetString("2")})
	if err != nil || debtOnly.NetAsset != nil || debtOnly.AssetTotal != nil {
		t.Fatal(debtOnly, err)
	}
	rows, err := s.List(ctx, 1)
	if err != nil || len(rows) != 5 || rows[0].SnapshotDate != "2026-02-01" || rows[4].SnapshotDate != "2025-12-31" {
		t.Fatal("ordering", rows, err)
	}
	if _, err = s.Get(ctx, 2, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user read", err)
	}
	if _, err = s.Save(ctx, 2, a.ID, AssetInput{SnapshotDate: a.SnapshotDate, Bank: assetString("1"), Version: a.Version}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user edit", err)
	}
	if err = s.Delete(ctx, 2, a.ID, a.Version); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-user delete", err)
	}
	if err = s.Delete(ctx, 1, b.ID, 1); !errors.Is(err, repository.ErrAssetChanged) {
		t.Fatal("stale deletion", err)
	}
	if err = s.Delete(ctx, 1, b.ID, b.Version); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, 1, b.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("delete", err)
	}
	var saved model.FinanceTransaction
	db.First(&saved, transaction.ID)
	if saved.AmountMinor != 1234 || saved.ExternalID != "source" {
		t.Fatal("ledger changed", saved)
	}
	export, err := NewExportService(db).Snapshot(ctx)
	if err != nil || len(export.AssetSnapshots) != 4 || len(export.FinanceTransactions) != 1 {
		t.Fatal("export missing assets", err)
	}
}
func TestAssetInputValidationAndAtomicFailure(t *testing.T) {
	_, db := lifeTestService(t)
	s := NewAssetService(repository.NewAssetRepository(db))
	ctx := context.Background()
	for _, input := range []AssetInput{{SnapshotDate: "2026-01-01"}, {SnapshotDate: "2026-02-30", Bank: assetString("1")}, {SnapshotDate: "2026-01-01", Bank: assetString("-0.01")}, {SnapshotDate: "2026-01-01", Bank: assetString("1.001")}, {SnapshotDate: "2026-01-01", Bank: assetString("1e3")}, {SnapshotDate: "2026-01-01", Bank: assetString("")}, {SnapshotDate: "2026-01-01", Bank: assetString("10000000000")}, {SnapshotDate: "2026-01-01", Bank: assetString("1"), Note: strings.Repeat("中", 1001)}} {
		if _, err := s.Save(ctx, 1, 0, input); !errors.Is(err, ErrAssetInput) {
			t.Fatal(input, err)
		}
	}
	in := completeAsset("2026-01-01")
	a, err := s.Save(ctx, 1, 0, in)
	if err != nil {
		t.Fatal(err)
	}
	db.Callback().Update().Before("gorm:update").Register("asset_fail", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "asset_snapshots" {
			tx.AddError(errors.New("injected"))
		}
	})
	in.Version = a.Version
	in.Bank = assetString("5")
	if _, err = s.Save(ctx, 1, a.ID, in); err == nil {
		t.Fatal("expected save error")
	}
	saved, err := s.Get(ctx, 1, a.ID)
	if err != nil || *saved.BankMinor != 10 || saved.Version != 1 {
		t.Fatal("partial edit", saved, err)
	}
	db.Callback().Update().Remove("asset_fail")
	in.Bank = assetString("9999999999.99")
	in.Platform = in.Bank
	in.Investment = in.Bank
	in.Other = in.Bank
	b, err := s.Save(ctx, 1, a.ID, in)
	if err != nil || *b.AssetTotal != 3999999999996 {
		t.Fatal("maximum precise sum", b, err)
	}
}

func TestAssetBackupRestoreRetainsSnapshotsAndLedger(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DatabasePath: filepath.Join(dir, "restore.db"), BackupDir: filepath.Join(dir, "backups"), BackupRetentionCount: 5}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := NewAssetService(repository.NewAssetRepository(db))
	ctx := context.Background()
	a, err := s.Save(ctx, 1, 0, completeAsset("2025-12-31"))
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&model.FinanceLedger{UserID: 1, Name: "kept"})
	db.Create(&model.FinanceTransaction{LedgerID: 1, ExternalID: "kept", AmountMinor: 1234})
	goal := model.LifeGoal{UserID: 1, Title: "preserved goal"}
	if err = db.Create(&goal).Error; err != nil {
		t.Fatal(err)
	}
	reference, err := s.SaveReference(ctx, 1, goal.ID, a.ID, GoalAssetInput{JudgmentDate: "2026-10-10", Note: "preserved context"}, true)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := database.CreateBackup(ctx, db, cfg.BackupDir, 5)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteReference(ctx, 1, goal.ID, a.ID, reference.Version); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, 1, a.ID, a.Version); err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.Close()
	if _, err = RestoreDatabase(cfg.DatabasePath, backup.Path, cfg.BackupDir, 5); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	conn, _ = db.DB()
	defer conn.Close()
	s = NewAssetService(repository.NewAssetRepository(db))
	restored, err := s.Get(ctx, 1, a.ID)
	if err != nil || !restored.Complete || *restored.NetAsset != -70 || restored.SnapshotDate != "2025-12-31" {
		t.Fatal(restored, err)
	}
	refs, err := s.GoalReferences(ctx, 1, goal.ID)
	if err != nil || len(refs) != 1 || refs[0].Note != reference.Note {
		t.Fatal("restore lost reference", refs, err)
	}
	var transaction model.FinanceTransaction
	if err = db.First(&transaction, 1).Error; err != nil || transaction.AmountMinor != 1234 {
		t.Fatal("restore lost ledger", transaction, err)
	}
}
