package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"learnos/internal/model"
	"learnos/internal/repository"
	"os"
	"testing"
)

func financeCSV(rows ...[]string) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(FinanceHeaders)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
	return b.Bytes()
}
func financeRow(id, date, kind, amount, category, sub, tags, note string) []string {
	r := make([]string, 18)
	r[0] = id
	r[1] = date
	r[2] = category
	r[3] = sub
	r[4] = kind
	r[5] = amount
	r[6] = "CNY"
	r[9] = note
	r[15] = tags
	return r
}
func financeTestService(t *testing.T) (*FinanceService, *gorm.DB) {
	_, db := lifeTestService(t)
	return NewFinanceService(repository.NewFinanceRepository(db)), db
}
func TestFinanceCSVValidation(t *testing.T) {
	valid := financeRow("x", "2026-01-02 12:03", "支出", "123.45", "餐饮", "午餐", "#生活#工作#生活", "逗号,引号\"及\n换行")
	p := ParseFinanceCSV(append([]byte{0xef, 0xbb, 0xbf}, financeCSV(valid)...))
	if len(p.Errors) != 0 || len(p.Rows) != 1 || p.Rows[0].AmountMinor != 12345 || p.Rows[0].Note != valid[9] || p.Rows[0].TagsJSON != `["生活","工作"]` {
		t.Fatalf("parse %+v", p)
	}
	cases := map[string][]byte{"empty": nil, "only header": financeCSV(), "bad utf8": {255}, "wrong header": []byte("ID,时间\nx,y\n"), "broken quoting": []byte("ID,时间,分类,二级分类,类型,金额,币种,账户1,账户2,备注,已报销,手续费,优惠券,记账者,账单标记,标签,账单图片,关联账单\n\"oops"), "duplicate": financeCSV(valid, valid), "oversize": make([]byte, FinanceMaxBytes+1)}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if len(ParseFinanceCSV(data).Errors) == 0 {
				t.Fatal("accepted invalid CSV")
			}
		})
	}
	for _, bad := range []struct {
		col   int
		value string
	}{{0, ""}, {1, "2026-02-30 12:00"}, {4, "未知"}, {5, "NaN"}, {5, "1.001"}, {5, "1e3"}, {6, ""}, {11, "oops"}, {12, "1.000"}} {
		r := append([]string{}, valid...)
		r[bad.col] = bad.value
		if p := ParseFinanceCSV(financeCSV(r)); len(p.Errors) == 0 || p.Errors[0].Line != 2 {
			t.Fatalf("invalid column %d accepted %+v", bad.col, p)
		}
	}
	for text, want := range map[string]int64{"0": 0, "0.1": 10, "-0.01": -1, "12.34": 1234, "9999999999.99": 999999999999} {
		got, e := FinanceMoney(text)
		if e != nil || got != want {
			t.Fatal(text, got, e)
		}
	}
}
func TestFinanceIncrementalUndoAndIsolation(t *testing.T) {
	s, db := financeTestService(t)
	ctx := context.Background()
	a := financeRow("a", "2026-01-02", "支出", "0.10", "餐饮", "", "#日常#工作", "早餐,\n备注")
	b := financeRow("b", "2026-02-03", "收入", "200.20", "薪资", "", "", "工资")
	c := financeRow("c", "2026-02-04", "支出", "0.20", "餐饮", "午餐", "#日常", "午餐")
	input := FinancePreviewInput{LedgerName: "个人", Filename: "sample.csv"}
	preview := func(rows ...[]string) FinancePreview {
		t.Helper()
		p, e := s.Preview(ctx, 1, input, financeCSV(rows...))
		if e != nil || p.Batch == nil {
			t.Fatalf("preview %+v %v", p, e)
		}
		return p
	}
	apply := func(p FinancePreview) model.FinanceBatch {
		t.Helper()
		r, e := s.Import(ctx, 1, p.Batch.ID, false, false, false)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	p := preview(a, b, c)
	first := apply(p)
	if first.Added != 3 {
		t.Fatal(first)
	}
	same := apply(p)
	if same.ID != first.ID {
		t.Fatal("replay batch")
	}
	var count int64
	db.Model(&model.FinanceTransaction{}).Count(&count)
	if count != 3 {
		t.Fatal(count)
	}
	p = preview(a, b, c)
	if p.Batch.Unchanged != 3 || p.Batch.Added != 0 {
		t.Fatal(p.Batch)
	}
	second := apply(p)
	f := FinanceFilter{Page: 1, PageSize: 30}
	sum, e := s.Summary(ctx, 1, first.LedgerID, f)
	if e != nil || sum.Income != 20020 || sum.Expense != 30 || sum.Difference != 19990 || len(sum.Tags) != 2 {
		t.Fatal(sum, e)
	}
	f.Tag = "日常"
	page, e := s.Transactions(ctx, 1, first.LedgerID, f)
	if e != nil || page.Total != 2 {
		t.Fatal(page, e)
	}
	f.Tag = "日"
	page, e = s.Transactions(ctx, 1, first.LedgerID, f)
	if e != nil || page.Total != 0 {
		t.Fatal("substring tag", page, e)
	}
	f.Tag = ""
	f.From = "2026-02-01"
	f.Kind = "expense"
	f.Keyword = "午餐"
	page, e = s.Transactions(ctx, 1, first.LedgerID, f)
	if e != nil || page.Total != 1 {
		t.Fatal(page, e)
	}
	if _, e = s.Summary(ctx, 999, first.LedgerID, f); !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal(e)
	}
	if _, e = s.Import(ctx, 999, first.ID, false, false, false); !errors.Is(e, gorm.ErrRecordNotFound) {
		t.Fatal(e)
	}
	original := p.Sample[0]
	_ = original
	a[5] = "12.34"
	a[9] = "新备注"
	updated := apply(preview(a, b, c))
	if updated.Updated != 1 {
		t.Fatal(updated)
	}
	if _, e = s.Undo(ctx, 1, second.ID); !errors.Is(e, repository.ErrFinanceConflict) {
		t.Fatal("out-of-order undo", e)
	}
	if _, e = s.Undo(ctx, 1, updated.ID); e != nil {
		t.Fatal(e)
	}
	rows, e := s.repo.Transactions(ctx, 1, first.LedgerID)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range rows {
		if row.ExternalID == "a" && (row.AmountMinor != 10 || row.Note != "早餐,\n备注" || row.TagsJSON != `["日常","工作"]` || row.RawJSON == "" || row.ContentHash == "") {
			t.Fatalf("incomplete undo %+v", row)
		}
	}
	if _, e = s.Undo(ctx, 1, updated.ID); e != nil {
		t.Fatal("undo replay", e)
	}
	input.Complete = true
	input.From = "2026-01-01"
	input.To = "2026-12-31"
	missing := preview(b, c)
	if missing.Batch.Missing != 1 {
		t.Fatal(missing.Batch)
	}
	if _, e = s.Import(ctx, 1, missing.Batch.ID, true, false, true); !errors.Is(e, ErrFinanceInput) {
		t.Fatal("missing confirmation", e)
	}
	removed, e := s.Import(ctx, 1, missing.Batch.ID, true, true, true)
	if e != nil || removed.Removed != 1 {
		t.Fatal(removed, e)
	}
	rows, _ = s.repo.Transactions(ctx, 1, first.LedgerID)
	if len(rows) != 2 {
		t.Fatal(len(rows))
	}
	if _, e = s.Undo(ctx, 1, removed.ID); e != nil {
		t.Fatal(e)
	}
	rows, _ = s.repo.Transactions(ctx, 1, first.LedgerID)
	if len(rows) != 3 {
		t.Fatal("undo removed", len(rows))
	}
	input.Complete = false
	stale := preview(b)
	fresh := apply(preview(a, b, c))
	if _, e = s.Import(ctx, 1, stale.Batch.ID, false, false, false); !errors.Is(e, repository.ErrFinanceConflict) {
		t.Fatal("stale preview", e)
	}
	if _, e = s.Undo(ctx, 1, fresh.ID); e != nil {
		t.Fatal(e)
	}
	// No-op later batches can be undone without destroying previously imported data.
	if _, e = s.Undo(ctx, 1, second.ID); e != nil {
		t.Fatal(e)
	}
	rows, _ = s.repo.Transactions(ctx, 1, first.LedgerID)
	if len(rows) != 3 {
		t.Fatal(len(rows))
	}
	if _, e = s.Undo(ctx, 1, first.ID); e != nil {
		t.Fatal(e)
	}
	rows, _ = s.repo.Transactions(ctx, 1, first.LedgerID)
	if len(rows) != 0 {
		t.Fatal(len(rows))
	}
}
func TestFinanceWriteFailureRollsBack(t *testing.T) {
	s, db := financeTestService(t)
	ctx := context.Background()
	r := financeRow("a", "2026-01-01", "支出", "12.34", "餐饮", "", "#日常", "private")
	p, e := s.Preview(ctx, 1, FinancePreviewInput{LedgerName: "个人", Filename: "x.csv"}, financeCSV(r))
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Callback().Create().Before("gorm:create").Register("finance_fail", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "finance_changes" {
			tx.AddError(errors.New("injected failure"))
		}
	}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Import(ctx, 1, p.Batch.ID, false, false, false); e == nil {
		t.Fatal("expected error")
	}
	var count int64
	db.Model(&model.FinanceTransaction{}).Count(&count)
	if count != 0 {
		t.Fatal("partial transactions", count)
	}
	l, _ := s.repo.Ledger(ctx, 1, p.Batch.LedgerID)
	if l.Revision != 0 {
		t.Fatal("revision not rolled back", l)
	}
	db.Callback().Create().Remove("finance_fail")
	if _, e = s.Import(ctx, 1, p.Batch.ID, false, false, false); e != nil {
		t.Fatal("retry", e)
	}
	if e = db.Callback().Delete().Before("gorm:delete").Register("finance_undo_fail", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "finance_transactions" {
			tx.AddError(errors.New("injected undo failure"))
		}
	}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Undo(ctx, 1, p.Batch.ID); e == nil {
		t.Fatal("expected undo error")
	}
	db.Model(&model.FinanceTransaction{}).Count(&count)
	if count != 1 {
		t.Fatal("partial undo", count)
	}
	db.Callback().Delete().Remove("finance_undo_fail")
}
func TestFinanceMixedCurrenciesAndRefunds(t *testing.T) {
	s, _ := financeTestService(t)
	ctx := context.Background()
	a := financeRow("a", "2026-01-01", "支出", "12.34", "购物", "", "", "消费")
	b := financeRow("b", "2026-01-02", "支出", "-2.34", "购物", "", "", "退款")
	c := financeRow("c", "2026-01-02", "转账", "100", "转账", "", "", "transfer")
	d := financeRow("d", "2026-01-02", "支出", "5", "购物", "", "", "usd")
	d[6] = "USD"
	p, e := s.Preview(ctx, 1, FinancePreviewInput{LedgerName: "个人", Filename: "x.csv"}, financeCSV(a, b, c, d))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Import(ctx, 1, p.Batch.ID, false, false, false)
	if e != nil {
		t.Fatal(e)
	}
	f := FinanceFilter{Page: 1, PageSize: 30}
	sum, e := s.Summary(ctx, 1, p.Batch.LedgerID, f)
	if e != nil || sum.Currency != "" || sum.Count != 0 || len(sum.Currencies) != 2 {
		t.Fatal("mixed", sum, e)
	}
	f.Currency = "CNY"
	sum, e = s.Summary(ctx, 1, p.Batch.LedgerID, f)
	if e != nil || sum.Expense != 1000 || sum.Income != 0 || sum.Count != 3 {
		t.Fatal(sum, e)
	}
}

