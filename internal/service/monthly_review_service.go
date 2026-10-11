package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"learnos/internal/links"
	"learnos/internal/model"
	"learnos/internal/repository"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"
)

var ErrReviewInput = errors.New("invalid monthly review")

const ReviewTimezone = "Asia/Shanghai"

type ReviewItem struct {
	RelatedEventID   uint   `json:"related_event_id,omitempty"`
	RelatedAvailable bool   `json:"related_available,omitempty"`
	Key              string `json:"key"`
	Kind             string `json:"kind"`
	SourceID         uint   `json:"source_id"`
	ParentID         uint   `json:"parent_id"`
	Title            string `json:"title"`
	Date             string `json:"date"`
	Detail           string `json:"detail"`
	State            string `json:"state"`
	Domain           string `json:"domain"`
	Milestone        bool   `json:"milestone"`
	URL              string `json:"url"`
	Available        bool   `json:"available"`
}
type ReviewFinance struct {
	LedgerID  uint            `json:"ledger_id"`
	Name      string          `json:"name"`
	Currency  string          `json:"currency"`
	Summary   *FinanceSummary `json:"summary"`
	Available bool            `json:"available"`
}
type ReviewAsset struct {
	Snapshot  AssetView `json:"snapshot"`
	Available bool      `json:"available"`
}
type ReviewFacts struct {
	Month       string          `json:"month"`
	Timezone    string          `json:"timezone"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	GeneratedAt time.Time       `json:"generated_at"`
	InProgress  bool            `json:"in_progress"`
	Actions     []ReviewItem    `json:"actions"`
	Learning    []ReviewItem    `json:"learning"`
	Events      []ReviewItem    `json:"events"`
	Goals       []ReviewItem    `json:"goals"`
	Finance     []ReviewFinance `json:"finance"`
	Assets      []ReviewAsset   `json:"assets"`
	Limits      []string        `json:"limits"`
}
type ReviewDocument struct {
	Review           *model.MonthlyReview       `json:"review"`
	FactsID          uint                       `json:"facts_id"`
	Facts            ReviewFacts                `json:"facts"`
	History          []model.MonthlyReviewFacts `json:"history"`
	ArchivedPersonal *model.MonthlyReview       `json:"archived_personal,omitempty"`
}
type MonthlyReviewService struct {
	repo *repository.MonthlyReviewRepository
}

func NewMonthlyReviewService(r *repository.MonthlyReviewRepository) *MonthlyReviewService {
	return &MonthlyReviewService{r}
}
func reviewMonth(month string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation(ReviewTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	t, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil || t.Format("2006-01") != month || t.Year() < 1900 || t.Year() > 9998 {
		return t, t, ErrReviewInput
	}
	return t, t.AddDate(0, 1, 0), nil
}
func reviewDate(t time.Time) string {
	loc, _ := time.LoadLocation(ReviewTimezone)
	return t.In(loc).Format("2006-01-02 15:04")
}
func (s *MonthlyReviewService) List(ctx context.Context, user uint) ([]model.MonthlyReview, error) {
	return s.repo.List(ctx, user)
}
func (s *MonthlyReviewService) Generate(ctx context.Context, user uint, month string) (ReviewDocument, error) {
	start, end, err := reviewMonth(month)
	if err != nil {
		return ReviewDocument{}, err
	}
	facts := ReviewFacts{Month: month, Timezone: ReviewTimezone, From: start.Format("2006-01-02"), To: end.AddDate(0, 0, -1).Format("2006-01-02"), GeneratedAt: time.Now().UTC(), InProgress: time.Now().Before(end), Actions: []ReviewItem{}, Learning: []ReviewItem{}, Events: []ReviewItem{}, Goals: []ReviewItem{}, Finance: []ReviewFinance{}, Assets: []ReviewAsset{}, Limits: []string{"仅按实际记录回顾，不评价效率或生活价值。", "时间戳按 Asia/Shanghai 自然月左闭右开；生活事件、阶段、账目和资产使用原日历日期。", "升级前任务完成时间可能由更新时间推断，项目、课时和结业缺少完成日期；这些旧完成记录不归入月份。", "学习回答仅表示有学习活动，不代表掌握；探索问题按提出时间展示，不推断完成。", "账本覆盖和已结束月份不能证明记录完整；收支差额不代表净资产变化。"}}
	var base uint
	err = s.repo.Read(ctx, func(tx *gorm.DB) error {
		sources, err := repository.CollectReviewSources(tx, user, month, start, end)
		if err != nil {
			return err
		}
		base = sources.Version
		add := func(out *[]ReviewItem, kind string, id, parent uint, title, date, detail, state, url string) {
			*out = append(*out, ReviewItem{Key: fmt.Sprintf("%s:%d", kind, id), Kind: kind, SourceID: id, ParentID: parent, Title: title, Date: date, Detail: detail, State: state, URL: url, Available: true})
		}
		tasks := sources.Tasks
		for _, t := range tasks {
			add(&facts.Actions, "task", t.ID, t.ProjectID, t.Title, reviewDate(*t.ActualCompletedAt), "", "任务完成", fmt.Sprintf("/projects?task=%d", t.ID))
		}
		projects := sources.Projects
		completed := map[string]bool{}
		for _, p := range projects {
			add(&facts.Actions, "project", p.ID, 0, p.Title, reviewDate(*p.CompletedAt), "", "项目完成", fmt.Sprintf("/projects?project=%d&view=board", p.ID))
			completed[fmt.Sprintf("project:%d", p.ID)] = true
		}
		courses := sources.Courses
		for _, c := range courses {
			add(&facts.Learning, "course", c.ID, 0, c.Name, reviewDate(*c.CompletedAt), "", "正式结业", fmt.Sprintf("/courses/%d/archive", c.ID))
			completed[fmt.Sprintf("course:%d", c.ID)] = true
		}
		lessons := sources.Lessons
		lessonDone := map[uint]bool{}
		for _, l := range lessons {
			lessonDone[l.ID] = true
			add(&facts.Learning, "lesson", l.ID, l.CourseID, l.Title, reviewDate(*l.CompletedAt), "", "课时完成", fmt.Sprintf("/courses/%d/archive", l.CourseID))
		}
		turns := sources.Turns
		seen := map[uint]bool{}
		for _, t := range turns {
			if seen[t.LessonID] || lessonDone[t.LessonID] {
				continue
			}
			seen[t.LessonID] = true
			title, ok := sources.LessonTitles[t.LessonID]
			if !ok {
				continue
			}
			add(&facts.Learning, "learning_activity", t.ID, t.CourseID, title, reviewDate(t.CreatedAt), "按课时合并本月回答活动，不计算掌握率。", "学习活动", fmt.Sprintf("/courses/%d/archive", t.CourseID))
		}
		questions := sources.Questions
		for _, q := range questions {
			add(&facts.Learning, "exploration", q.ID, q.CourseID, q.Question, reviewDate(q.CreatedAt), "", "提出探索问题", "/explore")
		}
		events := sources.Events
		for _, e := range events {
			if e.SourceID != nil && completed[fmt.Sprintf("%s:%d", e.SourceType, *e.SourceID)] {
				for _, items := range [][]ReviewItem{facts.Actions, facts.Learning} {
					for i := range items {
						if items[i].Kind == e.SourceType && items[i].SourceID == *e.SourceID {
							items[i].RelatedEventID = e.ID
							items[i].RelatedAvailable = true
							items[i].Milestone = e.Milestone
							items[i].Domain = e.PrimaryDomain
							items[i].Detail = "生活收录（" + e.OccurredOn + "）：" + e.Title + "\n" + e.Description
						}
					}
				}
				continue
			}
			add(&facts.Events, "event", e.ID, 0, e.Title, e.OccurredOn, e.Description, "生活经历", fmt.Sprintf("/life?event=%d", e.ID))
			i := len(facts.Events) - 1
			facts.Events[i].Domain = e.PrimaryDomain
			facts.Events[i].Milestone = e.Milestone
		}
		entries := sources.Entries
		for _, e := range entries {

			state := "阶段记录"
			if e.ToStatus != "" {
				state = e.FromStatus + " → " + e.ToStatus
			}
			detail := e.Content
			if e.Reason != "" {
				detail += "\n原因：" + e.Reason
			}
			add(&facts.Goals, "goal_entry", e.ID, e.GoalID, sources.GoalTitles[e.GoalID], e.OccurredOn, detail, state, fmt.Sprintf("/life?view=goals&goal=%d", e.GoalID))
		}
		finance := NewFinanceService(repository.NewFinanceRepository(tx))
		ledgers, err := finance.Ledgers(ctx, user)
		if err != nil {
			return err
		}
		for _, l := range ledgers {
			f := FinanceFilter{From: facts.From, To: facts.To, Page: 1, PageSize: 1}
			metadata, err := finance.Summary(ctx, user, l.ID, f)
			if err != nil {
				return err
			}
			if len(metadata.Currencies) == 0 {
				facts.Finance = append(facts.Finance, ReviewFinance{LedgerID: l.ID, Name: l.Name, Available: true})
			}
			for _, currency := range metadata.Currencies {
				f.Currency = currency
				summary, err := finance.Summary(ctx, user, l.ID, f)
				if err != nil {
					return err
				}
				row := ReviewFinance{LedgerID: l.ID, Name: l.Name, Currency: currency, Available: true}
				if summary.Count > 0 {
					summary.Years = nil
					summary.Currencies = nil
					summary.AllTags = nil
					summary.AllCategories = nil
					row.Summary = &summary
				}
				facts.Finance = append(facts.Finance, row)
			}
		}
		assets, err := NewAssetService(repository.NewAssetRepository(tx)).List(ctx, user)
		if err != nil {
			return err
		}
		for _, a := range assets {
			if a.SnapshotDate >= facts.From && a.SnapshotDate < end.Format("2006-01-02") {
				facts.Assets = append(facts.Assets, ReviewAsset{Snapshot: a, Available: true})
			}
		}
		return nil
	})
	if err != nil {
		return ReviewDocument{}, err
	}
	for _, items := range [][]ReviewItem{facts.Actions, facts.Learning} {
		sort.SliceStable(items, func(i, j int) bool { return items[i].Date > items[j].Date })
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		return ReviewDocument{}, err
	}
	draft, err := s.repo.Draft(ctx, model.MonthlyReviewFacts{UserID: user, Month: month, BaseVersion: base, FactsJSON: string(encoded), GeneratedAt: facts.GeneratedAt})
	return ReviewDocument{FactsID: draft.ID, Facts: facts}, err
}
func (s *MonthlyReviewService) snapshot(ctx context.Context, user, id uint) (ReviewDocument, error) {
	row, err := s.repo.Facts(ctx, user, id)
	if err != nil {
		return ReviewDocument{}, err
	}
	doc := ReviewDocument{FactsID: row.ID}
	if err = json.Unmarshal([]byte(row.FactsJSON), &doc.Facts); err != nil {
		return doc, err
	}
	if row.ArchivedPersonalJSON != "" {
		var personal model.MonthlyReview
		if err = json.Unmarshal([]byte(row.ArchivedPersonalJSON), &personal); err != nil {
			return doc, err
		}
		doc.ArchivedPersonal = &personal
	}
	err = s.repo.Read(ctx, func(tx *gorm.DB) error {
		for _, items := range [][]ReviewItem{doc.Facts.Actions, doc.Facts.Learning, doc.Facts.Events, doc.Facts.Goals} {
			for i := range items {
				ok, err := repository.ReviewSourceAvailable(tx, user, items[i].Kind, items[i].SourceID)
				if err != nil {
					return err
				}
				items[i].Available = ok
				if items[i].RelatedEventID > 0 {
					available, err := repository.ReviewSourceAvailable(tx, user, "event", items[i].RelatedEventID)
					if err != nil {
						return err
					}
					items[i].RelatedAvailable = available
				}
			}
		}
		for i := range doc.Facts.Finance {
			ok, err := repository.ReviewSourceAvailable(tx, user, "ledger", doc.Facts.Finance[i].LedgerID)
			if err != nil {
				return err
			}
			doc.Facts.Finance[i].Available = ok
		}
		for i := range doc.Facts.Assets {
			ok, err := repository.ReviewSourceAvailable(tx, user, "asset", doc.Facts.Assets[i].Snapshot.ID)
			if err != nil {
				return err
			}
			doc.Facts.Assets[i].Available = ok
		}
		return nil
	})
	return doc, err
}
func (s *MonthlyReviewService) Get(ctx context.Context, user uint, month string) (ReviewDocument, error) {
	if _, _, err := reviewMonth(month); err != nil {
		return ReviewDocument{}, err
	}
	row, err := s.repo.Get(ctx, user, month)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ReviewDocument{}, nil
	}
	if err != nil {
		return ReviewDocument{}, err
	}
	doc, err := s.snapshot(ctx, user, row.FactsID)
	if err != nil {
		return doc, err
	}
	doc.Review = &row
	doc.History, err = s.repo.History(ctx, user, month)
	return doc, err
}
func (s *MonthlyReviewService) HistorySnapshot(ctx context.Context, user uint, month string, id uint) (ReviewDocument, error) {
	row, err := s.repo.Facts(ctx, user, id)
	if err != nil {
		return ReviewDocument{}, err
	}
	if row.Month != month || row.ConfirmedAt == nil {
		return ReviewDocument{}, gorm.ErrRecordNotFound
	}
	return s.snapshot(ctx, user, id)
}

type ReviewSaveInput struct {
	FactsID        uint   `json:"facts_id"`
	Version        uint   `json:"version"`
	Memorable      string `json:"memorable"`
	Thoughts       string `json:"thoughts"`
	NextFocus      string `json:"next_focus"`
	ExternalURL    string `json:"external_url"`
	ConfirmReplace bool   `json:"confirm_replace"`
}

func (s *MonthlyReviewService) Save(ctx context.Context, user uint, month string, in ReviewSaveInput) (ReviewDocument, error) {
	if _, _, err := reviewMonth(month); err != nil {
		return ReviewDocument{}, err
	}
	if in.FactsID == 0 || !links.SafeExternal(in.ExternalURL) || len(in.ExternalURL) > 2048 {
		return ReviewDocument{}, ErrReviewInput
	}
	for _, v := range []string{in.Memorable, in.Thoughts, in.NextFocus} {
		if !utf8.ValidString(v) || len([]rune(v)) > 10000 || strings.ContainsRune(v, 0) {
			return ReviewDocument{}, ErrReviewInput
		}
	}
	old, err := s.repo.Get(ctx, user, month)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ReviewDocument{}, err
	}
	if err == nil && old.FactsID != in.FactsID && !in.ConfirmReplace {
		return ReviewDocument{}, ErrReviewInput
	}
	_, err = s.repo.Save(ctx, model.MonthlyReview{UserID: user, Month: month, FactsID: in.FactsID, Version: in.Version, Memorable: in.Memorable, Thoughts: in.Thoughts, NextFocus: in.NextFocus, ExternalURL: strings.TrimSpace(in.ExternalURL)})
	if err != nil {
		return ReviewDocument{}, err
	}
	return s.Get(ctx, user, month)
}
