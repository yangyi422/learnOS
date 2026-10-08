package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type ExportService struct{ db *gorm.DB }

func NewExportService(db *gorm.DB) *ExportService { return &ExportService{db: db} }

type ExportSnapshot struct {
	LifeEvents            []model.LifeEvent                   `json:"life_events"`
	LifeGoals             []model.LifeGoal                    `json:"life_goals"`
	LifeGoalEntries       []model.LifeGoalEntry               `json:"life_goal_entries"`
	ExportVersion         string                              `json:"export_version"`
	ExportedAt            time.Time                           `json:"exported_at"`
	Courses               []model.Course                      `json:"courses"`
	Units                 []model.CourseUnit                  `json:"course_units"`
	Lessons               []model.Lesson                      `json:"lessons"`
	Relations             []model.LessonRelation              `json:"lesson_relations"`
	LearningTurns         []model.LearningTurn                `json:"learning_turns"`
	Mastery               []model.MasteryRecord               `json:"mastery_records"`
	CognitiveStates       []model.CognitiveState              `json:"cognitive_states"`
	CognitiveEvidence     []model.CognitiveEvidence           `json:"cognitive_evidence"`
	CognitiveEvents       []model.CognitiveStateEvent         `json:"cognitive_state_events"`
	Misconceptions        []model.Misconception               `json:"misconceptions"`
	MisconceptionEvents   []model.MisconceptionEvent          `json:"misconception_events"`
	Challenges            []model.AssessmentChallenge         `json:"assessment_challenges"`
	ChallengeAttempts     []model.ChallengeAttempt            `json:"challenge_attempts"`
	ExplorationDirections []model.ExplorationDirection        `json:"exploration_directions"`
	ExplorationQuestions  []model.ExplorationQuestion         `json:"exploration_questions"`
	Blueprints            []model.CurriculumBlueprint         `json:"curriculum_blueprints"`
	BlueprintUnits        []model.CurriculumBlueprintUnit     `json:"curriculum_blueprint_units"`
	BlueprintLessons      []model.CurriculumBlueprintLesson   `json:"curriculum_blueprint_lessons"`
	BlueprintRelations    []model.CurriculumBlueprintRelation `json:"curriculum_blueprint_relations"`
	Sources               []model.KnowledgeSource             `json:"sources"`
	Evidence              []model.SourceEvidence              `json:"source_evidence"`
	GroundingLinks        []model.GroundingLink               `json:"grounding_links"`
	Credibility           []model.SourceCredibilityAssessment `json:"source_credibility_assessments"`
	DomainDrafts          []model.DomainInitializationDraft   `json:"domain_initialization_drafts"`
	Projects              []model.Project                     `json:"projects"`
	ProjectTasks          []model.ProjectTask                 `json:"project_tasks"`
	Records               []model.LightweightRecord           `json:"records"`
	InboxItems            []model.InboxItem                   `json:"inbox_items"`
}

func (s *ExportService) Snapshot(ctx context.Context) (*ExportSnapshot, error) {
	snapshot := &ExportSnapshot{ExportVersion: "1", ExportedAt: time.Now().UTC()}
	queries := []struct {
		dest interface{}
		name string
	}{
		{&snapshot.LifeEvents, "life events"}, {&snapshot.LifeGoals, "life goals"}, {&snapshot.LifeGoalEntries, "life goal entries"},
		{&snapshot.Courses, "courses"}, {&snapshot.Units, "course units"}, {&snapshot.Lessons, "lessons"}, {&snapshot.Relations, "lesson relations"},
		{&snapshot.LearningTurns, "learning turns"}, {&snapshot.Mastery, "mastery records"}, {&snapshot.CognitiveStates, "cognitive states"},
		{&snapshot.CognitiveEvidence, "cognitive evidence"}, {&snapshot.CognitiveEvents, "cognitive events"}, {&snapshot.Misconceptions, "misconceptions"},
		{&snapshot.MisconceptionEvents, "misconception events"}, {&snapshot.Challenges, "challenges"}, {&snapshot.ChallengeAttempts, "challenge attempts"},
		{&snapshot.ExplorationDirections, "exploration directions"}, {&snapshot.ExplorationQuestions, "exploration questions"}, {&snapshot.Blueprints, "blueprints"},
		{&snapshot.BlueprintUnits, "blueprint units"}, {&snapshot.BlueprintLessons, "blueprint lessons"}, {&snapshot.BlueprintRelations, "blueprint relations"},
		{&snapshot.Sources, "sources"}, {&snapshot.Evidence, "source evidence"}, {&snapshot.GroundingLinks, "grounding links"}, {&snapshot.Credibility, "credibility"}, {&snapshot.DomainDrafts, "domain initialization drafts"},
		{&snapshot.Projects, "projects"}, {&snapshot.ProjectTasks, "project tasks"}, {&snapshot.InboxItems, "inbox items"}, {&snapshot.Records, "lightweight records"},
	}
	for _, query := range queries {
		if err := s.db.WithContext(ctx).Find(query.dest).Error; err != nil {
			return nil, fmt.Errorf("export %s: %w", query.name, err)
		}
	}
	return snapshot, nil
}

