package service

import (
	"context"
	"encoding/json"
	"learnos/internal/model"
	"learnos/internal/repository"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FinanceService struct{ repo *repository.FinanceRepository }

func NewFinanceService(r *repository.FinanceRepository) *FinanceService { return &FinanceService{r} }

type FinancePreviewInput struct {
	LedgerName string
	Filename   string
	From       string
	To         string
	Complete   bool
}
type FinancePreview struct {
	Parse    FinanceParse             `json:"parse"`
	Batch    *model.FinanceBatch      `json:"batch"`
	Sample   []FinanceTransactionView `json:"sample"`
	Warnings []string                 `json:"warnings"`
}
type FinanceTransactionView struct {
	model.FinanceTransaction
	Tags   []string          `json:"tags"`
	Raw    map[string]string `json:"raw,omitempty"`
	Source string            `json:"source"`
}

func transactionView(a model.FinanceTransaction, name string, raw bool) FinanceTransactionView {
	v := FinanceTransactionView{FinanceTransaction: a, Tags: []string{}, Source: name}
	_ = json.Unmarshal([]byte(a.TagsJSON), &v.Tags)
	if raw {
		_ = json.Unmarshal([]byte(a.RawJSON), &v.Raw)
	}
	return v
}
func (s *FinanceService) Preview(ctx context.Context, user uint, input FinancePreviewInput, data []byte) (FinancePreview, error) {
	name := strings.TrimSpace(input.LedgerName)
	if name == "" || len(name) > 120 || len(input.Filename) > 255 {
		return FinancePreview{}, ErrFinanceInput
	}
	p := ParseFinanceCSV(data)
	out := FinancePreview{Parse: p, Sample: []FinanceTransactionView{}, Warnings: []string{}}
	if len(p.Errors) > 0 {
		return out, nil
	}
	if input.Complete && (!financeDate(input.From) || !financeDate(input.To) || input.From > input.To || p.From < input.From || p.To > input.To) {
		return out, ErrFinanceInput
	}
	b, e := s.repo.Preview(ctx, user, name, p.Rows, model.FinanceBatch{Filename: input.Filename, FileHash: p.FileHash, ScopeFrom: input.From, ScopeTo: input.To, CompleteScope: input.Complete})
	if e != nil {
		return out, e
	}
	out.Batch = &b
	for i, a := range p.Rows {
		if i == 5 {
			break
		}
		out.Sample = append(out.Sample, transactionView(a, name, false))
	}
	if b.Missing > 0 {
		out.Warnings = append(out.Warnings, "完整范围声明仅由你确认，CSV 本身不能证明导出完整；缺失交易默认保留。")
	}
	if b.Missing > 0 && b.Added > 0 {
		out.Warnings = append(out.Warnings, "同时出现新增与缺失 ID。请核对钱迹 ID 是否稳定；系统不会按日期或金额自动合并。")
	}
	out.Warnings = append(out.Warnings, "手续费、优惠券、报销字段保留原值，不额外计入收支；转账不计入收入或支出。")
	return out, nil
}
func (s *FinanceService) Ledgers(ctx context.Context, user uint) ([]model.FinanceLedger, error) {
	return s.repo.Ledgers(ctx, user)
}
func (s *FinanceService) Batches(ctx context.Context, user, ledger uint) ([]model.FinanceBatch, error) {
	return s.repo.Batches(ctx, user, ledger)
}
func (s *FinanceService) Import(ctx context.Context, user, id uint, remove, confirm, stable bool) (model.FinanceBatch, error) {
	if remove && (!confirm || !stable) {
		return model.FinanceBatch{}, ErrFinanceInput
	}
	return s.repo.Import(ctx, user, id, remove)
}
func (s *FinanceService) Undo(ctx context.Context, user, id uint) (model.FinanceBatch, error) {
	return s.repo.Undo(ctx, user, id)
}

type FinanceFilter struct {
	From             string
	To               string
	Currency         string
	Category         string
	Subcategory      string
	CategoryEmpty    bool
	SubcategoryEmpty bool
	Tag              string
	Kind             string
	Keyword          string
	Page             int
	PageSize         int
}

func (f FinanceFilter) Validate() error {
	if (f.From != "" && !financeDate(f.From)) || (f.To != "" && !financeDate(f.To)) || (f.From != "" && f.To != "" && f.From > f.To) || len(f.Keyword) > 200 || len(f.Tag) > 65536 || f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 {
		return ErrFinanceInput
	}
	if f.Kind != "" && f.Kind != "income" && f.Kind != "expense" && f.Kind != "transfer" {
		return ErrFinanceInput
	}
	return nil
}
func financeMatches(a model.FinanceTransaction, f FinanceFilter) bool {
	d := a.OccurredAt[:10]
	if f.CategoryEmpty && a.Category != "" || f.SubcategoryEmpty && a.Subcategory != "" {
		return false
	}
	if f.From != "" && d < f.From || f.To != "" && d > f.To || f.Currency != "" && a.Currency != f.Currency || f.Category != "" && a.Category != f.Category || f.Subcategory != "" && a.Subcategory != f.Subcategory || f.Kind != "" && a.Kind != f.Kind || f.Keyword != "" && !strings.Contains(strings.ToLower(a.Note), strings.ToLower(f.Keyword)) {
		return false
	}
	if f.Tag != "" {
		var tags []string
		_ = json.Unmarshal([]byte(a.TagsJSON), &tags)
		found := false
		for _, t := range tags {
			if t == f.Tag {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

type FinancePage struct {
	Items    []FinanceTransactionView `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func (s *FinanceService) Transactions(ctx context.Context, user, ledger uint, f FinanceFilter) (FinancePage, error) {
	out := FinancePage{Items: []FinanceTransactionView{}, Page: f.Page, PageSize: f.PageSize}
	if e := f.Validate(); e != nil {
		return out, e
	}
	l, e := s.repo.Ledger(ctx, user, ledger)
	if e != nil {
		return out, e
	}
	rows, total, e := s.repo.Page(ctx, user, ledger, repository.FinanceQuery{From: f.From, To: f.To, Currency: f.Currency, Category: f.Category, Subcategory: f.Subcategory, Tag: f.Tag, Kind: f.Kind, Keyword: f.Keyword, Page: f.Page, PageSize: f.PageSize, CategoryEmpty: f.CategoryEmpty, SubcategoryEmpty: f.SubcategoryEmpty})
	if e != nil {
		return out, e
	}
	out.Total = total
	for _, a := range rows {
		out.Items = append(out.Items, transactionView(a, l.Name, false))
	}
	return out, nil
}
func (s *FinanceService) Transaction(ctx context.Context, user, ledger, id uint) (FinanceTransactionView, error) {
	l, e := s.repo.Ledger(ctx, user, ledger)
	if e != nil {
		return FinanceTransactionView{}, e
	}
	a, e := s.repo.Transaction(ctx, user, ledger, id)
	if e != nil {
		return FinanceTransactionView{}, e
	}
	return transactionView(a, l.Name, true), nil
}

type FinanceAmount struct {
	Name     string          `json:"name"`
	Amount   int64           `json:"amount_minor,string"`
	Count    int             `json:"count"`
	Children []FinanceAmount `json:"children,omitempty"`
}
type FinanceMonth struct {
	Month   string `json:"month"`
	Income  int64  `json:"income_minor,string"`
	Expense int64  `json:"expense_minor,string"`
	Count   int    `json:"count"`
}
type FinanceSummary struct {
	Count         int             `json:"count"`
	Income        int64           `json:"income_minor,string"`
	Expense       int64           `json:"expense_minor,string"`
	Difference    int64           `json:"difference_minor,string"`
	Currency      string          `json:"currency"`
	From          string          `json:"coverage_from"`
	To            string          `json:"coverage_to"`
	LastImport    *time.Time      `json:"last_import"`
	Years         []string        `json:"years"`
	Currencies    []string        `json:"currencies"`
	AllTags       []string        `json:"all_tags"`
	AllCategories []string        `json:"all_categories"`
	Months        []FinanceMonth  `json:"months"`
	Categories    []FinanceAmount `json:"categories"`
	Tags          []FinanceAmount `json:"tags"`
}

func checkedFinanceAdd(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		return 0, ErrFinanceInput
	}
	return a + b, nil
}
func stringKeys(m map[string]bool) []string {
	a := []string{}
	for k := range m {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}
func amountList(m map[string]*FinanceAmount) []FinanceAmount {
	a := []FinanceAmount{}
	for _, v := range m {
		a = append(a, *v)
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].Amount == a[j].Amount {
			return a[i].Name < a[j].Name
		}
		return a[i].Amount > a[j].Amount
	})
	return a
}
func (s *FinanceService) Summary(ctx context.Context, user, ledger uint, f FinanceFilter) (FinanceSummary, error) {
	out := FinanceSummary{Months: []FinanceMonth{}, Categories: []FinanceAmount{}, Tags: []FinanceAmount{}}
	if e := f.Validate(); e != nil {
		return out, e
	}
	rows, e := s.repo.Transactions(ctx, user, ledger)
	if e != nil {
		return out, e
	}
	years, currencies, tags, cats := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, a := range rows {
		years[a.OccurredAt[:4]] = true
		currencies[a.Currency] = true
		cats[a.Category] = true
		var ts []string
		_ = json.Unmarshal([]byte(a.TagsJSON), &ts)
		for _, t := range ts {
			tags[t] = true
		}
		d := a.OccurredAt[:10]
		if out.From == "" || d < out.From {
			out.From = d
		}
		if d > out.To {
			out.To = d
		}
	}
	out.Years = stringKeys(years)
	out.Currencies = stringKeys(currencies)
	out.AllTags = stringKeys(tags)
	out.AllCategories = stringKeys(cats)
	if f.Currency == "" {
		if len(out.Currencies) == 1 {
			f.Currency = out.Currencies[0]
		} else if len(out.Currencies) > 1 {
			return out, nil
		}
	}
	out.Currency = f.Currency
	batches, e := s.repo.Batches(ctx, user, ledger)
	if e != nil {
		return out, e
	}
	for _, b := range batches {
		if b.Status == "imported" {
			out.LastImport = b.ImportedAt
			break
		}
	}
	months := map[string]*FinanceMonth{}
	categories := map[string]*FinanceAmount{}
	tagAmounts := map[string]*FinanceAmount{}
	children := map[string]map[string]*FinanceAmount{}
	add := func(m map[string]*FinanceAmount, k string, v int64) error {
		if m[k] == nil {
			m[k] = &FinanceAmount{Name: k}
		}
		m[k].Count++
		n, e := checkedFinanceAdd(m[k].Amount, v)
		m[k].Amount = n
		return e
	}
	for _, a := range rows {
		if !financeMatches(a, f) {
			continue
		}
		out.Count++
		key := a.OccurredAt[:7]
		if months[key] == nil {
			months[key] = &FinanceMonth{Month: key}
		}
		m := months[key]
		m.Count++
		switch a.Kind {
		case "income":
			out.Income, e = checkedFinanceAdd(out.Income, a.AmountMinor)
			if e != nil {
				return out, e
			}
			m.Income, e = checkedFinanceAdd(m.Income, a.AmountMinor)
		case "expense":
			out.Expense, e = checkedFinanceAdd(out.Expense, a.AmountMinor)
			if e != nil {
				return out, e
			}
			m.Expense, e = checkedFinanceAdd(m.Expense, a.AmountMinor)
			if e != nil {
				return out, e
			}
			if e = add(categories, a.Category, a.AmountMinor); e != nil {
				return out, e
			}
			if children[a.Category] == nil {
				children[a.Category] = map[string]*FinanceAmount{}
			}
			if e = add(children[a.Category], a.Subcategory, a.AmountMinor); e != nil {
				return out, e
			}
			var ts []string
			_ = json.Unmarshal([]byte(a.TagsJSON), &ts)
			for _, t := range ts {
				if e = add(tagAmounts, t, a.AmountMinor); e != nil {
					return out, e
				}
			}
		}
		if e != nil {
			return out, e
		}
	}
	out.Difference, e = checkedFinanceAdd(out.Income, -out.Expense)
	if e != nil {
		return out, e
	}
	for k, v := range categories {
		v.Children = amountList(children[k])
	}
	out.Categories = amountList(categories)
	out.Tags = amountList(tagAmounts)
	for _, v := range months {
		out.Months = append(out.Months, *v)
	}
	sort.Slice(out.Months, func(i, j int) bool { return out.Months[i].Month < out.Months[j].Month })
	return out, nil
}

// MoneyExact formats minor units for exports and sample verification, without floats.
func MoneyExact(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return sign + strconv.FormatInt(n/100, 10) + "." + strconv.FormatInt(n%100+100, 10)[1:]
}
