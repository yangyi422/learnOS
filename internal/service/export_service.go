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
}

func (s *ExportService) Snapshot(ctx context.Context) (*ExportSnapshot, error) {
	snapshot := &ExportSnapshot{ExportVersion: "1", ExportedAt: time.Now().UTC()}
	queries := []struct {
		dest interface{}
		name string
	}{
		{&snapshot.Courses, "courses"}, {&snapshot.Units, "course units"}, {&snapshot.Lessons, "lessons"}, {&snapshot.Relations, "lesson relations"},
		{&snapshot.LearningTurns, "learning turns"}, {&snapshot.Mastery, "mastery records"}, {&snapshot.CognitiveStates, "cognitive states"},
		{&snapshot.CognitiveEvidence, "cognitive evidence"}, {&snapshot.CognitiveEvents, "cognitive events"}, {&snapshot.Misconceptions, "misconceptions"},
		{&snapshot.MisconceptionEvents, "misconception events"}, {&snapshot.Challenges, "challenges"}, {&snapshot.ChallengeAttempts, "challenge attempts"},
		{&snapshot.ExplorationDirections, "exploration directions"}, {&snapshot.ExplorationQuestions, "exploration questions"}, {&snapshot.Blueprints, "blueprints"},
		{&snapshot.BlueprintUnits, "blueprint units"}, {&snapshot.BlueprintLessons, "blueprint lessons"}, {&snapshot.BlueprintRelations, "blueprint relations"},
		{&snapshot.Sources, "sources"}, {&snapshot.Evidence, "source evidence"}, {&snapshot.GroundingLinks, "grounding links"}, {&snapshot.Credibility, "credibility"}, {&snapshot.DomainDrafts, "domain initialization drafts"},
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
	out.WriteString("# LearnOS 学习数据导出\n\n")
	fmt.Fprintf(&out, "导出时间：%s\n\n", snapshot.ExportedAt.Format(time.RFC3339))
	out.WriteString("本文件由 LearnOS 生成，记录课程结构、学习记录、认知状态、误区、挑战、探索问题和来源关联。\n\n")
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
