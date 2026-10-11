package repository

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"learnos/internal/model"
	"time"
)

var ErrFinanceConflict = errors.New("finance data changed; preview again or undo later batches first")

type FinanceRepository struct{ db *gorm.DB }

// Never emit SQL with private financial content, even on failed writes.
func NewFinanceRepository(db *gorm.DB) *FinanceRepository {
	return &FinanceRepository{db.Session(&gorm.Session{Logger: logger.Discard})}
}
func (r *FinanceRepository) Ledgers(ctx context.Context, user uint) ([]model.FinanceLedger, error) {
	a := []model.FinanceLedger{}
	e := r.db.WithContext(ctx).Where("user_id = ?", user).Order("id").Find(&a).Error
	return a, e
}
func (r *FinanceRepository) Ledger(ctx context.Context, user, id uint) (model.FinanceLedger, error) {
	var a model.FinanceLedger
	e := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, user).First(&a).Error
	return a, e
}
func (r *FinanceRepository) Transactions(ctx context.Context, user, ledger uint) ([]model.FinanceTransaction, error) {
	if _, e := r.Ledger(ctx, user, ledger); e != nil {
		return nil, e
	}
	a := []model.FinanceTransaction{}
	e := r.db.WithContext(ctx).Where("ledger_id = ?", ledger).Order("occurred_at DESC, id DESC").Find(&a).Error
	return a, e
}
func (r *FinanceRepository) Batches(ctx context.Context, user, ledger uint) ([]model.FinanceBatch, error) {
	if _, e := r.Ledger(ctx, user, ledger); e != nil {
		return nil, e
	}
	a := []model.FinanceBatch{}
	e := r.db.WithContext(ctx).Omit("payload_json").Where("ledger_id = ?", ledger).Order("id DESC").Find(&a).Error
	return a, e
}
func financeDiff(rows, old []model.FinanceTransaction, b *model.FinanceBatch) {
	m := map[string]model.FinanceTransaction{}
	for _, a := range old {
		m[a.ExternalID] = a
	}
	seen := map[string]bool{}
	for _, a := range rows {
		seen[a.ExternalID] = true
		if p, ok := m[a.ExternalID]; !ok {
			b.Added++
		} else if p.ContentHash != a.ContentHash {
			b.Updated++
		} else {
			b.Unchanged++
		}
	}
	if b.CompleteScope {
		for _, a := range old {
			d := a.OccurredAt[:10]
			if d >= b.ScopeFrom && d <= b.ScopeTo && !seen[a.ExternalID] {
				b.Missing++
			}
		}
	}
}
func (r *FinanceRepository) Preview(ctx context.Context, user uint, name string, rows []model.FinanceTransaction, b model.FinanceBatch) (model.FinanceBatch, error) {
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		l := model.FinanceLedger{UserID: user, Name: name}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&l).Error; e != nil {
			return e
		}
		if e := tx.Where("user_id = ? AND name = ?", user, name).First(&l).Error; e != nil {
			return e
		}
		old := []model.FinanceTransaction{}
		if e := tx.Where("ledger_id = ?", l.ID).Find(&old).Error; e != nil {
			return e
		}
		b.LedgerID = l.ID
		b.BaseRevision = l.Revision
		b.Status = "preview"
		b.RowCount = len(rows)
		financeDiff(rows, old, &b)
		stored := []financeStoredRow{}
		for _, a := range rows {
			stored = append(stored, financeStoredRow{a, a.RawJSON, a.TagsJSON, a.ContentHash})
		}
		payload, e := json.Marshal(stored)
		if e != nil {
			return e
		}
		b.PayloadJSON = string(payload)
		return tx.Create(&b).Error
	})
	return b, e
}
func batchOwned(tx *gorm.DB, user, id uint) (model.FinanceBatch, model.FinanceLedger, error) {
	var b model.FinanceBatch
	var l model.FinanceLedger
	if e := tx.First(&b, id).Error; e != nil {
		return b, l, e
	}
	e := tx.Where("id = ? AND user_id = ?", b.LedgerID, user).First(&l).Error
	return b, l, e
}
func bump(tx *gorm.DB, l model.FinanceLedger) error {
	q := tx.Model(&model.FinanceLedger{}).Where("id = ? AND revision = ?", l.ID, l.Revision).Updates(map[string]interface{}{"revision": l.Revision + 1, "updated_at": time.Now().UTC()})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected != 1 {
		return ErrFinanceConflict
	}
	return nil
}