// Optional local-only sample: no private ledger content is written into the repo.
func TestFinanceRealSample(t *testing.T) {
	path := os.Getenv("LIFEOS_QIANJI_SAMPLE")
	if path == "" {
		t.Skip("real CSV path not provided")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	p := ParseFinanceCSV(data)
	if len(p.Errors) > 0 || p.Total != 1340 || p.Income != 16 || p.Expense != 1324 {
		t.Fatalf("counts total=%d income=%d expense=%d errors=%v", p.Total, p.Income, p.Expense, p.Errors)
	}
	var income, expense int64
	for _, a := range p.Rows {
		if a.Currency != "CNY" {
			t.Fatal("unexpected currency")
		}
		if a.Kind == "income" {
			income += a.AmountMinor
		} else if a.Kind == "expense" {
			expense += a.AmountMinor
		}
	}
	s, _ := financeTestService(t)
	ctx := context.Background()
	preview, e := s.Preview(ctx, 1, FinancePreviewInput{LedgerName: "local verification", Filename: "private.csv"}, data)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if _, e = s.Import(ctx, 1, preview.Batch.ID, false, false, false); e != nil {
			t.Fatal(e)
		}
	}
	repeat, e := s.Preview(ctx, 1, FinancePreviewInput{LedgerName: "local verification", Filename: "private.csv"}, data)
	if e != nil || repeat.Batch.Unchanged != 1340 {
		t.Fatal("repeat mismatch", e)
	}
	f := FinanceFilter{Page: 1, PageSize: 30, Currency: "CNY"}
	sum, e := s.Summary(ctx, 1, preview.Batch.LedgerID, f)
	if e != nil || sum.Count != 1340 || sum.Income != income || sum.Expense != expense {
		t.Fatal("aggregate mismatch", e)
	}
	t.Log(fmt.Sprintf("records=1340 income_count=16 expense_count=1324; exact sums and repeated import verified"))
}

