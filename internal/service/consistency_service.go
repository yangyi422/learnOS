package service

import (
	"context"
	"fmt"

	"learnos/internal/model"

	"gorm.io/gorm"
)

type ConsistencyService struct{ db *gorm.DB }

func NewConsistencyService(db *gorm.DB) *ConsistencyService { return &ConsistencyService{db: db} }

type ConsistencyIssue struct {
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	Code       string `json:"code"`
	Entity     string `json:"entity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

type ConsistencyReport struct {
	Healthy  bool               `json:"healthy"`
	Warnings []ConsistencyIssue `json:"warnings"`
	Errors   []ConsistencyIssue `json:"errors"`
}

func (s *ConsistencyService) Check(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{Healthy: true, Warnings: []ConsistencyIssue{}, Errors: []ConsistencyIssue{}}
	addError := func(code, entity, message string) {
		report.Healthy = false
		category, suggestion := consistencyIssueHelp(code)
		report.Errors = append(report.Errors, ConsistencyIssue{Severity: "error", Category: category, Code: code, Entity: entity, Message: message, Suggestion: suggestion})
	}
	var courses []model.Course
	if err := s.db.WithContext(ctx).Find(&courses).Error; err != nil {
		return nil, err
	}
	for _, course := range courses {
		var lessons []model.Lesson
		var relations []model.LessonRelation
		var units []model.CourseUnit
		if err := s.db.WithContext(ctx).Where("course_id = ?", course.ID).Find(&units).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("course_id = ?", course.ID).Find(&lessons).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("course_id = ?", course.ID).Find(&relations).Error; err != nil {
			return nil, err
		}
		if course.CurrentLessonID != nil {
			var lesson model.Lesson
			if err := s.db.WithContext(ctx).Where("id = ? AND course_id = ?", *course.CurrentLessonID, course.ID).First(&lesson).Error; err != nil {
				addError("COURSE_CURRENT_LESSON_MISSING", fmt.Sprintf("course:%d", course.ID), "CurrentLesson 不属于该 Course")
			}
		}
		lessonIDs := map[uint]bool{}
		unitIDs := map[uint]bool{}
		for _, unit := range units {
			unitIDs[unit.ID] = true
		}
		for _, lesson := range lessons {
			lessonIDs[lesson.ID] = true
			if !unitIDs[lesson.UnitID] {
				addError("LESSON_UNIT_MISMATCH", fmt.Sprintf("lesson:%d", lesson.ID), "Lesson 所属区域不存在或不属于该 Course")
			}
			if !lesson.ContentRole.Valid() {
				addError("LESSON_CONTENT_ROLE_INVALID", fmt.Sprintf("lesson:%d", lesson.ID), "Lesson 节点类型不在统一枚举中")
			}
		}
		if course.CurrentUnitID != nil && !unitIDs[*course.CurrentUnitID] {
			addError("COURSE_CURRENT_UNIT_MISSING", fmt.Sprintf("course:%d", course.ID), "CurrentUnit 不属于该 Course")
		}
		for _, relation := range relations {
			if !lessonIDs[relation.FromLessonID] || !lessonIDs[relation.ToLessonID] {
				addError("LESSON_RELATION_ENDPOINT_MISSING", fmt.Sprintf("relation:%d", relation.ID), "LessonRelation 端点不存在或不属于该 Course")
			}
		}
		if err := ValidateCourseGraph(course.ID, lessons, relations); err != nil {
			addError("KNOWLEDGE_GRAPH_INVALID", fmt.Sprintf("course:%d", course.ID), err.Error())
		}
	}

	var states []model.CognitiveState
	if err := s.db.WithContext(ctx).Find(&states).Error; err != nil {
		return nil, err
	}
	for _, state := range states {
		var lesson model.Lesson
		if err := s.db.WithContext(ctx).First(&lesson, state.LessonID).Error; err != nil || lesson.CourseID != state.CourseID {
			addError("COGNITIVE_LESSON_MISMATCH", fmt.Sprintf("cognitive_state:%d", state.ID), "CognitiveState 的 Course/Lesson 归属不一致")
		}
		if state.LastLearningTurnID != nil {
			var turn model.LearningTurn
			if err := s.db.WithContext(ctx).First(&turn, *state.LastLearningTurnID).Error; err != nil {
				addError("COGNITIVE_TURN_MISSING", fmt.Sprintf("cognitive_state:%d", state.ID), "LastLearningTurnID 不存在")
			} else if turn.CourseID != state.CourseID || turn.LessonID != state.LessonID {
				addError("COGNITIVE_TURN_MISMATCH", fmt.Sprintf("cognitive_state:%d", state.ID), "LastLearningTurn 与认知状态的 Course/Lesson 不一致")
			}
		}
	}
	var evidence []model.CognitiveEvidence
	if err := s.db.WithContext(ctx).Find(&evidence).Error; err != nil {
		return nil, err
	}
	for _, item := range evidence {
		var turn model.LearningTurn
		if err := s.db.WithContext(ctx).First(&turn, item.LearningTurnID).Error; err != nil {
			addError("COGNITIVE_EVIDENCE_TURN_MISSING", fmt.Sprintf("evidence:%d", item.ID), "CognitiveEvidence 对应 LearningTurn 不存在")
		} else if turn.CourseID != item.CourseID || turn.LessonID != item.LessonID {
			addError("COGNITIVE_EVIDENCE_SCOPE_MISMATCH", fmt.Sprintf("evidence:%d", item.ID), "CognitiveEvidence 与 LearningTurn 的 Course/Lesson 不一致")
		}
	}
	var masteryRecords []model.MasteryRecord
	if err := s.db.WithContext(ctx).Find(&masteryRecords).Error; err != nil {
		return nil, err
	}
	for _, item := range masteryRecords {
		var lesson model.Lesson
		if err := s.db.WithContext(ctx).First(&lesson, item.LessonID).Error; err != nil {
			addError("MASTERY_LESSON_MISSING", fmt.Sprintf("mastery:%d", item.ID), "MasteryRecord 对应 Lesson 不存在")
		} else if lesson.CourseID != item.CourseID {
			addError("MASTERY_COURSE_MISMATCH", fmt.Sprintf("mastery:%d", item.ID), "MasteryRecord 与 Lesson 的 Course 归属不一致")
		}
		if item.MasteryScore < 0 || item.MasteryScore > 1 {
			addError("MASTERY_SCORE_INVALID", fmt.Sprintf("mastery:%d", item.ID), "MasteryScore 必须介于 0 与 1 之间")
		}
	}
	var misconceptions []model.Misconception
	if err := s.db.WithContext(ctx).Find(&misconceptions).Error; err != nil {
		return nil, err
	}
	for _, item := range misconceptions {
		if !existsByID(ctx, s.db, &model.Lesson{}, item.LessonID) {
			addError("MISCONCEPTION_LESSON_MISSING", fmt.Sprintf("misconception:%d", item.ID), "Misconception 对应 Lesson 不存在")
		}
	}
	var attempts []model.ChallengeAttempt
	if err := s.db.WithContext(ctx).Find(&attempts).Error; err != nil {
		return nil, err
	}
	for _, item := range attempts {
		if !existsByID(ctx, s.db, &model.AssessmentChallenge{}, item.ChallengeID) {
			addError("CHALLENGE_MISSING", fmt.Sprintf("attempt:%d", item.ID), "ChallengeAttempt 对应 Challenge 不存在")
		}
	}
	var directions []model.ExplorationDirection
	if err := s.db.WithContext(ctx).Find(&directions).Error; err != nil {
		return nil, err
	}
	for _, item := range directions {
		if !existsByID(ctx, s.db, &model.Lesson{}, item.TargetLessonID) {
			addError("EXPLORATION_TARGET_MISSING", fmt.Sprintf("direction:%d", item.ID), "探索方向目标 Lesson 不存在")
		}
	}
	var questions []model.ExplorationQuestion
	if err := s.db.WithContext(ctx).Find(&questions).Error; err != nil {
		return nil, err
	}
	for _, item := range questions {
		if item.SourceDirectionID != nil && !existsByID(ctx, s.db, &model.ExplorationDirection{}, *item.SourceDirectionID) {
			addError("EXPLORATION_DIRECTION_MISSING", fmt.Sprintf("question:%d", item.ID), "探索问题来源方向不存在")
		}
	}
	var blueprintLessons []model.CurriculumBlueprintLesson
	if err := s.db.WithContext(ctx).Find(&blueprintLessons).Error; err != nil {
		return nil, err
	}
	for _, item := range blueprintLessons {
		if item.AppliedLessonID != nil && !existsByID(ctx, s.db, &model.Lesson{}, *item.AppliedLessonID) {
			addError("BLUEPRINT_LESSON_MAPPING_MISSING", fmt.Sprintf("blueprint_lesson:%d", item.ID), "BlueprintLesson 映射的正式 Lesson 不存在")
		}
	}
	var links []model.GroundingLink
	if err := s.db.WithContext(ctx).Find(&links).Error; err != nil {
		return nil, err
	}
	for _, item := range links {
		var evidenceItem model.SourceEvidence
		if err := s.db.WithContext(ctx).First(&evidenceItem, item.EvidenceID).Error; err != nil {
			addError("GROUNDING_EVIDENCE_MISSING", fmt.Sprintf("link:%d", item.ID), "GroundingLink 对应 Evidence 不存在")
			continue
		}
		if !existsByID(ctx, s.db, &model.KnowledgeSource{}, evidenceItem.SourceID) {
			addError("GROUNDING_SOURCE_MISSING", fmt.Sprintf("link:%d", item.ID), "Evidence 对应 Source 不存在")
		}
	}
	return report, nil
}

func consistencyIssueHelp(code string) (string, string) {
	switch code {
	case "COURSE_CURRENT_LESSON_MISSING", "COURSE_CURRENT_UNIT_MISSING", "LESSON_UNIT_MISMATCH", "LESSON_CONTENT_ROLE_INVALID", "LESSON_RELATION_ENDPOINT_MISSING", "KNOWLEDGE_GRAPH_INVALID", "BLUEPRINT_LESSON_MAPPING_MISSING":
		return "课程结构", "先创建备份，再检查课程当前节点、区域归属和知识关系；不要直接删除历史记录。"
	case "COGNITIVE_LESSON_MISMATCH", "COGNITIVE_TURN_MISSING", "COGNITIVE_TURN_MISMATCH", "COGNITIVE_EVIDENCE_TURN_MISSING", "COGNITIVE_EVIDENCE_SCOPE_MISMATCH", "MASTERY_LESSON_MISSING", "MASTERY_COURSE_MISMATCH", "MASTERY_SCORE_INVALID":
		return "学习证据", "先创建备份，再核对学习记录与认知证据的关联；建议通过修复脚本补齐引用。"
	case "MISCONCEPTION_LESSON_MISSING", "CHALLENGE_MISSING":
		return "学习历史", "保留原始记录并创建备份，再检查关联的 Lesson 或迁移挑战是否缺失。"
	case "EXPLORATION_TARGET_MISSING", "EXPLORATION_DIRECTION_MISSING":
		return "探索数据", "创建备份后重新生成对应探索方向；不要把旧问题错误归属到当前领域。"
	case "GROUNDING_EVIDENCE_MISSING", "GROUNDING_SOURCE_MISSING":
		return "来源与证据", "检查知识来源和证据条目的关联，确认来源后再重建连接。"
	default:
		return "其他", "创建备份并记录问题代码，再进行有针对性的修复。"
	}
}

func existsByID(ctx context.Context, db *gorm.DB, value interface{}, id uint) bool {
	return db.WithContext(ctx).First(value, id).Error == nil
}