type financeStoredRow struct {
	Transaction model.FinanceTransaction
	RawJSON     string
	TagsJSON    string
	ContentHash string
}

func snapshot(a model.FinanceTransaction) string {
	b, _ := json.Marshal(financeStoredRow{a, a.RawJSON, a.TagsJSON, a.ContentHash})
	return string(b)
}
func restoreSnapshot(value string) (model.FinanceTransaction, error) {
	var s financeStoredRow
	e := json.Unmarshal([]byte(value), &s)
	a := s.Transaction
	a.RawJSON = s.RawJSON
	a.TagsJSON = s.TagsJSON
	a.ContentHash = s.ContentHash
	return a, e
}
func (r *FinanceRepository) Import(ctx context.Context, user, id uint, remove bool) (model.FinanceBatch, error) {
	var result model.FinanceBatch
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		b, l, e := batchOwned(tx, user, id)
		if e != nil {
			return e
		}
		result = b
		if b.Status == "imported" {
			return nil
		}
		if b.Status != "preview" || l.Revision != b.BaseRevision || time.Since(b.CreatedAt) > 24*time.Hour {
			return ErrFinanceConflict
		}
		if remove && !b.CompleteScope {
			return ErrFinanceConflict
		}
		if e = bump(tx, l); e != nil {
			return e
		}
		var stored []financeStoredRow
		if e = json.Unmarshal([]byte(b.PayloadJSON), &stored); e != nil {
			return e
		}
		rows := []model.FinanceTransaction{}
		for _, s := range stored {
			a := s.Transaction
			a.RawJSON = s.RawJSON
			a.TagsJSON = s.TagsJSON
			a.ContentHash = s.ContentHash
			rows = append(rows, a)
		}
		old := []model.FinanceTransaction{}
		if e = tx.Where("ledger_id = ?", l.ID).Find(&old).Error; e != nil {
			return e
		}
		m := map[string]model.FinanceTransaction{}
		for _, a := range old {
			m[a.ExternalID] = a
		}
		seen := map[string]bool{}
		for _, a := range rows {
			seen[a.ExternalID] = true
			a.LedgerID = l.ID
			a.BatchID = b.ID
			prev, ok := m[a.ExternalID]
			if ok && prev.ContentHash == a.ContentHash {
				continue
			}
			change := model.FinanceChange{BatchID: b.ID}
			if ok {
				change.BeforeJSON = snapshot(prev)
				a.ID = prev.ID
				a.CreatedAt = prev.CreatedAt
			}
			if e = tx.Save(&a).Error; e != nil {
				return e
			}
			change.TransactionID = a.ID
			change.AfterJSON = snapshot(a)
			if e = tx.Create(&change).Error; e != nil {
				return e
			}
		}
		if remove {
			for _, a := range old {
				d := a.OccurredAt[:10]
				if d < b.ScopeFrom || d > b.ScopeTo || seen[a.ExternalID] {
					continue
				}
				change := model.FinanceChange{BatchID: b.ID, TransactionID: a.ID, BeforeJSON: snapshot(a)}
				if e = tx.Create(&change).Error; e != nil {
					return e
				}
				if e = tx.Delete(&a).Error; e != nil {
					return e
				}
				b.Removed++
			}
		}
		now := time.Now().UTC()
		b.Status = "imported"
		b.RemoveMissing = remove
		b.ImportedAt = &now
		b.PayloadJSON = ""
		if e = tx.Save(&b).Error; e != nil {
			return e
		}
		result = b
		return nil
	})
	return result, e
}
func (r *FinanceRepository) Undo(ctx context.Context, user, id uint) (model.FinanceBatch, error) {
	var result model.FinanceBatch
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		b, l, e := batchOwned(tx, user, id)
		if e != nil {
			return e
		}
		result = b
		if b.Status == "undone" {
			return nil
		}
		if b.Status != "imported" {
			return ErrFinanceConflict
		}
		var latest model.FinanceBatch
		if e = tx.Where("ledger_id = ? AND status = ?", l.ID, "imported").Order("id DESC").First(&latest).Error; e != nil {
			return e
		}
		if latest.ID != id {
			return ErrFinanceConflict
		}
		if e = bump(tx, l); e != nil {
			return e
		}
		changes := []model.FinanceChange{}
		if e = tx.Where("batch_id = ?", id).Order("id DESC").Find(&changes).Error; e != nil {
			return e
		}
		for _, c := range changes {
			if c.BeforeJSON == "" {
				if e = tx.Delete(&model.FinanceTransaction{}, c.TransactionID).Error; e != nil {
					return e
				}
			} else {
				a, decodeErr := restoreSnapshot(c.BeforeJSON)
				if decodeErr != nil {
					return decodeErr
				}
				savedAt := a.UpdatedAt
				if e = tx.Save(&a).Error; e != nil {
					return e
				}
				if e = tx.Model(&model.FinanceTransaction{}).Where("id = ?", a.ID).UpdateColumn("updated_at", savedAt).Error; e != nil {
					return e
				}
			}
		}
		now := time.Now().UTC()
		b.Status = "undone"
		b.UndoneAt = &now
		if e = tx.Save(&b).Error; e != nil {
			return e
		}
		result = b
		return nil
	})
	return result, e
}