func TestFinanceUncategorizedPaginationAndExport(t *testing.T) {
	s, db := financeTestService(t)
	ctx := context.Background()
	rows := [][]string{}
	for i := 0; i < 35; i++ {
		rows = append(rows, financeRow(fmt.Sprint(i), "2026-01-01", "支出", "0.01", "", "", "#日常", "原始"))
	}
	rows = append(rows, financeRow("other", "2026-01-01", "支出", "1.00", "其他", "二级", "", "其他"))
	p, e := s.Preview(ctx, 1, FinancePreviewInput{LedgerName: "个人", Filename: "test.csv"}, financeCSV(rows...))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Import(ctx, 1, p.Batch.ID, false, false, false); e != nil {
		t.Fatal(e)
	}
	f := FinanceFilter{Page: 2, PageSize: 30, CategoryEmpty: true, SubcategoryEmpty: true}
	list, e := s.Transactions(ctx, 1, p.Batch.LedgerID, f)
	if e != nil || list.Total != 35 || len(list.Items) != 5 {
		t.Fatal(list, e)
	}
	sum, e := s.Summary(ctx, 1, p.Batch.LedgerID, f)
	if e != nil || sum.Expense != 35 {
		t.Fatal(sum, e)
	}
	export := NewExportService(db)
	snapshot, e := export.Snapshot(ctx)
	if e != nil || len(snapshot.FinanceTransactions) != 36 || len(snapshot.FinanceChanges) != 36 {
		t.Fatal("export count", e)
	}
	if snapshot.FinanceTransactions[0].RawJSON == "" || snapshot.FinanceTransactions[0].TagsJSON == "" {
		t.Fatal("export lost raw fields")
	}
}