func (s *ExportService) JSON(ctx context.Context) ([]byte, error) {
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(snapshot, "", "  ")
}

func (s *ExportService) Markdown(ctx context.Context) ([]byte, error) {
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("# LearnOS 数据导出\n\n")
	fmt.Fprintf(&out, "导出时间：%s\n\n", snapshot.ExportedAt.Format(time.RFC3339))
	out.WriteString("本文件由 LearnOS 生成，记录生活档案、长期目标及历史、收集箱、项目任务、课程结构、学习记录、认知状态、误区、挑战、探索问题和来源关联。\n\n")
	out.WriteString("## 收集箱\n\n")
	for _, item := range snapshot.InboxItems {
		fmt.Fprintf(&out, "- %s（%s）\n", item.Content, item.Status)
	}
	out.WriteString("\n")
	out.WriteString("## 生活档案\n\n")
	for _, event := range snapshot.LifeEvents {
		fmt.Fprintf(&out, "### %s · %s\n\n%s\n\n- 事件 ID：%d\n- 里程碑：%t\n- 领域：%s / %s\n- 创建：%s\n- 修改：%s\n", event.OccurredOn, event.Title, event.Description, event.ID, event.Milestone, event.PrimaryDomain, event.SecondaryDomain, event.CreatedAt.Format(time.RFC3339), event.UpdatedAt.Format(time.RFC3339))
		if event.SourceID != nil {
			fmt.Fprintf(&out, "- 来源：%s / %d / %s\n", event.SourceType, *event.SourceID, event.SourceTitle)
		}
		if event.GoalID != nil {
			fmt.Fprintf(&out, "- 目标 ID：%d\n", *event.GoalID)
		}
		if event.ExternalURL != "" {
			fmt.Fprintf(&out, "- 外部链接：%s %s\n", event.LinkName, event.ExternalURL)
		}
		out.WriteString("\n")
	}
	for _, goal := range snapshot.LifeGoals {
		fmt.Fprintf(&out, "### 长期目标：%s\n\n%s\n\n%s\n\n- 目标 ID：%d\n- 状态：%s\n- 领域：%s\n- 创建：%s\n- 修改：%s\n", goal.Title, goal.Why, goal.CurrentNote, goal.ID, goal.Status, goal.Domain, goal.CreatedAt.Format(time.RFC3339), goal.UpdatedAt.Format(time.RFC3339))
		if goal.ProjectID != nil {
			fmt.Fprintf(&out, "- 项目 ID：%d\n", *goal.ProjectID)
		}
		if goal.ExternalURL != "" {
			fmt.Fprintf(&out, "- 外部链接：%s %s\n", goal.LinkName, goal.ExternalURL)
		}
		for _, entry := range snapshot.LifeGoalEntries {
			if entry.GoalID == goal.ID {
				fmt.Fprintf(&out, "- %s：%s", entry.OccurredOn, entry.Content)
				if entry.ToStatus != "" {
					fmt.Fprintf(&out, "（%s → %s）", entry.FromStatus, entry.ToStatus)
				}
				fmt.Fprintf(&out, " %s\n", entry.Reason)
			}
		}
		out.WriteString("\n")
	}
	out.WriteString("## 轻量记录\n\n")
	for _, record := range snapshot.Records {
		fmt.Fprintf(&out, "### 记录 %d\n\n%s\n\n- 创建：%s\n- 修改：%s\n", record.ID, record.Content, record.CreatedAt.Format(time.RFC3339), record.UpdatedAt.Format(time.RFC3339))
		if record.ProjectID != nil {
			fmt.Fprintf(&out, "- 项目 ID：%d\n", *record.ProjectID)
		}
		if record.ExternalURL != "" {
			fmt.Fprintf(&out, "- 外部链接：%s\n", record.ExternalURL)
		}
		if record.ArchivedAt != nil {
			fmt.Fprintf(&out, "- 已归档：%s\n", record.ArchivedAt.Format(time.RFC3339))
		}
		out.WriteString("\n")
	}
	out.WriteString("## 项目看板\n\n")
	for _, project := range snapshot.Projects {
		fmt.Fprintf(&out, "### %s\n\n- 状态：%s\n- 目标：%s\n\n", project.Title, project.Status, project.Description)
		for _, task := range snapshot.ProjectTasks {
			if task.ProjectID == project.ID {
				fmt.Fprintf(&out, "- %s（%s · %s", task.Title, task.Status, task.Priority)
				if task.DueDate != nil {
					fmt.Fprintf(&out, " · 到期 %s", *task.DueDate)
				}
				out.WriteString("）\n")
			}
		}
		out.WriteString("\n")
	}
	out.WriteString("## 课程概览\n\n")
	for _, course := range snapshot.Courses {
		fmt.Fprintf(&out, "### %s\n\n- 状态：%s\n- 进度：%d%%\n- 目标：%s\n\n", course.Name, course.Status, course.Progress, course.Goal)
		out.WriteString("#### Lesson\n\n")
		for _, lesson := range snapshot.Lessons {
			if lesson.CourseID != course.ID {
				continue
			}
			fmt.Fprintf(&out, "- **%s**（ID %d）\n  - 核心问题：%s\n  - 状态：%s\n  - 认知校验：%s\n\n", lesson.Title, lesson.ID, lesson.CoreQuestion, lesson.Status, lesson.GroundingStatus)
		}
	}
	out.WriteString("## 学习记录\n\n")
	for _, turn := range snapshot.LearningTurns {
		fmt.Fprintf(&out, "### %s\n\n- Lesson ID：%d\n- 结果：%s\n- 回答：%s\n- 反馈：%s\n- 时间：%s\n\n", turn.Question, turn.LessonID, turn.Result, turn.UserAnswer, turn.Feedback, turn.CreatedAt.Format(time.RFC3339))
	}
	out.WriteString("## 认知历史\n\n")
	for _, event := range snapshot.CognitiveEvents {
		fmt.Fprintf(&out, "- Lesson %d：%s → %s，%s（%s）\n", event.LessonID, event.FromLevel, event.ToLevel, event.Reason, event.CreatedAt.Format(time.RFC3339))
	}
	out.WriteString("\n## 误区\n\n")
	for _, item := range snapshot.Misconceptions {
		fmt.Fprintf(&out, "- **%s** → %s（状态：%s）\n", item.OriginalUnderstanding, item.CorrectUnderstanding, item.Status)
	}
	out.WriteString("\n## 探索问题\n\n")
	for _, item := range snapshot.ExplorationQuestions {
		fmt.Fprintf(&out, "- %s（%s）\n  - %s\n", item.Question, item.Status, item.WhyThisQuestion)
	}
	out.WriteString("\n## 来源与 Grounding\n\n")
	for _, source := range snapshot.Sources {
		fmt.Fprintf(&out, "- %s（%s）\n", source.Title, source.SourceType)
	}
	out.WriteString("\n## 领域初始化草案\n\n")
	for _, draft := range snapshot.DomainDrafts {
		fmt.Fprintf(&out, "- **%s**（状态：%s，ID：%d）\n", draft.DomainName, draft.Status, draft.ID)
	}
	return []byte(out.String()), nil
}