// Lists page at the database, with exact tag membership (not substring matching).
type FinanceQuery struct {
	From, To, Currency, Category, Subcategory, Tag, Kind, Keyword string
	CategoryEmpty, SubcategoryEmpty                               bool
	Page, PageSize                                                int
}

func (r *FinanceRepository) Page(ctx context.Context, user, ledger uint, f FinanceQuery) ([]model.FinanceTransaction, int, error) {
	if _, e := r.Ledger(ctx, user, ledger); e != nil {
		return nil, 0, e
	}
	q := r.db.WithContext(ctx).Model(&model.FinanceTransaction{}).Where("ledger_id = ?", ledger)
	if f.From != "" {
		q = q.Where("occurred_at >= ?", f.From+" 00:00:00")
	}
	if f.To != "" {
		q = q.Where("occurred_at <= ?", f.To+" 23:59:59")
	}
	for key, value := range map[string]string{"currency": f.Currency, "category": f.Category, "subcategory": f.Subcategory, "kind": f.Kind} {
		if value != "" {
			q = q.Where(key+" = ?", value)
		}
	}
	if f.CategoryEmpty {
		q = q.Where("category = ?", "")
	}
	if f.SubcategoryEmpty {
		q = q.Where("subcategory = ?", "")
	}
	if f.Tag != "" {
		q = q.Where("EXISTS (SELECT 1 FROM json_each(finance_transactions.tags_json) WHERE json_each.value = ?)", f.Tag)
	}
	if f.Keyword != "" {
		q = q.Where("instr(lower(note), lower(?)) > 0", f.Keyword)
	}
	var total int64
	if e := q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	rows := []model.FinanceTransaction{}
	e := q.Omit("raw_json").Order("occurred_at DESC, id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&rows).Error
	return rows, int(total), e
}
func (r *FinanceRepository) Transaction(ctx context.Context, user, ledger, id uint) (model.FinanceTransaction, error) {
	var a model.FinanceTransaction
	if _, e := r.Ledger(ctx, user, ledger); e != nil {
		return a, e
	}
	e := r.db.WithContext(ctx).Where("ledger_id = ? AND id = ?", ledger, id).First(&a).Error
	return a, e
}

// Export includes private raw data only in the authenticated download, never logs.
type FinanceExportRow = financeStoredRow

func NewFinanceExportRow(a model.FinanceTransaction) FinanceExportRow {
	return financeStoredRow{a, a.RawJSON, a.TagsJSON, a.ContentHash}
}