// Source amounts, monthly/category drill-down and multi-tag totals stay independent.
func TestFinanceSecondPassAccuracy(t *testing.T) {
	s, _ := financeTestService(t)
	ctx := context.Background()
	input := FinancePreviewInput{LedgerName: "synthetic", Filename: "synthetic.csv"}
	rows := [][]string{
		financeRow("one", "2026-01-01", "支出", "0.10", "餐饮", "早餐", "#生活#工作#生活", "报销退款字样不改变类型"),
		financeRow("two", "2026-01-31 23:59:59", "支出", "0.20", "餐饮", "午餐", "#生活", ""),
		financeRow("three", "2026-02-01", "收入", "1.00", "收入", "", "", ""),
	}
	importRows := func(input FinancePreviewInput, rows ...[]string) model.FinanceBatch {
		t.Helper()
		p, err := s.Preview(ctx, 1, input, financeCSV(rows...))
		if err != nil || p.Batch == nil {
			t.Fatalf("preview failed: %v", err)
		}
		b, err := s.Import(ctx, 1, p.Batch.ID, false, false, false)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	first := importRows(input, rows...)
	repeated := importRows(input, rows...)
	if repeated.Added != 0 || repeated.Unchanged != 3 {
		t.Fatal("duplicate import", repeated)
	}
	// A partial export updates the existing ID, keeping all omitted history.
	changed := append([]string{}, rows[0]...)
	changed[5] = "0.11"
	b := importRows(input, changed)
	if b.Added != 0 || b.Updated != 1 {
		t.Fatal("update became insert", b)
	}
	f := FinanceFilter{Page: 1, PageSize: 30, Currency: "CNY"}
	sum, err := s.Summary(ctx, 1, first.LedgerID, f)
	if err != nil || sum.Count != 3 || sum.Expense != 31 || sum.Income != 100 || sum.Difference != 69 || sum.From != "2026-01-01" || sum.To != "2026-02-01" {
		t.Fatal(sum, err)
	}
	if len(sum.Tags) != 2 || sum.Tags[0].Amount != 31 || sum.Tags[1].Amount != 11 {
		t.Fatal("tags changed expense denominator", sum.Tags)
	}
	f.From, f.To, f.Kind, f.Category = "2026-01-01", "2026-01-31", "expense", "餐饮"
	category, err := s.Summary(ctx, 1, first.LedgerID, f)
	list, listErr := s.Transactions(ctx, 1, first.LedgerID, f)
	var exact int64
	for _, row := range list.Items {
		exact += row.AmountMinor
	}
	if err != nil || listErr != nil || list.Total != 2 || exact != 31 || category.Expense != exact || len(category.Months) != 1 || category.Months[0].Expense != exact {
		t.Fatal("month/category mismatch", category, list, err, listErr)
	}
	for _, child := range category.Categories[0].Children {
		f.Subcategory = child.Name
		page, err := s.Transactions(ctx, 1, first.LedgerID, f)
		var amount int64
		for _, row := range page.Items {
			amount += row.AmountMinor
		}
		if err != nil || page.Total != child.Count || amount != child.Amount {
			t.Fatal("subcategory mismatch", child, page, err)
		}
	}
	// Identical source IDs in another ledger must stay independent.
	input.LedgerName = "another synthetic ledger"
	other := importRows(input, rows[0])
	if other.Added != 1 || other.LedgerID == first.LedgerID {
		t.Fatal("ledger isolation", other)
	}
}
